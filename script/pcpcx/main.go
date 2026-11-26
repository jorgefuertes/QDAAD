// pcpcx turns the PC edition's illustrations into a PC tree's.
//
//	go run ./script/pcpcx PART1.CGA examples/AO/PC/part1/gfx/scr
//
// This is the one conversion in here that is not a port. La Aventura Original
// had a PC release and these are its own drawings, out of its own archive; what
// changes is only the mode they are drawn in. The 1989 release was CGA mode 4,
// four colours on a 320 by 200 screen; a version 3 interpreter draws VGA256 on
// the same 320 by 200, and reads a .MSD built from a PCX. So the pixels come
// across untouched and what this writes is a container, not a redrawing.
//
// Which is why there is no dither here and no palette guessing. The four
// colours are mode 4 palette 1 at high intensity — black, cyan, magenta, white
// — and they are the four the decompiler already renders these drawings with,
// so a PNG out of work/decomp and a PCX out of here are the same picture.
//
// The height is each drawing's own and not a constant. La Aventura Original's
// rooms are 256 by 112 and its title is 256 by 176, and one height for both
// cuts the frame off the title — the same thing IMGLINES does to the Spectrum's
// tallest drawings. See pictureLines in internal/pack.
//
// It needs work/, so a fresh checkout does not run it: what the tree carries is
// the PCX files it produced.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jorgefuertes/QDAAD/script/cgapic"
)

// The screen a VGA256 release draws on, which is what a drawing has to fit in
// rather than what it has to be. These are 256 across and go out at 256: the
// .MSD the packager builds carries a fixed position, so it is the header that
// centres them on the screen and not sixty-four columns of black.
//
// Which is also why these files are not SCRMAKER's kind. It takes a whole
// screen and crops from the top left, so it refuses a PCX that is not 320 or
// 640 wide; this writes the drawing and nothing around it.
const screenWidth = 320

// The four colours of CGA mode 4, palette 1, high intensity.
//
// Read off the drawings the decompiler renders rather than off a table of CGA
// modes: work/decomp/ao/pc/part1/gfx/png holds these as RGB, and these are the
// four values in them. A palette that merely looked like CGA would make this
// tree disagree with every PNG in the checkout.
var palette = [cgapic.Colours][3]byte{
	{0x00, 0x00, 0x00}, // black
	{0x55, 0xFF, 0xFF}, // cyan
	{0xFF, 0x55, 0xFF}, // magenta
	{0xFF, 0xFF, 0xFF}, // white
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: pcpcx <archive.CGA> <directory>")
		os.Exit(2)
	}

	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(archive, dir string) error {
	data, err := os.ReadFile(archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if strings.EqualFold(filepath.Ext(archive), ".scr") {
		return cover(data, dir)
	}

	numbers := []int{}
	total := 0

	for number := range cgapic.Slots {
		pixels, width, height, ok := cgapic.Picture(data, number)
		if !ok {
			continue
		}

		if width > screenWidth {
			return fmt.Errorf("picture %d is %d wide and a screen is %d",
				number, width, screenWidth)
		}

		body := pcx(pixels, width, height)

		name := filepath.Join(dir, fmt.Sprintf("%03d.pcx", number))
		if err := os.WriteFile(name, body, 0o644); err != nil {
			return err
		}

		numbers = append(numbers, number)
		total += len(body)
	}

	if len(numbers) == 0 {
		return fmt.Errorf("%s holds no illustrations", archive)
	}

	sort.Ints(numbers)
	fmt.Printf("%d illustrations into %s: %v\n", len(numbers), dir, numbers)
	fmt.Printf("%d KiB of PCX, before the packager turns them into .MSD\n", total/1024)

	return nil
}

// pcx writes an 8-bit PCX.
//
// The format is older than anything else here and it shows: the palette of a
// 256-colour file is not in the header but in the last 769 bytes of the file,
// behind a marker, because the header was fixed before anybody wanted that
// many colours.
func pcx(pixels []byte, width, height int) []byte {
	out := make([]byte, 128)

	out[0] = 0x0A // the only thing identifying the format
	out[1] = 5    // version
	out[2] = 1    // run encoded
	out[3] = 8    // bits a pixel

	word := func(at, value int) {
		out[at] = byte(value)
		out[at+1] = byte(value >> 8)
	}

	word(8, width-1)   // xmax, with xmin left at zero
	word(10, height-1) // ymax
	word(12, 72)       // the dots an inch nothing reads
	word(14, 72)
	out[65] = 1     // one plane
	word(66, width) // bytes a line
	word(68, 1)     // a colour palette

	for y := range height {
		out = append(out, encode(pixels[y*width:(y+1)*width])...)
	}

	out = append(out, 12) // the marker that says 256 colours follow

	for i := range 256 {
		if i < len(palette) {
			out = append(out, palette[i][0], palette[i][1], palette[i][2])

			continue
		}

		out = append(out, 0, 0, 0)
	}

	return out
}

// encode run-encodes one scanline.
//
// A byte with its top two bits set is a count in the low six and the value
// follows, so a run is at most 63 — and a value with those bits set has to be
// written as a run of one even when it stands alone, or a reader takes it for a
// count. Which is how a PCX of a busy picture comes out bigger than the picture.
func encode(row []byte) []byte {
	out := make([]byte, 0, len(row))

	for at := 0; at < len(row); {
		run := 1
		for at+run < len(row) && row[at+run] == row[at] && run < 63 {
			run++
		}

		if run > 1 || row[at] >= 0xC0 {
			out = append(out, 0xC0|byte(run), row[at])
		} else {
			out = append(out, row[at])
		}

		at += run
	}

	return out
}

// The cover, which is a different file and a different shape.
//
// A PC .SCR is a dump of CGA mode 4 video memory: 320 by 200, two bits to the
// pixel, and the rows in two halves — the even ones from the start and the odd
// ones from 0x2000 — because that is how the hardware addressed them. There is
// no header and no compression, so this is the one picture here that arrives as
// the screen it was.
//
// It comes out as loading.pcx, which the packager turns into DAAD.MSD: the
// picture the interpreter puts up before the adventure starts.
const (
	cgaBank     = 0x2000
	cgaRowBytes = 80
	coverWidth  = 320
	coverHeight = 200
	coverName   = "loading.pcx"
)

func cover(data []byte, dir string) error {
	if len(data) < cgaBank+cgaRowBytes*coverHeight/2 {
		return fmt.Errorf("a CGA screen is 16384 bytes and this is %d", len(data))
	}

	pixels := make([]byte, coverWidth*coverHeight)

	for y := range coverHeight {
		base := (y / 2) * cgaRowBytes
		if y%2 == 1 {
			base += cgaBank
		}

		for x := range coverWidth {
			b := data[base+x/cgapic.PixelsPerByte]
			pixels[y*coverWidth+x] = (b >> (6 - 2*(x%cgapic.PixelsPerByte))) & 0x03
		}
	}

	name := filepath.Join(dir, coverName)
	if err := os.WriteFile(name, pcx(pixels, coverWidth, coverHeight), 0o644); err != nil {
		return err
	}

	fmt.Printf("the cover into %s\n", name)

	return nil
}
