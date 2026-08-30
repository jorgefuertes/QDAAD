package media

import (
	"fmt"
	"sort"
	"strings"
)

// The CP/M filesystem, as AMSDOS wrote it on the Amstrad disks.
//
// Not every DSK carries one. Some of these disks were formatted for the game's
// own loader and hold nothing but a run of sectors, but others are ordinary
// CP/M volumes with a directory naming their files, and the two cannot be told
// apart from the container: both are DSK images of 40 tracks. So the directory
// is read, and the disk is taken to have a filesystem only if it comes out
// naming something.
//
// The directory sits at the front of the data area, 64 entries of 32 bytes.
// There is no size field: a file is described by one entry per 16 KB extent,
// each listing the blocks it occupies and how many 128-byte records of the last
// one are used, so a file has to be put back together from all of its extents.
const (
	cpmEntrySize    = 32
	cpmEntries      = 64 // the directory is two blocks
	cpmBlockSize    = 1024
	cpmRecordSize   = 128
	cpmBlocksPerExt = 16
	cpmDeleted      = 0xE5
	cpmMaxUser      = 0x0F
	cpmMaxRecords   = 128 // one extent is 16 KB, which is 128 records
)

// Fields of a directory entry.
const (
	cpmUser     = 0x00
	cpmName     = 0x01
	cpmNameLen  = 8
	cpmExtLen   = 3
	cpmExtentLo = 0x0C
	cpmExtentHi = 0x0E
	cpmRecords  = 0x0F
	cpmBlocks   = 0x10
)

// cpmFile is one extent of one file, before the extents are joined up.
type cpmExtent struct {
	name    string
	number  int
	records int
	blocks  []int
}

// cpmFiles reads the directory and returns the files it names, whole.
//
// Where the data area starts is not written down anywhere either. It depends on
// how the disk was formatted — the sector numbering says which, &C1 upwards for
// a data disk with no reserved tracks and &41 upwards for a system one with
// two — but rather than trust the numbering we try each in turn and keep the
// one that yields a directory, which is the same test that has to be made
// anyway to know whether there is a filesystem at all.
func cpmFiles(payload []byte, trackSize int) []File {
	for _, reserved := range []int{0, 1, 2} {
		start := reserved * trackSize
		if start+cpmEntries*cpmEntrySize > len(payload) {
			break
		}

		extents := cpmDirectory(payload[start:])
		if len(extents) == 0 {
			continue
		}

		// A file whose blocks fall outside the disk says the entries were not a
		// directory after all, which is a thing to move on from rather than to
		// report: a loader disk has sectors that read as entries here and there.
		files, err := cpmAssemble(extents, payload[start:])
		if err != nil {
			continue
		}

		if len(files) > 0 {
			return files
		}
	}

	return nil
}

// cpmDirectory reads the 64 entries, keeping the ones that describe a file.
//
// An entry that is not one has to be discarded rather than taken as the end of
// the directory: a disk that has had files deleted and written again has live
// entries after dead ones.
func cpmDirectory(area []byte) []cpmExtent {
	var extents []cpmExtent

	for i := range cpmEntries {
		at := i * cpmEntrySize
		if at+cpmEntrySize > len(area) {
			break
		}

		entry := area[at : at+cpmEntrySize]
		if entry[cpmUser] == cpmDeleted || entry[cpmUser] > cpmMaxUser {
			continue
		}

		name, ok := cpmName11(entry[cpmName : cpmName+cpmNameLen+cpmExtLen])
		if !ok {
			continue
		}

		records := int(entry[cpmRecords])
		if records == 0 || records > cpmMaxRecords {
			continue
		}

		extents = append(extents, cpmExtent{
			name:    name,
			number:  int(entry[cpmExtentLo]) + int(entry[cpmExtentHi])*32,
			records: records,
			blocks:  cpmBlockList(entry[cpmBlocks:]),
		})
	}

	return extents
}

// cpmAllowed says whether a character can appear in a CP/M name. The set is
// narrow — upper case, digits and a handful of symbols — and being strict about
// it is what keeps a disk formatted for its own loader from looking like one
// with a directory: sectors of program or picture data are full of characters
// that cannot be in a name.
func cpmAllowed(c rune) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == ' ':
		return true
	default:
		return strings.ContainsRune("$-_@#!&+", c)
	}
}

// cpmName11 turns the eleven bytes of a name into one, or says it is not a
// name. The top bit of each is a flag — read-only, system, archived — and has
// to come off before the byte is a character.
func cpmName11(raw []byte) (string, bool) {
	var name, extension strings.Builder

	for i, b := range raw {
		c := rune(b & 0x7F)
		if !cpmAllowed(c) {
			return "", false
		}

		if i < cpmNameLen {
			name.WriteRune(c)
		} else {
			extension.WriteRune(c)
		}
	}

	stem := strings.TrimRight(name.String(), " ")
	if stem == "" {
		return "", false
	}

	suffix := strings.TrimRight(extension.String(), " ")
	if suffix == "" {
		return stem, true
	}

	return stem + "." + suffix, true
}

// cpmBlockList reads the sixteen block numbers of an extent. They are one byte
// each on these disks, which hold well under 256 blocks; a zero means the slot
// is unused, and block zero is the directory itself, so it can never be one.
func cpmBlockList(raw []byte) []int {
	blocks := make([]int, 0, cpmBlocksPerExt)

	for i := range cpmBlocksPerExt {
		if i >= len(raw) {
			break
		}

		if raw[i] == 0 {
			continue
		}

		blocks = append(blocks, int(raw[i]))
	}

	return blocks
}

// cpmAssemble joins the extents of each file, in order, and cuts the result to
// the length the record counts give.
func cpmAssemble(extents []cpmExtent, area []byte) ([]File, error) {
	byName := map[string][]cpmExtent{}

	var order []string

	for _, e := range extents {
		if _, seen := byName[e.name]; !seen {
			order = append(order, e.name)
		}

		byName[e.name] = append(byName[e.name], e)
	}

	files := make([]File, 0, len(order))

	for _, name := range order {
		parts := byName[name]
		sort.SliceStable(parts, func(i, j int) bool { return parts[i].number < parts[j].number })

		var data []byte

		for _, e := range parts {
			extent := make([]byte, 0, len(e.blocks)*cpmBlockSize)

			for _, b := range e.blocks {
				at := b * cpmBlockSize
				if at+cpmBlockSize > len(area) {
					return nil, fmt.Errorf("%s: block %d falls outside the disk", name, b)
				}

				extent = append(extent, area[at:at+cpmBlockSize]...)
			}

			// The record count is what says where the file ends. Every extent
			// but the last is full, and the last one is only partly used.
			if want := e.records * cpmRecordSize; want < len(extent) {
				extent = extent[:want]
			}

			data = append(data, extent...)
		}

		files = append(files, File{Name: name, Data: data})
	}

	return files, nil
}
