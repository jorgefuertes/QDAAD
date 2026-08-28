package media

import (
	"fmt"
	"strings"
)

// A D64 is a raw dump of a 1541 floppy. Nothing at the front identifies it, so
// the size is what gives it away.
//
// The tracks do not all hold the same number of sectors: the outer ones are
// longer, so the drive fits more into them. That is why a track and sector pair
// cannot be turned into an offset by multiplying.
var d64SectorsPerTrack = []struct {
	upToTrack int
	sectors   int
}{
	{17, 21},
	{24, 19},
	{30, 18},
	{35, 17},
	{40, 17}, // the extra tracks of a 40-track image
}

const (
	d64SectorSize   = 256
	d64DirTrack     = 18
	d64DirSector    = 1
	d64EntrySize    = 32
	d64NameLength   = 16
	d64NamePadding  = 0xA0
	d64DiskNameAt   = 0x90 // in the block that holds the availability map
	d64ClosedFile   = 0x80
	d64EntriesBlock = 8
)

// Fields of a directory entry, from its start.
const (
	d64Type       = 2
	d64FirstTrack = 3
	d64FirstSec   = 4
	d64Name       = 5
)

// D64 reads a Commodore 64 disk image.
type D64 struct {
	data []byte
}

func NewD64(data []byte) (*D64, error) {
	if len(data) < d64Offset(d64DirTrack, d64DirSector)+d64SectorSize {
		return nil, fmt.Errorf("too short for a 1541 disk: %d bytes", len(data))
	}

	return &D64{data: data}, nil
}

func (d *D64) Format() string {
	return "Commodore 1541 disk image"
}

// Name returns the disk label, which sits in the block holding the map of free
// blocks rather than in the directory.
func (d *D64) Name() string {
	block := d.sector(d64DirTrack, 0)
	if block == nil {
		return ""
	}

	return petscii(block[d64DiskNameAt : d64DiskNameAt+d64NameLength])
}

func (d *D64) Files() ([]File, error) {
	var files []File

	track, sector := d64DirTrack, d64DirSector

	for seen := 0; track != 0; seen++ {
		if seen > len(d.data)/d64SectorSize {
			return nil, fmt.Errorf("the directory loops")
		}

		block := d.sector(track, sector)
		if block == nil {
			return nil, fmt.Errorf("the directory runs off the disk at track %d sector %d", track, sector)
		}

		for i := range d64EntriesBlock {
			entry := block[i*d64EntrySize : (i+1)*d64EntrySize]

			// A type byte of zero means the slot was never used; without the
			// closed flag the file was left open and its length is not to be
			// trusted.
			if entry[d64Type]&d64ClosedFile == 0 {
				continue
			}

			data, err := d.chain(int(entry[d64FirstTrack]), int(entry[d64FirstSec]))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", petscii(entry[d64Name:d64Name+d64NameLength]), err)
			}

			files = append(files, File{
				Name: petscii(entry[d64Name : d64Name+d64NameLength]),
				Data: data,
			})
		}

		// The first two bytes of a directory block point at the next one.
		track, sector = int(block[0]), int(block[1])
	}

	return files, nil
}

// chain follows the blocks of a file. Every block spends its first two bytes on
// the link to the next, which is why a file is never a contiguous run of bytes
// in the image.
func (d *D64) chain(track, sector int) ([]byte, error) {
	var out []byte

	for seen := 0; track != 0; seen++ {
		if seen > len(d.data)/d64SectorSize {
			return nil, fmt.Errorf("the block chain loops")
		}

		block := d.sector(track, sector)
		if block == nil {
			return nil, fmt.Errorf("block at track %d sector %d falls outside the disk", track, sector)
		}

		next, offset := int(block[0]), int(block[1])

		if next == 0 {
			// On the last block the second byte is not a sector but the index
			// of the last byte in use.
			if offset < 2 {
				return out, nil
			}

			return append(out, block[2:offset+1]...), nil
		}

		out = append(out, block[2:]...)
		track, sector = next, offset
	}

	return out, nil
}

func (d *D64) sector(track, sector int) []byte {
	off := d64Offset(track, sector)
	if off < 0 || off+d64SectorSize > len(d.data) {
		return nil
	}

	return d.data[off : off+d64SectorSize]
}

// d64Offset turns a track and sector into a position in the image, adding up
// the tracks before it. Tracks are numbered from one.
func d64Offset(track, sector int) int {
	if track < 1 {
		return -1
	}

	offset, from := 0, 1

	for _, band := range d64SectorsPerTrack {
		for t := from; t <= band.upToTrack; t++ {
			if t == track {
				if sector < 0 || sector >= band.sectors {
					return -1
				}

				return (offset + sector) * d64SectorSize
			}

			offset += band.sectors
		}

		from = band.upToTrack + 1
	}

	return -1
}

// petscii turns a name from the disk into something readable, dropping the
// shifted spaces the 1541 pads names with.
func petscii(raw []byte) string {
	var sb strings.Builder

	for _, c := range raw {
		if c == d64NamePadding {
			break
		}

		switch {
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			sb.WriteByte(c)
		case c >= 0xC1 && c <= 0xDA:
			// The shifted half of the character set holds the letters again.
			sb.WriteByte(c - 0xC1 + 'A')
		case c >= 32 && c < 127:
			sb.WriteByte(c)
		default:
			sb.WriteByte('?')
		}
	}

	return strings.TrimRight(sb.String(), " ")
}
