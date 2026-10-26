package vm

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/stretchr/testify/require"
)

// The phrase in quotes, which is how an adventure lets the player speak.
//
// A line like DECIR "SIGUEME" holds two sentences for two different readers.
// The verb is the player's and is parsed at once; the phrase belongs to
// whoever the verb hands it to, and is parsed later by PARSE 1 into the same
// flags. Both halves have to come out of one line, so taking the quotes out is
// the first thing the parser does rather than something it does afterwards.
func TestTakingThePhraseOutOfTheQuotes(t *testing.T) {
	for _, tc := range []struct {
		name, line, rest, quoted string
	}{
		{
			name:   "the ordinary case",
			line:   `DECIR "SIGUEME"`,
			rest:   "DECIR ",
			quoted: "SIGUEME",
		},
		{
			name:   "with something after it",
			line:   `DECIR "HOLA" AL ELFO`,
			rest:   "DECIR  AL ELFO",
			quoted: "HOLA",
		},
		{
			name:   "nothing in quotes leaves the line alone",
			line:   "COGER LLAVE",
			rest:   "COGER LLAVE",
			quoted: "",
		},
		{
			// The forgiving reading, and the one a player who forgot the
			// second quote deserves: everything after the first is the phrase.
			name:   "an unclosed quote runs to the end",
			line:   `DECIR "SIGUEME`,
			rest:   "DECIR ",
			quoted: "SIGUEME",
		},
		{
			name:   "an empty phrase is a phrase that said nothing",
			line:   `DECIR ""`,
			rest:   "DECIR ",
			quoted: "",
		},
		{
			// Only the first pair counts, which is what the reference does: one
			// buffer, filled once.
			name:   "a second pair is not a second phrase",
			line:   `DECIR "UNO" Y "DOS"`,
			rest:   `DECIR  Y "DOS"`,
			quoted: "UNO",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rest, quoted := splitQuoted(tc.line)
			require.Equal(t, tc.rest, rest, "what is left for the player's own sentence")
			require.Equal(t, tc.quoted, quoted, "what is kept for PARSE 1")
		})
	}
}

// And PARSE 1 reads that phrase without asking for anything.
//
// Asking is the whole of what went wrong before: PARSE ignored its operand and
// always wanted a line, so the quoted phrase was dropped and the player's next
// command was eaten in its place. A console that is never reached is the proof
// — if PARSE 1 prompts, this hangs or takes input that is not there.
func TestParseOneAsksNobodyAnything(t *testing.T) {
	m := &Machine{
		console: refusingConsole{t},
		watch:   NoWatcher{},
		vocab:   vocabularyOf("SIGUE", 106, ddbKindNoun),
		literal: "SIGUEME",
	}

	// False means understood: the operand convention is that PARSE's condition
	// passes when the sentence made no sense.
	require.False(t, m.parse(1), "the phrase in hand was not understood")
	require.Equal(t, byte(106), m.flags[FlagNoun1], "SIGUEME did not reach the sentence")
}

// vocabularyOf is a one-word vocabulary, padded to five characters the way the
// database stores them so that the trimming is exercised too.
func vocabularyOf(word string, value, kind byte) []ddbfile.VocabularyEntry {
	return []ddbfile.VocabularyEntry{{Word: word + " ", Value: value, Kind: kind}}
}

// refusingConsole fails the test if anything asks it for input.
type refusingConsole struct{ t *testing.T }

func (c refusingConsole) Print(string) {}
func (c refusingConsole) AnyKey()      {}

func (c refusingConsole) Input() (string, bool) {
	c.t.Fatal("PARSE 1 asked the player for a line; the phrase was already in hand")

	return "", false
}
