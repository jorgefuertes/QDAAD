package pack

// The shape of a CPC picture, which is what everything that carries one has to
// know.
//
// A mode 1 screen is 16384 bytes and video memory lays it out in eight
// interleaved bands: band b begins at 0xC000 + b*0x800 and holds one scan line
// out of every character row. So a picture that is not the full height of the
// screen is not a run of bytes but eight of them, one to a band, each as many
// bytes long as the picture is tall in character rows.
//
// The interpreter does the same thing with the files it opens itself: the loop
// at 0x0A16 of DCPCISF3.BIN copies band after band to those eight addresses,
// 0x800 apart, and stops at the count it was given. cpcbank.go is that loop
// with ZX0 in front of it, and this is the arithmetic both of them are built
// on.
//
// There was a second way through here and it is gone. A disk used to carry the
// pictures as files of their own, uncompressed, which is what the interpreter
// was written for and what a 464 with a drive could read; it is removed because
// nothing compresses a picture on that path, so nine of the twenty-eight of La
// Aventura Original fit on a 180K disk and the other nineteen took the text of
// their locations with them — see CPCDisk. A disk is a 6128 disk now, and a
// machine without the memory for it loads from tape.
const (
	// cpcPictureSuffix is the extension the interpreter builds its filenames
	// with: three digits and this. 0x0CCA writes the digits into the mould at
	// 0x0CF2 and opens what comes out, which is why a picture carried on the
	// disk has to be called exactly that.
	cpcPictureSuffix = ".CPC"
	// cpcPaletteSize is the sixteen pens a picture carries whatever the mode,
	// which is what the interpreter's own ink buffer holds.
	cpcPaletteSize = 16

	// cpcScreenBase is where the display lives.
	cpcScreenBase = 0xC000

	// The shape of a mode 1 display: eighty bytes to a row, eight bands of
	// 0x800, and twenty-five character rows between them.
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
//
// Asking for a height is worth less than it was now that the pictures are
// compressed wherever they go: what it buys is a smaller bank, and what it
// costs is the bottom of every drawing that is taller than the number. The
// frame the title of La Aventura Original is printed inside is twenty-two rows
// where the rest are fourteen, so any number at all takes its bottom edge off.
// Nought, which is the default, carries each picture down to its own last row
// and is what the example passes.
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
