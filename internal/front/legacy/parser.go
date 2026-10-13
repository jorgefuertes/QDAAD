package legacy

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// parser walks the live token stream and fills in a database.
//
// It is recursive descent over a slice with an index, which is all the language
// needs: a section runs until the marker of the next one, and nothing has to be
// looked at twice. The compiler of reference walks a linked list instead, and
// pays for it with an UnScan that rewinds two nodes; here a step back is at.
type parser struct {
	toks []Token
	at   int
	// symbols is the same table the preprocessor filled in, and it is still
	// being filled in. Six of the symbols are counts of what the source itself
	// declares, and the _VOC_ of every vocabulary word is another; none of them
	// can exist before the section that produces them has been read.
	symbols Symbols
	opts    Options
	db      *ddb.DDB
}

// Parse reads the token stream into a database.
func Parse(res *Result, opts Options) (*ddb.DDB, error) {
	p := &parser{toks: res.Tokens, symbols: res.Symbols, opts: opts, db: ddb.New()}

	p.db.Version = opts.Version
	p.db.MachineID = opts.Machine
	p.db.Language = opts.Language

	if err := p.run(); err != nil {
		return nil, err
	}

	return p.db, nil
}

// define adds a symbol the source produced rather than declared.
//
// It overwrites on purpose, unlike #define, which refuses a name it has seen
// before. These are ours and we write each of them once, and a source that has
// already used the name would have collided with the compiler of reference too.
func (p *parser) define(name string, value int) {
	p.symbols[name] = value
}

// The sections come in a fixed order and none of them is optional, which is
// what lets each one run until it meets the next marker.
func (p *parser) run() error {
	for _, step := range []func() error{
		p.control,
		p.vocabulary,
		p.systemMessages,
		p.userMessages,
		p.objectTexts,
		p.locationTexts,
		p.connections,
		p.objects,
		p.processes,
	} {
		if err := step(); err != nil {
			return err
		}
	}

	return nil
}

// peek looks at the token under the cursor without consuming it. The stream
// always ends in EOF, so there is always one to look at.
func (p *parser) peek() Token {
	if p.at >= len(p.toks) {
		return Token{Kind: EOF}
	}

	return p.toks[p.at]
}

func (p *parser) next() Token {
	t := p.peek()
	if p.at < len(p.toks) {
		p.at++
	}

	return t
}

// atSection says whether the cursor is on a given section marker, which is how
// every section knows it has ended.
func (p *parser) atSection(id SectionID) bool {
	t := p.peek()

	return t.Kind == Section && t.Section == id
}

// expectSection consumes the marker that opens a section.
func (p *parser) expectSection(id SectionID) error {
	if !p.atSection(id) {
		return p.errorf("%s expected", id)
	}

	p.next()

	return nil
}

// expect consumes a token of the kind given, or says what was there instead.
func (p *parser) expect(kind Kind, what string) (Token, error) {
	t := p.peek()
	if t.Kind != kind {
		return t, p.errorf("%s expected, but %s is there", what, t.Describe())
	}

	return p.next(), nil
}

// errorf reports at the position of the token under the cursor, which is the
// place a person can go and fix.
// The position goes in through the format rather than through Sprintf, so that
// a %w in the caller's format still wraps.
func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("%s: "+format, append([]any{p.peek().Where()}, args...)...)
}

// value works out the number a token stands for.
//
// The order is the compiler's: a quoted string is arithmetic, an underscore is
// the null word, a number is itself, and an identifier is a symbol. What is
// missing here on purpose is the vocabulary, which only some places accept and
// which those places try first.
func (p *parser) value(t Token) (int, error) {
	switch t.Kind {
	case Number:
		return t.Num, nil

	case Wildcard:
		return int(ddb.NO_WORD_ID), nil

	case String:
		return evaluate(t, p.symbols)

	case Ident:
		n, known := p.symbols[t.Text]
		if !known {
			return 0, fmt.Errorf("%s: %q is not defined", t.Where(), t.Text)
		}

		return n, nil
	}

	return 0, fmt.Errorf("%s: %s is not a value", t.Where(), t.Describe())
}

// byteValue is value with the range a byte allows, which every number in a
// database has to fit.
func (p *parser) byteValue(t Token, what string) (ddb.ID, error) {
	n, err := p.value(t)
	if err != nil {
		return 0, err
	}

	if n < 0 || n > 255 {
		return 0, fmt.Errorf("%s: %s is %d, which does not fit in a byte", t.Where(), what, n)
	}

	return ddb.ID(n), nil
}
