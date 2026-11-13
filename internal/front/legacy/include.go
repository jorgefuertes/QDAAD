package legacy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Line is one line of source together with where the author wrote it.
type Line struct {
	File string
	Num  int // within File, counting from one
	Text string
}

// #include is the one directive resolved before anything else, and it has rules
// of its own that are not the rules of the others.
const (
	includeWord = "#include"
	// Where the file name starts. The compiler of reference takes it from a
	// fixed column rather than parsing, so "#include  x" names " x".
	includeNameAt = 10
)

// Resolve reads a source and returns its lines with the includes pulled in.
//
// This is the only part of the front end that works on lines. Everything after
// it works on tokens, which is why the origin of each line is kept: it ends up
// in every token and is what makes an error point at the right place.
func Resolve(path string) ([]Line, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}

	out := make([]Line, 0, len(lines))

	for _, l := range lines {
		name, isInclude := includedName(l.Text)
		if !isInclude {
			out = append(out, l)

			continue
		}

		if name == "" {
			return nil, fmt.Errorf("%s:%d: #include names no file", l.File, l.Num)
		}

		// The name is relative to the file that asked for it, which is what
		// lets a source tree be moved without editing it.
		included, err := readLines(filepath.Join(filepath.Dir(l.File), name))
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", l.File, l.Num, err)
		}

		for _, inner := range included {
			if _, nested := includedName(inner.Text); nested {
				return nil, fmt.Errorf("%s:%d: nested includes are not allowed",
					inner.File, inner.Num)
			}
		}

		out = append(out, included...)
	}

	return out, nil
}

// includedName says whether a line is an #include and, if so, what it names.
//
// The word is matched in lower case and at the start of the line, as the
// compiler of reference does. The name runs from a fixed column to the end,
// minus any comment, and carries no quotes.
func includedName(text string) (string, bool) {
	if !strings.HasPrefix(text, includeWord) {
		return "", false
	}

	// Guard against #includesomething, which is not this directive.
	if len(text) > len(includeWord) && !isBlank(text[len(includeWord)]) {
		return "", false
	}

	name := ""
	if len(text) > includeNameAt-1 {
		name = text[includeNameAt-1:]
	}

	if at := strings.IndexByte(name, ';'); at >= 0 {
		name = name[:at]
	}

	return strings.TrimSpace(name), true
}

func isBlank(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r'
}

// readLines reads one file and splits it, checking that it is UTF-8.
//
// Nothing else is accepted and nothing is converted: a source has an encoding
// and that is the one. The sources DAAD READY ships are ISO-8859-1 and have to
// be converted before they get here.
func readLines(path string) ([]Line, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("%s is not UTF-8; convert it first", path)
	}

	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	split := strings.Split(text, "\n")

	// A trailing newline leaves an empty last field that is not a line.
	if len(split) > 0 && split[len(split)-1] == "" {
		split = split[:len(split)-1]
	}

	lines := make([]Line, 0, len(split))
	for i, t := range split {
		lines = append(lines, Line{File: path, Num: i + 1, Text: t})
	}

	return lines, nil
}
