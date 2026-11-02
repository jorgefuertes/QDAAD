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
// What is the same whatever the geometry.
const (
	dataSides      = 1
	dataSectorSize = 512
	dataEntrySize  = 32
	// dataRecordSize is CP/M's own unit, which is not the sector and not the
	// block. A directory entry counts its last extent in records.
	dataRecordSize = 128
	// dataFiller is what an unwritten byte is on a freshly formatted disk.
	dataFiller = 0xE5
	// dataGap3 is the gap the formatter leaves between sectors. It is in the
	// track header because a real controller needs it; nothing here does.
	dataGap3 = 0x4E
)

// geometry is a disk format: how the sectors are laid out and how the
// filesystem on top of them is cut up.
//
// There are two, and they are not interchangeable. The parameters are not free
// either — a disk with any of them wrong is a disk the machine reads as
// unformatted, however well-formed the file is — so both are read off disks
// known to work rather than reasoned about.
type geometry struct {
	tracks        int
	sectorsTrack  int
	firstSectorID byte
	// reserved is how many tracks the filesystem leaves alone in front of its
	// data area, which is CP/M's OFF.
	reserved int
	// blockSize is the unit a file is allocated in. A file of one byte takes a
	// whole one.
	blockSize int
	// dirBlocks is how many blocks the directory takes, and so how many names
	// the disk can hold: a block of 1 KiB is 32 entries.
	dirBlocks int
	// spec says whether the first sector of the reserved track carries a
	// +3DOS disk specification. Without reserved tracks there is nowhere to
	// put one, because that sector is the first block of the directory.
	spec bool
	// diskType is the specification's first byte, and what makes +3DOS read
	// the rest of it rather than take the sector for directory.
	diskType byte
}

func (g geometry) dirEntries() int { return g.dirBlocks * g.blockSize / dataEntrySize }

// dataGeometry is the CPC's DATA format, which a +3 reads too.
//
// They are not two formats that happen to agree: the +3 took the CPC's 3-inch
// drive and its DATA format with it. It is recognised by its sector IDs rather
// than by anything written on it, which is why it needs no reserved track and
// has none — the directory is in the very first sector.
//
// Nothing in CP/M makes that obvious — a PCW reserves one and an AMSDOS disk
// reserves two — so it was read off a disk known to work. DAAD's own +3 disk
// has "PART1   SDG" in track 0's first sector, 512 bytes into the image.
var dataGeometry = geometry{
	tracks: 40, sectorsTrack: 9, firstSectorID: 0xC1,
	reserved: 0, blockSize: 1024, dirBlocks: 2,
}

// plus3Geometry is the one La Aventura Original's own disk uses, which is the
// PCW and +3 format proper: sector IDs from 1, one reserved track, and a disk
// specification in the first sector of it telling +3DOS the rest.
//
// The specification is why this exists. A DATA disk is 64 names and there is no
// arguing with it, because nothing on the disk says how big the directory is —
// the machine knows, from the sector IDs. A disk of this format says, in the
// byte at +7 of its first sector, and +3DOS believes it. Four blocks instead of
// two is 128 names for two kilobytes of a disk with eighty spare, and that is
// the difference between an adventure of 69 pictures fitting and not.
//
// The bytes are the original's, read out of it, with that one changed:
//
//	00 00 28 09 02 01 03 02 2A 52
//	|  |  |  |  |  |  |  |  |  '- gap when formatting
//	|  |  |  |  |  |  |  |  '---- gap when reading and writing
//	|  |  |  |  |  |  |  '------- directory blocks: 2 there, 4 here
//	|  |  |  |  |  |  '---------- block size, as log2 minus seven: 1 KiB
//	|  |  |  |  |  '------------- reserved tracks
//	|  |  |  |  '---------------- sector size, same reading: 512
//	|  |  |  '------------------- sectors per track
//	|  |  '---------------------- tracks per side
//	|  '------------------------- sides, and the order they are written in
//	'---------------------------- disk type
var plus3Geometry = geometry{
	tracks: 40, sectorsTrack: 9, firstSectorID: 0x01,
	reserved: 1, blockSize: 1024, dirBlocks: 4,
	spec: true, diskType: 0,
}

