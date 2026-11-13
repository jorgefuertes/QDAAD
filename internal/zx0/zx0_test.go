package zx0

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The coder against the machine's own decoder.
//
// Decompress is a translation of the 68 bytes of Z80 the interpreter runs, so
// a stream it reads back whole is a stream the interpreter reads back whole.
// That the decoder is the format and not our reading of it is settled
// elsewhere, against Saukas' own coder: make compare-zx0.
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

// The coder compresses, which is the whole point of it and not something a
// round trip notices: a stream of nothing but literals reads back just as
// whole as a good one, so without this a coder that had stopped finding
// matches would pass every other test here.
//
// A run of one byte is the easy case on purpose. How much better it could be
// done is not this test's business — compare-zx0 measures that against Saukas'
// own coder — and what is asked for here is only that the matches are being
// found at all.
func TestTheCoderFindsItsMatches(t *testing.T) {
	data := bytes.Repeat([]byte{0x5A}, 4096)

	require.Less(t, len(Compress(data)), len(data)/16,
		"4096 bytes of one value should pack to almost nothing")

	// A block of literals still costs next to nothing to carry, which is what
	// keeps an incompressible picture from coming out half as big again.
	noise := everyByte()

	require.LessOrEqual(t, len(Compress(noise)), len(noise)+8,
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

// Every offset the format can express, and every length that is coded a
// different way.
//
// This is the part that could not be written before. An offset is the one
// thing in the stream whose coding does not read off the decoder at a glance:
// its halves are built in different registers, the low bit of the low byte is
// shifted out to become the first bit of the length, and the carry riding
// into that rotate decides what the whole thing means. So rather than trust
// the reasoning, this walks all of it — 32,640 offsets, each at the lengths
// that take a different path through the coder.
func TestEveryOffsetAndLengthComesBack(t *testing.T) {
	// A source with no repeats of its own, so that the only copy in the
	// stream is the one being asked for.
	source := make([]byte, maxOffset+64)
	for i := range source {
		source[i] = byte(i*7 + i/251)
	}

	for _, length := range []int{2, 3, 4, 5, 8, 17, 64, 255, 256, 257} {
		t.Run(fmt.Sprintf("length %d", length), func(t *testing.T) {
			for offset := 1; offset <= maxOffset; offset++ {
				w := &writer{}

				// Literals enough to copy from, then the copy.
				w.elias(offset)
				w.bytes(source[:offset])
				w.bit(1)
				w.match(offset, length)
				w.end()

				back, err := Decompress(w.out)
				if err != nil {
					t.Fatalf("offset %d: %v", offset, err)
				}

				want := make([]byte, 0, offset+length)
				want = append(want, source[:offset]...)

				for range length {
					want = append(want, want[len(want)-offset])
				}

				if !bytes.Equal(want, back) {
					t.Fatalf("offset %d, length %d: came back as %d bytes", offset, length, len(back))
				}
			}
		})
	}
}
