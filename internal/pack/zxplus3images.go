package pack

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/zx0"
)

// The pictures of a +3 build, which are files on the disk and not a page in
// memory.
//
// A 48K build carries what it can fit above the database, cut into runs of
// video memory and indexed by a table; a +3 build carries none of that and
// opens a file when a PICTURE asks for one — the interpreter holds the name
// "UTO.128" as a mould and writes the number into it. So the room limit goes
// away and all of them go on the disk.
//
// The file is five bytes of geometry and then the whole screen compressed as
// one ZX0 stream. Both halves of that are first-hand: SCRMAKER.PAS writes it
//
//	BlockWrite(outFile, WidthInCharacters, 1);   // width DIV 8
//	BlockWrite(outFile, HeightInCharacters, 1);  // height DIV 8
//	BlockWrite(outFile, WidthInColumns, 1);      // width DIV 6
//	BlockWrite(outFile, X0InChars, 1);           // X0 DIV 8
//	BlockWrite(outFile, Y0InChars, 1);           // Y0 DIV 8
//	Blockwrite(outFile, buffer[0], BytesRead);   // the .zx0 file, whole
//
// and PLUS3.ASM reads it into five bytes of variables — WidthInCharacters,
// HeightInCharacters, WidthInColumns, FixedXInChars, FixedYInChars — and then
// uncompresses what follows into PictureBuffer, a 6912-byte buffer at 0xC200,
// in one go. It never decompresses to the screen, which is why nothing here
// untwists anything: what goes in the stream is a screen dump, exactly as it
// sits in video memory, and the interpreter blits the buffer afterwards.
//
// The seven bytes that plus3cache.php talks about are simply wrong, and the
// six of ZXDAAD128 belong to a different interpreter. Four attempts went on
// that pair of red herrings; see docs/DAAD_v2_v3_Research/09-graficos.md.
const (
	// plus3PictureSuffix is the extension the interpreter builds its
	// filenames with: three digits and this.
	plus3PictureSuffix = ".128"
	// plus3TopOfScreen is the Y every picture here is pinned to. The
	// alternative is to let it float — any Y at or past the height of the
	// screen means "wherever the window happens to be" — and floating puts
	// the drawing under the text instead of above it, because by the time a
	// PICTURE runs the window is the one the adventure last printed in.
	plus3TopOfScreen = 0
)

// plus3Picture turns a Spectrum screen into the file the +3 interpreter opens.
func plus3Picture(screen []byte) ([]byte, error) {
	if len(screen) != screenSize {
		return nil, fmt.Errorf("a Spectrum screen is %d bytes, not %d", screenSize, len(screen))
	}

	// Everything below the picture area is wiped first, which is what scrcrop
	// does in the reference pipeline and why it still hands over 6912 bytes:
	// the adventure writes its text down there, and a blank half costs almost
	// nothing once compressed.
	cropped := make([]byte, screenSize)
	copy(cropped, screen)

	for y := imageLines; y < 192; y++ {
		at := (y>>6)*2048 + (y&7)*256 + ((y>>3)&7)*bytesWide
		clear(cropped[at : at+bytesWide])
	}

	clear(cropped[screenSize-scrAttributes+imageLines/8*bytesWide:])

	out := []byte{
		canvasWidth / 8, // width in 8x8 characters
		imageLines / 8,  // height in 8x8 characters
		canvasWidth / 6, // width in 6x8 columns, which is the text font
		0,                // fixed X: every picture is as wide as the screen
		plus3TopOfScreen, // pinned to the top, with the text below it
	}

	return append(out, zx0.Compress(cropped)...), nil
}

// The screen a picture is cut out of.
const (
	canvasWidth   = 256
	scrAttributes = 768
)

// plus3PictureName is what a picture is called on the disk.
func plus3PictureName(number int) string {
	return fmt.Sprintf("%03d%s", number, plus3PictureSuffix)
}
