package decompiler

import (
	"sort"

	"github.com/jorgefuertes/QDAAD/internal/media"
)

// The MSX editions do not put a database on the disk where it can be found. The
// disk is an ordinary MS-DOS one and its files are BLOAD images — the header the
// MSX BASIC loader understands: a marker byte and three addresses, saying where
// the bytes go, where they end and what to call once they are there.
//
// Nothing readable is at those addresses. Every part loads into the same buffer
// at 0x9000 and the routine at the end of it, the one the header names, moves
// what was loaded down into the RAM the game really runs in — which BASIC cannot
// use, and which the loader pages in around each copy:
//
//	CF00: f3           di
//	CF01: 3a 3f e0     ld a,(0xe03f)     ; the slot the game lives in
//	CF04: d3 a8        out (0xa8),a      ; page it in
//	CF0C: 21 00 90     ld hl,0x9000      ; from the buffer
//	CF0F: 11 00 01     ld de,0x0100      ; to where it belongs
//	CF12: 01 00 3f     ld bc,0x3f00
//	CF15: ed b0        ldir
//	CF17: 3a 40 e0     ld a,(0xe040)
//	CF1A: d3 a8        out (0xa8),a      ; and page BASIC back
//
// So the database is never on the disk in one piece and never at the address it
// is loaded to. Putting the copies back together is what makes it findable, and
// the copies say where they go themselves.
const (
	bloadMarker = 0xFE
	bloadHeader = 7       // the marker, then the start, end and entry addresses
	msxMemory   = 0x10000 // what the copies are laid out in
)

// msxChunk is one BLOAD file: where it was loaded, and the bytes of it.
type msxChunk struct {
	name string
	load int
	body []byte
}

// msxCopy is one block move a chunk performs once it is loaded.
type msxCopy struct {
	from, to, size int
}

// msxChunks picks the BLOAD files out of what a disk holds. Anything else on it
// — the BASIC loader, a title screen — is left alone.
func msxChunks(files []media.File) []msxChunk {
	var out []msxChunk

	for _, f := range files {
		if len(f.Data) <= bloadHeader || f.Data[0] != bloadMarker {
			continue
		}

		out = append(out, msxChunk{
			name: f.Name,
			load: int(f.Data[1]) | int(f.Data[2])<<8,
			body: f.Data[bloadHeader:],
		})
	}

	return out
}

// copies finds the block moves in a chunk, by the shape of the instructions
// rather than by where they sit: `ld hl,nn / ld de,nn / ld bc,nn / ldir`.
//
// Reading them out of the code is what keeps this from being guesswork. The
// destination is not written down anywhere else, it is not the same for every
// chunk, and it is not the address the chunk loads to.
func (c msxChunk) copies() []msxCopy {
	const moveSize = 11

	var out []msxCopy

	word := func(at int) int { return int(c.body[at]) | int(c.body[at+1])<<8 }

	for at := 0; at+moveSize <= len(c.body); at++ {
		if c.body[at] != 0x21 || c.body[at+3] != 0x11 || c.body[at+6] != 0x01 ||
			c.body[at+9] != 0xED || c.body[at+10] != 0xB0 {
			continue
		}

		from, to, size := word(at+1), word(at+4), word(at+7)

		// A move that reads outside the chunk or writes outside the machine is
		// not one of these: the pattern is common enough to turn up by chance.
		if start := from - c.load; start >= 0 && size > 0 &&
			start+size <= len(c.body) && to+size <= msxMemory {
			out = append(out, msxCopy{from: from, to: to, size: size})
		}
	}

	return out
}

// msxSets groups the chunks into the loads they belong to.
//
// A disk holds more than one: Chichén Itzá keeps both parts of the adventure on
// the same one, as CIA1 to CIA7 and CIB1 to CIB7. They cannot be laid out
// together — both write a part to 0x0100 and the second would bury the first —
// so what tells them apart is their names, which differ only in the number that
// says which piece of its own load each is.
// The directory names them in whatever order they were written, so each set is
// put back into the order its numbers give: that is the order the loader asks
// for them, and where two chunks write to the same place it decides which wins.
func msxSets(chunks []msxChunk) [][]msxChunk {
	split := func(name string) (string, int) {
		at := len(name)
		for at > 0 && name[at-1] >= '0' && name[at-1] <= '9' {
			at--
		}

		piece := 0
		for _, c := range name[at:] {
			piece = piece*10 + int(c-'0')
		}

		return name[:at], piece
	}

	stem := func(name string) string { s, _ := split(name); return s }

	var order []string

	sets := map[string][]msxChunk{}

	for _, c := range chunks {
		s := stem(c.name)
		if _, seen := sets[s]; !seen {
			order = append(order, s)
		}

		sets[s] = append(sets[s], c)
	}

	out := make([][]msxChunk, 0, len(order))

	for _, s := range order {
		set := sets[s]

		sort.SliceStable(set, func(i, j int) bool {
			_, a := split(set[i].name)
			_, b := split(set[j].name)

			return a < b
		})

		out = append(out, set)
	}

	return out
}

// msxImage lays one set of copies out where they go and gives back the memory
// the game would have been running in.
//
// The chunks are taken in the order the disk names them, which is the order the
// loader asks for them, so where two write to the same place the later one wins
// — as it would have on the machine.
func msxImage(chunks []msxChunk) []byte {
	memory := make([]byte, msxMemory)
	wrote := false

	for _, c := range chunks {
		for _, m := range c.copies() {
			copy(memory[m.to:], c.body[m.from-c.load:m.from-c.load+m.size])

			wrote = true
		}
	}

	if !wrote {
		return nil
	}

	return memory
}
