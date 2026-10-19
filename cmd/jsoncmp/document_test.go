package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
