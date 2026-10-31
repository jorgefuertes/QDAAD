package pack

import (
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

// plus3Cached is one picture that went into the buffer.
type plus3Cached struct {
	number int
	body   []byte // the .128 as it would have been on its own: header and data
}

// chooseCached decides which pictures go into the buffer and which stay as
// files of their own.
//
// Smallest first, which is what the problem asks for: the buffer is there to
// take pictures out of the directory, so the measure of a good choice is how
// many of them it rescues and not how much disk time it saves. A blank-ish
// screen compresses to a few dozen bytes, so the first several cost almost
// nothing — ten of La Aventura Original's second part come to 431 bytes.
//
// Ties go to the lower number so that two builds of the same adventure are the
// same file.
func chooseCached(pictures map[int][]byte) (cached []plus3Cached, loose []int, err error) {
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

	used := 0

	for _, n := range order {
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
