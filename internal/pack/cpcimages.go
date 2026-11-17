package pack

import (
	"fmt"
	"sort"

	"github.com/jorgefuertes/QDAAD/internal/media"
)

// The pictures of a CPC build, which are files on the disk and not part of the
// binary.
//
// The interpreter opens them by name as a PICTURE asks for one — it carries the
// string "000.CPC" as a mould and writes the number into it, the same trick the
// +3's carries "UTO.128" for — and it opens them through the firmware's
// cassette vectors, which AMSDOS takes over when there is a disk.
//
// The format is read off the interpreter we ship, DCPCISF3.BIN, at 0x09C0. It
// knows two shapes. One is a whole uncompressed screen, taken when the file is
// exactly 16384 bytes long: CAS IN DIRECT drops it on the display and the
// palette comes out of its last sixteen bytes, which fall in the part of video
// memory nothing shows. The other is this one, and it is the one written here
// because it is the only one that can be shorter than a screen:
//
//	byte 0      how many bands follow in each two-kilobyte record
//	bytes 1-16  the palette, one firmware ink to a pen
//	bytes 17-18 how many bytes of each band are carried
//	then        the bands, and every record padded out to 2048
//
// A band is one of the eight interleaved groups video memory is laid out in:
// band b begins at 0xC000 + b*0x800 and holds one scan line out of every
// character row. The loop at 0x0A16 copies the bands to those eight addresses
// in order, uncompressed, and reads another record whenever it has used the
// ones in hand — which is why the records have to be padded and why the number
// of bands in each is written in the file rather than worked out.
//
// So what makes a picture small here is carrying fewer rows of it, and nothing
// else: there is no compression anywhere in the path. DAAD's own IMGLINES
// defaults to 96 lines, and the reason it is 96 rather than anything else falls
// out of the arithmetic — twelve character rows are 960 bytes a band, which is
// the most that lets two bands share a record.
const (
	// cpcPictureSuffix is the extension the interpreter builds its filenames
	// with: three digits and this.
	cpcPictureSuffix = ".CPC"
	// cpcPaletteSize is the sixteen pens the file carries whatever the mode.
	cpcPaletteSize = 16
	// cpcPictureHeader is the byte of bands, the palette and the band length.
	cpcPictureHeader = 1 + cpcPaletteSize + 2
	// cpcRecordSize is the record AMSDOS reads a file in, and the buffer the
	// interpreter hands to CAS IN OPEN.
	cpcRecordSize = 2048

	// cpcScreenBase is where the display lives.
	cpcScreenBase = 0xC000

	// The shape of a mode 1 display: eighty bytes to a row and eight bands.
	cpcPackBytesWide = 80
	cpcPackBands     = 8
	cpcPackBandSize  = 0x800
	cpcPackRows      = 25
	cpcCellRows      = 8
	cpcPackScreen    = cpcPackBands * cpcPackBandSize
)

// cpcPictureRows is how many character rows of a screen to carry: as many as
// were asked for, or the last one with a pixel in it.
//
// The same choice as the Spectrum's pictureLines and for the same reason. DAAD
// cuts every picture to one height for all of them, and the drawings of 1989 are
// not all the same size — those of La Aventura Original fill the fourteen rows
// their own records declare, and the 96 lines of IMGLINES would take two of them
// off the bottom.
func cpcPictureRows(screen []byte, lines int) int {
	// A height asked for wins, rounded up to the character row it falls in:
	// a band is copied in whole rows and there is no saying half of one.
	if lines > 0 {
		return min((lines+cpcCellRows-1)/cpcCellRows, cpcPackRows)
	}

	for row := cpcPackRows; row > 0; row-- {
		for band := range cpcPackBands {
			line := band*cpcPackBandSize + (row-1)*cpcPackBytesWide

			for _, b := range screen[line : line+cpcPackBytesWide] {
				if b != 0 {
					return row
				}
			}
		}
	}

	return 1
}

// cpcPicture turns a mode 1 screen and its inks into the file the interpreter
// opens.
func cpcPicture(screen, palette []byte, lines int) ([]byte, error) {
	if len(screen) != cpcPackScreen {
		return nil, fmt.Errorf("a CPC screen is %d bytes, not %d", cpcPackScreen, len(screen))
	}

	if len(palette) == 0 {
		return nil, fmt.Errorf("a CPC picture needs its inks and there is no .pal beside it")
	}

	length := cpcPictureRows(screen, lines) * cpcPackBytesWide

	// How many bands go in a record. The first record carries the header as
	// well, and the interpreter uses the one count for every record, so it is
	// the first that decides.
	perRecord := min(max((cpcRecordSize-cpcPictureHeader)/length, 1), cpcPackBands)

	out := make([]byte, 0, cpcRecordSize*((cpcPackBands+perRecord-1)/perRecord))

	out = append(out, byte(perRecord))
	out = append(out, palette[:min(len(palette), cpcPaletteSize)]...)

	for len(out) < cpcPictureHeader-2 {
		out = append(out, 0)
	}

	out = append(out, byte(length), byte(length>>8))

	for band := range cpcPackBands {
		at := band * cpcPackBandSize
		out = append(out, screen[at:at+length]...)

		// A record is read whole, so the bands have to fall where the
		// interpreter expects to find them: the next group starts at the next
		// record and the gap between is nothing in particular.
		if (band+1)%perRecord == 0 {
			for len(out)%cpcRecordSize != 0 {
				out = append(out, 0)
			}
		}
	}

	for len(out)%cpcRecordSize != 0 {
		out = append(out, 0)
	}

	return out, nil
}

// cpcPictureName is what a picture is called on the disk.
func cpcPictureName(number int) string {
	return fmt.Sprintf("%03d%s", number, cpcPictureSuffix)
}

// cpcPictureFiles turns the build's pictures into the files that go on the disk,
// in number order so that a listing reads like the index it is.
//
// Each one carries an AMSDOS header, the same as the binary does: it is what
// says the file is a binary and how long it is, and a file without one is an
// ASCII file as far as the disk is concerned.
func cpcPictureFiles(opts Options) ([]media.File, error) {
	numbers := make([]int, 0, len(opts.Images))
	for number := range opts.Images {
		numbers = append(numbers, number)
	}

	sort.Ints(numbers)

	files := make([]media.File, 0, len(numbers))

	for i, number := range numbers {
		opts.Progress.at(i+1, len(numbers), fmt.Sprintf("packing picture %03d", number))

		name := cpcPictureName(number)

		body, err := cpcPicture(opts.Images[number], opts.Palettes[number], opts.Lines)
		if err != nil {
			return nil, fmt.Errorf("picture %03d: %w", number, err)
		}

		header, err := media.AMSDOSHeader(name, cpcScreenBase, cpcScreenBase, len(body))
		if err != nil {
			return nil, err
		}

		files = append(files, media.File{Name: name, Data: append(header, body...)})
	}

	return files, nil
}
