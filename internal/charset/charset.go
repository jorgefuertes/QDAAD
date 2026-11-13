// Package charset turns the text of a source into the codes a DAAD
// interpreter shows, and those codes back into a source.
//
// DAAD uses a character set of its own and it is the same on every machine —
// "throughout the whole DAAD system the same character set of 256 characters is
// used", quoted in docs/DAAD_v2_v3_Research/03-secciones.md §2.3 — so nothing
// here takes a target. Codes 16 to 31 hold the letters Spanish and Portuguese
// need, where ASCII keeps control codes; a handful of codes below 16 are the
// interpreter's own commands, and the rest below 128 is plain ASCII.
//
// This lives in the compiler rather than beside the comparison tool it grew in
// because it is a step of compiling, not of testing: something has to turn "#n"
// into a carriage return and "á" into code 21 before any byte reaches a
// database. The decompiler walks the same table the other way, which is what
// makes the round trip a property a test can state.
//
// Encode accepts more spellings than Decode writes: "#r" and "#n" are both the
// carriage return, "#A" to "#P" are the sixteen accented codes written for an
// editor that cannot carry the letters themselves, and the escapes of the 1991
// compiler are read as well as the modern ones. Decode picks one spelling of
// each, so Encode(Decode(b)) is b, while Decode(Encode(t)) is t only for a text
// already written the canonical way.
//
// The compiler of reference has a second conversion for the databases it builds
// with -7, where the accents are flattened to plain ASCII letters and the
// forced space is dropped altogether — ConvertAscii7Chars at
// UJSONExport.pas:45-153. This package is the other one, ConvertChars, which is
// what every target gets unless that switch is asked for. A 7-bit mode is a
// thing to add beside this one if it is ever wanted; it is not a mode this
// table can be bent into, because it loses letters on purpose.
package charset

import (
	"fmt"
	"strings"
)

// The codes that are not characters. They are what the interpreter acts on
// rather than prints, and every one of them has an escape.
const (
	EndOfText   = 0x0A // never data: it terminates a text
	ClearWindow = 0x0B // #b
	WaitKey     = 0x0C // #k
	Newline     = 0x0D // #n, and #r for the same thing
	GraphicSet  = 0x0E // #g
	TextSet     = 0x0F // #t
	ForcedSpace = 0x7F // #f
)

// firstLowChar is where the accented letters start.
const firstLowChar = 0x10

// lowCharset is the one table this package is built on: the sixteen codes DAAD
// keeps its accented letters at. The index is the code less firstLowChar, and
// both directions are derived from it so they cannot drift apart.
var lowCharset = [16]rune{
	'ª', '¡', '¿', '«', '»', 'á', 'é', 'í',
	'ó', 'ú', 'ñ', 'Ñ', 'ç', 'Ç', 'ü', 'Ü',
}

// lowByRune is lowCharset read backwards, with the one letter that lands
// outside it: ß shares its code with the forced space.
var lowByRune = func() map[rune]byte {
	m := make(map[rune]byte, len(lowCharset)+1)
	for i, r := range lowCharset {
		m[r] = byte(firstLowChar + i)
	}

	m['ß'] = ForcedSpace

	return m
}()

// graphicChars are the letters that do not fit in the low charset. They travel
// as a drawing: switch to the second character set, the slot the letter was put
// in there, switch back. Every accented letter no Spanish adventure needed is
// here, which is most of what the other languages need.
//
// See UJSONExport.pas:164-265, which is where the slots are decided.
var graphicChars = map[rune]byte{
	// Uppercase vowels with an acute accent.
	'Á': 0x7B, 'É': 0x7C, 'Í': 0x7D, 'Ó': 0x7E, 'Ú': 0x7F,
	// Lowercase vowels with the other accents.
	'à': 0x10, 'ã': 0x11, 'ä': 0x12, 'â': 0x13,
	'è': 0x14, 'ë': 0x15, 'ê': 0x16,
	'ì': 0x17, 'ï': 0x18, 'î': 0x19,
	'ò': 0x1A, 'õ': 0x1B, 'ö': 0x1C, 'ô': 0x1D,
	'ù': 0x1E, 'û': 0x1F,
	// The same, uppercase.
	'À': 0x20, 'Ã': 0x21, 'Ä': 0x22, 'Â': 0x23,
	'È': 0x24, 'Ë': 0x25, 'Ê': 0x26,
	'Ì': 0x27, 'Ï': 0x28, 'Î': 0x29,
	'Ò': 0x2A, 'Õ': 0x2B, 'Ö': 0x2C, 'Ô': 0x2D,
	'Ù': 0x2E, 'Û': 0x2F,
	// The letters the northern languages need.
	'ý': 0x3A, 'Ý': 0x3B, 'þ': 0x3C, 'Þ': 0x3D, 'å': 0x3E, 'Å': 0x3F,
	'ð': 0x5B, 'Ð': 0x5C, 'ø': 0x5D, 'Ø': 0x5E,
}

