package legacy

import (
	"fmt"
	"strings"
)

// Symbols is the table #define fills in.
type Symbols map[string]int

// externals records what the source asked to be linked in. The front end only
// takes note: which file a name resolves to depends on the target machine, and
// that is the back end's business.
type externals struct {
	extern []string
	interp []string
	sfx    []string
}

// Preprocess consumes the raw token stream and returns the live one.
//
// It resolves #define and the conditionals and drops the branches that are not
// taken, so the parser never learns those directives exist. What it does not
// touch are the five that emit bytes into a condact block — #db, #dw, #hex,
// #incbin and #userptr — which are grammar, not preprocessing, and go through
// untouched.
//
// The pass has to be sequential rather than two runs over the stream: the value
// of a #define can name an earlier one, and a conditional depends on what is
// defined by the time it is reached.
func Preprocess(tokens []Token) ([]Token, Symbols, error) {
	p := &preprocessor{
		in:      tokens,
		symbols: Symbols{},
	}

	if err := p.run(); err != nil {
		return nil, nil, err
	}

	if p.open > 0 {
		return nil, nil, fmt.Errorf("%d #endif missing at the end of the source", p.open)
	}

	return p.out, p.symbols, nil
}

type preprocessor struct {
	in      []Token
	at      int
	out     []Token
	symbols Symbols
	linked  externals
	// open counts the conditionals whose branch is being kept, so that a
	// missing #endif can be reported at the end.
	open int
}

func (p *preprocessor) run() error {
	for p.at < len(p.in) {
		t := p.in[p.at]

		if t.Kind != Directive || t.Directive.forParser() {
			p.out = append(p.out, t)
			p.at++

			continue
		}

		if err := p.directive(t); err != nil {
			return err
		}
	}

	return nil
}

func (p *preprocessor) directive(t Token) error {
	p.at++

	switch t.Directive {
	case DirDefine:
		return p.define(t)

	case DirIfdef, DirIfndef:
		return p.conditional(t)

	case DirElse:
		// Reaching an #else means the branch above it was the live one, so
		// everything from here to the #endif is dead.
		if p.open == 0 {
			return fmt.Errorf("%s: #else without #ifdef", t.Where())
		}

		return p.skipDead(t)

	case DirEndif:
		if p.open == 0 {
			return fmt.Errorf("%s: #endif without #ifdef", t.Where())
		}

		p.open--

		return nil

	case DirEcho:
		text, err := p.expectString(t, "#echo")
		if err != nil {
			return err
		}

		fmt.Println(text)

		return nil

	case DirExtern, DirInt, DirSfx:
		return p.external(t)

	case DirDebug, DirClassic:
		// Flags for the back end. Nothing to resolve here.
		return nil
	}

	return fmt.Errorf("%s: %s is not handled by the preprocessor", t.Where(), t.Directive)
}

func (p *preprocessor) define(t Token) error {
	name, err := p.expect(Ident, t, "#define needs a name")
	if err != nil {
		return err
	}

	if p.at >= len(p.in) {
		return fmt.Errorf("%s: #define %s needs a value", t.Where(), name.Text)
	}

	value := p.in[p.at]
	p.at++

	n, err := p.valueOf(value)
	if err != nil {
		return err
	}

	if _, already := p.symbols[name.Text]; already {
		return fmt.Errorf("%s: %q is defined twice", name.Where(), name.Text)
	}

	p.symbols[name.Text] = n

	return nil
}

// conditional keeps or drops the branch, depending on whether the symbol is
// known. The operand has to be quoted, which is a rule of the language and not
// of this implementation.
func (p *preprocessor) conditional(t Token) error {
	symbol, err := p.expectString(t, t.Directive.String())
	if err != nil {
		return err
	}

	_, known := p.symbols[symbol]
	if t.Directive == DirIfndef {
		known = !known
	}

	p.open++

	if known {
		return nil
	}

	return p.skipDead(t)
}

// skipDead walks past a branch that is not taken, counting the conditionals
// nested inside it so that only the matching #endif — or the #else of this very
// conditional — ends the skipping.
func (p *preprocessor) skipDead(from Token) error {
	depth := 1

	for p.at < len(p.in) {
		t := p.in[p.at]
		p.at++

		if t.Kind != Directive {
			continue
		}

		switch t.Directive {
		case DirIfdef, DirIfndef:
			depth++
		case DirEndif:
			depth--
			if depth == 0 {
				p.open--

				return nil
			}
		case DirElse:
			// Only the #else of the conditional that started the skip brings
			// us back; one nested deeper is part of the dead branch.
			if depth == 1 {
				return nil
			}
		}
	}

	return fmt.Errorf("%s: %s is never closed by an #endif", from.Where(), from.Directive)
}

func (p *preprocessor) external(t Token) error {
	name, err := p.expectString(t, t.Directive.String())
	if err != nil {
		return err
	}

	switch t.Directive {
	case DirExtern:
		p.linked.extern = append(p.linked.extern, name)
	case DirInt:
		p.linked.interp = append(p.linked.interp, name)
	case DirSfx:
		p.linked.sfx = append(p.linked.sfx, name)
	}

	return nil
}

// valueOf reads the value of a #define: a number, another symbol, or a quoted
// arithmetic expression.
func (p *preprocessor) valueOf(t Token) (int, error) {
	switch t.Kind {
	case Number:
		return t.Num, nil

	case Ident:
		n, known := p.symbols[t.Text]
		if !known {
			return 0, fmt.Errorf("%s: %q is not defined", t.Where(), t.Text)
		}

		return n, nil

	case String:
		return evaluate(t, p.symbols)
	}

	return 0, fmt.Errorf("%s: %s is not a value", t.Where(), t.Describe())
}

func (p *preprocessor) expect(want Kind, after Token, what string) (Token, error) {
	if p.at >= len(p.in) {
		return Token{}, fmt.Errorf("%s: %s, and the source ends", after.Where(), what)
	}

	t := p.in[p.at]
	if t.Kind != want {
		return Token{}, fmt.Errorf("%s: %s, but %s is there", t.Where(), what, t.Describe())
	}

	p.at++

	return t, nil
}

func (p *preprocessor) expectString(after Token, what string) (string, error) {
	t, err := p.expect(String, after,
		strings.TrimSpace(what)+" needs its operand between quotes")
	if err != nil {
		return "", err
	}

	return t.Text, nil
}
