// pcwpic turns the PC edition's illustrations into an Amstrad PCW tree's.
//
//	go run ./script/pcwpic PART1.CGA examples/AO/Amstrad/PCW/part1/gfx/scr
//	go run ./script/pcwpic PART1.SCR examples/AO/Amstrad/PCW/part1/gfx
//
// A .SCR is the other thing: the cover, which on a PCW is a file of its own
// called PARTE000.PIC and which the interpreter puts up by itself before the
// adventure starts. It comes out as gfx/presentation.pic.
//
// La Aventura Original was never released for the PCW, so a PCW tree has to
// take its drawings from somewhere. The PC's are the ones to take: CGA mode 4
// is 320 by 200 in four colours and the PCW is 720 by 256 in one, so what is
// lost is colour and nothing else — where the Spectrum's are 256 by 192 with
// one ink per eight-by-eight cell, and the Amstrad's are vectors that version
// 3 has no engine for.
//
// What makes this short is that the .CGA archive is laid out like the PCW's
// .DAT — six-byte header, 256 slots of ten, pictures behind — and packed like
// it too, with markers chosen per picture. Reading it is script/cgapic, which
// pcpcx shares.
//
// What comes out is one file per picture, named by the number a PICTURE names
// it with, holding the bitmap qpack expects: rows of 64 bytes, the high bit of
// each the leftmost pixel, a set bit white.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jorgefuertes/QDAAD/script/cgapic"
)

const (
	// The conversion, and it is Aventuras AD's own arithmetic rather than
	// ours. Cozumel shipped on both machines and the two archives can be held
	// side by side: its PC illustrations are 256 by 104 and its PCW ones are
	// 512 by 104, the same locations and the same count of them. They doubled
	// the width and left the height alone, which is what a PCW pixel being
	// about half as wide as it is tall asks for.
	//
	// The height is each picture's own and not a constant. La Aventura
	// Original's rooms are 256 by 112, which makes fourteen character rows
	// here — but its title is 256 by 176, which makes twenty-two, and forcing
	// that one to the height of a room cuts the frame off.
	widen = 2
)

// The four colours of CGA mode 4, palette 1, as how bright they are to the
// eye. Cyan and magenta are nearly the same brightness and flattening them
// together loses the water against the sky, so they are pulled apart by hand:
// what matters on a monochrome screen is that four colours stay four tones.
var tone = [4]int{0, 130, 85, 255}

// bayer is the threshold each pixel is held against, by where it sits in a
// two-by-two tile.
//
// An ordered dither and not Floyd–Steinberg, for two reasons and the second is
// the one that decided it. It looks like the machine: the drawings on the 1989
// PCW disks are regular hatching, not noise, because somebody dithered them by
// hand, and theirs repeat every two pixels — 0xCC and 0x33, one after the
// other, all down a Cozumel illustration.
//
// And it packs, which is the same thing said in bytes. A run-length scheme
// lives on bytes that repeat, and the tile's period is how often they do. The
// same thirty illustrations come to 150 KiB with the error diffused, 95 with a
// four-by-four tile and 82 with this one, and 82 is what leaves room for CP/M
// on the disk as well.
var bayer = [2][2]int{
	{0, 2},
	{3, 1},
}

// white says whether a tone is drawn at that place in the tile.
//
// Four tones over four thresholds, so each colour of the source gets a hatch
// of its own: nothing, a quarter, three quarters, and solid.
func white(tone, x, y int) bool { return tone > bayer[y%2][x%2]*64+32 }

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: pcwpic <archive.CGA> <directory>")
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
		pixels, width, height, ok := cgapic.Packed(data, number)
		if !ok {
			continue
		}

		body := encode(dither(pixels, width, height), width*widen, height)

		name := filepath.Join(dir, fmt.Sprintf("%03d.scr", number))
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
	fmt.Printf("%d KiB of bitmaps, before qpack packs them\n", total/1024)

	return nil
}

