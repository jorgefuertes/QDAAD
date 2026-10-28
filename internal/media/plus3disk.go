package media

import (
	"fmt"
	"strings"
)

// Writing a +3 disk, which is two formats one inside the other.
//
// The outer one is the DSK file every emulator reads: a header saying how many
// tracks there are and how big each is, then the tracks, each with a header of
// its own listing the sectors it holds. It describes a floppy rather than
// holding a filesystem, which is why a damaged disk can be stored in one. Our
// own reader opens these already — see dsk.go — and this writes what it reads.
//
// The inner one is +3DOS, which is CP/M 2.2 with Amstrad's numbers. A directory
// of 32-byte entries at the front of the data area, files allocated in blocks
// of 1 KiB, and the block numbers of a file listed in its own directory entry.
// Nothing in it is indexed or ordered: finding a file means reading the whole
// directory, which is why 64 entries was thought generous.
//
// The parameters are the ones the +3 ROM uses for the format it writes itself.
// They are not free: a disk with any of them different is a disk the +3 will
// read as unformatted, however well-formed this file is.
const (
	plus3Tracks       = 40
	plus3Sides        = 1
	plus3SectorsTrack = 9
	plus3SectorSize   = 512
	// plus3FirstSectorID is where the +3's sector numbering starts. A CPC disk
	// uses 0xC1 upwards for a data-format disk and this is the same scheme.
	plus3FirstSectorID = 0xC1
	// plus3Reserved is how many tracks the filesystem leaves alone before its
	// data area, which is CP/M's OFF. On a +3 it is none: the directory is in
	// the first sector of track 0.
	//
	// Nothing in CP/M makes that obvious — a PCW reserves one and an AMSDOS
	// disk reserves two — so it is read off a disk known to work. DAAD's own
	// +3 disk has "PART1   SDG" in track 0's first sector, 512 bytes into the
	// image. A disk written with one reserved track formats perfectly and
	// CATs as empty, because the +3 looks at track 0, finds the 0xE5 of a
	// freshly formatted disk, and believes it.
	plus3Reserved = 0
	// plus3BlockSize is the unit a file is allocated in. A file of one byte
	// takes a whole one.
	plus3BlockSize = 1024
	// plus3DirBlocks is how many blocks the directory takes, and 64 entries is
	// what that comes to.
	plus3DirBlocks  = 2
	plus3DirEntries = plus3DirBlocks * plus3BlockSize / plus3EntrySize
	plus3EntrySize  = 32
	// plus3RecordSize is CP/M's own unit, which is not the sector and not the
	// block. A directory entry counts its last extent in records.
	plus3RecordSize = 128
	// plus3Filler is what an unwritten byte is on a freshly formatted disk.
	plus3Filler = 0xE5
	// plus3Gap3 is the gap the formatter leaves between sectors. It is in the
	// track header because a real controller needs it; nothing here does.
	plus3Gap3 = 0x4E
)

