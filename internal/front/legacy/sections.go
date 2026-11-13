package legacy

import (
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// control reads /CTL, which holds the null word and nothing else.
//
// It is a relic that ends up in byte 2 of the binary. The compiler of reference
// accepts one or more underscores and anything else is an error, so that is
// what is asked for here.
func (p *parser) control() error {
	if err := p.expectSection(SecCTL); err != nil {
		return err
	}

	seen := false

	for !p.atSection(SecVOC) {
		t := p.next()
		if t.Kind != Wildcard {
			return p.errorf("/VOC expected, but %s is there", t.Describe())
		}

		seen = true
	}

	if !seen {
		return p.errorf("/CTL names no null word")
	}

	// The underscore is what the source writes and what the binary carries.
	p.db.NullWord = '_'

	return nil
}

// The word types, in the spelling the source uses. The compiler of reference
// compares without regard to case, so this does too.
var wordKinds = map[string]ddb.WordKind{
	"VERB":        ddb.Verb,
	"ADVERB":      ddb.Adverb,
	"NOUN":        ddb.Noun,
	"ADJECTIVE":   ddb.Adjective,
	"PREPOSITION": ddb.Preposition,
	"CONJUGATION": ddb.Conjunction,
	"PRONOUN":     ddb.Pronoun,
}

// vocabulary reads /VOC: a word, its value and its type, one per record.
//
// It has to come before everything that follows: connections name a direction
// by its word and objects name themselves with a noun and an adjective.
func (p *parser) vocabulary() error {
	if err := p.expectSection(SecVOC); err != nil {
		return err
	}

	for !p.atSection(SecSTX) {
		word := p.next()
		if word.Kind != Ident && word.Kind != Number {
			return p.errorf("a vocabulary word or /STX expected, but %s is there",
				word.Describe())
		}

		// A word may be written as a number — the digits are words in Chichén
		// Itzá — so the text is what names it, whatever kind the scanner gave.
		name := word.Text
		if word.Kind == Number {
			name = word.Describe()
		}

		valueTok := p.next()

		value, err := p.byteValue(valueTok, "the value of "+name)
		if err != nil {
			return err
		}

		kindTok, err := p.expect(Ident, "a word type")
		if err != nil {
			return err
		}

		kind, known := wordKinds[strings.ToUpper(kindTok.Text)]
		if !known {
			return p.errorf("%q is not a word type", kindTok.Text)
		}

		if err := p.db.Words().NewLegacy(value, kind, name); err != nil {
			return p.errorf("%s %d %s: %w", name, value, kindTok.Text, err)
		}

		p.define(vocSymbol(name), int(value))
	}

	return nil
}

// vocSymbol is the name a vocabulary word answers to as a symbol, so that a
// process can write _VOC_NORTE instead of repeating the number.
//
// It is not the word as the database stores it. The stored form is cut to five
// characters and the symbol is not, so a long word has a symbol no vocabulary
// entry is spelled like — which is what the compiler of reference does, in
// UVocabularyTree.pas:70, because it takes the name before the cut.
//
// The accented capitals turn into their lowercase selves, which looks wrong
// and is not: the DAAD character set has no uppercase Á, so that code point is
// where an accented A lives whatever its case. Folding here rather than in
// ddb.NormalizeWord is deliberate — there it would be the emitter's business
// leaking into the model, here it is only how a symbol is spelled.
func vocSymbol(word string) string {
	folded := strings.Map(func(r rune) rune {
		if lower, accented := accentedCaps[r]; accented {
			return lower
		}

		return r
	}, strings.ToUpper(word))

	return "_VOC_" + folded
}

var accentedCaps = map[rune]rune{
	'Á': 'á', 'É': 'é', 'Í': 'í', 'Ó': 'ó',
	'Ú': 'ú', 'Ü': 'ü', 'Ñ': 'ñ', 'Ç': 'ç',
}

// textSection reads one of the four text tables, which all share a grammar:
// a numbered list entry and the text it names, on one line.
//
// The numbering has to run consecutively from zero. The compiler of reference
// insists on it, and the binary could not hold anything else: a text table is
// indexed by position, so a gap is not a thing it can express.
func (p *parser) textSection(open, close SectionID, store func(ddb.ID16, string) error) error {
	if err := p.expectSection(open); err != nil {
		return err
	}

	want := 0

	for !p.atSection(close) {
		entry := p.next()
		if entry.Kind != ListEntry {
			return p.errorf("a numbered entry or %s expected, but %s is there",
				close, entry.Describe())
		}

		if entry.Text != "" {
			return p.errorf("%s is named where a number was expected", entry.Describe())
		}

		if entry.Num != want {
			return p.errorf("entries must run consecutively from zero: %d expected, %d found",
				want, entry.Num)
		}

		text, err := p.expect(String, "a text between quotes")
		if err != nil {
			return err
		}

		if err := store(ddb.ID16(entry.Num), text.Text); err != nil {
			return p.errorf("entry %d: %w", entry.Num, err)
		}

		want++
	}

	return nil
}

func (p *parser) systemMessages() error {
	return p.textSection(SecSTX, SecMTX, func(id ddb.ID16, text string) error {
		return p.db.Messages().NewLegacy(id, ddb.SystemMessage, text)
	})
}

func (p *parser) userMessages() error {
	return p.textSection(SecMTX, SecOTX, func(id ddb.ID16, text string) error {
		return p.db.Messages().NewLegacy(id, ddb.UserMessage, text)
	})
}

// objectTexts reads /OTX. The text goes on the object it describes, so the
// objects have to exist by the time /OBJ fills the rest of them in — which is
// why the store is created here and completed later.
//
// How many there are becomes two symbols, which is how a source writes a loop
// over every object without hard-coding a number that a later edit invalidates.
func (p *parser) objectTexts() error {
	if err := p.textSection(SecOTX, SecLTX, func(id ddb.ID16, text string) error {
		return p.db.Objects().NewLegacy(ddb.Object{
			ID:          ddb.ID(id),
			Description: text,
		})
	}); err != nil {
		return err
	}

	p.defineCount("OBJECT", p.db.Objects().Len())

	return nil
}

func (p *parser) locationTexts() error {
	if err := p.textSection(SecLTX, SecCON, func(id ddb.ID16, text string) error {
		return p.db.Locations().NewLegacy(ddb.ID(id), text)
	}); err != nil {
		return err
	}

	p.defineCount("LOCATION", p.db.Locations().Len())

	return nil
}

// defineCount adds the NUM_ and LAST_ pair for a table.
//
// LAST_ of an empty table is -1, which is what the compiler of reference leaves
// behind and is not an accident worth correcting: a loop written as
// "0 to LAST_OBJECT" has to run zero times when there are none.
func (p *parser) defineCount(what string, n int) {
	p.define("NUM_"+what+"S", n)
	p.define("LAST_"+what, n-1)
}
