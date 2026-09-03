package legacy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// scanText runs the scanner over one line of source.
func scanText(t *testing.T, text string) []Token {
	t.Helper()

	got, err := Scan([]Line{{File: "t.sce", Num: 1, Text: text}})
	require.NoError(t, err)

	// The end marker is not what these tests are about.
	require.NotEmpty(t, got)
	require.Equal(t, EOF, got[len(got)-1].Kind)

	return got[:len(got)-1]
}

func kinds(tokens []Token) []Kind {
	out := make([]Kind, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, t.Kind)
	}

	return out
}

// The longest match wins, and only then the first rule. Both halves of that
// matter here: /CTL is a section but /CTLX is a named entry, and a lone
// underscore is the wildcard while a longer one is a name.
func TestScannerTakesTheLongestMatch(t *testing.T) {
	got := scanText(t, "/CTL /CTLX _ _ABC * 12 /12")

	require.Equal(t,
		[]Kind{Section, ListEntry, Wildcard, Ident, Wildcard, Number, ListEntry},
		kinds(got))

	require.Equal(t, SecCTL, got[0].Section)
	require.Equal(t, "CTLX", got[1].Text)
	require.Equal(t, "_ABC", got[3].Text)
	require.Equal(t, 12, got[5].Num)
	require.Equal(t, 12, got[6].Num)
}

// A string runs to the LAST quote of its line, which is what lets a text hold
// quotes of its own with nothing to escape them.
func TestScannerReadsToTheLastQuote(t *testing.T) {
	got := scanText(t, `/0 "Te dice: "hola", y se va."`)

	require.Equal(t, []Kind{ListEntry, String}, kinds(got))
	require.Equal(t, `Te dice: "hola", y se va.`, got[1].Text)
}

// A comment ends the line wherever it starts, but a semicolon inside a string
// is text and does not open one.
func TestScannerComments(t *testing.T) {
	require.Empty(t, scanText(t, "; everything here is a comment"))

	got := scanText(t, `MESSAGE 3 ; and this is not a parameter`)
	require.Equal(t, []Kind{Ident, Number}, kinds(got))

	got = scanText(t, `/0 "a ; inside is text"`)
	require.Equal(t, []Kind{ListEntry, String}, kinds(got))
	require.Equal(t, "a ; inside is text", got[1].Text)
}

// The position is the author's: file, line and column, counting from one.
func TestScannerReportsWhereItIs(t *testing.T) {
	got, err := Scan([]Line{
		{File: "a.sce", Num: 7, Text: "  > _ _  MESSAGE 3"},
	})
	require.NoError(t, err)

	require.Equal(t, "a.sce:7:3", got[0].Where())
	require.Equal(t, Entry, got[0].Kind)
	require.Equal(t, "a.sce:7:10", got[3].Where())
	require.Equal(t, "MESSAGE", got[3].Text)
}

func TestScannerDirectivesAndTheirSynonyms(t *testing.T) {
	got := scanText(t, "#define #if #ifdef #defb #db #DEFW")

	require.Equal(t,
		[]DirectiveID{DirDefine, DirIfdef, DirIfdef, DirDB, DirDB, DirDW},
		[]DirectiveID{
			got[0].Directive, got[1].Directive, got[2].Directive,
			got[3].Directive, got[4].Directive, got[5].Directive,
		})
}

func TestScannerLabelsIndirectionAndSignedNumbers(t *testing.T) {
	got := scanText(t, "$again SKIP -3 LET @31 1")

	require.Equal(t,
		[]Kind{Label, Ident, Number, Ident, Indirect, Number, Number},
		kinds(got))

	require.Equal(t, "again", got[0].Text)
	require.Equal(t, -3, got[2].Num)
	require.Equal(t, 31, got[4+1].Num)
}

func TestScannerRejectsWhatItCannotRead(t *testing.T) {
	for _, tc := range []struct{ text, says string }{
		{`"never closed`, "opened and never closed"},
		{"#nosuch", "is not a directive"},
		{"/", "lone / is not anything"},
		{"$", "needs a name after the $"},
		{"~", "is not a character this language uses"},
	} {
		_, err := Scan([]Line{{File: "t.sce", Num: 1, Text: tc.text}})
		require.ErrorContains(t, err, tc.says, "scanning %q", tc.text)
	}
}

// The section markers are matched whole and without regard to case, and /TOK is
// among them even though version 3 dropped it: it has to be recognised so that
// the parser knows where it ends and can skip it.
func TestScannerKnowsEverySection(t *testing.T) {
	got := scanText(t, "/CTL /VOC /STX /MTX /OTX /LTX /CON /OBJ /PRO /END /TOK /tok")

	want := []SectionID{
		SecCTL, SecVOC, SecSTX, SecMTX, SecOTX, SecLTX,
		SecCON, SecOBJ, SecPRO, SecEND, SecTOK, SecTOK,
	}

	require.Len(t, got, len(want))

	for i, w := range want {
		require.Equal(t, Section, got[i].Kind)
		require.Equal(t, w, got[i].Section)
	}
}
