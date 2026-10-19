package charset_test

import (
	"fmt"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/charset"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	for _, tc := range []struct {
		name string
		from string
		want string
	}{
		{"plain text is left alone", "Hello there.", "Hello there."},
		{"a Spanish letter takes its low code", "años", "a\x1Aos"},
		{
			"every accent of the low charset", "ªáéíóúñÑçÇüÜ¡¿«»",
			"\x10\x15\x16\x17\x18\x19\x1A\x1B\x1C\x1D\x1E\x1F\x11\x12\x13\x14",
		},
		{"a letter outside it travels as a graphic", "Á", "\x0E\x7B\x0F"},
		{"the modern escapes", "#n#b#k#g#t#f", "\x0D\x0B\x0C\x0E\x0F\x7F"},
		{"#r is the carriage return as well", "#r", "\x0D"},
		{"#s is a space, which an editor cannot eat", "a#sb", "a b"},
		{"the sixteen written as letters", "#A#P", "\x10\x1F"},
		{
			"the escapes of 1991 are read too", `\n\b\k\g\t\f`,
			"\x0D\x0B\x0C\x0E\x0F\x7F",
		},
		{"a backslash on its own is a character", `C:\dir`, `C:\dir`},
		{"ß shares its code with the forced space", "ß", "\x7F"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, charset.Encode(tc.from))
		})
	}
}

func TestDecode(t *testing.T) {
	for _, tc := range []struct {
		name string
		from []byte
		want string
	}{
		{"plain text", []byte("Hello there."), "Hello there."},
		{"a low code comes back as its letter", []byte("a\x1Aos"), "años"},
		{
			"the control codes come back with a hash",
			[]byte{0x0D, 0x0B, 0x0C, 0x0E, 0x0F, 0x7F},
			"#n#b#k#g#t#f",
		},
		{"a graphic is not recomposed", []byte{0x0E, 0x7B, 0x0F}, "#g{#t"},
		{"a backslash needs nothing done to it", []byte(`C:\dir`), `C:\dir`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := charset.Decode(tc.from)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// A code with no source form is said so rather than given an invented escape.
//
// It is what the \xNN of the old decompiler got wrong: nothing read it back, so
// a text that came out of a bad read looked like a source and compiled into
// something else.
func TestDecodeRefusesACodeWithNoSourceForm(t *testing.T) {
	for _, code := range []byte{0x00, 0x01, 0x0A, 0x80, 0xFF} {
		t.Run(fmt.Sprintf("0x%02X", code), func(t *testing.T) {
			_, err := charset.DecodeByte(code)
			require.ErrorContains(t, err, "no source form")
		})
	}
}

// Every code that has a source form has to survive the round trip.
//
// This is the property the whole package exists for: the decompiler writes what
// Decode says and the compiler reads it with Encode, so a code that does not
// come back is a text the round trip loses.
func TestEveryCodeSurvivesTheRoundTrip(t *testing.T) {
	for code := range 256 {
		source, err := charset.DecodeByte(byte(code))
		if err != nil {
			continue
		}

		t.Run(fmt.Sprintf("0x%02X", code), func(t *testing.T) {
			require.Equal(t, string(rune(code)), charset.Encode(source),
				"%q did not encode back to the code it came from", source)
		})
	}
}

// The one place the round trip is not safe, stated so that it is a decision and
// not a surprise.
//
// A text holding a hash followed by one of the escape letters is read as the
// escape, because that is what the hash is for. It is the same in the compiler
// of reference and there is nowhere to put the difference: the format has no
// way of writing a literal hash. Across the sixty-six decompiled sources the
// only literal hash is in "#g#f#$#t", where the character after it is a dollar
// and nothing is lost.
func TestALiteralHashBeforeAnEscapeLetterIsTheEscape(t *testing.T) {
	require.Equal(t, "\x0D", charset.Encode("#n"))

	// And the one that does occur, which is safe.
	codes := []byte{charset.GraphicSet, charset.ForcedSpace, '#', '$', charset.TextSet}

	source, err := charset.Decode(codes)
	require.NoError(t, err)
	require.Equal(t, "#g#f#$#t", source)
	require.Equal(t, string(codes), charset.Encode(source))
}
