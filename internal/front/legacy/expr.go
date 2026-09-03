package legacy

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// evaluate works out the value of a quoted arithmetic expression.
//
// Wherever the language wants a value it also takes a string, and reads it as
// arithmetic over the symbols defined so far. That is why the operand of a
// conditional has to be quoted: the quotes are what mark an expression, not a
// piece of text.
//
// Integer arithmetic throughout. The compiler of reference evaluates in
// floating point and truncates, which only shows in a division, and truncation
// of a positive quotient is what Go's integer division already does.
func evaluate(t Token, symbols Symbols) (int, error) {
	e := &expr{text: []rune(t.Text), symbols: symbols, at: 0}

	value, err := e.sum()
	if err != nil {
		return 0, fmt.Errorf("%s: in %q: %w", t.Where(), t.Text, err)
	}

	e.skipBlanks()

	if e.at < len(e.text) {
		return 0, fmt.Errorf("%s: in %q: %q is left over",
			t.Where(), t.Text, string(e.text[e.at:]))
	}

	return value, nil
}

type expr struct {
	text    []rune
	at      int
	symbols Symbols
}

// sum is addition and subtraction, which bind loosest.
func (e *expr) sum() (int, error) {
	value, err := e.product()
	if err != nil {
		return 0, err
	}

	for {
		e.skipBlanks()

		if e.at >= len(e.text) || (e.text[e.at] != '+' && e.text[e.at] != '-') {
			return value, nil
		}

		op := e.text[e.at]
		e.at++

		right, err := e.product()
		if err != nil {
			return 0, err
		}

		if op == '+' {
			value += right
		} else {
			value -= right
		}
	}
}

// product is multiplication, division and remainder.
func (e *expr) product() (int, error) {
	value, err := e.atom()
	if err != nil {
		return 0, err
	}

	for {
		e.skipBlanks()

		if e.at >= len(e.text) {
			return value, nil
		}

		op := e.text[e.at]
		if op != '*' && op != '/' && op != '%' {
			return value, nil
		}

		e.at++

		right, err := e.atom()
		if err != nil {
			return 0, err
		}

		if (op == '/' || op == '%') && right == 0 {
			return 0, fmt.Errorf("division by zero")
		}

		switch op {
		case '*':
			value *= right
		case '/':
			value /= right
		case '%':
			value %= right
		}
	}
}

// atom is a number, a symbol, a parenthesised sum, or a negation.
func (e *expr) atom() (int, error) {
	e.skipBlanks()

	if e.at >= len(e.text) {
		return 0, fmt.Errorf("a value is missing")
	}

	c := e.text[e.at]

	switch {
	case c == '(':
		e.at++

		value, err := e.sum()
		if err != nil {
			return 0, err
		}

		e.skipBlanks()

		if e.at >= len(e.text) || e.text[e.at] != ')' {
			return 0, fmt.Errorf("a parenthesis is opened and never closed")
		}

		e.at++

		return value, nil

	case c == '-':
		e.at++

		value, err := e.atom()

		return -value, err

	case c == '+':
		e.at++

		return e.atom()

	case unicode.IsDigit(c):
		start := e.at
		for e.at < len(e.text) && unicode.IsDigit(e.text[e.at]) {
			e.at++
		}

		return strconv.Atoi(string(e.text[start:e.at]))

	case isIdent(c):
		start := e.at
		for e.at < len(e.text) && isIdent(e.text[e.at]) {
			e.at++
		}

		name := string(e.text[start:e.at])

		value, known := e.symbols[name]
		if !known {
			return 0, fmt.Errorf("%q is not defined", name)
		}

		return value, nil
	}

	return 0, fmt.Errorf("%q is not part of an expression", string(c))
}

func (e *expr) skipBlanks() {
	for e.at < len(e.text) && strings.ContainsRune(" \t", e.text[e.at]) {
		e.at++
	}
}
