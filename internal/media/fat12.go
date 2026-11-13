package media

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// The Atari ST wrote MS-DOS floppies, so its images carry a plain FAT12 with a
// BIOS parameter block in the boot sector. Everything below is that layout, not
// anything of Atari's own.
//
// Note the numbers in the parameter block are little-endian even though the
// 68000 is not: the format was inherited whole from the PC.
const (
	bpbBytesPerSector    = 11
	bpbSectorsPerCluster = 13
	bpbReservedSectors   = 14
	bpbNumberOfFATs      = 16
	bpbRootEntries       = 17
	bpbTotalSectors      = 19
	bpbSectorsPerFAT     = 22
)

// A directory entry is 32 bytes.
const (
	dirEntrySize   = 32
	dirName        = 0
	dirAttributes  = 11
	dirFirstluster = 26
	dirFileSize    = 28

	dirEnd     = 0x00 // no more entries in this directory
	dirDeleted = 0xE5

	attrVolumeLabel = 0x08
	attrDirectory   = 0x10
	attrLongName    = 0x0F // a VFAT name fragment, not a file
)

// FAT12 reads a floppy image holding a FAT12 filesystem.
type FAT12 struct {
	data []byte

	bytesPerSector    int
	sectorsPerCluster int
	rootEntries       int

	fatStart  int // byte offset of the first file allocation table
	rootStart int // byte offset of the root directory
	dataStart int // byte offset of the first data cluster
}

func NewFAT12(data []byte) (*FAT12, error) {
	if len(data) < 512 {
		return nil, fmt.Errorf("too short to hold a boot sector: %d bytes", len(data))
	}

	word := func(off int) int { return int(binary.LittleEndian.Uint16(data[off:])) }

	f := &FAT12{
		data:              data,
		bytesPerSector:    word(bpbBytesPerSector),
		sectorsPerCluster: int(data[bpbSectorsPerCluster]),
		rootEntries:       word(bpbRootEntries),
	}

	reserved := word(bpbReservedSectors)
	fats := int(data[bpbNumberOfFATs])
	sectorsPerFAT := word(bpbSectorsPerFAT)

	if f.bytesPerSector == 0 || f.sectorsPerCluster == 0 || fats == 0 || sectorsPerFAT == 0 {
		return nil, fmt.Errorf("the boot sector does not describe a FAT filesystem")
	}

	f.fatStart = reserved * f.bytesPerSector
	f.rootStart = f.fatStart + fats*sectorsPerFAT*f.bytesPerSector
	f.dataStart = f.rootStart + f.rootEntries*dirEntrySize

	if f.dataStart > len(data) {
		return nil, fmt.Errorf("the filesystem does not fit in the image")
	}

	return f, nil
}

func (f *FAT12) Format() string {
	return fmt.Sprintf("FAT12, %d bytes per sector, %d per cluster",
		f.bytesPerSector, f.sectorsPerCluster)
}

// Payload returns the image as it stands. A raw sector dump needs no
// assembling: the sectors are already in order and contiguous, which is what
// makes it worth searching when the directory turns out to be empty.
//
// A plausible boot sector is no promise of a filesystem. The MSX edition of El
// Jabato keeps one and then formats the rest of the disk for its own loader, so
// the root directory holds Z80 code where the entries should be.
func (f *FAT12) Payload() []byte {
	return f.data
}

func (f *FAT12) Files() ([]File, error) {
	root := f.data[f.rootStart : f.rootStart+f.rootEntries*dirEntrySize]

	var files []File

	if err := f.walk(root, "", &files, 0); err != nil {
		return nil, err
	}

	return files, nil
}

func (f *FAT12) walk(dir []byte, prefix string, files *[]File, depth int) error {
	const maxDepth = 16

	if depth > maxDepth {
		return fmt.Errorf("directories nested too deep: the image is probably corrupt")
	}

	for off := 0; off+dirEntrySize <= len(dir); off += dirEntrySize {
		entry := dir[off : off+dirEntrySize]

		switch entry[dirName] {
		case dirEnd:
			return nil
		case dirDeleted:
			continue
		}

		attributes := entry[dirAttributes]
		if attributes&attrLongName == attrLongName || attributes&attrVolumeLabel != 0 {
			continue
		}

		name := entryName(entry)
		if name == "." || name == ".." {
			continue
		}

		cluster := int(binary.LittleEndian.Uint16(entry[dirFirstluster:]))
		size := int(binary.LittleEndian.Uint32(entry[dirFileSize:]))

		if attributes&attrDirectory != 0 {
			// A directory has no size of its own: it is read whole.
			contents, err := f.chain(cluster, -1)
			if err != nil {
				return fmt.Errorf("%s%s: %w", prefix, name, err)
			}

			if err := f.walk(contents, prefix+name+"/", files, depth+1); err != nil {
				return err
			}

			continue
		}

		data, err := f.chain(cluster, size)
		if err != nil {
			return fmt.Errorf("%s%s: %w", prefix, name, err)
		}

		*files = append(*files, File{Name: prefix + name, Data: data})
	}

	return nil
}

// chain follows the cluster list of a file. A size of -1 reads to the end of
// the chain, which is what a directory needs.
func (f *FAT12) chain(cluster, size int) ([]byte, error) {
	const (
		firstCluster = 2
		endOfChain   = 0xFF8 // and anything above
	)

	clusterBytes := f.sectorsPerCluster * f.bytesPerSector
	out := make([]byte, 0, max(size, clusterBytes))

	// A file of no bytes has no cluster to start from.
	if size == 0 {
		return out, nil
	}

	for seen := 0; cluster >= firstCluster && cluster < endOfChain; seen++ {
		if seen > len(f.data)/clusterBytes {
			return nil, fmt.Errorf("the cluster chain loops")
		}

		start := f.dataStart + (cluster-firstCluster)*clusterBytes
		if start+clusterBytes > len(f.data) {
			return nil, fmt.Errorf("cluster %d falls outside the image", cluster)
		}

		out = append(out, f.data[start:start+clusterBytes]...)

		if size >= 0 && len(out) >= size {
			break
		}

		cluster = f.next(cluster)
	}

	if size < 0 {
		return out, nil
	}

	if len(out) < size {
		return nil, fmt.Errorf("file is short: %d bytes of the %d declared", len(out), size)
	}

	return out[:size], nil
}

// next reads one entry of the allocation table. Entries are twelve bits, so
// they come in pairs sharing a byte, and which half to take depends on whether
// the number is odd.
func (f *FAT12) next(cluster int) int {
	off := f.fatStart + cluster*3/2
	if off+1 >= len(f.data) {
		return 0
	}

	pair := int(f.data[off]) | int(f.data[off+1])<<8

	if cluster%2 == 0 {
		return pair & 0x0FFF
	}

	return pair >> 4
}

// entryName joins the name and the extension of a directory entry, both of them
// padded with spaces.
func entryName(entry []byte) string {
	name := strings.TrimRight(string(entry[dirName:dirName+8]), " ")
	extension := strings.TrimRight(string(entry[dirName+8:dirName+11]), " ")

	if extension == "" {
		return name
	}

	return name + "." + extension
}
