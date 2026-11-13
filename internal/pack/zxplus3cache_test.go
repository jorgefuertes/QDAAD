package pack

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/zx0"

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

// overrunning is a picture count that will not fit in the directory, whatever
// the directory holds today.
//
// It used to be the literal 70, which was six over a disk of 64 names. The disk
// says 128 now and 70 of them fit with room to spare, so a test that asked for
// 70 and expected a buffer was testing nothing — it got an empty one and said
// so. Counted from the limit, it goes on meaning what it was written to mean.
func overrunning() int { return media.Plus3Names() + 6 }

// The directory the unit tests overrun, and how many pictures overrun it.
//
// Small on purpose. What the buffer takes depends on how far past the limit the
// pictures go and not on the limit itself, so eighteen against twelve is the
// same question as a hundred and thirty-four against a hundred and twenty-eight
// — and it is the one that does not put a hundred and thirty-four whole screens
// through the coder. A disk's real directory is media.Plus3Names(); the tests
// that need that one are the ones that go through ZXDisk.
const (
	fewNames    = 12
	fewOverruns = fewNames + 6
)

// overrun is the fixture the buffer tests share: more pictures than the
// directory holds, already sorted into the ones the buffer took and the ones
// that stayed files, with each picture's packaged size beside it.
//
// Shared because every one of those screens goes through the coder twice over
// — once for chooseCached to sort by size and once for the sizes here — and a
// whole screen is a quarter of a second. Nothing in it is written to, which is
// what makes handing the same maps to four tests safe rather than a trap.
var overrun = sync.OnceValue(func() overrunFixture {
	pictures := screens(fewOverruns)

	cached, loose, err := chooseCached(pictures, fewNames, plus3OtherFiles)
	if err != nil {
		return overrunFixture{err: err}
	}

	sizes := make(map[int]int, len(pictures))

	for n, body := range pictures {
		one, err := plus3Picture(body)
		if err != nil {
			return overrunFixture{err: err}
		}

		sizes[n] = len(one)
	}

	return overrunFixture{pictures: pictures, cached: cached, loose: loose, sizes: sizes}
})

type overrunFixture struct {
	pictures map[int][]byte
	cached   []plus3Cached
	loose    []int
	// sizes is what each picture comes to as a file, by picture number.
	sizes map[int]int
	err   error
}

