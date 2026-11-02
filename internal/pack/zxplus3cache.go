package pack

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

// The buffered pictures of a +3 release, which DAAD calls DAAD.GRA.
//
// A picture on a +3 is a file of its own, opened by name when a PICTURE names
// it. That works and it is what the interpreter does first, but it spends a
// directory entry per picture and a +3DOS directory has 64 of them. La Aventura
// Original's second part is 69 pictures and four other files, so nine of them
// had nowhere to be named — with 81 KB of the disk still empty, because what
// ran out was names and not room.
//
// The interpreter has the answer already and we were not using it. At start-up
// it looks for DAAD.GRA, and if it is there it reads the whole thing into page
// 1 and keeps it; every PICTURE then looks in there before it goes near the
// disk. One directory entry for as many pictures as fit, and the ones that fit
// draw without a disk access at all.
//
// All of this is read out of PLUS3.ASM — Plus3Initialise, LoadCachedImages and
// CheckCachedImages — which is the +3 interpreter's own source.
const (
	// plus3CacheAt is where the file's contents land: page 1, right after the
	// 6912-byte picture buffer at 0xC200. The offsets in the index are
	// absolute addresses in that page, not offsets into the file, so this
	// number is part of the format and not a detail of the loader.
	plus3CacheAt = 0xDD00
	// plus3CacheTop is one past the last byte of the page.
	plus3CacheTop = 0x10000
	// plus3CacheRoom is everything from the one to the other: index and
	// pictures together.
	plus3CacheRoom = plus3CacheTop - plus3CacheAt
	// plus3CacheEntry is one row of the index.
	//
	// PLUS3.ASM's own comment says three bytes — "one byte for the number of
	// image (locno) and then two that for the offset" — and it is wrong. The
	// code walks the table with "LD BC,5 / ADD HL,BC" and reads a size after
	// the offset. The code is what runs.
	plus3CacheEntry = 5
	// plus3CacheEnd is the picture number that closes the index. It is also
	// what the interpreter writes at plus3CacheAt when there is no file at
	// all, so that the same walk finds an empty table.
	plus3CacheEnd = 255
	// plus3CacheName is what the interpreter opens, and nothing else will do.
	plus3CacheName = "DAAD.GRA"
)

// The buffered path of the interpreter uncompresses with interrupts on, and it
// should not.
//
// Paging a bank in ends with an EI — the routine is DI, out to 0x7FFD, EI, RET
// — so whatever runs next runs interruptible with a bank other than the usual
// one at 0xC000. The interpreter knows: the path that reads a picture out of
// its own file puts a DI in front of the decompressor and says why, in
// PLUS3.ASM's own words, "the pageinRAMx calls enable interrupts, we must
// ensure they are not activated while UNCOMPRESS runs".
//
// CheckCachedImages, which is the path a buffered picture takes, does not. It
// never mattered while nothing used DAAD.GRA. It matters now, and it fails the
// way a timing bug fails: most pictures most of the time, and then a screen
// that comes out black or a machine that resets, depending on where the 50 Hz
// interrupt lands inside the decompression.
//
// So the image gets a DI, in the one place it is missing.
const (
	// plus3UncompressCall is the instruction we move: the decompressor being
	// called with the buffer already in DE. The bytes around it are what make
	// it findable without an address — a five-byte LDIR of the picture's
	// header, then DE pointing at the picture buffer at 0xC200, then the call,
	// then the POP that drops the caller's return address.
	//
	//	01 05 00      ld bc,5
	//	ed b0         ldir
	//	c1            pop bc
	//	11 00 c2      ld de,0xC200
	//	cd lo hi      call UNCOMPRESS   <- this one
	//	c1            pop bc
	plus3CallAt = 10
	// plus3StubSize is DI, the call we took over, and RET.
	plus3StubSize = 5
)

var (
	plus3UncompressBefore = []byte{0x01, 0x05, 0x00, 0xED, 0xB0, 0xC1, 0x11, 0x00, 0xC2, 0xCD}
	plus3UncompressAfter  = []byte{0xC1}
)

// plus3DisableInterrupts puts the missing DI in front of the decompressor, by
// sending the call through a stub of three instructions appended to the image.
//
// Appending is what makes this safe to do to somebody else's binary: nothing
// inside it moves, so every address in it still means what it meant. The stub
// goes in the gap between the end of the interpreter and the database, which on
// a +3 is the best part of a kilobyte.
//
// origin is where the image loads, and limit is the first address it must not
// reach — the database's.
func plus3DisableInterrupts(image []byte, origin, limit int) ([]byte, error) {
	at := -1

	for from := 0; ; {
		found := bytes.Index(image[from:], plus3UncompressBefore)
		if found < 0 {
			break
		}

		found += from
		from = found + 1

		after := found + len(plus3UncompressBefore) + 2
		if after+len(plus3UncompressAfter) > len(image) ||
			!bytes.Equal(image[after:after+len(plus3UncompressAfter)], plus3UncompressAfter) {
			continue
		}

		if at >= 0 {
			return nil, errors.New(
				"the interpreter calls the decompressor from two places that look alike, " +
					"so this cannot tell which one buffers pictures")
		}

		at = found
	}

	if at < 0 {
		return nil, errors.New(
			"this interpreter has no buffered-picture path to fix: it does not call the " +
				"decompressor with the picture buffer in DE")
	}

	stub := origin + len(image)
	if stub+plus3StubSize > limit {
		return nil, fmt.Errorf(
			"no room for the stub: the interpreter ends at 0x%04X and the database starts at 0x%04X",
			stub, limit)
	}

	call := at + plus3CallAt
	was := int(image[call]) | int(image[call+1])<<8

	out := append([]byte(nil), image...)
	out[call], out[call+1] = byte(stub), byte(stub>>8)

	return append(out, 0xF3, 0xCD, byte(was), byte(was>>8), 0xC9), nil
}

