package pack

import (
	"fmt"
	"sort"

	"github.com/jorgefuertes/QDAAD/internal/media"
)

// The pictures of a PC build, which are files the interpreter opens and the
// only ones here that come in as PCX.
//
// The format is .MSD, and it is SCRMAKER's own rather than anything a paint
// program writes: sixteen bytes of header, the whole palette, and the pixels
// one byte each. It is first-hand from both ends — SCRMAKER.PAS:690-818 writes
// it and pic.pas:346-400 reads it — which is the only reason the palette shift
// below is a fact and not a guess.
//
// There is no compression. The header has a bit for it, offset 12 bit 1, and
// the comment above the writer says the pixels use "same encoding as PCX"; the
// writer sets that byte to zero and dumps the rows raw, and loadMSD never reads
// byte 12 at all. So a picture is 16 + 768 + width×height bytes and a 320×96
// one is 31504 of them, which is why a PC release is megabytes where a tape is
// kilobytes. It goes on a hard disk and nobody minded.
const (
	// pcPictureSuffix is what the interpreter looks for first, before .pcx and
	// .VGA (pic.pas:113-122).
	pcPictureSuffix = ".MSD"
	// pcLoadingName is the picture shown before anything else. The interpreter
	// builds its filename from the picture number, except for one: 65535 is
	// called DAAD (pic.pas:103).
	pcLoadingName = "DAAD"
	// pcHeaderSize is the header, and pcPaletteSize the palette that follows
	// it. The palette is always whole, whatever the limits in the header say.
	pcHeaderSize  = 16
	pcPaletteSize = 768
	// pcCanvasWidth and pcCanvasHeight are the screen a VGA256 build draws on,
	// and the size every PCX going into one has to be. The reference refuses
	// anything else outright (SCRMAKER.PAS:744-749).
	pcCanvasWidth  = 320
	pcCanvasHeight = 200
	// pcReservedColours is how far the picture's palette is pushed up to make
	// room for DAAD's own sixteen. See pcPalette.
	pcReservedColours = 16
)

// pcStandardPalette is the sixteen colours DAAD keeps for itself at the bottom
// of a PC palette, and they are the Spectrum's: black, blue, red, magenta,
// green, cyan, yellow and white, each dim and bright.
//
// They are there because the text is drawn in them. A PC release picks its
// inks with the same numbers an eight-bit one does — INK 6 is yellow on every
// machine DAAD targets — so the bottom of the palette has to be that palette,
// whatever the drawing wanted those entries for. Which is what the shift is:
// the drawing's own colours move up sixteen places and these take their seats.
//
// Read off StandardZXPalette, SCRMAKER.PAS:43-58, and read rather than
// reconstructed: neither half is what it looks like it should be. A dim colour
// is 0xC0 and a bright one 0xF0, not 0xFF — writing the obvious 0xFF came out
// one byte different from the reference and that byte was bright blue.
var pcStandardPalette = [pcReservedColours * 3]byte{
	0x00, 0x00, 0x00, // black
	0x00, 0x00, 0xC0, // blue
	0xC0, 0x00, 0x00, // red
	0xC0, 0x00, 0xC0, // magenta
	0x00, 0xC0, 0x00, // green
	0x00, 0xC0, 0xC0, // cyan
	0xC0, 0xC0, 0x00, // yellow
	0xC0, 0xC0, 0xC0, // white
	0x00, 0x00, 0x00, // bright black, which is still black
	0x00, 0x00, 0xF0, // bright blue
	0xF0, 0x00, 0x00, // bright red
	0xF0, 0x00, 0xF0, // bright magenta
	0x00, 0xF0, 0x00, // bright green
	0x00, 0xF0, 0xF0, // bright cyan
	0xF0, 0xF0, 0x00, // bright yellow
	0xF0, 0xF0, 0xF0, // bright white
}

