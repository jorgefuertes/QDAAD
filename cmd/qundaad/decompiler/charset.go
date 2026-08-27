package decompiler

import "strings"

// DAAD uses a character set of its own, the same on every machine. Codes 16 to
// 31 hold the letters Spanish and Portuguese need; everything else below 128 is
// plain ASCII. See docs/DAAD_v2_v3_Research/03-secciones.md section 2.3.
var lowCharset = [16]rune{
	'ª', '¡', '¿', '«', '»', 'á', 'é', 'í',
	'ó', 'ú', 'ñ', 'Ñ', 'ç', 'Ç', 'ü', 'Ü',
}

const (
	codeEndOfText   = 0x0A // never data: it terminates the string
	codeClearWindow = 0x0B // \b
	codeWaitKey     = 0x0C // \k
	codeNewline     = 0x0D // a null line in the source
	codeGraphicSet  = 0x0E // \g
	codeTextSet     = 0x0F // \t
	codeBackslash   = 0x5C // the escape character itself
	codeForcedSpace = 0x7F // \f
)

// decodeChar turns one DAAD code into its source form.
//
// The escape character of a 1991 source is the backslash, not the hash that
// the modern compiler uses (manual section 5.1, note g). A newline comes back
// as a real one: the writer turns it into the null line the compiler expects.
func decodeChar(code byte) string {
	switch {
	case code >= 16 && code < 32:
		// Accented letters can go in directly when the database declares a
		// language, which is the readable choice. The escapes \A to \P produce
		// the same codes if a plain ASCII source is ever needed.
		return string(lowCharset[code-16])
	case code == codeNewline:
		return "\n"
	case code == codeGraphicSet:
		return `\g`
	case code == codeTextSet:
		return `\t`
	case code == codeClearWindow:
		return `\b`
	case code == codeWaitKey:
		return `\k`
	case code == codeForcedSpace:
		return `\f`
	case code == codeBackslash:
		return `\\`
	case code >= 32 && code < 127:
		return string(rune(code))
	}

	// No source form: keep it visible rather than emit something that would
	// compile into a different byte.
	return `\x` + hex2(code)
}

func hex2(b byte) string {
	const digits = "0123456789abcdef"

	return string([]byte{digits[b>>4], digits[b&0x0F]})
}

// sourceLines lays a decoded text out the way the compiler reads it back.
//
// Two rules from the manual (section 5.1, notes i and j) drive this: it joins
// consecutive lines with a space, and turns a null line into a carriage
// return. So each run of text goes on a single line, and the carriage returns
// inside it become blank lines. An empty text produces no lines at all: a
// blank one would compile into a carriage return that was never there.
func sourceLines(text string) []string {
	if text == "" {
		return nil
	}

	parts := strings.Split(text, "\n")
	lines := make([]string, 0, len(parts)*2)

	for i, part := range parts {
		if i > 0 {
			lines = append(lines, "") // the null line that means a return
		}

		lines = append(lines, protectSpaces(part))
	}

	return lines
}

// protectSpaces writes the spaces at either end as \s. They are significant —
// the compiler joins lines with a space of its own — and text editors are known
// to strip them, which is why the escape exists at all.
func protectSpaces(line string) string {
	trimmed := strings.Trim(line, " ")
	if trimmed == line {
		return line
	}

	leading := len(line) - len(strings.TrimLeft(line, " "))
	trailing := len(line) - len(strings.TrimRight(line, " "))

	return strings.Repeat(`\s`, leading) + trimmed + strings.Repeat(`\s`, trailing)
}

// tokenText renders a compression token as it expands into a text: spaces are
// spaces.
func tokenText(raw []byte) string {
	var sb strings.Builder

	for _, c := range raw {
		sb.WriteString(decodeChar(c))
	}

	return sb.String()
}

// tokenSource renders a token for the /TOK section, where the sources of the
// time wrote spaces as the null word character.
//
// Careful: PCDAAD and jDAAD expand that character back to a space at run time,
// but msx2daad and NextDAAD do not. A token whose spaces matter is therefore
// not portable across every interpreter.
func tokenSource(token string, nullWord byte) string {
	return strings.ReplaceAll(token, " ", string(rune(nullWord)))
}