// DataDisk writes a 180 KiB disk image in the CPC's DATA format, which a
// Spectrum +3 reads too.
//
// DAAD's own CPC disk is the other one, SYSTEM format — sector IDs from 0x41,
// two reserved tracks and the sectors interleaved on the track — which is
// bootable with |CPM and nine kilobytes smaller. DATA is what this writes:
// nothing on a CPC here needs to boot from CP/M, and a disk without reserved
// tracks is a disk with more room on it.
//
// A +3 release does not come through here any more. It wants the directory of
// 128 names that only a disk specification can ask for, and a DATA disk has
// nowhere to put one. See Plus3Disk.
func DataDisk(files []File) ([]byte, error) {
	return diskImage(dataGeometry, files)
}

// Plus3Disk writes a 177 KiB disk image in the +3's own format, specification
// and all.
//
// It costs a track against DataDisk — the reserved one the specification lives
// in — and buys the thing that track says: a directory of 128 names instead of
// 64. For an adventure that is pictures-as-files, names are what runs out long
// before room does.
func Plus3Disk(files []File) ([]byte, error) {
	return diskImage(plus3Geometry, files)
}

// Plus3Names is how many files a disk Plus3Disk writes will hold.
//
// It is here rather than written down again where it is needed, because it is a
// consequence of the geometry and not a fact of its own: change the directory
// blocks and this follows.
func Plus3Names() int { return plus3Geometry.dirEntries() }

// diskImage writes the files given onto a disk of the geometry given.
//
// They go on in the order given, which is the order the directory lists them in
// and so the order a CAT shows. Nothing is sorted: a release has a loader that
// ought to be first.
func diskImage(g geometry, files []File) ([]byte, error) {
	// The data area, sized by the track and not by the block, because that is
	// how it is written out. What the filesystem can hand to a file is whole
	// blocks of it, and nine sectors a track is four and a half of them, so a
	// disk with an odd number of reserved tracks ends on half a block that
	// nothing can be put in. 175 blocks of the 175.5 there are.
	data := make([]byte, (g.tracks-g.reserved)*g.sectorsTrack*dataSectorSize)

	for i := range data {
		data[i] = dataFiller
	}

	// The directory lives in the first blocks of the data area and starts out
	// empty, which on a CP/M disk means filled with 0xE5 like everything else.
	blocks := len(data) / g.blockSize
	entries := g.dirEntries()

	next := g.dirBlocks
	dir := 0

	for _, f := range files {
		name, ext, err := dataName(f.Name)
		if err != nil {
			return nil, err
		}

		want := (len(f.Data) + g.blockSize - 1) / g.blockSize

		if next+want > blocks {
			return nil, fmt.Errorf(
				"%s does not fit: %d blocks wanted and %d left of the %d on the disk",
				f.Name, want, blocks-next, blocks)
		}

		copy(data[next*g.blockSize:], f.Data)

		// One entry per sixteen blocks, because an entry has room for sixteen
		// block numbers and no more. A file longer than 16 KiB is several
		// entries — "extents" — numbered from zero. Every file in a DAAD
		// release is well inside one, but a two-part adventure's graphics are
		// not always going to be.
		const blocksPerExtent = 16

		recordsPerExtent := blocksPerExtent * g.blockSize / dataRecordSize

		// Extents are counted from the blocks and not from the length, so that
		// a file of zero bytes still gets the one entry that says it exists.
		extents := max((want+blocksPerExtent-1)/blocksPerExtent, 1)

		for extent := range extents {
			if dir >= entries {
				return nil, fmt.Errorf("the directory is full at %d entries", entries)
			}

			first := extent * blocksPerExtent

			// The record count is of this extent alone, so every extent but
			// the last is full and the last holds what is left over.
			left := len(f.Data) - first*g.blockSize
			records := min(max((left+dataRecordSize-1)/dataRecordSize, 0), recordsPerExtent)

			entry := data[dir*dataEntrySize:][:dataEntrySize]

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

			for i := range min(want-first, blocksPerExtent) {
				entry[16+i] = byte(next + first + i)
			}

			dir++
		}

		next += want
	}

	return dataImage(g, data), nil
}

