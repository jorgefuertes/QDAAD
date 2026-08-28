package media_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

// La Aventura Original for the Amiga. The expected contents were checked
// against an independent extraction of the same image.
const amigaImage = "../../work/AO/Amiga/ORIGINAL.ADF"

func openAmiga(t *testing.T) media.Volume {
	t.Helper()

	if _, err := os.Stat(amigaImage); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	v, err := media.Open(amigaImage)
	require.NoError(t, err)

	return v
}

func TestOpenAmigaImage(t *testing.T) {
	v := openAmiga(t)
	require.Equal(t, "Amiga ADF (OFS)", v.Format(),
		"the adventure shipped on the old filesystem, the one with a header per data block")

	adf, ok := v.(*media.ADF)
	require.True(t, ok)
	require.Equal(t, "Aventura Original", adf.Name())
}

func TestAmigaFiles(t *testing.T) {
	files, err := openAmiga(t).Files()
	require.NoError(t, err)

	byName := make(map[string]int, len(files))
	for _, f := range files {
		byName[f.Name] = len(f.Data)
	}

	// Sizes as the directory declares them: the reader trims the padding of the
	// last block, so a mismatch here means the block list was walked wrong.
	want := map[string]int{
		"part1.ddb":          14464,
		"part2.ddb":          20352,
		"part1.chr":          2176,
		"part2.chr":          2176,
		"part1.dat":          168704,
		"part2.dat":          288732,
		"part1.scr":          32127,
		"AD.AGP":             512,
		"AD1.AGP":            512,
		"original.ad":        17252,
		"s/startup-sequence": 28,
		"l/disk-validator":   1848,
		"devs/keymaps/e":     1356,
		"system/setmap":      4500,
	}

	for name, size := range want {
		got, found := byName[name]
		require.True(t, found, "%s is missing; found %v", name, keys(byName))
		require.Equal(t, size, got, "size of %s", name)
	}

	require.Len(t, keys(byName), 17, "the disk holds seventeen files, plus the empty c/ directory")
}

// TestAmigaFilesAreWholeFiles guards the part that is easy to get wrong. On the
// old filesystem a file is scattered: every 512-byte block spends 24 bytes on a
// header, so carving the image would give 488 bytes of payload followed by
// rubbish. A database read back from here has to parse as one.
func TestAmigaFilesAreWholeFiles(t *testing.T) {
	files, err := openAmiga(t).Files()
	require.NoError(t, err)

	var ddb []byte

	for _, f := range files {
		if filepath.Base(f.Name) == "part1.ddb" {
			ddb = f.Data
		}
	}

	require.NotNil(t, ddb, "part1.ddb is not in the image")

	// The header of a database: version, the null word character, and counters
	// that have to match the PC edition of the same adventure.
	require.Equal(t, byte(1), ddb[0], "DAAD version")
	require.Equal(t, byte(95), ddb[2], "the null word character")
	require.Equal(t, byte(9), ddb[3], "objects")
	require.Equal(t, byte(49), ddb[4], "locations")
	require.Equal(t, byte(140), ddb[5], "user messages")
	require.Equal(t, byte(63), ddb[6], "system messages")
	require.Equal(t, byte(17), ddb[7], "processes")

	// And the vocabulary, which sits right after a 32-byte header. Reading it
	// undoes the obfuscation every word carries.
	word := make([]byte, 5)
	for i := range word {
		word[i] = ddb[32+i] ^ 0xFF
	}

	require.Equal(t, "\x15RBOL", string(word), "the first word is ÁRBOL, with its accent")
}

func keys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}
