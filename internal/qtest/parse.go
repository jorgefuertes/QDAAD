package qtest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Reading a script.
//
// One order per line, keywords in whatever case the author likes, a semicolon
// starts a comment — which is what a comment is in a DSF too, so there is one
// fewer thing to remember. A text argument is the rest of the line, or what is
// between quotes if it starts with one, and the quotes are what to reach for
// when the text has a comma or edge spaces that matter.
//
// Every error says the file, the line and the column, the way the front end
// does it. A script that cannot be read is said to be wrong before the
// adventure is started rather than halfway through it.

// defaultSeconds is the clock an EXPECT gets when it does not ask for one.
//
// Ten rather than none, because wanting to wait is the common case: an
// adventure with a TIME running says things on its own, and making every line
// spell out a number would be noise over the one thing a test is about.
const defaultSeconds = 10

// includeDepth is how far INCLUDE may nest. A cycle is caught by name, so this
// is only the belt to that braces.
const includeDepth = 16

// Parse reads a script from a file.
func Parse(path string) (*Script, error) {
	p := &parser{script: &Script{}}

	if err := p.file(path, Place{}, nil); err != nil {
		return nil, err
	}

	return p.script, nil
}

// ParseText reads a script that is not a file, which is what a test wants and
// what makes an INCLUDE-less script easy to write inline.
func ParseText(name, body string) (*Script, error) {
	p := &parser{script: &Script{}}

	if err := p.text(name, body, nil); err != nil {
		return nil, err
	}

	return p.script, nil
}

type parser struct {
	script *Script

	// named and gamed are which of the one-per-script directives have been
	// given, so that saying one twice is an error rather than the last one
	// quietly winning.
	named, gamed bool

	// open is where a STEP was opened that is still waiting for its END. It
	// belongs to the file being read — text puts it back on the way out of an
	// INCLUDE — so that an included script brings whole steps and never half
	// of one, which is what makes a failure inside a prologue nameable.
	open *Place
}

// file reads one script file. from is where the INCLUDE that asked for it was
// written, so that a missing file is reported against the line that named it.
func (p *parser) file(path string, from Place, stack []string) error {
	if len(stack) > includeDepth {
		return fmt.Errorf("%s: INCLUDE went more than %d deep", from, includeDepth)
	}

	full, err := filepath.Abs(path)
	if err != nil {
		full = path
	}

	for _, seen := range stack {
		if seen == full {
			return fmt.Errorf("%s: %s includes itself, round and round", from, path)
		}
	}

	body, err := os.ReadFile(path)
	if err != nil {
		if from.File == "" {
			return err
		}

		return fmt.Errorf("%s: %w", from, err)
	}

	return p.text(path, string(body), append(stack, full))
}

// text reads a whole script, which is a line at a time and nothing cleverer.
func (p *parser) text(name, body string, stack []string) error {
	// An INCLUDE is read from the middle of whatever included it, and the step
	// it is inside is that file's business rather than this one's.
	outer := p.open
	p.open = nil

	defer func() { p.open = outer }()

	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")

	for n, raw := range lines {
		line := uncomment(raw)

		word, rest := cut(line)
		if word == "" {
			continue
		}

		at := Place{File: name, Line: n + 1, Col: column(raw)}

		if err := p.order(word, rest, at, strings.TrimSpace(line), stack); err != nil {
			return err
		}
	}

	if p.open != nil {
		return fmt.Errorf("%s: this STEP was never closed, so nothing would say it passed", p.open)
	}

	return nil
}

