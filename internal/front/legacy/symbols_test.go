package legacy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// parseSymbols runs the whole front end and hands back the table as it stands
// when the last section has been read.
//
// The map the preprocessor built is the one the parser writes into, so what
// comes back here has both halves: what the source declared with #define and
// what the source produced by declaring objects, locations and words.
func parseSymbols(t *testing.T, source string) Symbols {
	t.Helper()

	var lines []Line
	for i, text := range strings.Split(source, "\n") {
		lines = append(lines, Line{File: "t.sce", Num: i + 1, Text: text})
	}

	tokens, err := Scan(lines)
	require.NoError(t, err)

	opts := testOptions()

	res, err := Preprocess(tokens, opts)
	require.NoError(t, err)

	_, err = Parse(res, opts)
	require.NoError(t, err)

	return res.Symbols
}

// The counts a source produces by declaring things. A source reads them to
// write a loop over every object without a number that goes stale.
func TestCountsBecomeSymbols(t *testing.T) {
	s := parseSymbols(t, bare)

	// One object and two locations, as the bare source declares.
	require.Equal(t, 1, s["NUM_OBJECTS"])
	require.Equal(t, 0, s["LAST_OBJECT"])
	require.Equal(t, 2, s["NUM_LOCATIONS"])
	require.Equal(t, 1, s["LAST_LOCATION"])

	// The only object starts in the player's hands and nothing is worn.
	require.Equal(t, 1, s["NUM_CARRIED"])
	require.Equal(t, 0, s["NUM_WORN"])
}

// Every vocabulary word can be named as a symbol, which is how a process says
// _VOC_COGER rather than repeating the number the /VOC line gave it.
func TestVocabularyWordsBecomeSymbols(t *testing.T) {
	s := parseSymbols(t, bare)

	require.Equal(t, 1, s["_VOC_N"])
	require.Equal(t, 20, s["_VOC_COGER"])
	require.Equal(t, 30, s["_VOC_LLAVE"])
	require.Equal(t, 2, s["_VOC_VIEJA"])
}

// The symbol is built from the word as written, not from the five characters
// the database stores, so a long word has a symbol no entry is spelled like.
func TestVocabularySymbolIsNotTruncated(t *testing.T) {
	require.Equal(t, "_VOC_EXAMINAR", vocSymbol("examinar"))
	require.Equal(t, "_VOC_N", vocSymbol("n"))

	// An accented capital folds to its lowercase self: the DAAD character set
	// has no uppercase Á, so that code point is where an accented A lives.
	require.Equal(t, "_VOC_CAMIóN", vocSymbol("camión"))
	require.Equal(t, "_VOC_ñ", vocSymbol("ñ"))
}

// A source can use what it declared, which is the whole point of the counts
// existing: this one places an object at the last location by name.
func TestASourceCanUseTheCountsItProduced(t *testing.T) {
	s := parseSymbols(t, `
/CTL
_
/VOC
N     1  noun
COGER 20 verb
LLAVE 30 noun
VIEJA 2  adjective
/STX
/0 "It is dark."
/MTX
/0 "You cannot."
/OTX
/0 "a key"
/LTX
/0 "A cave."
/1 "A hall."
/CON
/0
/1
/OBJ
/0  WORN 3 Y N  _ _ _ _ _ _ _ _ _ _ _ _ _ _ Y _  LLAVE VIEJA
/PRO 0
> COGER LLAVE   PLACE 0 LAST_LOCATION
                DONE
/END
`)

	require.Equal(t, 0, s["NUM_CARRIED"])
	require.Equal(t, 1, s["NUM_WORN"])
}
