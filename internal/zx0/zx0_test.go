package zx0

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The coder against the machine's own decoder.
//
// This is the only check there can be until there is a reference stream to
// compare with: Decompress is a translation of the 68 bytes of Z80 the
// interpreter runs, so a stream it reads back whole is a stream the
// interpreter reads back whole.
func TestWhatIsWrittenIsWhatComesBack(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"one byte", []byte{0x42}},
		{"a byte that is all ones", []byte{0xFF}},
		{"eight bytes, which is where a group of bits ends", bytes.Repeat([]byte{0xA5}, 8)},
		{"a run of the same byte", bytes.Repeat([]byte{0x00}, 300)},
		{"every byte there is", everyByte()},
		{"a screen's worth", bytes.Repeat([]byte{0x12, 0x34}, 3456)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			back, err := Decompress(Compress(tc.data))
			require.NoError(t, err)
			require.True(t, bytes.Equal(tc.data, back), "it did not come back whole")
		})
	}

	t.Run("a thousand random blocks", func(t *testing.T) {
		r := rand.New(rand.NewSource(1)) //nolint:gosec // a fixed seed, so a failure repeats

		for range 1000 {
			data := make([]byte, 1+r.Intn(2000))
			r.Read(data) //nolint:staticcheck // never fails on a rand source

			back, err := Decompress(Compress(data))
			require.NoError(t, err)
			require.True(t, bytes.Equal(data, back), "a block of %d bytes came back wrong", len(data))
		}
	})
}

// And against the real thing: the screens the decompiler writes out of La
// Aventura Original, which are what a build would actually carry.
func TestTheAdventuresOwnScreensSurviveTheRoundTrip(t *testing.T) {
	const dir = "../../work/decomp/ao/spectrum/tzx/part1/gfx/scr"

	names, err := filepath.Glob(filepath.Join(dir, "*.scr"))
	require.NoError(t, err)

	if len(names) == 0 {
		t.Skipf("no screens in %s: run the decompiler over the tape first", dir)
	}

	for _, name := range names {
		t.Run(filepath.Base(name), func(t *testing.T) {
			screen, err := os.ReadFile(name)
			require.NoError(t, err)
			require.Len(t, screen, 6912)

			back, err := Decompress(Compress(screen))
			require.NoError(t, err)
			require.Equal(t, screen, back)
		})
	}
}

// Nothing at all has no stream: the format cannot say "no literals here",
// because the shortest length Elias gamma writes is one.
func TestAnEmptyBlockHasNoStream(t *testing.T) {
	require.Nil(t, Compress(nil))
	require.Nil(t, Compress([]byte{}))
}

// Literals cost almost nothing to carry, which is the one thing that makes
// shipping them acceptable until the matches are written: a block comes out
// the size it went in plus a handful of bytes, not half as big again.
func TestLiteralsCostAlmostNothing(t *testing.T) {
	data := bytes.Repeat([]byte{0x5A}, 4096)

	require.LessOrEqual(t, len(Compress(data)), len(data)+8,
		"the overhead of a block of literals grew")
}

// A stream that stops in the middle says so instead of reading off the end.
func TestATruncatedStreamIsAnError(t *testing.T) {
	full := Compress(bytes.Repeat([]byte{0x7E}, 64))

	for cut := range len(full) {
		_, err := Decompress(full[:cut])
		require.Error(t, err, "a stream cut to %d bytes was read as whole", cut)
	}
}

func everyByte() []byte {
	out := make([]byte, 256)
	for i := range out {
		out[i] = byte(i)
	}

	return out
}