// order reads one line that has something on it.
//
//nolint:gocognit,gocyclo,cyclop // one case per keyword; splitting it only hides the language
func (p *parser) order(word, rest string, at Place, source string, stack []string) error {
	switch strings.ToUpper(word) {
	case "TEST":
		name, _, err := words(rest)
		if err != nil || name == "" {
			return fmt.Errorf("%s: TEST needs a name", at)
		}

		if p.named {
			return fmt.Errorf("%s: the test was already given a name", at)
		}

		p.script.Name, p.named = name, true

	case "GAME":
		path, _, err := words(rest)
		if err != nil || path == "" {
			return fmt.Errorf("%s: GAME needs a database to play", at)
		}

		if p.gamed {
			return fmt.Errorf("%s: GAME was already said", at)
		}

		p.script.Game, p.gamed = beside(at.File, path), true

	case "SEED":
		seed, err := number(rest, 0, 0x7FFFFFFF)
		if err != nil {
			return fmt.Errorf("%s: SEED: %w", at, err)
		}

		if p.script.HasSeed {
			return fmt.Errorf("%s: SEED was already said", at)
		}

		p.script.Seed, p.script.HasSeed = int64(seed), true

	case "INCLUDE":
		path, _, err := words(rest)
		if err != nil || path == "" {
			return fmt.Errorf("%s: INCLUDE needs a file", at)
		}

		return p.file(beside(at.File, path), at, stack)

	case "STEP":
		name, _, err := words(rest)
		if err != nil || name == "" {
			return fmt.Errorf("%s: STEP needs a name, because the name is what gets reported", at)
		}

		if p.open != nil {
			return fmt.Errorf("%s: this STEP is inside the one opened at %s, and they do not nest", at, p.open)
		}

		here := at
		p.open = &here

		p.add(step{op: opStep, at: at, source: source, text: name})

	case "END":
		if rest != "" {
			return fmt.Errorf("%s: END takes nothing; it closes the STEP above it", at)
		}

		if p.open == nil {
			return fmt.Errorf("%s: END with no STEP open", at)
		}

		p.open = nil

		p.add(step{op: opEnd, at: at, source: source})

	case "EXPECT":
		return p.expect(rest, at, source)

	case "KEY":
		name, more := cut(rest)
		if name == "" || more != "" {
			return fmt.Errorf("%s: KEY takes one key, like RETURN, SPACE or A", at)
		}

		typed, enter, err := keystroke(name)
		if err != nil {
			return fmt.Errorf("%s: %w", at, err)
		}

		p.add(step{op: opKey, at: at, source: source, text: name, typed: typed, enter: enter})

	case "INPUT":
		text, tail, err := words(rest)
		if err != nil {
			return fmt.Errorf("%s: INPUT: %w", at, err)
		}

		if text == "" || tail != "" {
			return fmt.Errorf("%s: INPUT needs something to type, and nothing after it", at)
		}

		p.add(step{op: opInput, at: at, source: source, text: text})

	case "WAIT":
		secs, err := number(rest, 0, 0xFFFF)
		if err != nil {
			return fmt.Errorf("%s: WAIT: %w", at, err)
		}

		p.add(step{op: opWait, at: at, source: source, secs: secs})

	case "DEBUG":
		return p.debug(rest, at, source)

	case "ASSERT":
		return p.assert(rest, at, source)

	case "TRACE":
		switch strings.ToUpper(strings.TrimSpace(rest)) {
		case "ON":
			p.script.tracing = true

			p.add(step{op: opTrace, at: at, source: source, on: true})
		case "OFF":
			p.script.tracing = true

			p.add(step{op: opTrace, at: at, source: source})
		default:
			return fmt.Errorf("%s: TRACE is ON or OFF", at)
		}

	default:
		return fmt.Errorf("%s: %q is not an order this language has", at, word)
	}

	return nil
}

func (p *parser) add(s step) { p.script.steps = append(p.script.steps, s) }

// expect reads an EXPECT, which is the one order with an optional second
// argument and so the one with any ambiguity in it.
func (p *parser) expect(rest string, at Place, source string) error {
	kind := opExpect

	if first, more := cut(rest); strings.EqualFold(first, "NOT") {
		kind, rest = opExpectNot, more
	}

	quoted := strings.HasPrefix(rest, `"`)

	text, tail, err := words(rest)
	if err != nil {
		return fmt.Errorf("%s: EXPECT: %w", at, err)
	}

	secs, asked := defaultSeconds, false

	switch {
	case tail != "":
		secs, err = number(strings.TrimPrefix(tail, ","), 0, 0xFFFF)
		if err != nil {
			return fmt.Errorf("%s: EXPECT: %w", at, err)
		}

		asked = true

	case !quoted:
		// An unquoted text may end in ", 5", and then the number is the
		// budget rather than part of what the adventure should say. Quoting is
		// how to mean a text that really does end in a comma and a number.
		if shorter, n, ok := trailingNumber(text); ok {
			text, secs, asked = shorter, n, true
		}
	}

	if text == "" {
		return fmt.Errorf("%s: EXPECT needs something to wait for", at)
	}

	if kind == opExpectNot && asked {
		return fmt.Errorf("%s: EXPECT NOT has nothing to wait for, so it takes no seconds: "+
			"it is about what has been said already", at)
	}

	p.add(step{op: kind, at: at, source: source, text: text, secs: secs})

	return nil
}

func (p *parser) debug(rest string, at Place, source string) error {
	what, more := cut(rest)

	s := step{op: opDebug, at: at, source: source}

	switch strings.ToUpper(what) {
	case "FLAG":
		num, err := number(more, 0, 0xFF)
		if err != nil {
			return fmt.Errorf("%s: DEBUG FLAG: %w", at, err)
		}

		s.ask, s.num = askFlag, num

	case "FLAGS":
		switch {
		case more == "":
			s.ask = askFlags
		case strings.EqualFold(more, "SET"):
			s.ask = askFlagsSet
		default:
			return fmt.Errorf("%s: DEBUG FLAGS takes nothing, or SET for only the ones that hold something", at)
		}

	case "AT", "PICTURE", "OBJECTS", "SAID":
		if more != "" {
			return fmt.Errorf("%s: DEBUG %s takes nothing", at, strings.ToUpper(what))
		}

		switch strings.ToUpper(what) {
		case "AT":
			s.ask = askAt
		case "PICTURE":
			s.ask = askPicture
		case "OBJECTS":
			s.ask = askObjects
		default:
			s.ask = askSaid
		}

	default:
		return fmt.Errorf("%s: DEBUG knows FLAG, FLAGS, AT, PICTURE, OBJECTS and SAID, not %q", at, what)
	}

	p.add(s)

	return nil
}

