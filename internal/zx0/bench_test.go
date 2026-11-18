package zx0

import (
	"os"
	"path/filepath"
	"testing"
)

// What a whole screen costs to compress, which is the size a +3 build hands
// over: one picture is one 6912-byte block.
func BenchmarkCompressAWholeScreen(b *testing.B) {
	benchScreen(b, "../../examples/AO/ZXSpectrum/+3/part1/gfx/scr", nil)
}

// And what an Amstrad one costs, which is the size that made any of this worth
// measuring: 16384 bytes against 6912, and the search is quadratic.
//
// The rows below the picture go in blanked, as cpcCropped hands them over, so
// that this is the block a CPC build actually compresses and not a denser one.
func BenchmarkCompressAnAmstradScreen(b *testing.B) {
	benchScreen(b, "../../examples/AO/Amstrad/CPC/part1/gfx/scr", func(screen []byte) []byte {
		// Fourteen character rows of eighty bytes, which is what --lines 112
		// leaves, in each of the eight bands video memory interleaves.
		cropped := append([]byte(nil), screen...)

		for band := range 8 {
			clear(cropped[band*0x800+14*80 : (band+1)*0x800])
		}

		return cropped
	})
}

func benchScreen(b *testing.B, dir string, prepare func([]byte) []byte) {
	names, err := filepath.Glob(filepath.Join(dir, "*.scr"))
	if err != nil || len(names) == 0 {
		b.Skipf("no screens in %s", dir)
	}

	screen, err := os.ReadFile(names[0])
	if err != nil {
		b.Fatal(err)
	}

	if prepare != nil {
		screen = prepare(screen)
	}

	b.SetBytes(int64(len(screen)))
	b.ResetTimer()

	for b.Loop() {
		Compress(screen)
	}
}