// Reading the buffer back the way the Z80 does: walk the index five bytes at a
// time until the number is 255, and for a hit take the offset as an address in
// page one and the five bytes there as the picture's header.
//
// See CheckCachedImages in PLUS3.ASM. The point of doing it this way rather
// than against our own structs is that the format is the interpreter's, and a
// test that agrees with the writer but not with the reader proves nothing.
func TestTheBufferReadsBackTheWayTheInterpreterWalksIt(t *testing.T) {
	f := overrun()
	require.NoError(t, f.err)

	cached, loose := f.cached, f.loose
	require.NotEmpty(t, cached)
	require.NotEmpty(t, loose, "that many names do not all fit in the directory")

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
	f := overrun()
	require.NoError(t, f.err)

	pictures, cached, loose, sizes := f.pictures, f.cached, f.loose, f.sizes

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

// Every buffered picture decompresses back to the screen it came from.
//
// The walk is the interpreter's — five bytes an entry, the offset read as an
// address in page one, the five-byte header copied out and the rest handed to
// the decompressor — and what comes out the other end is compared with what
// went in. An index that points one byte out, or a size that counts the header
// twice, gives a screen that is almost right, and almost right on a Spectrum
// is a picture full of flashing attributes.
func TestEveryBufferedPictureComesBackOut(t *testing.T) {
	f := overrun()
	require.NoError(t, f.err)

	pictures, cached := f.pictures, f.cached
	require.NotEmpty(t, cached)

	file, err := plus3Cache(cached)
	require.NoError(t, err)

	page := file[2:]
	seen := 0

	for at := 0; page[at] != plus3CacheEnd; at += plus3CacheEntry {
		number := int(page[at])
		address := int(binary.LittleEndian.Uint16(page[at+1:]))
		size := int(binary.LittleEndian.Uint16(page[at+3:]))

		from := address - plus3CacheAt + plus3PictureHeader

		out, err := zx0.Decompress(page[from : from+size])
		require.NoError(t, err, "picture %d", number)

		require.Equal(t, screenSize, len(out), "picture %d is not a whole screen", number)

		// The screen the packer would have written on its own, which is the
		// source cropped to its own height.
		want, err := plus3Picture(pictures[number])
		require.NoError(t, err)
		require.Equal(t, want[plus3PictureHeader:], page[from:from+size],
			"picture %d is not the stream the packer makes", number)

		seen++
	}

	require.Equal(t, len(cached), seen)
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

// The buffer is for one thing, so an adventure that does not need it does not
// get one.
//
// This is the whole of the rule and it is worth stating on its own: the
// directory holds as many names as the disk's own specification says, a release
// spends four on things that are not pictures, and the buffer costs a fifth.
// Below that there is nothing to rescue, and a picture read out of its own file
// goes through the path that has always worked.
//
// The directory it overruns is a small one, and not the 128 a +3 disk holds.
//
// What decides how many pictures move is len(pictures) + others - names, so the
// answer turns on the distance past the limit and not on the limit: eighteen
// pictures against twelve names buffers the same eleven that a hundred and
// thirty-four against a hundred and twenty-eight do. Asking the question at
// twelve costs eighteen screens through the coder instead of a hundred and
// thirty-four, and a whole screen through the coder is a quarter of a second —
// see internal/zx0, and the shortest path it now takes.
//
// That the real disk holds 128 and that nothing shipped comes near it is a
// separate matter, and TestWhatThePlus3DirectoryHoldsToday says it without
// packaging anything.
func TestTheBufferIsOnlyThereWhenTheDirectoryWouldOverrun(t *testing.T) {
	const names = 12

	for _, tc := range []struct {
		pictures, cached int
		note             string
	}{
		{pictures: 3, cached: 0},
		{pictures: names - plus3OtherFiles, cached: 0, note: "these fit exactly, with nothing to rescue"},
		{pictures: names - plus3OtherFiles + 1, cached: 2, note: "one over, and the buffer costs a name of its own"},
		{pictures: names + 6, cached: 11},
	} {
		t.Run(fmt.Sprint(tc.pictures), func(t *testing.T) {
			cached, loose, err := chooseCached(screens(tc.pictures), names, plus3OtherFiles)
			require.NoError(t, err)

			require.Len(t, cached, tc.cached, tc.note)
			require.Len(t, loose, tc.pictures-tc.cached)

			used := len(loose) + plus3OtherFiles
			if len(cached) > 0 {
				used++
			}

			require.LessOrEqual(t, used, names, "the directory still overruns")
		})
	}
}

// What the disk holds today, and how far from it the adventures are.
//
// The buffer exists because of a limit that moved. These disks were written in
// the CPC's DATA format at first, which carries no specification at all: the
// directory is 64 names and there is no arguing with it, and La Aventura
// Original's second part wanted 73 — 69 pictures and four other files — with
// 81 KB of the disk still empty. What ran out was names and not room.
//
// Asking for 128 is sixteen bytes in the first sector of track 0, and between
// that and dropping the drawings nothing can ask for, part two came down to 46
// names and part one to 33. So nothing shipped needs the buffer, and 46 would
// have fitted in a DATA disk's 64 anyway.
//
// It is written down here because it is the kind of fact that stops being true
// quietly. The buffer stays — an adventure with a picture for each of a hundred
// locations is not a strange adventure — but a build of La Aventura Original
// reaching for it again would mean the geometry had moved, and that is worth
// hearing about from a test rather than from a disk full of flashing
// attributes.
func TestWhatThePlus3DirectoryHoldsToday(t *testing.T) {
	require.Equal(t, 128, media.Plus3Names(), "the +3 disk asks for four directory blocks")
	require.Equal(t, 64, media.DataNames(), "a CPC DATA disk has no say in it")

	// Part two of La Aventura Original as it is packaged: 41 pictures the
	// adventure can ask for, and the four files the loader opens.
	const partTwoPictures = 41

	require.LessOrEqual(t, partTwoPictures+plus3OtherFiles, media.Plus3Names(),
		"the adventure needs the buffer again, so something moved")
	require.LessOrEqual(t, partTwoPictures+plus3OtherFiles, media.DataNames(),
		"it would no longer fit the disk format these started out in")
}

// An adventure with no pictures at all has no buffer to write.
func TestTheEdgesOfTheBuffer(t *testing.T) {
	cached, loose, err := chooseCached(screens(3), fewNames, plus3OtherFiles)
	require.NoError(t, err)
	require.Empty(t, cached, "three names fit without help")
	require.Len(t, loose, 3)

	cached, loose, err = chooseCached(nil, fewNames, plus3OtherFiles)
	require.NoError(t, err)
	require.Empty(t, cached)
	require.Empty(t, loose)

	file, err := plus3Cache(nil)
	require.NoError(t, err)
	require.Empty(t, file, "no pictures is no file, not an empty one")
}
