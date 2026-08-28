// Package media opens the disk and tape images the adventures of the era were
// shipped on, and hands back the files inside them.
//
// Every platform used its own container and its own filesystem, so a database
// cannot simply be carved out of an image: AmigaDOS interleaves a header in
// every data block and Commodore links its sectors, so a file is rarely a
// contiguous run of bytes.
package media

import (
	"bytes"
	"fmt"
	"os"
)

// File is one file taken out of an image.
type File struct {
	Name string
	Data []byte
}

// Volume is an image that holds files.
type Volume interface {
	// Format names the container, for reporting.
	Format() string
	// Files returns everything in the image, directories walked through.
	Files() ([]File, error)
}

// Open identifies an image and returns a reader for it.
//
// Identification goes by content wherever the format has a signature, and by
// size where it does not: a Commodore disk image and an Atari one are both raw
// dumps with nothing at the front to tell them apart.
func Open(path string) (Volume, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	switch {
	case bytes.HasPrefix(data, []byte("DOS")):
		return NewADF(data)

	case bytes.HasPrefix(data, []byte("MV - CPCEMU")),
		bytes.HasPrefix(data, []byte("EXTENDED CPC DSK")):
		return nil, notYet("Amstrad CPC / Spectrum +3 disk image")

	case bytes.HasPrefix(data, []byte("ZXTape!")):
		return nil, notYet("ZX Spectrum tape")

	case bytes.HasPrefix(data, []byte("C64-TAPE-RAW")):
		// This one holds the timings of the pulses on the tape rather than
		// bytes, so reading it means decoding the loader as well.
		return nil, notYet("Commodore 64 tape, stored as raw pulses")

	case looksLikeFAT(data):
		return NewFAT12(data)

	case len(data) == d64Size:
		return NewD64(data)
	}

	return nil, fmt.Errorf("%s: unknown image format", path)
}

// Sizes that identify the raw dumps, which carry no signature.
const (
	d64Size = 174848 // 35 tracks of a 1541
)

// looksLikeFAT recognises an Atari ST image by its BIOS parameter block: the ST
// wrote MS-DOS floppies, so the boot sector is a normal FAT12 one.
func looksLikeFAT(data []byte) bool {
	const bpb = 11

	if len(data) < 512 {
		return false
	}

	bytesPerSector := int(data[bpb]) | int(data[bpb+1])<<8
	sectorsPerCluster := int(data[bpb+2])

	return bytesPerSector == 512 && sectorsPerCluster > 0 && sectorsPerCluster <= 8
}

func notYet(what string) error {
	return fmt.Errorf("%s: recognised, but reading it is not implemented yet", what)
}