// pcPicture turns a PCX into the file the interpreter opens.
//
// lines is how much of the screen the drawing occupies, or zero to take the
// PCX's own height.
//
// Zero is the useful answer and DAAD's is not. The reference takes a whole
// 320x200 screen and crops it to IMGLINES, one height for every drawing in the
// build, and that is the wrong number for an adventure of 1989: La Aventura
// Original's PC rooms are 112 lines and its title is 176, so a single height
// either cuts the title's frame off or pads every room with sixty-four blank
// rows — and the height in the header is what the interpreter puts the text
// window under, so the padding is not free either.
//
// Taking the file's own height means accepting a PCX that is not a whole
// screen, which the reference refuses outright (SCRMAKER.PAS:744-749). That is
// a deliberate difference and the same one pictureLines makes on the Spectrum:
// each drawing is as tall as it is. A full-screen PCX with lines given is still
// exactly what SCRMAKER writes, which is what script/compare-scrmaker.sh holds
// this against.
func pcPicture(file []byte, lines int) ([]byte, error) {
	image, err := media.ReadPCX(file)
	if err != nil {
		return nil, err
	}

	if image.Width > pcCanvasWidth {
		return nil, fmt.Errorf("a VGA256 screen is %d pixels wide and this picture is %d",
			pcCanvasWidth, image.Width)
	}

	if image.Height > pcCanvasHeight {
		return nil, fmt.Errorf("a VGA256 screen is %d lines and this picture is %d",
			pcCanvasHeight, image.Height)
	}

	if lines == 0 {
		lines = image.Height
	}

	if lines < 0 || lines > image.Height {
		return nil, fmt.Errorf("asked for %d lines of a picture that has %d",
			lines, image.Height)
	}

	pixels, err := pcShift(image.Pixels)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 0, pcHeaderSize+pcPaletteSize+image.Width*lines)

	// The signature is the project's own name in hex, which is a joke that has
	// outlived everyone who had to type it.
	out = append(out, 0xDA, 0xAD)
	out = append(out, byte(image.Width), byte(image.Width>>8))
	out = append(out, byte(lines), byte(lines>>8))

	// The palette limits, which say which entries the interpreter may fade and
	// cycle. The whole range, as every PC build in the kit asks for.
	out = append(out, 0, 255)

	// Where it goes: centred across, and pinned to the top.
	//
	// Centred because a drawing need not be as wide as the screen and these are
	// not — La Aventura Original's PC illustrations are 256 across a screen of
	// 320 — and a picture written at zero sits against the left edge with
	// sixty-four columns of black beside it. The header is what places it, so
	// this costs nothing: the pixels are the same and only fixedX moves.
	//
	// Pinned to the top rather than floating, for the reason the Spectrum's
	// pictures are: by the time a PICTURE runs, the window is the one the
	// adventure last printed in, so letting it float puts the drawing under the
	// text. Floating is what a Y of 400 would ask for — and that number is
	// worth not misreading, because SCRMAKER's own comment says 640 while its
	// code says MAX_PC_HEIGHT, which is 400. The interpreter agrees with the
	// code: isFloat := fixedY = 400, pic.pas:382.
	at := (pcCanvasWidth - image.Width) / 2

	out = append(out, byte(at), byte(at>>8), 0, 0)

	// The compression flag, off, and three bytes nobody reads.
	out = append(out, 0, 0, 0, 0)

	out = append(out, pcPalette(image.Palette)...)

	// The top of the picture, row by row, cut to the height. The rest of the
	// screen is where the text goes and is not carried.
	for y := range lines {
		out = append(out, pixels[y*image.Width:(y+1)*image.Width]...)
	}

	return out, nil
}

// pcShift moves every pixel up by sixteen, to sit above DAAD's own colours.
//
// A drawing using the top sixteen entries has nowhere to go and is refused
// rather than wrapped around, which is the reference's answer too
// (getShiftedColour, SCRMAKER.PAS:676-684). It is worth the error: a wrap would
// paint those pixels in the text colours and look almost right.
func pcShift(pixels []byte) ([]byte, error) {
	const highest = 256 - pcReservedColours

	out := make([]byte, len(pixels))

	for i, p := range pixels {
		if p >= highest {
			return nil, fmt.Errorf(
				"this picture uses colour %d, and %d to 255 have to be free so that the "+
					"palette can move up to make room for DAAD's own sixteen", p, highest)
		}

		out[i] = p + pcReservedColours
	}

	return out, nil
}

// pcPalette builds the palette that goes in the file: DAAD's sixteen, then the
// picture's own moved up to follow them.
//
// The last sixteen entries of the picture's palette fall off the end, which is
// the arithmetic rather than a decision — 240 entries is what is left of 256
// once sixteen are taken — and it is why pcShift refuses a picture that uses
// them.
func pcPalette(from [pcPaletteSize]byte) []byte {
	out := make([]byte, pcPaletteSize)

	copy(out, pcStandardPalette[:])
	copy(out[pcReservedColours*3:], from[:pcPaletteSize-pcReservedColours*3])

	return out
}

// pcPictureName is what a picture is called in the release: three digits and
// the extension, which is the mould the interpreter writes the number into.
func pcPictureName(number int) string {
	return fmt.Sprintf("%03d%s", number, pcPictureSuffix)
}

// pcPictures turns every PCX into its .MSD, in number order so that two builds
// of one adventure come out the same.
func pcPictures(
	pictures map[int][]byte, lines int, report Progress,
) (map[int][]byte, []int, error) {
	numbers := make([]int, 0, len(pictures))
	for n := range pictures {
		numbers = append(numbers, n)
	}

	sort.Ints(numbers)

	bodies, err := compressPictures(numbers, report, func(n int) ([]byte, error) {
		one, err := pcPicture(pictures[n], lines)
		if err != nil {
			return nil, fmt.Errorf("picture %d: %w", n, err)
		}

		return one, nil
	})
	if err != nil {
		return nil, nil, err
	}

	built := make(map[int][]byte, len(numbers))
	for i, n := range numbers {
		built[n] = bodies[i]
	}

	return built, numbers, nil
}