//nolint:gocognit,cyclop // a case per thing that can be asserted
func (p *parser) assert(rest string, at Place, source string) error {
	fields := strings.Fields(spaceOperators(rest))
	if len(fields) == 0 {
		return fmt.Errorf("%s: ASSERT needs something to check", at)
	}

	s := step{op: opAssert, at: at, source: source, hasVal: true}

	what := strings.ToUpper(fields[0])
	args := fields[1:]

	switch what {
	case "ENDED":
		if len(args) != 0 {
			return fmt.Errorf("%s: ASSERT ENDED takes nothing", at)
		}

		s.check, s.hasVal = chkEnded, false

	case "FLAG":
		if len(args) != 3 {
			return fmt.Errorf("%s: it is ASSERT FLAG <number> = <value>", at)
		}

		num, err := number(args[0], 0, 0xFF)
		if err != nil {
			return fmt.Errorf("%s: ASSERT FLAG: %w", at, err)
		}

		how, err := comparison(args[1])
		if err != nil {
			return fmt.Errorf("%s: ASSERT FLAG: %w", at, err)
		}

		val, err := number(args[2], 0, 0xFF)
		if err != nil {
			return fmt.Errorf("%s: ASSERT FLAG: %w", at, err)
		}

		s.check, s.num, s.cmp, s.val = chkFlag, num, how, val

	case "AT", "PICTURE", "SCORE", "TURNS":
		if len(args) != 2 {
			return fmt.Errorf("%s: it is ASSERT %s = <value>", at, what)
		}

		how, err := comparison(args[0])
		if err != nil {
			return fmt.Errorf("%s: ASSERT %s: %w", at, what, err)
		}

		top := 0xFF
		if what == "TURNS" {
			top = 0xFFFF
		}

		val, err := number(args[1], 0, top)
		if err != nil {
			return fmt.Errorf("%s: ASSERT %s: %w", at, what, err)
		}

		s.cmp, s.val = how, val

		switch what {
		case "AT":
			s.check = chkAt
		case "PICTURE":
			s.check = chkPicture
		case "SCORE":
			s.check = chkScore
		default:
			s.check = chkTurns
		}

	case "CARRIED", "WORN", "HERE", "PRESENT":
		if len(args) != 1 {
			return fmt.Errorf("%s: it is ASSERT %s <object>", at, what)
		}

		num, err := number(args[0], 0, 0xFF)
		if err != nil {
			return fmt.Errorf("%s: ASSERT %s: %w", at, what, err)
		}

		s.num, s.hasVal = num, false

		switch what {
		case "CARRIED":
			s.check = chkCarried
		case "WORN":
			s.check = chkWorn
		case "HERE":
			s.check = chkHere
		default:
			s.check = chkPresent
		}

	case "OBJECT":
		if len(args) != 3 || !strings.EqualFold(args[1], "AT") {
			return fmt.Errorf("%s: it is ASSERT OBJECT <number> AT <location>", at)
		}

		num, err := number(args[0], 0, 0xFF)
		if err != nil {
			return fmt.Errorf("%s: ASSERT OBJECT: %w", at, err)
		}

		val, err := number(args[2], 0, 0xFF)
		if err != nil {
			return fmt.Errorf("%s: ASSERT OBJECT: %w", at, err)
		}

		s.check, s.num, s.val = chkObject, num, val

	default:
		return fmt.Errorf("%s: ASSERT does not know %q", at, fields[0])
	}

	p.add(s)

	return nil
}

// uncomment cuts a line at its comment, which a quoted text is allowed to
// contain.
func uncomment(line string) string {
	quoted := false

	for i, c := range line {
		switch c {
		case '"':
			quoted = !quoted
		case ';':
			if !quoted {
				return line[:i]
			}
		}
	}

	return line
}

// column is where the first word of a line starts, counting from one.
func column(line string) int {
	return utf8.RuneCountInString(line) - utf8.RuneCountInString(strings.TrimLeft(line, " \t")) + 1
}

// cut takes the first word off a line and trims what is left.
func cut(line string) (word, rest string) {
	line = strings.TrimSpace(line)

	at := strings.IndexFunc(line, unicode.IsSpace)
	if at < 0 {
		return line, ""
	}

	return line[:at], strings.TrimSpace(line[at:])
}

