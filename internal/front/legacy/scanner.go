package legacy

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Scan turns the resolved lines into tokens.
//
// It takes lines rather than one run of bytes on purpose. Three rules of the
// language are tied to the line and would need special care otherwise: a string
// runs to the last quote of its line and never crosses the end, a comment runs
// to the end of the line, and the position of every token is a line and a
// column. Scanning a line at a time gets all three for free — and the parser,
// which is what this is all for, never sees a line again.
func Scan(lines []Line) ([]Token, error) {
	var tokens []Token

	skipping := false

	for _, l := range lines {
		// /TOK is skipped whole, and it has to be skipped here rather than in
		// the parser: what it holds are compression tokens, which are pieces of
		// text and not source — "\f" and "_una_" are two of them — so they
		// would not survive being read as tokens at all. See startsSection.
		if skipping {
			if !startsSection(l.Text) {
				continue
			}

			skipping = false
		}

		got, err := scanLine(l)
		if err != nil {
			return nil, err
		}

		if len(got) == 1 && got[0].Kind == Section && got[0].Section == SecTOK {
			skipping = true

			continue
		}

		tokens = append(tokens, got...)
	}

	if len(lines) > 0 {
		last := lines[len(lines)-1]
		tokens = append(tokens, Token{
			Kind: EOF, File: last.File, Line: last.Num, Col: len(last.Text) + 1,
		})
	}

	return tokens, nil
}

// startsSection says whether a line opens a section, without lexing it.
//
// It is needed to find the end of a skipped section, whose contents cannot be
// read as tokens: the only way out is to look for the next marker by hand.
func startsSection(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}

	_, found := lookupSection(strings.ToUpper(fields[0]))

	return found
}

func scanLine(l Line) ([]Token, error) {
	var tokens []Token

	runes := []rune(l.Text)

	for at := 0; at < len(runes); {
		if runes[at] == ' ' || runes[at] == '\t' || runes[at] == '\r' {
			at++

			continue
		}

		// A comment ends the line, whatever is left of it.
		if runes[at] == ';' {
			break
		}

		t, width, err := scanToken(runes, at, l)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, t)
		at += width
	}

	return tokens, nil
}

// scanToken reads one token starting at "from" and says how wide it was.
func scanToken(runes []rune, from int, l Line) (Token, int, error) {
	here := Token{File: l.File, Line: l.Num, Col: from + 1}
	c := runes[from]

	switch {
	case c == '/':
		return scanSlash(runes, from, here)

	case c == '#':
		return scanDirective(runes, from, here)

	// Both quotings run to the last quote of the line, so a text may hold the
	// quote character without escaping it.
	case c == '"' || c == '\'':
		return scanString(runes, from, here, c)

	case c == '$':
		word := identRun(runes, from+1)
		if word == "" {
			return here, 0, lexError(here, "a label needs a name after the $")
		}

		here.Kind, here.Text = Label, word

		return here, 1 + len([]rune(word)), nil

	case c == '>':
		here.Kind = Entry

		return here, 1, nil

	case c == '@':
		here.Kind = Indirect

		return here, 1, nil

	// The wildcard is one character. "_" alone is the wildcard, but "_ABC" is
	// an identifier, because the longer match wins.
	case c == '*':
		here.Kind = Wildcard

		return here, 1, nil

	case c == '_':
		if word := identRun(runes, from); len([]rune(word)) > 1 {
			here.Kind, here.Text = Ident, word

			return here, len([]rune(word)), nil
		}

		here.Kind = Wildcard

		return here, 1, nil

	case c == '-' || isDigit(c):
		return scanNumber(runes, from, here)

	case isIdent(c):
		word := identRun(runes, from)
		here.Kind, here.Text = Ident, word

		return here, len([]rune(word)), nil
	}

	return here, 0, lexError(here, "%q is not a character this language uses", string(c))
}

// scanSlash tells a section marker from a list entry.
//
// The three rules start the same way, so the whole run is read first and then
// decided. Reading the run is what makes "/CTLX" a named entry rather than the
// /CTL section: the longest match wins.
func scanSlash(runes []rune, from int, here Token) (Token, int, error) {
	run := identRun(runes, from+1)
	if run == "" {
		return here, 0, lexError(here, "a lone / is not anything")
	}

	width := 1 + len([]rune(run))

	if n, err := strconv.Atoi(run); err == nil && !strings.HasPrefix(run, "-") {
		here.Kind, here.Num = ListEntry, n

		return here, width, nil
	}

	if id, found := lookupSection("/" + strings.ToUpper(run)); found {
		here.Kind, here.Section = Section, id

		return here, width, nil
	}

	here.Kind, here.Text = ListEntry, run

	return here, width, nil
}

func scanDirective(runes []rune, from int, here Token) (Token, int, error) {
	run := identRun(runes, from+1)
	if run == "" {
		return here, 0, lexError(here, "a lone # is not a directive")
	}

	width := 1 + len([]rune(run))

	id, found := lookupDirective("#" + strings.ToLower(run))
	if !found {
		return here, 0, lexError(here, "#%s is not a directive", run)
	}

	here.Kind, here.Directive = Directive, id

	return here, width, nil
}

func scanString(runes []rune, from int, here Token, quote rune) (Token, int, error) {
	// The last quote of the line closes it, not the next one. That is what
	// lets a text hold quotes of its own with nothing to escape them.
	end := -1
	for at := len(runes) - 1; at > from; at-- {
		if runes[at] == quote {
			end = at

			break
		}
	}

	if end < 0 {
		return here, 0, lexError(here, "a string is opened and never closed")
	}

	here.Kind, here.Text = String, string(runes[from+1:end])

	return here, end - from + 1, nil
}

func scanNumber(runes []rune, from int, here Token) (Token, int, error) {
	at := from
	if runes[at] == '-' {
		at++
	}

	start := at
	for at < len(runes) && isDigit(runes[at]) {
		at++
	}

	if at == start {
		return here, 0, lexError(here, "a minus sign with no number after it")
	}

	n, err := strconv.Atoi(string(runes[from:at]))
	if err != nil {
		return here, 0, lexError(here, "%s does not fit in a number", string(runes[from:at]))
	}

	here.Kind, here.Num, here.Text = Number, n, string(runes[from:at])

	return here, at - from, nil
}

// identRun reads the identifier characters starting at "from".
func identRun(runes []rune, from int) string {
	at := from
	for at < len(runes) && isIdent(runes[at]) {
		at++
	}

	return string(runes[from:at])
}

// isIdent says whether a character can be part of an identifier.
//
// The comma is in there because the character class of the reference lexer has
// one — almost certainly a slip of whoever wrote it, who meant to separate and
// typed the separator inside the class. It is kept so that a source that
// compiles there compiles here.
func isIdent(c rune) bool {
	return c == '_' || c == ',' || unicode.IsLetter(c) || unicode.IsDigit(c)
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func lexError(at Token, format string, args ...any) error {
	return fmt.Errorf("%s: %s", at.Where(), fmt.Sprintf(format, args...))
}