// dataName splits a name into the eight and three characters a CP/M directory
// holds, space-padded and upper case.
//
// The high bit of each character of the extension is a file attribute on CP/M —
// read-only, system, archived — and leaving them clear is what makes a plain
// file. Nothing here sets them, so a name whose characters would need the high
// bit is refused rather than quietly turned into an attribute.
func dataName(name string) (eight, three []byte, err error) {
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

// dataImage wraps the data area in the DSK file an emulator reads, with the
// reserved tracks in front of it.
func dataImage(g geometry, data []byte) []byte {
	trackSize := dskTrackHeader + g.sectorsTrack*dataSectorSize

	out := make([]byte, dskHeaderSize)

	copy(out, "MV - CPCEMU Disk-File\r\nDisk-Info\r\n")
	copy(out[0x22:], "QDAAD")
	out[0x30] = byte(g.tracks)
	out[0x31] = dataSides
	out[0x32] = byte(trackSize)
	out[0x33] = byte(trackSize >> 8)

	for track := range g.tracks {
		head := make([]byte, dskTrackHeader)

		copy(head, trackSignature+"\r\n")
		head[0x10] = byte(track)
		head[0x11] = 0 // side
		head[0x14] = 2 // 128 << 2 is 512
		head[0x15] = byte(g.sectorsTrack)
		head[0x16] = dataGap3
		head[0x17] = dataFiller

		for sector := range g.sectorsTrack {
			at := 0x18 + sector*8
			head[at] = byte(track)
			head[at+1] = 0
			head[at+2] = g.firstSectorID + byte(sector)
			head[at+3] = 2
			// The two FDC status bytes stay zero: nothing here is a read error.
		}

		out = append(out, head...)

		// A reserved track is written as a formatted but empty one rather than
		// left out, because the filesystem not using it does not mean the drive
		// will not read it — and on this format the first sector of the first
		// one is the specification, which is the whole point.
		body := make([]byte, g.sectorsTrack*dataSectorSize)

		if track < g.reserved {
			for i := range body {
				body[i] = dataFiller
			}

			if track == 0 && g.spec {
				copy(body, plus3Spec(g))
			}
		} else {
			copy(body, data[(track-g.reserved)*len(body):])
		}

		out = append(out, body...)
	}

	return out
}

// plus3Spec writes the disk specification: the first sixteen bytes of the first
// sector of track 0, which is how a disk tells +3DOS its own shape instead of
// being recognised by its sector numbering.
//
// Only the first sixteen bytes mean anything. What the rest is matters all the
// same, because of what the +3 does with this sector before +3DOS ever sees it:
// its boot menu reads all 512 and, if they sum to 3 modulo 256, takes the disk
// for bootable and *runs* them. So the sector is left as formatted filler and
// the sum checked — a disk of ours that fell on 3 by accident would be a disk
// the +3 ran 512 bytes of 0xE5 from, and 0xE5 is RST 0x20.
//
// Our release is not booted that way in any case: it is loaded from the menu's
// own Loader option, which is LOAD "DISK", and that works bootable or not.
func plus3Spec(g geometry) []byte {
	spec := make([]byte, dataSectorSize)

	for i := range spec {
		spec[i] = dataFiller
	}

	// Sizes are written as a power of two, counted from 128 bytes: a 512-byte
	// sector is 2 and a 1 KiB block is 3.
	spec[0] = g.diskType
	spec[1] = 0 // one side, and so no question of what order the sides go in
	spec[2] = byte(g.tracks)
	spec[3] = byte(g.sectorsTrack)
	spec[4] = log2from128(dataSectorSize)
	spec[5] = byte(g.reserved)
	spec[6] = log2from128(g.blockSize)
	spec[7] = byte(g.dirBlocks)

	// The gaps, which are the original disk's. Nothing here reads them — they
	// are for a controller talking to a real drive — but a disk that gives a
	// gap no +3 would is a disk written for a machine other than the +3.
	spec[8] = 0x2A // reading and writing
	spec[9] = 0x52 // formatting

	// Bytes 10 to 15 are reserved and stay as filler.

	sum := 0
	for _, b := range spec {
		sum += int(b)
	}

	// Being bootable is not a thing to become by arithmetic. If the filler ever
	// adds up to the signature, one of the reserved bytes moves until it does
	// not: they mean nothing to +3DOS, and the sector meaning nothing to the
	// boot menu is the point.
	for sum%256 == plus3Bootable {
		spec[14]++
		sum++
	}

	return spec
}

// plus3Bootable is what the 512 bytes of track 0 sector 1 add up to, modulo
// 256, on a disk the +3's boot menu will load and execute.
const plus3Bootable = 3

// log2from128 writes a size the way a disk specification does: how many times
// it doubles 128.
func log2from128(size int) byte {
	out := byte(0)
	for n := size; n > dataRecordSize; n >>= 1 {
		out++
	}

	return out
}
