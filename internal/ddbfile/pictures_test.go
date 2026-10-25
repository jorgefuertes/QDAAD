package ddbfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The illustration condacts come out when the build has no pictures, and
// nothing else does.
func TestPicturesComeOutWhenAskedFor(t *testing.T) {
	all := [][]Entry{{{
		Verb: 1, Noun: 2,
		Condacts: []Condact{
			{Opcode: opPICTURE, Params: []byte{4}},
			{Opcode: opDISPLAY, Params: []byte{0}},
			{Opcode: 29}, // CLS
		},
	}}}

	got := withoutPictures(all)

	require.Equal(t, []Condact{{Opcode: 29}}, got[0][0].Condacts)
	require.Equal(t, byte(1), got[0][0].Verb, "the entry itself is untouched")
	require.Equal(t, byte(2), got[0][0].Noun)

	// And the original is left alone, so that asking twice is not a different
	// answer and nobody else's view of the database changes underneath them.
	require.Len(t, all[0][0].Condacts, 3)
}

// A reader with no conversion set hands back what the binary says, which is
// the default and the thing every other caller depends on.
func TestWithoutAConversionNothingIsConverted(t *testing.T) {
	r := openDatabase(t, databases[0].path)

	require.False(t, r.Converting())

	before, err := r.Processes()
	require.NoError(t, err)

	r.Convert(doubleEverything{})
	require.True(t, r.Converting())

	after, err := r.Processes()
	require.NoError(t, err)

	require.Len(t, after, 2*len(before), "the conversion is asked and believed")
}

// doubleEverything is a conversion that does something obvious and nothing
// useful, which is all a test of the boundary needs.
type doubleEverything struct{}

func (doubleEverything) Processes(all [][]Entry) [][]Entry  { return append(all, all...) }
func (doubleEverything) Texts(t []string) []string          { return t }
func (doubleEverything) SystemMessages(t []string) []string { return t }