// words reads a text argument: what is between quotes if it starts with one,
// and otherwise the whole of what is left. tail is anything after a closing
// quote.
func words(rest string) (text, tail string, err error) {
	rest = strings.TrimSpace(rest)

	if !strings.HasPrefix(rest, `"`) {
		return rest, "", nil
	}

	end := strings.Index(rest[1:], `"`)
	if end < 0 {
		return "", "", errors.New("a quote that never closes")
	}

	return unescape(rest[1 : 1+end]), strings.TrimSpace(rest[end+2:]), nil
}

// unescape reads the few escapes a quoted text may hold.
//
// A line break is the one worth having: a TEST or a STEP whose name is two
// lines is a nicer heading than one long one, and there is no other way to
// write a newline on a line of a script.
//
// Anything else a backslash is in front of is left exactly as it was found. A
// quoted text is as likely to be a path as a sentence, and turning the
// backslashes of one into something else would be a surprise with no upside.
func unescape(text string) string {
	if !strings.ContainsRune(text, '\\') {
		return text
	}

	var out strings.Builder

	for i := 0; i < len(text); i++ {
		if text[i] != '\\' || i+1 >= len(text) {
			out.WriteByte(text[i])

			continue
		}

		switch text[i+1] {
		case 'n':
			out.WriteByte('\n')
		case 't':
			out.WriteByte('\t')
		case '"':
			out.WriteByte('"')
		case '\\':
			out.WriteByte('\\')
		default:
			out.WriteByte(text[i])

			continue
		}

		i++
	}

	return out.String()
}

// trailingNumber splits an unquoted text ending in ", 5".
func trailingNumber(text string) (shorter string, n int, ok bool) {
	comma := strings.LastIndex(text, ",")
	if comma < 0 {
		return text, 0, false
	}

	n, err := strconv.Atoi(strings.TrimSpace(text[comma+1:]))
	if err != nil || n < 0 {
		return text, 0, false
	}

	return strings.TrimSpace(text[:comma]), n, true
}

// number reads a whole number and says what is wrong with it if it is not one.
func number(text string, low, high int) (int, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("a number between %d and %d was wanted", low, high)
	}

	n, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number", text)
	}

	if n < low || n > high {
		return 0, fmt.Errorf("%d is outside %d to %d", n, low, high)
	}

	return n, nil
}

// spaceOperators pulls =, !=, > and < away from whatever they sit against, so
// that "FLAG 10=5" reads the same as "FLAG 10 = 5".
//
// It is one pass rather than a run of replacements: replacing "=" after "!="
// takes the second one apart again.
func spaceOperators(text string) string {
	var out strings.Builder

	for i := 0; i < len(text); i++ {
		switch c := text[i]; c {
		case '!':
			if i+1 < len(text) && text[i+1] == '=' {
				out.WriteString(" != ")
				i++

				continue
			}

			out.WriteByte(c)
		case '<':
			if i+1 < len(text) && text[i+1] == '>' {
				out.WriteString(" <> ")
				i++

				continue
			}

			out.WriteString(" < ")
		case '=':
			if i+1 < len(text) && text[i+1] == '=' {
				i++
			}

			out.WriteString(" = ")
		case '>':
			out.WriteString(" > ")
		default:
			out.WriteByte(c)
		}
	}

	return out.String()
}

func comparison(text string) (cmp, error) {
	switch text {
	case "=", "==":
		return cmpEq, nil
	case "!=", "<>":
		return cmpNot, nil
	case ">":
		return cmpGt, nil
	case "<":
		return cmpLt, nil
	}

	return cmpEq, fmt.Errorf("%q is not one of =, !=, > or <", text)
}

// keystroke works out what a named key does.
//
// What it types is what goes into the line being typed; the pauses do not care
// which key it was, because ANYKEY waits for a key and never asks which.
func keystroke(name string) (typed string, enter bool, err error) {
	switch strings.ToUpper(name) {
	case "RETURN", "ENTER":
		return "", true, nil
	case "SPACE":
		return " ", false, nil
	case "ESC", "ESCAPE":
		// Nothing a DAAD parser can read, so it types nothing. It is still a
		// keypress, which is all a pause wants.
		return "", false, nil
	}

	if utf8.RuneCountInString(name) == 1 {
		return name, false, nil
	}

	return "", false, fmt.Errorf("%q is not a key: RETURN, SPACE, ESC or one character", name)
}

// beside resolves a path named inside a script against the script's own
// directory, which is where the author was looking when they typed it.
func beside(script, path string) string {
	if filepath.IsAbs(path) || script == "" {
		return path
	}

	return filepath.Join(filepath.Dir(script), path)
}