// plus3Cached is one picture that went into the buffer.
type plus3Cached struct {
	number int
	body   []byte // the .128 as it would have been on its own: header and data
}

// plus3DirEntries is what a +3DOS directory holds, and plus3OtherFiles is how
// many of them a release always spends on things that are not pictures: the
// loader, the interpreter, the database and the graphics block. A loading
// screen is a fifth when there is one, and the buffer itself a sixth.
const (
	plus3DirEntries = 64
	plus3OtherFiles = 4
)

// chooseCached decides which pictures go into the buffer and which stay as
// files of their own.
//
// As few as will do, and smallest first.
//
// The buffer is there for one reason — a +3DOS directory holds 64 entries and
// an adventure can have more pictures than that — so an adventure whose
// pictures all fit gets no buffer at all, and one that overruns gets exactly
// enough of them moved to bring it back under. Smallest first because the
// measure is how many names it frees and not how much disk time it saves: a
// blank-ish screen compresses to a few dozen bytes, and the nine the second
// part of La Aventura Original needs come to under 400 of them.
//
// It used to fill the buffer to the top of the page, which is more pictures
// buffered and strictly more of them drawn without touching the disk. That is
// the better trade only if the buffered path is sound, and it is not: it
// decompresses with interrupts enabled, which plus3DisableInterrupts puts
// right, and something in it is still coming out as a screen full of flashing
// attributes. Until that is understood, the fewer pictures go through it the
// better — and an adventure like the first part, whose pictures all fit, goes
// back to not using it at all.
//
// Ties go to the lower number so that two builds of the same adventure are the
// same file.
func chooseCached(pictures map[int][]byte, others int) (cached []plus3Cached, loose []int, err error) {
	numbers := make([]int, 0, len(pictures))
	for n := range pictures {
		numbers = append(numbers, n)
	}

	sort.Ints(numbers)

	bodies := make(map[int][]byte, len(numbers))

	for _, n := range numbers {
		body, err := plus3Picture(pictures[n])
		if err != nil {
			return nil, nil, fmt.Errorf("picture %d: %w", n, err)
		}

		bodies[n] = body
	}

	order := append([]int(nil), numbers...)
	sort.SliceStable(order, func(i, j int) bool {
		return len(bodies[order[i]]) < len(bodies[order[j]])
	})

	// How many have to move for the names to fit. The buffer costs a name of
	// its own, so it only helps once it takes more than one picture with it.
	over := len(numbers) + others - plus3DirEntries
	if over > 0 {
		over++
	}

	used := 0

	for _, n := range order {
		if len(cached) >= over {
			break
		}

		// What this one would add: its bytes, its row of the index, and the
		// terminator the index always ends with.
		grown := used + len(bodies[n]) + plus3CacheEntry
		if grown+plus3CacheEntry > plus3CacheRoom {
			break
		}

		used = grown

		cached = append(cached, plus3Cached{number: n, body: bodies[n]})
	}

	in := make(map[int]bool, len(cached))
	for _, one := range cached {
		in[one.number] = true
	}

	for _, n := range numbers {
		if !in[n] {
			loose = append(loose, n)
		}
	}

	// Back into picture order, which is the order the index reads in and the
	// order a person expects to see.
	sort.Slice(cached, func(i, j int) bool { return cached[i].number < cached[j].number })

	return cached, loose, nil
}

// plus3Cache builds the DAAD.GRA file: two bytes saying how much follows, the
// index, and the pictures.
//
// The two bytes are the length of the rest, which is there because the
// interpreter cannot ask +3DOS how long a file is without reading past its end,
// and reading past the end of a file is the one thing +3DOS will not forgive.
func plus3Cache(cached []plus3Cached) ([]byte, error) {
	if len(cached) == 0 {
		return nil, nil
	}

	index := (len(cached) + 1) * plus3CacheEntry
	body := make([]byte, 0, index)
	data := make([]byte, 0, plus3CacheRoom)

	for _, one := range cached {
		if one.number >= plus3CacheEnd {
			return nil, fmt.Errorf(
				"picture %d cannot be buffered: %d closes the index",
				one.number, plus3CacheEnd)
		}

		at := plus3CacheAt + index + len(data)

		// The size is the compressed data, which is the file without the
		// header the interpreter copies out of it first.
		size := len(one.body) - plus3PictureHeader

		body = append(body, byte(one.number),
			byte(at), byte(at>>8),
			byte(size), byte(size>>8))

		data = append(data, one.body...)
	}

	body = append(body, plus3CacheEnd, 0, 0, 0, 0)

	if len(body)+len(data) > plus3CacheRoom {
		return nil, fmt.Errorf(
			"the buffered pictures come to %d bytes and page one has room for %d",
			len(body)+len(data), plus3CacheRoom)
	}

	rest := len(body) + len(data)

	out := make([]byte, 0, rest+2)
	out = append(out, byte(rest), byte(rest>>8))
	out = append(out, body...)

	return append(out, data...), nil
}
