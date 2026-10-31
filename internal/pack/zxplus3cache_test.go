package pack

import (
	"encoding/binary"
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
