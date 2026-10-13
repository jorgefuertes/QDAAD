package legacy

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// processes reads the /PRO sections, which are the program of the adventure.
//
// There is one section per process and they run to /END. A process holds
// entries, an entry is a verb and a noun and the condacts that run when the
// player's sentence matches them.
func (p *parser) processes() error {
	for p.atSection(SecPRO) {
		p.next()

		number, err := p.byteValue(p.next(), "the process number")
		if err != nil {
			return err
		}

		entries, err := p.entries()
		if err != nil {
			return p.errorf("process %d: %w", number, err)
		}

		if err := p.db.Processes().NewLegacy(ddb.Process{ID: number, Entries: entries}); err != nil {
			return p.errorf("process %d: %w", number, err)
		}
	}

	return p.expectSection(SecEND)
}

// entries reads the entries of one process, up to the next /PRO or /END.
//
// Several "> verb noun" lines in a row share the block that follows them. The
// block is read once and given to each, which is how a source says "these two
// sentences do the same thing" without writing it twice.
func (p *parser) entries() ([]ddb.Entry, error) {
	var out []ddb.Entry

	for !p.atSection(SecPRO) && !p.atSection(SecEND) && p.peek().Kind != EOF {
		var heads []ddb.Entry

		for p.peek().Kind == Entry {
			p.next()

			verb, noun, err := p.verbNoun()
			if err != nil {
				return nil, err
			}

			heads = append(heads, ddb.Entry{Verb: verb, Noun: noun})
		}

		if len(heads) == 0 {
			return nil, p.errorf("an entry sign > expected, but %s is there", p.peek().Describe())
		}

		block, err := p.condacts()
		if err != nil {
			return nil, err
		}

		for i := range heads {
			heads[i].Condacts = block
		}

		out = append(out, heads...)
	}

	return out, nil
}

// verbNoun reads the two words that open an entry.
//
// The first slot takes a verb, or a noun low enough to be used as one; the
// second takes a noun. Both are named by a word and not by a value: the
// compiler of reference looks the text up in the vocabulary and never reads it
// as a number. A bare number is accepted here as well — see the plan, §2.3 —
// for the value that has no word of the kind the slot wants.
func (p *parser) verbNoun() (verb, noun ddb.ID, err error) {
	if verb, err = p.entryWord(ddb.Verb, true); err != nil {
		return 0, 0, err
	}

	noun, err = p.entryWord(ddb.Noun, false)

	return verb, noun, err
}

func (p *parser) entryWord(kind ddb.WordKind, convertible bool) (ddb.ID, error) {
	t := p.next()
	if t.Kind == Wildcard {
		return ddb.NO_WORD_ID, nil
	}

	name := t.Text
	if t.Kind == Number {
		name = t.Describe()
	}

	if w, err := p.db.Words().GetByTypeAndSynonym(kind, name); err == nil {
		return w.ID, nil
	}

	// A noun of low enough value doubles as a verb, which is what lets "NORTE"
	// be typed on its own.
	if convertible {
		if w, err := p.db.Words().GetByTypeAndSynonym(ddb.Noun, name); err == nil &&
			w.ID <= ddb.MAX_CONVERTIBLE_NAME {
			return w.ID, nil
		}
	}

	if t.Kind == Number {
		return p.byteValue(t, "the entry word")
	}

	return 0, p.errorf("%q is not a word this entry can use", name)
}

// condacts reads the block that follows an entry, up to whatever ends it: the
// next entry, the next process, or the end of the source.
func (p *parser) condacts() ([]ddb.Condact, error) {
	var out []ddb.Condact

	for {
		t := p.peek()
		if t.Kind == Entry || t.Kind == EOF || p.atSection(SecPRO) || p.atSection(SecEND) {
			return out, nil
		}

		if t.Kind == Label {
			return nil, p.errorf("labels are not read yet")
		}

		if t.Kind == Directive {
			return nil, p.errorf("%s inside a process is not read yet", t.Directive)
		}

		name, err := p.expect(Ident, "a condact")
		if err != nil {
			return nil, err
		}

		// An unknown name here is a condact spelled wrong, and saying so beats
		// what the compiler of reference does: it takes anything it cannot
		// place as the end of the block, and the error surfaces somewhere else.
		op, def, known := ddb.LookupCondactByName(name.Text)
		if !known {
			return nil, p.errorf("%q is not a condact", name.Text)
		}

		params, err := p.params(op, def)
		if err != nil {
			return nil, p.errorf("%s: %w", def.Name, err)
		}

		out = append(out, ddb.Condact{Opcode: op, Params: params})
	}
}

