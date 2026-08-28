package media_test

import (
	"os"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

const atariImage = "../../work/AO/AtariST/ORIGINAL.ST"

func openAtari(t *testing.T) media.Volume {
	t.Helper()

	if _, err := os.Stat(atariImage); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	v, err := media.Open(atariImage)
	require.NoError(t, err)

	return v
}

// The Atari image carries no signature of its own: it is recognised by the
// BIOS parameter block, because the ST wrote plain MS-DOS floppies.
func TestOpenAtariImage(t *testing.T) {
	v := openAtari(t)
	require.Equal(t, "FAT12, 512 bytes per sector, 2 per cluster", v.Format())
}

func TestAtariFiles(t *testing.T) {
	files, err := openAtari(t).Files()
	require.NoError(t, err)

	byName := make(map[string][]byte, len(files))
	for _, f := range files {
		byName[f.Name] = f.Data
	}

	want := map[string]int{
		"PART1.DDB":    14464,
		"PART2.DDB":    20352,
		"PART1.CHR":    2176,
		"PART2.CHR":    2176,
		"PART1.CH0":    2176,
		"PART2.CH0":    2176,
		"PART1.DAT":    168570,
		"PART2.DAT":    288598,
		"PART1.SCR":    32066,
		"ORIGINAL.PRG": 15987,
		"JTB1.AGP":     512,
		"PANIC.BUT":    12,
		"STARTGEM.INF": 15,
	}

	for name, size := range want {
		data, found := byName[name]
		require.True(t, found, "%s is missing", name)
		require.Len(t, data, size, "size of %s", name)
	}

	require.Len(t, files, len(want), "the disk holds exactly these files")
}

// A file has to come out whole: with two sectors to a cluster, reading the
// chain wrong gives something that still looks plausible for the first
// kilobyte and then falls apart.
func TestAtariDatabaseIsWhole(t *testing.T) {
	files, err := openAtari(t).Files()
	require.NoError(t, err)

	var ddb []byte

	for _, f := range files {
		if f.Name == "PART1.DDB" {
			ddb = f.Data
		}
	}

	require.NotNil(t, ddb)

	require.Equal(t, byte(1), ddb[0], "DAAD version")
	require.Equal(t, byte(95), ddb[2], "the null word character")
	require.Equal(t, byte(9), ddb[3], "objects")
	require.Equal(t, byte(49), ddb[4], "locations")
	require.Equal(t, byte(140), ddb[5], "user messages")
	require.Equal(t, byte(17), ddb[7], "processes")

	word := make([]byte, 5)
	for i := range word {
		word[i] = ddb[32+i] ^ 0xFF
	}

	require.Equal(t, "\x15RBOL", string(word), "the first word of the vocabulary")
}

// The Atari and the Amiga editions were built for machines that align to a
// word, and both come out the same size: 126 bytes longer than the PC one,
// which needs no padding.
func TestAtariAndAmigaAgree(t *testing.T) {
	sizeOf := func(v media.Volume, name string) int {
		files, err := v.Files()
		require.NoError(t, err)

		for _, f := range files {
			if f.Name == name {
				return len(f.Data)
			}
		}

		t.Fatalf("%s is not in the image", name)

		return 0
	}

	require.Equal(t,
		sizeOf(openAmiga(t), "part1.ddb"),
		sizeOf(openAtari(t), "PART1.DDB"),
		"the two 68000 editions hold the same database")
}