// escapes are the sequences an author writes instead of a code. They are
// applied over the whole text and after the per-character pass, which is what
// lets the graphic markers that pass wrote be resolved by the same rule — and
// means an author who typed "#g" by hand is treated the same way.
//
// The order is ConvertChars at UJSONExport.pas:248-263 followed line for line
// and it matters: "#e" expands into a graphic that the two rules after it then
// resolve.
//
// The last five are the escapes of the 1991 compiler, which the reference does
// not know — it reads "\n" and "\r" at :260-261 but not these. Our own
// decompiler wrote them until this package existed and those sources are out
// there, so they are read. They are never written.
var escapes = []struct{ from, to string }{
	{"#e", "#g\x60#t"}, // the euro sign, a graphic like the rest
	{"#g", string(rune(GraphicSet))},
	{"#t", string(rune(TextSet))},
	{"#b", string(rune(ClearWindow))},
	{"#s", " "},
	{"#f", string(rune(ForcedSpace))},
	{"#k", string(rune(WaitKey))},
	{"#n", string(rune(Newline))},
	{"#r", string(rune(Newline))},

	{`\g`, string(rune(GraphicSet))},
	{`\t`, string(rune(TextSet))},
	{`\b`, string(rune(ClearWindow))},
	{`\f`, string(rune(ForcedSpace))},
	{`\k`, string(rune(WaitKey))},
	{`\n`, string(rune(Newline))},
	{`\r`, string(rune(Newline))},
}

// Encode turns a text as the author wrote it into the codes DAAD holds it as.
//
// What comes back is a string of code points and not bytes, which is what the
// intermediate form needs — DAAD READY writes these as \u escapes in its JSON,
// so the two can be compared — and it is also what lets a character this table
// has no code for travel through to be rejected with its name intact. Turning
// the result into bytes, and refusing anything that does not fit in one, is the
// database emitter's to do.
func Encode(text string) string {
	var b strings.Builder

	for _, r := range text {
		if low, spanish := lowByRune[r]; spanish {
			b.WriteRune(rune(low))

			continue
		}

		if slot, graphic := graphicChars[r]; graphic {
			fmt.Fprintf(&b, "#g%c#t", rune(slot))

			continue
		}

		b.WriteRune(r)
	}

	out := b.String()

	for _, e := range escapes {
		out = strings.ReplaceAll(out, e.from, e.to)
	}

	// #A to #P are the sixteen accented codes written as letters, so that a
	// source can carry them through an editor that cannot hold the letters.
	for c := 'A'; c <= 'P'; c++ {
		out = strings.ReplaceAll(out, "#"+string(c), string(rune(c-'A'+firstLowChar)))
	}

	return out
}

// EncodeBytes is Encode with the result narrowed to the bytes a database
// holds, which is where a character this table has no code for stops being
// somebody else's problem.
//
// Encode leaves such a character alone so that it can be named here: a source
// that writes "Ω" produces a code point no DAAD machine has, and the useful
// thing to say is which character it was and not that byte 0xCE is out of
// range.
func EncodeBytes(text string) ([]byte, error) {
	encoded := Encode(text)
	out := make([]byte, 0, len(encoded))

	for _, r := range encoded {
		if r > 0xFF {
			return nil, fmt.Errorf("%q has no code in the DAAD character set", r)
		}

		out = append(out, byte(r))
	}

	return out, nil
}

// Decode turns the codes of a text back into a source.
//
// The escapes it writes are the modern ones, with a hash, which is what the
// templates DAAD READY ships are written with and therefore the one spelling
// worth emitting.
func Decode(raw []byte) (string, error) {
	var b strings.Builder

	for _, code := range raw {
		s, err := DecodeByte(code)
		if err != nil {
			return b.String(), err
		}

		b.WriteString(s)
	}

	return b.String(), nil
}

// DecodeByte is Decode for one code, which is how a reader that stops at a
// terminator or walks an obfuscated word needs it.
//
// Two codes are ambiguous and the choice is made here rather than guessed at
// by every caller. 0x7F is the forced space, ß, and the slot Ú sits in as a
// graphic: it comes back as "#f". And a graphic is not undone — 0x0E 0x7B 0x0F
// comes back as "#g{#t" and not as "Á" — because what the round trip has to
// preserve is the codes, and recomposing the letter would be a guess about
// which of the two spellings the author used.
func DecodeByte(code byte) (string, error) {
	if code >= firstLowChar && code < firstLowChar+byte(len(lowCharset)) {
		return string(lowCharset[code-firstLowChar]), nil
	}

	switch code {
	case Newline:
		return "#n", nil
	case GraphicSet:
		return "#g", nil
	case TextSet:
		return "#t", nil
	case ClearWindow:
		return "#b", nil
	case WaitKey:
		return "#k", nil
	case ForcedSpace:
		return "#f", nil
	}

	// The printable range goes in as it stands, the backslash included: the
	// escape character of a source is the hash now, so a backslash is an
	// ordinary character and needs nothing done to it.
	if code >= 0x20 && code < 0x7F {
		return string(rune(code)), nil
	}

	// Anything left has no spelling a compiler would read back. Saying so beats
	// inventing one: the \xNN this used to write was readable by nothing, ours
	// or theirs, and it would have turned a bad read into a source that looks
	// fine and compiles into something else.
	return "", fmt.Errorf("code 0x%02X has no source form", code)
}