// dither turns four-colour pixels into one-colour ones, twice as wide.
//
// Four tones over the four thresholds of the tile, and nothing cleverer. It
// was tried: the drawings arrive stippled already — a CGA artist had four
// colours and got more by alternating pixels — so the thought was to resolve
// that stipple and to take edges hard, which would have kept a letter cut into
// rock legible. What it actually produced was a title screen that is a black
// rectangle. The stipple is not noise over the drawing, it is the drawing, and
// taking it apart loses more than it saves.
func dither(pixels []byte, width, height int) []byte {
	wide := width * widen
	stride := width / 4
	out := make([]byte, wide/8*height)

	for y := range height {
		for x := range width {
			b := pixels[y*stride+x/4]
			v := tone[(b>>(6-2*(x%4)))&0x03]

			// Two PCW pixels for each CGA one, each held against its own
			// place in the tile so that the hatching stays even.
			for i := range widen {
				at := x*widen + i
				if white(v, at, y) {
					out[y*(wide/8)+at/8] |= 0x80 >> (at % 8)
				}
			}
		}
	}

	return out
}

// encode puts the measurements in front of the bitmap.
//
// Four bytes, two little-endian words. Without them a file is a run of bits
// and nothing says how wide a row is — and the pictures are not all one shape:
// a room is 112 rows and the title is 176.
func encode(bitmap []byte, width, height int) []byte {
	return append([]byte{
		byte(width), byte(width >> 8),
		byte(height), byte(height >> 8),
	}, bitmap...)
}

// The cover, which is a different shape and a different file.
//
// A PC .SCR is a dump of CGA mode 4 video memory: 320 by 200, two bits to the
// pixel, and the rows in two halves — the even ones from the start and the odd
// ones from 0x2000 — because that is how the hardware addressed them.
//
// What comes out is 640 by 200 of one bit, the same doubling as a drawing, and
// laid out the way the PCW's own screen is: eight bytes to a character cell,
// which are its eight scanlines, eighty cells across and twenty-five down.
// That is 16000 bytes, and it is what PARTE000.PIC is on the 1989 disks.
const (
	cgaBank     = 0x2000
	cgaRowBytes = 80
	coverWidth  = 640
	coverHeight = 200
	cellHeight  = 8
)

func cover(data []byte, dir string) error {
	if len(data) < cgaBank+cgaRowBytes*coverHeight/2 {
		return fmt.Errorf("a CGA screen is 16384 bytes and this is %d", len(data))
	}

	grey := make([]int, coverWidth*coverHeight)

	for y := range coverHeight {
		// The even rows come from the start and the odd ones from the second
		// bank, which is how CGA addressed them.
		base := (y / 2) * cgaRowBytes
		if y%2 == 1 {
			base += cgaBank
		}

		for x := range coverWidth / widen {
			b := data[base+x/4]
			v := tone[(b>>(6-2*(x%4)))&0x03]

			for i := range widen {
				grey[y*coverWidth+x*widen+i] = v
			}
		}
	}

	bits := make([]byte, coverWidth/8*coverHeight)

	for y := range coverHeight {
		for x := range coverWidth {
			if white(grey[y*coverWidth+x], x, y) {
				bits[y*(coverWidth/8)+x/8] |= 0x80 >> (x % 8)
			}
		}
	}

	// And out in the machine's own order: a cell is its eight scanlines, one
	// after the other, which is not how a bitmap is laid out anywhere else.
	out := make([]byte, 0, len(bits))

	for row := range coverHeight / cellHeight {
		for col := range coverWidth / 8 {
			for line := range cellHeight {
				out = append(out, bits[(row*cellHeight+line)*(coverWidth/8)+col])
			}
		}
	}

	name := filepath.Join(dir, "presentation.pic")
	if err := os.WriteFile(name, out, 0o644); err != nil {
		return err
	}

	fmt.Printf("the cover into %s: %d by %d, %d bytes\n", name, coverWidth, coverHeight, len(out))

	return nil
}
