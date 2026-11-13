package qtest

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Finding what an EXPECT is looking for.
//
// Two liberties are taken with the text, and both are there because the
// alternative is a test that fails for a reason that has nothing to do with
// the adventure.
//
// Case is ignored. An adventure shouts its location names and whispers its
// messages, and which it does is a decision its author is allowed to change
// without every test having to be retyped.
//
// Any run of space in the text matches any run of space in what was said. The
// machine prints a line break wherever the database has a #n, so a message that
// reads as one sentence arrives with a newline in the middle of it and nobody
// writing a test knows which middle. Writing one space and meaning "some space
// here" is the reading that keeps the test about the words.

// indexFold finds text in said, and gives the byte it starts at and the byte
// just past its end, which is what the buffer is consumed up to. Both are -1
// when it is not there.
func indexFold(said, text string) (start, end int) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, 0
	}

	// range over a string stops at every rune start, which is where a match
	// may begin and nowhere else.
	for at := range said {
		if n := matchAt(said[at:], text); n >= 0 {
			return at, at + n
		}
	}

	return -1, -1
}

// matchAt says how many bytes of said the text takes up at the front of it, or
// -1 if it is not there.
func matchAt(said, text string) int {
	at := 0

	for text != "" {
		want, size := utf8.DecodeRuneInString(text)
		text = text[size:]

		if unicode.IsSpace(want) {
			text = strings.TrimLeftFunc(text, unicode.IsSpace)

			rest := said[at:]

			run := len(rest) - len(strings.TrimLeftFunc(rest, unicode.IsSpace))
			if run == 0 {
				return -1
			}

			at += run

			continue
		}

		got, size := utf8.DecodeRuneInString(said[at:])
		if size == 0 || !sameFold(got, want) {
			return -1
		}

		at += size
	}

	return at
}

// sameFold is two runes being the same letter. Lowering both is enough for
// every alphabet a DAAD charset can hold.
func sameFold(a, b rune) bool { return a == b || unicode.ToLower(a) == unicode.ToLower(b) }
