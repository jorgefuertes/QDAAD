package back

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A reference is one byte, and which byte it is matters more than it looks.
//
// Entry zero of every table the reference carries is a placeholder no text
// contains, so the first token anything points at is entry one, at 128. Off by
// one here and every compressed text expands into somebody else's words — and
// still expands, which is what makes it worth a test of its own.
func TestAReferenceIsOneByteAndStartsPastThePlaceholder(t *testing.T) {
	require.Equal(t, []byte{127}, []byte(reference(0)), "the placeholder")
	require.Equal(t, []byte{128}, []byte(reference(1)), "the first real token")
	require.Equal(t, []byte{255}, []byte(reference(128)), "the last one a byte can name")

	// Written the obvious way this would be two bytes of UTF-8.
	require.Len(t, reference(128), 1)
}

// A token earns its place or it is dropped. The arithmetic is the reference's:
// the first use is counted as costing a byte, and every use after it as saving
// the token's length less one.
func TestOnlyTokensThatPayAreKept(t *testing.T) {
	table := []string{"\x00", " the ", " zzz ", " and "}

	t.Run("one used twice pays and stays", func(t *testing.T) {
		kept, savings := worthKeeping(table, []string{"a the b the c"})

		require.Equal(t, []string{"\x00", " the "}, kept)
		require.Equal(t, len(" the ")-1-1, savings, "two uses: one costs, one earns")
	})

	t.Run("one used once does not", func(t *testing.T) {
		kept, _ := worthKeeping(table, []string{"a the b"})

		require.Equal(t, []string{"\x00"}, kept)
	})

	t.Run("one used by nothing does not", func(t *testing.T) {
		kept, _ := worthKeeping(table, []string{"nothing here"})

		require.Equal(t, []string{"\x00"}, kept)
	})

	// The placeholder is kept whatever it is worth, because the numbering
	// starts after it.
	t.Run("the placeholder is never dropped", func(t *testing.T) {
		kept, _ := worthKeeping(table, []string{""})

		require.Equal(t, []string{"\x00"}, kept)
	})
}

// An earlier token takes what it finds before a later one is measured, so the
// two are not judged against the same text.
func TestATokenIsMeasuredOnWhatTheOnesBeforeItLeft(t *testing.T) {
	// " de la " contains " de ", so once the shorter one has been substituted
	// the longer has nothing left to find.
	table := []string{"\x00", " de ", " de la "}

	kept, _ := worthKeeping(table, []string{"a de b de c de la d de la e"})

	require.Equal(t, []string{"\x00", " de "}, kept,
		"the longer token is left with nothing to match")
}

// The texts come back written in the tokens that survived, numbered by where
// they ended up rather than where they started.
func TestTheTextsAreRewrittenInTheTokensThatSurvived(t *testing.T) {
	// " zzz " is in the table and in no text, so it is dropped and the token
	// after it moves up a place.
	table := []string{"\x00", " zzz ", " the "}
	old := compressionTokens["ES"]

	compressionTokens["ES"] = table
	defer func() { compressionTokens["ES"] = old }()

	bodies := []string{"a the b the c"}

	tokens, savings := compress("ES", bodies)

	require.Equal(t, []string{"\x00", " the "}, tokens)
	require.Positive(t, savings)
	require.Equal(t, "a"+reference(1)+"b"+reference(1)+"c", bodies[0],
		"numbered by its place in the table that was written")
}

// Every token ends with its top bit set, which is the only thing that says
// where one stops and the next starts.
func TestTheTableMarksTheEndOfEachToken(t *testing.T) {
	l := &linker{}
	l.writeTokens([]string{"ab", "c"})

	require.Equal(t, []byte{'a', 'b' | 0x80, 'c' | 0x80}, l.out)
}

// The tables are the reference's and are not ours to drift from.
func TestTheCarriedTablesAreTheOnesDaadReadyHas(t *testing.T) {
	for _, language := range []string{"ES", "EN", "PT", "DE", "FR"} {
		t.Run(language, func(t *testing.T) {
			table, known := compressionTokens[language]
			require.True(t, known)

			require.LessOrEqual(t, len(table), maxTokens,
				"a byte cannot name more than this")

			for i, token := range table {
				require.NotEmpty(t, token, "token %d", i)
				require.Less(t, len(token), 16, "token %d is not a sentence", i)

				for j := range len(token) {
					require.Less(t, token[j], byte(0x80),
						"token %d byte %d: the top bit is the table's own mark", i, j)
				}
			}
		})
	}

	require.Equal(t, "\x00", compressionTokens["ES"][0], "the placeholder")
	require.Contains(t, compressionTokens["ES"], " que ")
	require.Contains(t, compressionTokens["EN"], " the ")
}

// And what it is all for: a text that goes in long comes out shorter.
func TestCompressionActuallyShortensTheTexts(t *testing.T) {
	bodies := []string{
		"Est\x15s en una cueva de piedra con una puerta de madera.",
		"Hay una puerta de madera en la pared de piedra de la cueva.",
	}

	before := len(bodies[0]) + len(bodies[1])

	_, savings := compress("ES", bodies)

	require.Positive(t, savings)
	require.Less(t, len(bodies[0])+len(bodies[1]), before)
	require.Contains(t, strings.Join(bodies, ""), reference(1))
}
