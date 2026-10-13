package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The conversion is only ever exercised end to end when fpc is installed, and
// it is the piece most likely to be subtly wrong: a mapping off by one puts a
// different letter on the screen and the comparison would report it as our
// fault rather than as the table's.
func TestDaadEncode(t *testing.T) {
	for _, tc := range []struct {
		name string
		from string
		want string
	}{
		{
			name: "plain text is left alone",
			from: "You cannot do that.",
			want: "You cannot do that.",
		},
		{
			name: "the Spanish letters go to the low charset",
			from: "Está demasiado oscuro",
			want: "Est\x15 demasiado oscuro",
		},
		{
			name: "the inverted question mark too",
			from: "¿Seguro?",
			want: "\x12Seguro?",
		},
		{
			name: "every Spanish vowel",
			from: "áéíóú",
			want: "\x15\x16\x17\x18\x19",
		},
		{
			name: "the ñ in both cases",
			from: "añoAÑO",
			want: "a\x1AoA\x1BO",
		},
		{
			name: "a new line",
			from: "Done.#n",
			want: "Done.\x0D",
		},
		{
			name: "a backslash-n means the same thing",
			from: `Done.\n`,
			want: "Done.\x0D",
		},
		{
			name: "a letter with no place in the low charset travels as a graphic",
			from: "Á",
			want: "\x0E\x7B\x0F",
		},
		{
			name: "the wait escape",
			from: "One.#kTwo.",
			want: "One.\x0CTwo.",
		},
		{
			name: "a hard space",
			from: "a#sb",
			want: "a b",
		},
		{
			name: "the lettered codes",
			from: "#A#P",
			want: "\x10\x1F",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, daadEncode(tc.from))
		})
	}
}

// The other front end writes files a JSON reader will not take, and the two
// ways it does it have to be told apart: a backslash it meant and one that is
// a character of the adventure.
func TestRepairEscapes(t *testing.T) {
	// The six characters drf writes to stand for one character code. Built
	// here rather than typed, so that nothing in this file is the control
	// character itself.
	code := `\` + `u0015`

	for _, tc := range []struct {
		name string
		from string
		want string
	}{
		{
			// A code the other side wrote on purpose, which has to survive as
			// the escape it is rather than become a backslash and a letter u.
			name: "a character code is left whole",
			from: `{"Text":"Est` + code + `"}`,
			want: `{"Text":"Est` + code + `"}`,
		},
		{
			name: "an escaped quote is left whole",
			from: `{"Text":"say \"hello\""}`,
			want: `{"Text":"say \"hello\""}`,
		},
		{
			name: "a stray backslash becomes a character",
			from: `{"Text":"a\kb"}`,
			want: `{"Text":"a\\kb"}`,
		},
		{
			// This is the one that matters. JSON would have taken \b as a
			// backspace, so without the repair the text quietly gains a
			// character the adventure never had.
			name: "one that JSON would have taken as an escape too",
			from: `{"Text":"a\b"}`,
			want: `{"Text":"a\\b"}`,
		},
		{
			name: "a backslash outside a string is not touched",
			from: `{"Path":1}` + `\`,
			want: `{"Path":1}` + `\`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, string(repairEscapes([]byte(tc.from))))
		})
	}
}

// A file written in the wrong encoding still has to be read, because that is
// what the vocabulary of an accented adventure arrives as.
func TestLatin1Rescue(t *testing.T) {
	// "CADÁVER" as ISO-8859-1: the Á is one byte, 0xC1, which is not valid
	// UTF-8 on its own.
	latin1 := []byte{'C', 'A', 'D', 0xC1, 'V', 'E', 'R'}

	require.Equal(t, "CADÁVER", string(latin1ToUTF8(latin1)))

	// Something already valid is handed straight back.
	require.Equal(t, "CADÁVER", string(latin1ToUTF8([]byte("CADÁVER"))))
}
