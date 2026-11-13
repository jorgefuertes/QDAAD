package zx0

import (
	"os"
	"path/filepath"
	"testing"
)

// What a whole screen costs to compress, which is the size a +3 build hands
// over: one picture is one 6912-byte block.
func BenchmarkCompressAWholeScreen(b *testing.B) {
	const dir = "../../examples/AO/ZXSpectrum/+3/part1/gfx/scr"

	names, err := filepath.Glob(filepath.Join(dir, "*.scr"))
	if err != nil || len(names) == 0 {
		b.Skipf("no screens in %s", dir)
	}

	screen, err := os.ReadFile(names[0])
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(screen)))
	b.ResetTimer()

	for b.Loop() {
		Compress(screen)
	}
}
