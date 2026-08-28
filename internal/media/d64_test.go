package media_test

import (
	"os"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

const commodoreImage = "../../work/AO/Commodore64/Aventura Original.d64"

func openCommodore(t *testing.T) media.Volume {
	t.Helper()

	if _, err := os.Stat(commodoreImage); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	v, err := media.Open(commodoreImage)
	require.NoError(t, err)

	return v
}

// Like the Atari one, this image carries no signature: it is a raw dump, and
// only its size tells it apart.
func TestOpenCommodoreImage(t *testing.T) {
	v := openCommodore(t)
	require.Equal(t, "Commodore 1541 disk image", v.Format())

	d, ok := v.(*media.D64)
	require.True(t, ok)
	require.Equal(t, "AV. ORIGINAL", d.Name())
}

// The Commodore edition shipped no databases of its own: each part is a single
// program with the interpreter, the database and the graphics inside it. That
// is why there is nothing here named like the .DDB files of the other machines.
func TestCommodoreFiles(t *testing.T) {
	files, err := openCommodore(t).Files()
	require.NoError(t, err)

	want := map[string]int{
		"AVENTURA1":    60311,
		"AVENTURA2":    60286,
		"LEEME/README": 8752,
	}

	byName := make(map[string][]byte, len(files))
	for _, f := range files {
		byName[f.Name] = f.Data
	}

	for name, size := range want {
		data, found := byName[name]
		require.True(t, found, "%s is missing", name)
		require.Len(t, data, size, "size of %s", name)
	}

	require.Len(t, files, len(want), "the disk holds exactly these files")
}

// TestCommodoreFilesAreWholeFiles guards the part that is easy to get wrong.
// CBM DOS links its sectors: every 256-byte block spends its first two bytes
// pointing at the next one, and the blocks of a file are not in order in the
// image. Carving would give 254 good bytes and then rubbish.
//
// The proof is the database buried in the first program. Its header only reads
// as a header if the chain was followed correctly, because the bytes around it
// come from blocks that lie elsewhere on the disk.
func TestCommodoreFilesAreWholeFiles(t *testing.T) {
	files, err := openCommodore(t).Files()
	require.NoError(t, err)

	var program []byte

	for _, f := range files {
		if f.Name == "AVENTURA1" {
			program = f.Data
		}
	}

	require.NotNil(t, program)

	// Where the database sits inside the program.
	const at = 13972

	ddb := program[at:]

	require.Equal(t, byte(1), ddb[0], "DAAD version")
	require.Equal(t, byte(2), ddb[1]>>4, "machine: Commodore 64")
	require.Equal(t, byte(95), ddb[2], "the null word character")

	// The counters of the adventure, the same ones the other machines carry.
	require.Equal(t, byte(9), ddb[3], "objects")
	require.Equal(t, byte(49), ddb[4], "locations")
	require.Equal(t, byte(140), ddb[5], "user messages")
	require.Equal(t, byte(63), ddb[6], "system messages")
	require.Equal(t, byte(17), ddb[7], "processes")

	word := make([]byte, 5)
	for i := range word {
		word[i] = ddb[32+i] ^ 0xFF
	}

	require.Equal(t, "\x15RBOL", string(word), "the first word is ÁRBOL, with its accent")
}
