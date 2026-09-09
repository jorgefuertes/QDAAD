package decompiler

import (
	"os"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

const msxImageFile = "../../../work/aventuras/Chichén_Itzá/MSX/chichen_itza.dsk"

func msxFiles(t *testing.T) []media.File {
	t.Helper()

	if _, err := os.Stat(msxImageFile); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	v, err := media.Open(msxImageFile)
	require.NoError(t, err)

	files, err := v.Files()
	require.NoError(t, err)

	return files
}

// names is the make-up of each set, which is what a failure here should show
// rather than the tens of kilobytes each chunk carries.
func names(sets [][]msxChunk) [][]string {
	out := make([][]string, 0, len(sets))

	for _, set := range sets {
		group := make([]string, 0, len(set))
		for _, c := range set {
			group = append(group, c.name)
		}

		out = append(out, group)
	}

	return out
}

// The disk holds both parts of the adventure, as two runs of chunks that both
// write to the same addresses. Keeping them apart is what stops the second
// burying the first, and their names are the only thing that says which is
// which.
//
// The title screen is a BLOAD file too and comes out as a set of its own, which
// costs nothing: it performs no copy, so it builds no memory and is passed over.
func TestMSXSetsAreKeptApart(t *testing.T) {
	// In the order the loader asks for them, which is not the order the
	// directory names them: the disk lists CIA3 first and CIA1 near the end.
	require.Equal(t, [][]string{
		{"CIA1", "CIA2", "CIA3", "CIA4", "CIA5", "CIA6", "CIA7"},
		{"CIB1", "CIB2", "CIB3", "CIB4", "CIB5", "CIB6", "CIB7"},
		{"CI_LOGO.BIN"},
	}, names(msxSets(msxChunks(msxFiles(t)))))
}

// The copies are read out of the code that performs them, and this is what they
// come to. Nothing else on the disk says where a chunk ends up.
func TestMSXCopiesAreReadFromTheCode(t *testing.T) {
	sets := msxSets(msxChunks(msxFiles(t)))
	require.NotEmpty(t, sets)

	got := map[string][]msxCopy{}

	for _, c := range sets[0] {
		if moves := c.copies(); len(moves) > 0 {
			got[c.name] = moves
		}
	}

	require.Equal(t, map[string][]msxCopy{
		"CIA2": {{from: 0x8900, to: 0x4000, size: 6912}},
		"CIA3": {{from: 0x9000, to: 0x0100, size: 16128}},
		"CIA4": {{from: 0x9000, to: 0x4000, size: 16384}},
		"CIA7": {{from: 0xD200, to: 0x8000, size: 360}},
	}, got)
}

// And the proof that the reconstruction is right: a database comes out of the
// memory those copies build, at the address the first of them writes to. No
// single file on the disk holds one.
func TestMSXImageHoldsADatabase(t *testing.T) {
	files := msxFiles(t)

	for _, f := range files {
		require.Empty(t, FindDatabases(f.Data), "%s should hold no database on its own", f.Name)
	}

	databases := 0

	for i, set := range msxSets(msxChunks(files)) {
		memory := msxImage(set)
		if memory == nil {
			continue
		}

		found := FindDatabases(memory)
		require.Len(t, found, 1, "set %d", i)
		require.Equal(t, 0x0100, found[0].Offset())

		databases++
	}

	require.Equal(t, 2, databases, "one for each part of the adventure")
}
