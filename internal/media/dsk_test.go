package media_test

import (
	"os"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

const (
	amstradImage  = "../../work/aventuras/La_Aventura_Original/AmstradCPC/ORIGINAL.DSK"
	spectrumImage = "../../work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original.dsk"
	pcwImage      = "../../work/aventuras/Cozumel/AmstradPCW/Cozumel_A.dsk"

	// Written by a tool whose opening line reads "MV - CPC format Disk Image
	// (DU54)" rather than the usual "MV - CPCEMU Disk-File".
	pcwTemplosImage = "../../work/aventuras/Los_templos_sagrados/AmstradPCW/templosa.DSK"
)

func openDSK(t *testing.T, path string) media.Image {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	v, err := media.Open(path)
	require.NoError(t, err)

	image, ok := v.(media.Image)
	require.True(t, ok, "a disk with no filesystem has to offer its sectors")

	return image
}

// The two editions use the two variants of the container: the Amstrad one has a
// fixed track size, the Spectrum one a table of sizes.
func TestOpenDSKImages(t *testing.T) {
	require.Equal(t,
		"CPCEMU disk image, 40 tracks, 1 side(s), written by CPDRead 1.12",
		openDSK(t, amstradImage).Format())

	require.Equal(t,
		"extended CPCEMU disk image, 42 tracks, 1 side(s), written by CPDRead v3.24",
		openDSK(t, spectrumImage).Format())
}

// Neither disk carries a filesystem, and saying so is the point: an empty list
// would read as an empty disk.
func TestDSKHasNoFiles(t *testing.T) {
	for _, path := range []string{amstradImage, spectrumImage} {
		files, err := openDSK(t, path).Files()
		require.Error(t, err)
		require.ErrorContains(t, err, "no filesystem")
		require.Nil(t, files)
	}
}

// TestDSKPayloadIsInOrder guards the part that is easy to get wrong. Both disks
// were formatted for their own loader: the Amstrad one holds 1 KiB sectors from
// track 1 onwards, and the Spectrum one starts each track at a different sector
// number so the drive need not wait for the disk to come round.
//
// Read in the order the track lists them rather than the order they are
// numbered, the Spectrum payload comes out interleaved. The proof that it does
// not is the database buried in it, whose header only reads as one if the
// sectors either side of it are in their place.
func TestDSKPayloadIsInOrder(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		size    int
		at      int
		machine byte
	}{
		{name: "amstrad", path: amstradImage, size: 132608, at: 0x7640, machine: 3},
		{name: "spectrum", path: spectrumImage, size: 195584, at: 0x507D, machine: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := openDSK(t, tc.path).Payload()
			require.Len(t, payload, tc.size)

			ddb := payload[tc.at:]

			require.Equal(t, byte(1), ddb[0], "DAAD version")
			require.Equal(t, tc.machine, ddb[1]>>4, "machine")
			require.Equal(t, byte(95), ddb[2], "the null word character")

			// The counters of the first part, the same on every machine.
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
		})
	}
}

// The Amstrad PCW disks are the other kind: ordinary CP/M volumes with a
// directory, on the same container the loader disks use. Reading it is what
// tells the two apart, and the character set is the check that says the lengths
// come out right — 2176 bytes is a 128-byte AMSDOS header and 256 glyphs of
// eight rows, and nothing else.
func TestDSKReadsAmsdosDirectory(t *testing.T) {
	files, err := openDSK(t, pcwImage).Files()
	require.NoError(t, err)

	sizes := map[string]int{}
	for _, f := range files {
		sizes[f.Name] = len(f.Data)
	}

	require.Equal(t, map[string]int{
		"PARTE001.DDB": 33280,
		"PARTE001.DAT": 111360,
		"PARTE001.CHR": 2176,
		"PARTE000.PIC": 16000,
		"AD.COM":       8960,
	}, sizes)
}

// Only the first eight characters of the opening line identify the original
// variant: the disks of Los Templos Sagrados were written by a tool that put
// its own wording after them, and refusing those would lose the edition.
func TestDSKAcceptsAnotherWording(t *testing.T) {
	require.Equal(t,
		"CPCEMU disk image, 40 tracks, 1 side(s)",
		openDSK(t, pcwTemplosImage).Format())
}
