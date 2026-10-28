package decompiler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Where a source is told to find the adventure's own font.
//
// It travels in one of two places and both are the adventure's: inside the
// database on the eight-bit machines, in the same block as the drawings, and
// beside it as a file on the sixteen-bit ones. Either way the build needs it —
// a part built with somebody else's font plays with the little icons beside
// its objects turned into letters — so either way the source has to say so.
func TestWhereASourceLooksForItsOwnFont(t *testing.T) {
	const charsetSize = 256 * 8

	t.Run("inside the database, as the eight-bit editions keep it", func(t *testing.T) {
		db := database{vectors: &vectors{graphics: make([]byte, charsetSize)}}

		require.Equal(t, "gfx/chr/udg.chr", charsetPath(db, Options{}))
	})

	t.Run("beside it, as the sixteen-bit editions shipped it", func(t *testing.T) {
		db := database{assets: []asset{
			{name: "PART1.CGA"},
			{name: "PART1.CHR"},
		}}

		require.Equal(t, "gfx/raw/PART1.CHR", charsetPath(db, Options{Binaries: true}))
	})

	// An Atari ST part carries a .CH0 as well: the same glyphs for its other
	// screen mode, and not what a build wants.
	t.Run("the alternate font is not the font", func(t *testing.T) {
		db := database{assets: []asset{{name: "PART1.CH0"}, {name: "PART1.CHR"}}}

		require.Equal(t, "gfx/raw/PART1.CHR", charsetPath(db, Options{Binaries: true}))

		only := database{assets: []asset{{name: "PART1.CH0"}}}
		require.Empty(t, charsetPath(only, Options{Binaries: true}))
	})

	// Without the originals there is no file to point at, and a source naming
	// one that was never written is a build that stops.
	t.Run("nothing is named when nothing is kept", func(t *testing.T) {
		db := database{assets: []asset{{name: "PART1.CHR"}}}

		require.Empty(t, charsetPath(db, Options{Binaries: false}))
	})

	// The vector editions keep theirs whatever the flag says, because it is
	// written out of the database rather than copied from a file.
	t.Run("the one inside the database survives --no-binaries", func(t *testing.T) {
		db := database{vectors: &vectors{graphics: make([]byte, charsetSize)}}

		require.Equal(t, "gfx/chr/udg.chr", charsetPath(db, Options{Binaries: false}))
	})

	t.Run("an adventure with no font of its own says nothing", func(t *testing.T) {
		require.Empty(t, charsetPath(database{}, Options{Binaries: true}))

		// A graphics block too short to hold one is not one.
		short := database{vectors: &vectors{graphics: make([]byte, charsetSize-1)}}
		require.Empty(t, charsetPath(short, Options{Binaries: true}))
	})
}