// Plus3Disk writes a 180 KiB +3 disk image holding the files given.
//
// The files go on in the order given, which is the order the directory lists
// them in and so the order a CAT shows. Nothing is sorted: a release has a
// loader that ought to be first.
func Plus3Disk(files []File) ([]byte, error) {
	// The data area, as a run of blocks. The directory lives in the first two
	// and starts out empty, which on a CP/M disk means filled with 0xE5.
	dataBlocks := (plus3Tracks-plus3Reserved)*plus3SectorsTrack*plus3SectorSize/plus3BlockSize
	data := make([]byte, dataBlocks*plus3BlockSize)

	for i := range data {
		data[i] = plus3Filler
	}

	next := plus3DirBlocks
	dir := 0

	for _, f := range files {
		name, ext, err := plus3Name(f.Name)
		if err != nil {
			return nil, err
		}

		blocks := (len(f.Data) + plus3BlockSize - 1) / plus3BlockSize

		if next+blocks > dataBlocks {
			return nil, fmt.Errorf(
				"%s does not fit: %d blocks wanted and %d left of the %d on a +3 disk",
				f.Name, blocks, dataBlocks-next, dataBlocks)
		}

		copy(data[next*plus3BlockSize:], f.Data)

		// One entry per sixteen blocks, because an entry has room for sixteen
		// block numbers and no more. A file longer than 16 KiB is several
		// entries — "extents" — numbered from zero. Every file in a DAAD
		// release is well inside one, but a two-part adventure's graphics are
		// not always going to be.
		const (
			blocksPerExtent  = 16
			recordsPerExtent = blocksPerExtent * plus3BlockSize / plus3RecordSize
		)

		// Extents are counted from the blocks and not from the length, so that
		// a file of zero bytes still gets the one entry that says it exists.
		extents := max((blocks+blocksPerExtent-1)/blocksPerExtent, 1)

		for extent := range extents {
			if dir >= plus3DirEntries {
				return nil, fmt.Errorf("the directory is full at %d entries", plus3DirEntries)
			}

			first := extent * blocksPerExtent

			// The record count is of this extent alone, so every extent but
			// the last is full and the last holds what is left over.
			left := len(f.Data) - first*plus3BlockSize
			records := min(max((left+plus3RecordSize-1)/plus3RecordSize, 0), recordsPerExtent)

			entry := data[dir*plus3EntrySize:][:plus3EntrySize]

			// Cleared first, and not only for tidiness. The directory starts
			// out as 0xE5 filler like the rest of a formatted disk, so every
			// byte of an entry that is not written stays 0xE5 — and 0xE5 in an
			// unused block slot is block 229, which is off the end of a 175
			// block disk. Our own reader throws away a directory with one of
			// those in it, on the grounds that it cannot be a directory, and it
			// is right to.
			clear(entry)

			entry[0] = 0 // user zero, which is the only one anything here uses
			copy(entry[1:9], name)
			copy(entry[9:12], ext)
			entry[12] = byte(extent)
			entry[15] = byte(records)

			for i := range min(blocks-first, blocksPerExtent) {
				entry[16+i] = byte(next + first + i)
			}

			dir++
		}

		next += blocks
	}

	return plus3Image(data), nil
}

// plus3Name splits a name into the eight and three characters a CP/M directory
// holds, space-padded and upper case.
//
// The high bit of each character of the extension is a file attribute on CP/M —
// read-only, system, archived — and leaving them clear is what makes a plain
// file. Nothing here sets them, so a name whose characters would need the high
// bit is refused rather than quietly turned into an attribute.
func plus3Name(name string) (eight, three []byte, err error) {
	base, ext, _ := strings.Cut(strings.ToUpper(name), ".")

	if base == "" || len(base) > 8 || len(ext) > 3 {
		return nil, nil, fmt.Errorf("%q is not a name a +3 disk can hold: eight and three", name)
	}

	for _, r := range base + ext {
		if r > 0x7E || r < 0x20 || strings.ContainsRune(`<>.,;:=?*[]`, r) {
			return nil, nil, fmt.Errorf("%q cannot be on a +3 disk: %q is not allowed in a name",
				name, r)
		}
	}

	return []byte(base + strings.Repeat(" ", 8-len(base))),
		[]byte(ext + strings.Repeat(" ", 3-len(ext))), nil
}

// plus3Image wraps the data area in the DSK file an emulator reads, with the
// reserved track in front of it.
func plus3Image(data []byte) []byte {
	trackSize := dskTrackHeader + plus3SectorsTrack*plus3SectorSize

	out := make([]byte, dskHeaderSize)

	copy(out, "MV - CPCEMU Disk-File\r\nDisk-Info\r\n")
	copy(out[0x22:], "QDAAD")
	out[0x30] = plus3Tracks
	out[0x31] = plus3Sides
	out[0x32] = byte(trackSize)
	out[0x33] = byte(trackSize >> 8)

	for track := range plus3Tracks {
		head := make([]byte, dskTrackHeader)

		copy(head, trackSignature+"\r\n")
		head[0x10] = byte(track)
		head[0x11] = 0 // side
		head[0x14] = 2 // 128 << 2 is 512
		head[0x15] = plus3SectorsTrack
		head[0x16] = plus3Gap3
		head[0x17] = plus3Filler

		for sector := range plus3SectorsTrack {
			at := 0x18 + sector*8
			head[at] = byte(track)
			head[at+1] = 0
			head[at+2] = byte(plus3FirstSectorID + sector)
			head[at+3] = 2
			// The two FDC status bytes stay zero: nothing here is a read error.
		}

		out = append(out, head...)

		// Track 0 is the one the filesystem does not use, so it is written as
		// a formatted but empty track rather than left out.
		body := make([]byte, plus3SectorsTrack*plus3SectorSize)

		if track < plus3Reserved {
			for i := range body {
				body[i] = plus3Filler
			}
		} else {
			copy(body, data[(track-plus3Reserved)*len(body):])
		}

		out = append(out, body...)
	}

	return out
}
