package pack

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// screens makes pictures that compress to different sizes, so that choosing by
// size has something to choose between.
//
// The ink is a pattern that does not repeat, because a screen ZX0 can see a
// pattern in comes out tiny however much of it is covered, and then every
// picture fits and the choosing has nothing to do.
func screens(count int) map[int][]byte {
	out := make(map[int][]byte, count)

	for n := range count {
		one := make([]byte, screenSize)
		for at := range min((n+1)*230, screenSize) {
			one[at] = byte(at*7 + at/251 + n)
		}

		out[n] = one
	}

	return out
}

// Reading the buffer back the way the Z80 does: walk the index five bytes at a
// time until the number is 255, and for a hit take the offset as an address in
// page one and the five bytes there as the picture's header.
//
// See CheckCachedImages in PLUS3.ASM. The point of doing it this way rather
// than against our own structs is that the format is the interpreter's, and a
// test that agrees with the writer but not with the reader proves nothing.
func TestTheBufferReadsBackTheWayTheInterpreterWalksIt(t *testing.T) {
	cached, loose, err := chooseCached(screens(30))
	require.NoError(t, err)
	require.NotEmpty(t, cached)
	require.NotEmpty(t, loose, "thirty of these do not all fit")

	file, err := plus3Cache(cached)
	require.NoError(t, err)

	// Two bytes saying how much follows, because the interpreter cannot ask
	// +3DOS how long a file is without reading past its end.
	require.Equal(t, len(file)-2, int(binary.LittleEndian.Uint16(file)),
		"the header counts everything after itself")

	// What lands in page one, and where.
	page := file[2:]
	require.LessOrEqual(t, len(page), plus3CacheRoom, "it has to fit under 0x10000")

	found := map[int][]byte{}

	at := 0

	for {
		number := int(page[at])
		if number == plus3CacheEnd {
			break
		}

		address := int(binary.LittleEndian.Uint16(page[at+1:]))
		size := int(binary.LittleEndian.Uint16(page[at+3:]))

		require.GreaterOrEqual(t, address, plus3CacheAt, "picture %d", number)
		require.Less(t, address+plus3PictureHeader+size, plus3CacheTop, "picture %d", number)

		// The interpreter copies the header out of the buffer and then
		// uncompresses from just after it.
		from := address - plus3CacheAt
		found[number] = page[from : from+plus3PictureHeader+size]

		at += plus3CacheEntry
	}

	require.Len(t, found, len(cached))

	// And every one of them is the file it would have been on its own, header
	// and data, byte for byte.
	for _, one := range cached {
		require.Equal(t, one.body, found[one.number], "picture %d", one.number)
	}
}

// The buffer is there to take pictures out of the directory, so what it takes
// are the ones that cost least: smallest first, until the page is full.
func TestTheSmallestPicturesAreTheOnesBuffered(t *testing.T) {
	pictures := screens(30)

	cached, loose, err := chooseCached(pictures)
	require.NoError(t, err)

	sizes := map[int]int{}
	for n, body := range pictures {
		one, err := plus3Picture(body)
		require.NoError(t, err)

		sizes[n] = len(one)
	}

	biggest := 0
	for _, one := range cached {
		biggest = max(biggest, sizes[one.number])
	}

	for _, n := range loose {
		require.GreaterOrEqual(t, sizes[n], biggest,
			"picture %d stayed out while a bigger one went in", n)
	}

	// Nothing is lost between the two: every picture is in one place or the
	// other, and never in both.
	require.Len(t, cached, len(pictures)-len(loose))
}

// The interpreter's buffered-picture path uncompresses with interrupts on, and
// a build that uses the buffer gets a DI put in front of it.
//
// Both interpreters we ship are checked, in both languages, because the patch
// is found by the bytes around it rather than by an address — and the only
// thing that makes that safe is that the bytes are there and are there once.
func TestTheBufferedPathGetsItsDI(t *testing.T) {
	for _, language := range []string{"DSP3S3.BIN", "DSP3E3.BIN"} {
		t.Run(language, func(t *testing.T) {
			before, err := os.ReadFile(filepath.Join("..", "runtime", "assets", "zxplus3", language))
			require.NoError(t, err)

			const origin, database = 0x6000, 0x8400

			after, err := plus3DisableInterrupts(before, origin, database)
			require.NoError(t, err)
			require.Len(t, after, len(before)+plus3StubSize)

			// The call now goes to the stub, and the stub does what the call
			// did with interrupts off.
			at := bytes.Index(before, plus3UncompressBefore) + plus3CallAt
			stub := origin + len(before)

			require.Equal(t, uint16(stub), binary.LittleEndian.Uint16(after[at:]))

			was := binary.LittleEndian.Uint16(before[at:])
			require.Equal(t,
				[]byte{0xF3, 0xCD, byte(was), byte(was >> 8), 0xC9},
				after[len(before):],
				"DI, the call it took over, and RET")

			// And nothing else moved, which is the whole reason for appending
			// rather than inserting: every address in there still means what
			// it meant.
			require.Equal(t, before[:at], after[:at])
			require.Equal(t, before[at+2:], after[at+2:len(before)])
		})
	}
}

// What it will not do is guess. An image without the path, or without room
// after it, is said so rather than patched somewhere plausible.
func TestWhatTheDICannotBePutInto(t *testing.T) {
	plain := make([]byte, 100)

	_, err := plus3DisableInterrupts(plain, 0x6000, 0x8400)
	require.ErrorContains(t, err, "no buffered-picture path")

	withPath := append([]byte(nil), plus3UncompressBefore...)
	withPath = append(withPath, 0xB3, 0x65)
	withPath = append(withPath, plus3UncompressAfter...)

	_, err = plus3DisableInterrupts(withPath, 0x8000, 0x8000+len(withPath))
	require.ErrorContains(t, err, "no room for the stub")

	twice := append(append([]byte(nil), withPath...), withPath...)
	_, err = plus3DisableInterrupts(twice, 0x6000, 0x8400)
	require.ErrorContains(t, err, "two places that look alike")
}

// An adventure whose pictures all fit leaves no files of its own behind, and
// one with none at all has no buffer to write.
func TestTheEdgesOfTheBuffer(t *testing.T) {
	cached, loose, err := chooseCached(screens(3))
	require.NoError(t, err)
	require.Len(t, cached, 3)
	require.Empty(t, loose)

	cached, loose, err = chooseCached(nil)
	require.NoError(t, err)
	require.Empty(t, cached)
	require.Empty(t, loose)

	file, err := plus3Cache(nil)
	require.NoError(t, err)
	require.Empty(t, file, "no pictures is no file, not an empty one")
}
