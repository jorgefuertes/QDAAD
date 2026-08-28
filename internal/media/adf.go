package media

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// An ADF is a raw dump of an Amiga floppy: 2 sides of 80 tracks of 11 sectors
// of 512 bytes, and nothing else. There is no header, so the size is what
// identifies it.
const (
	adfBlockSize = 512
	adfDoubleDen = 2 * 80 * 11 * adfBlockSize
	adfRootBlock = 880 // always, on a double density disk
	adfHashSize  = 72
)

// Offsets inside a block. AmigaDOS puts the interesting fields of a header at
// the front and the identifying ones at the very end.
const (
	offHighSeq   = 8
	offFirstData = 16
	offDataSize  = 12
	offNextData  = 16
	offHashTable = 24
	offDataStart = 24 // where the payload begins, on the old filesystem
	offByteSize  = 324
	offNameLen   = 432
	offHashChain = 496
	offExtension = 504
	offSecType   = 508
)

// Secondary types, at the end of a header block.
const (
	secTypeUserDir = 2
	secTypeFile    = -3
)

// ADF reads an Amiga disk image.
type ADF struct {
	data []byte
	// oldFileSystem tells the two layouts apart. On OFS every data block spends
	// 24 of its 512 bytes on a header, which is why a file is never a
	// contiguous run of bytes; on FFS the whole block is payload.
	oldFileSystem bool
}

func NewADF(data []byte) (*ADF, error) {
	if len(data) < adfDoubleDen {
		return nil, fmt.Errorf("too short for an Amiga floppy: %d bytes", len(data))
	}

	// The low bit of the fourth byte of the boot block picks the filesystem.
	return &ADF{data: data, oldFileSystem: data[3]&1 == 0}, nil
}

func (a *ADF) Format() string {
	if a.oldFileSystem {
		return "Amiga ADF (OFS)"
	}

	return "Amiga ADF (FFS)"
}

// Name returns the disk label, which lives in the root block.
func (a *ADF) Name() string {
	return a.nameOf(adfRootBlock)
}

func (a *ADF) Files() ([]File, error) {
	var files []File

	if err := a.walk(adfRootBlock, "", &files, 0); err != nil {
		return nil, err
	}

	return files, nil
}

// walk follows the hash table of a directory. Nothing needs hashing to read a
// disk: every chain is walked whole, so the names come out without computing
// where they would have been stored.
func (a *ADF) walk(block int, prefix string, files *[]File, depth int) error {
	const maxDepth = 16 // a floppy that nests deeper is not a floppy

	if depth > maxDepth {
		return errors.New("directories nested too deep: the image is probably corrupt")
	}

	for i := range adfHashSize {
		entry := int(a.long(block, offHashTable+4*i))

		for entry != 0 {
			if !a.valid(entry) {
				return fmt.Errorf("block %d points outside the disk", entry)
			}

			name := prefix + a.nameOf(entry)

			switch a.signedLong(entry, offSecType) {
			case secTypeFile:
				data, err := a.contents(entry)
				if err != nil {
					return fmt.Errorf("%s: %w", name, err)
				}

				*files = append(*files, File{Name: name, Data: data})

			case secTypeUserDir:
				if err := a.walk(entry, name+"/", files, depth+1); err != nil {
					return err
				}
			}

			entry = int(a.long(entry, offHashChain))
		}
	}

	return nil
}

// contents joins the data blocks of a file.
//
// The header lists them in reverse order and a long file continues in
// extension blocks, so the list is gathered first and read afterwards.
func (a *ADF) contents(header int) ([]byte, error) {
	size := int(a.long(header, offByteSize))

	var blocks []int

	for h := header; h != 0; h = int(a.long(h, offExtension)) {
		if !a.valid(h) {
			return nil, fmt.Errorf("extension block %d points outside the disk", h)
		}

		used := int(a.long(h, offHighSeq))
		if used > adfHashSize {
			return nil, fmt.Errorf("block %d claims %d data blocks, more than fit", h, used)
		}

		// Stored last first, so the list is read backwards.
		for i := range used {
			blocks = append(blocks, int(a.long(h, offHashTable+4*(adfHashSize-1-i))))
		}
	}

	out := make([]byte, 0, size)

	for _, b := range blocks {
		if !a.valid(b) {
			return nil, fmt.Errorf("data block %d points outside the disk", b)
		}

		payload := a.data[b*adfBlockSize : (b+1)*adfBlockSize]

		if a.oldFileSystem {
			n := int(a.long(b, offDataSize))
			if n > adfBlockSize-offDataStart {
				return nil, fmt.Errorf("data block %d claims %d bytes, more than it holds", b, n)
			}

			payload = payload[offDataStart : offDataStart+n]
		}

		out = append(out, payload...)
	}

	if len(out) < size {
		return nil, fmt.Errorf("file is short: %d bytes of the %d declared", len(out), size)
	}

	// On the fast filesystem the last block is full, so the tail is trimmed.
	return out[:size], nil
}

// nameOf reads the BCPL string a header block ends with: a length followed by
// that many characters.
func (a *ADF) nameOf(block int) string {
	const maxName = 30

	base := block * adfBlockSize

	n := int(a.data[base+offNameLen])
	if n > maxName {
		n = maxName
	}

	return string(a.data[base+offNameLen+1 : base+offNameLen+1+n])
}

func (a *ADF) valid(block int) bool {
	return block > 0 && (block+1)*adfBlockSize <= len(a.data)
}

func (a *ADF) long(block, offset int) uint32 {
	return binary.BigEndian.Uint32(a.data[block*adfBlockSize+offset:])
}

func (a *ADF) signedLong(block, offset int) int32 {
	return int32(a.long(block, offset))
}