// params reads the operands of one condact. How many there are comes from the
// table and never from the source, which is what makes a token stream readable
// at all: nothing marks where a condact ends.
func (p *parser) params(op ddb.Opcode, def ddb.CondactDef) ([]ddb.Param, error) {
	out := make([]ddb.Param, 0, def.NumParams())

	for i, kind := range def.Params {
		var param ddb.Param

		// The @ says the operand names a flag whose content is used instead.
		if p.peek().Kind == Indirect {
			p.next()

			param.Indirect = true
		}

		value, err := p.operand(op, kind)
		if err != nil {
			return nil, p.errorf("parameter %d: %w", i+1, err)
		}

		param.Value = value
		out = append(out, param)
	}

	return out, nil
}

// operand works out one parameter.
//
// The order is the compiler's, and it matters: a vocabulary word may be spelled
// the same as a symbol, and which one wins depends on what the condact asked
// for. A literal string is handled first of all, because for the message
// condacts it is not a value at all — it is a text that gets filed and replaced
// by its number.
func (p *parser) operand(op ddb.Opcode, kind ddb.ParamKind) (ddb.ID, error) {
	t := p.peek()

	if t.Kind == String && takesLiteral(op) {
		p.next()

		return p.fileMessage(op, t.Text)
	}

	// Where the condact names a kind of word, that kind is tried first.
	if want, named := wordKindOf(kind); named && (t.Kind == Ident || t.Kind == Number) {
		name := t.Text
		if t.Kind == Number {
			name = t.Describe()
		}

		if w, err := p.db.Words().GetByTypeAndSynonym(want, name); err == nil {
			p.next()

			return w.ID, nil
		}
	}

	if isSkip(op) {
		return p.skipOffset()
	}

	return p.byteValue(p.next(), "the operand")
}

// skipOffset reads how far a SKIP jumps.
//
// It is the one operand that can be written negative, because jumping back is
// what a loop is, and it travels as a byte: the interpreter adds it to the
// entry number and lets it wrap. So -2 is stored as 254, which is the same
// arithmetic the compiler of reference does at USintactic.pas:709.
//
// A byte cannot hold a jump further back than 128, and the check is here
// rather than left to the wrap: -200 would silently become 56 and jump the
// wrong way.
func (p *parser) skipOffset() (ddb.ID, error) {
	t := p.next()

	n, err := p.value(t)
	if err != nil {
		return 0, err
	}

	if n < 0 {
		if n < -128 {
			return 0, fmt.Errorf("%s: a jump of %d is further back than a byte reaches", t.Where(), n)
		}

		n += 256
	}

	if n > 255 {
		return 0, fmt.Errorf("%s: a jump of %d does not fit in a byte", t.Where(), n)
	}

	return ddb.ID(n), nil
}

func isSkip(op ddb.Opcode) bool {
	def, found := ddb.LookupCondact(op)

	return found && def.Name == "SKIP"
}

// wordKindOf says which kind of word a parameter names, when it names one.
func wordKindOf(kind ddb.ParamKind) (ddb.WordKind, bool) {
	switch kind {
	case ddb.ParamVerb:
		return ddb.Verb, true
	case ddb.ParamNoun:
		return ddb.Noun, true
	case ddb.ParamAdjective:
		return ddb.Adjective, true
	case ddb.ParamAdverb:
		return ddb.Adverb, true
	case ddb.ParamPreposition:
		return ddb.Preposition, true
	}

	return 0, false
}

// The condacts that take a text written in place rather than a number.
var literalCondacts = map[string]ddb.MessageKind{
	"MESSAGE": ddb.UserMessage,
	"MES":     ddb.UserMessage,
	"SYSMESS": ddb.SystemMessage,
}

func takesLiteral(op ddb.Opcode) bool {
	def, found := ddb.LookupCondact(op)
	if !found {
		return false
	}

	_, takes := literalCondacts[def.Name]

	return takes
}

// fileMessage puts a text written inside a condact into the table it belongs
// to, and gives back the number that replaces it.
func (p *parser) fileMessage(op ddb.Opcode, text string) (ddb.ID, error) {
	def, _ := ddb.LookupCondact(op)

	kind := literalCondacts[def.Name]

	existing, _ := p.db.Messages().GetAllMessages(kind)
	id := ddb.ID16(len(existing))

	if id.Int() > ddb.MAX_MESSSAGE.Int() {
		return 0, p.errorf("no room left in the message table for another text")
	}

	if err := p.db.Messages().NewLegacy(id, kind, text); err != nil {
		return 0, p.errorf("filing the text: %w", err)
	}

	return ddb.ID(id), nil
}
