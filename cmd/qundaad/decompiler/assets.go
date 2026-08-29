package decompiler

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// Where the things that travelled with a database are written, alongside the
// source: the character sets in one directory and the graphics in the other.
const (
	fontsDir    = "chr"
	graphicsDir = "gfx"
)

// asset is a file that shipped with a database: a character set, a screen, or
// an archive of the illustrations.
type asset struct {
	name string
	data []byte
	// amiga marks a file that came off an Amiga disk. Its pictures share a
	// format with the Atari ones but not the order of the bit planes, and
	// nothing inside the file says which it is.
	amiga bool
}

// The Amstrad header that the tooling of the time put on its binaries. Aventuras
// AD worked on the CPC and the PCW, and the files reached the other machines
// with it still on the front, character sets included.
const (
	amsdosHeaderSize = 128
	amsdosChecksumAt = 67
)

// A character set is 256 glyphs of eight rows, one byte to the row. The glyphs
// are drawn six pixels wide inside the eight — one of the files is called
// AO6X8.BIN, which says as much.
const (
	glyphHeight = 8
	numGlyphs   = 256
	fontSize    = numGlyphs * glyphHeight
	atlasAcross = 16 // glyphs to a row in the PNG
)

// The screens are raw dumps of the video memory of the machine they were drawn
// for, so their size is what tells them apart.
const (
	cgaScreenSize  = 16384           // two banks of 8000 bytes used, the rest padding
	cgaBankSize    = 0x2000          // the odd scanlines start here
	egaScreenSize  = 1 + 4*8000      // a byte naming the mode, then four bit planes
	screenWidth    = 320             //
	screenHeight   = 200             //
	cgaBytesPerRow = screenWidth / 4 // four pixels to the byte
	egaBytesPerRow = screenWidth / 8 // eight to the byte, in each plane
	egaPlaneSize   = 8000
)

// The 68000 machines used Degas, the drawing program of the Atari ST, and its
// low resolution format: a word naming the resolution, sixteen more holding the
// palette, and then the picture. Both the Atari and the Amiga editions came
// this way.
//
// A file can be longer than that. Degas Elite appended colour cycling data, and
// the Amiga editions carry a hundred-odd bytes more of something; neither is
// needed to draw the picture, so both are ignored.
const (
	degasHeaderSize = 2 + 2*degasColours
	degasColours    = 16
	degasLowRes     = 0
	degasScreenSize = degasHeaderSize + 4*egaPlaneSize

	// A row is stored sixteen pixels at a time: one word from each of the four
	// planes, then the next sixteen pixels.
	degasBytesPerRow = screenWidth / 8 * 4
)

// The four colours of CGA mode 4 in its usual palette, and the sixteen of EGA
// mode 0Dh. Neither is written down in the files: the program sets the palette
// when it starts, so these are the defaults of the mode.
var (
	cgaPalette = [4]color.RGBA{
		{0x00, 0x00, 0x00, 0xFF}, {0x55, 0xFF, 0xFF, 0xFF},
		{0xFF, 0x55, 0xFF, 0xFF}, {0xFF, 0xFF, 0xFF, 0xFF},
	}

	egaPalette = [16]color.RGBA{
		{0x00, 0x00, 0x00, 0xFF}, {0x00, 0x00, 0xAA, 0xFF},
		{0x00, 0xAA, 0x00, 0xFF}, {0x00, 0xAA, 0xAA, 0xFF},
		{0xAA, 0x00, 0x00, 0xFF}, {0xAA, 0x00, 0xAA, 0xFF},
		{0xAA, 0x55, 0x00, 0xFF}, {0xAA, 0xAA, 0xAA, 0xFF},
		{0x55, 0x55, 0x55, 0xFF}, {0x55, 0x55, 0xFF, 0xFF},
		{0x55, 0xFF, 0x55, 0xFF}, {0x55, 0xFF, 0xFF, 0xFF},
		{0xFF, 0x55, 0x55, 0xFF}, {0xFF, 0x55, 0xFF, 0xFF},
		{0xFF, 0xFF, 0x55, 0xFF}, {0xFF, 0xFF, 0xFF, 0xFF},
	}
)

// writeAssets saves what came alongside the database, in two directories: the
// character sets in one and everything drawn in the other.
//
// What is written are pictures, numbered from one, and beside each the bytes it
// was made from. The names of the original files are not carried over: the
// directory already says which part of the adventure this is, and PART1.SCR.png
// inside part1/gfx says it twice and adds an extension that means nothing.
func writeAssets(assets []asset, outputDir string, opts Options) error {
	var fonts, graphics []asset

	for _, a := range assets {
		if isFont(a.name) {
			fonts = append(fonts, a)
		} else {
			graphics = append(graphics, a)
		}
	}

	if err := writeFonts(fonts, filepath.Join(outputDir, fontsDir), opts); err != nil {
		return err
	}

	return writeGraphics(graphics, filepath.Join(outputDir, graphicsDir), opts)
}

// writeFonts draws each character set as a sheet of its glyphs. A part carries
// one or two of them, and they are numbered in the order the disk holds them.
func writeFonts(fonts []asset, into string, opts Options) error {
	if len(fonts) == 0 {
		return nil
	}

	if err := os.MkdirAll(into, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", into, err)
	}

	for i, a := range fonts {
		if err := keepOriginal(a, into, opts); err != nil {
			return err
		}

		img, drawn := draw(a)
		if !drawn {
			continue
		}

		if err := writePNG(filepath.Join(into, numbered(i+1, ".png")), img); err != nil {
			return err
		}
	}

	return nil
}

// writeGraphics draws everything else into one directory: the loading screen as
// scr.png, and the illustrations of the archives numbered from one.
//
// The archives are numbered through, not restarted for each, because a part has
// only ever one of them and numbering per archive would invite a collision that
// never shows up in what we have.
func writeGraphics(graphics []asset, into string, opts Options) error {
	if len(graphics) == 0 {
		return nil
	}

	if err := os.MkdirAll(into, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", into, err)
	}

	drawings, screens := 0, 0

	for _, a := range graphics {
		if err := keepOriginal(a, into, opts); err != nil {
			return err
		}

		if pieces, index, isArchive := splitArchive(a); isArchive {
			if err := writePieces(pieces, index, into, &drawings, opts); err != nil {
				return err
			}

			continue
		}

		img, drawn := draw(a)
		if !drawn {
			continue
		}

		// A part has one loading screen. Should another ever turn up it is
		// numbered rather than quietly overwriting the first.
		name := "scr.png"
		if screens++; screens > 1 {
			name = fmt.Sprintf("scr%d.png", screens)
		}

		if err := writePNG(filepath.Join(into, name), img); err != nil {
			return err
		}
	}

	return nil
}

// keepOriginal writes a file as it came, under the name it came with, unless
// the caller asked for only what can be read.
func keepOriginal(a asset, into string, opts Options) error {
	if !opts.Binaries {
		return nil
	}

	// The name it came with, and .bin when it came without an extension.
	name := filepath.Base(a.name)
	if filepath.Ext(name) == "" {
		name += ".bin"
	}

	if err := os.WriteFile(filepath.Join(into, name), a.data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}

	return nil
}

// numbered gives a piece its name: three digits and an extension.
func numbered(at int, extension string) string {
	return fmt.Sprintf("%03d%s", at, extension)
}

// writePieces writes the illustrations of one archive, each as it lies in it
// and, where the drawing can be read, as a picture beside it.
//
// The index goes with them, naming for each what is known: where it lies, how
// big it is, which locations use it and in what palette. For the archives whose
// drawings are still unread that index is the whole of what we have.
func writePieces(pieces []piece, index, into string, from *int, opts Options) error {
	var listing strings.Builder

	listing.WriteString(index)

	for _, p := range pieces {
		*from++

		if opts.Binaries {
			at := filepath.Join(into, numbered(*from, ".bin"))
			if err := os.WriteFile(at, p.data, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", at, err)
			}
		}

		if p.img == nil {
			continue
		}

		if err := writePNG(filepath.Join(into, numbered(*from, ".png")), p.img); err != nil {
			return err
		}
	}

	at := filepath.Join(into, "index.txt")
	if err := os.WriteFile(at, []byte(listing.String()), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", at, err)
	}

	return nil
}

// draw turns an asset into a picture, if it is one of the kinds we can read.
func draw(a asset) (image.Image, bool) {
	body := withoutAmsdosHeader(a.data)

	switch {
	case isFont(a.name) && len(body) >= fontSize:
		return fontAtlas(body), true

	case len(body) == cgaScreenSize:
		return cgaScreen(body), true

	case len(body) == egaScreenSize:
		// The first byte names the video mode; the planes follow it.
		return egaScreen(body[1:]), true

	case isDegasScreen(a.name, body):
		return degasScreen(body, a.amiga), true
	}

	return nil, false
}

func beWord(d []byte) int { return int(d[0])<<8 | int(d[1]) }

// isDegasScreen is deliberately narrow. A Degas file opens with a zero word for
// low resolution, and so does every archive of illustrations, which would
// otherwise be drawn as one enormous screen of nonsense. So the name has to say
// it is a screen, and the size has to be a screen's size, give or take the odd
// trailer some of them carry.
func isDegasScreen(name string, body []byte) bool {
	const mostTrailing = 256

	if !strings.EqualFold(filepath.Ext(name), ".SCR") {
		return false
	}

	if len(body) < degasScreenSize || len(body) > degasScreenSize+mostTrailing {
		return false
	}

	return beWord(body) == degasLowRes
}

// degasScreen draws a low resolution Degas picture.
//
// The palette comes with it, which is what makes these worth reading: unlike
// the PC screens, the colours are not guesswork. Each entry is a word holding
// three nibbles, one to a component.
//
// The two machines lay the bit planes out differently and the file does not
// say which it is. The Atari interleaves them a word at a time along each row,
// as its hardware reads them; the Amiga keeps each plane whole, one after
// another. Read one as the other and the picture comes out tiled and scrambled,
// which is why this is told by where the file was found rather than guessed.
func degasScreen(data []byte, amiga bool) image.Image {
	palette := degasPalette(data[2:degasHeaderSize])

	img := image.NewNRGBA(image.Rect(0, 0, screenWidth, screenHeight))

	for y := range screenHeight {
		for x := range screenWidth {
			colour := 0

			for plane := range 4 {
				var at, shift int

				if amiga {
					at = plane*egaPlaneSize + y*egaBytesPerRow + x/8
					shift = 7 - x%8
				} else {
					// A word from each plane, then the next sixteen pixels.
					at = y*degasBytesPerRow + (x/16)*8 + plane*2 + (x%16)/8
					shift = 7 - x%8
				}

				if data[degasHeaderSize+at]&(1<<shift) != 0 {
					colour |= 1 << plane
				}
			}

			img.Set(x, y, palette[colour])
		}
	}

	return img
}

// degasPalette turns the sixteen words into colours.
//
// How wide a component is depends on the hardware and the file does not say:
// the Atari ST gives each three bits, and the Amiga and the STE four. It is
// read off the values instead — a component above seven cannot be a three-bit
// one — which is a question of colour depth and not of machine. Chichén Itzá
// is a four-bit picture on an Atari disk, drawn for the STE.
func degasPalette(words []byte) [degasColours]color.NRGBA {
	var palette [degasColours]color.NRGBA

	most := 0

	for i := range degasColours {
		word := beWord(words[i*2:])
		for _, shift := range []int{8, 4, 0} {
			most = max(most, (word>>shift)&0x0F)
		}
	}

	full := 7
	if most > 7 {
		full = 15
	}

	for i := range degasColours {
		word := beWord(words[i*2:])

		component := func(shift int) uint8 {
			return uint8(min((word>>shift)&0x0F, full) * 0xFF / full)
		}

		palette[i] = color.NRGBA{R: component(8), G: component(4), B: component(0), A: 0xFF}
	}

	return palette
}

func isFont(name string) bool {
	switch strings.ToUpper(filepath.Ext(name)) {
	case ".CHR", ".CH0":
		return true
	}

	return false
}

// withoutAmsdosHeader drops the 128-byte header if there is one. It is only
// there if it adds up: the last two bytes of it hold the sum of the rest.
func withoutAmsdosHeader(data []byte) []byte {
	if len(data) < amsdosHeaderSize {
		return data
	}

	sum := 0
	for _, b := range data[:amsdosChecksumAt] {
		sum += int(b)
	}

	declared := int(data[amsdosChecksumAt]) | int(data[amsdosChecksumAt+1])<<8
	if sum != declared || sum == 0 {
		return data
	}

	return data[amsdosHeaderSize:]
}

// fontAtlas lays the 256 glyphs out in a grid, sixteen to the row, at the size
// they were drawn: black ink on white, the way a specimen sheet is read.
//
// One pixel of the file is one pixel of the image. Scaling it up would be
// kinder to the eye and a lie about what is in there.
func fontAtlas(font []byte) image.Image {
	const across = atlasAcross

	var (
		ink    = color.NRGBA{0x00, 0x00, 0x00, 0xFF}
		ground = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
	)

	down := numGlyphs / across
	img := image.NewNRGBA(image.Rect(0, 0, across*glyphHeight, down*glyphHeight))

	for at := range img.Pix {
		img.Pix[at] = 0xFF
	}

	for code := range numGlyphs {
		left := (code % across) * glyphHeight
		top := (code / across) * glyphHeight

		for row := range glyphHeight {
			bits := font[code*glyphHeight+row]

			for column := range glyphHeight {
				shade := ground
				if bits&(0x80>>column) != 0 {
					shade = ink
				}

				img.SetNRGBA(left+column, top+row, shade)
			}
		}
	}

	return img
}

// cgaScreen draws a dump of CGA mode 4. The scanlines are not in order: the
// even ones fill the first half of the buffer and the odd ones the second, so
// that the hardware could read a whole field without waiting.
func cgaScreen(data []byte) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, screenWidth, screenHeight))

	for y := range screenHeight {
		row := (y%2)*cgaBankSize + (y/2)*cgaBytesPerRow

		for x := range screenWidth {
			at := row + x/4
			if at >= len(data) {
				continue
			}

			// Four pixels to the byte, the leftmost in the top bits.
			shift := 6 - 2*(x%4)
			img.Set(x, y, cgaPalette[(data[at]>>shift)&0x03])
		}
	}

	return img
}

// egaScreen draws a dump of EGA mode 0Dh. Each of the four bit planes holds one
// bit of the colour of every pixel, so a pixel is read by taking the same bit
// out of all four.
func egaScreen(data []byte) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, screenWidth, screenHeight))

	for y := range screenHeight {
		for x := range screenWidth {
			at := y*egaBytesPerRow + x/8
			mask := byte(0x80 >> (x % 8))

			colour := 0

			for plane := range 4 {
				if data[plane*egaPlaneSize+at]&mask != 0 {
					colour |= 1 << plane
				}
			}

			img.Set(x, y, egaPalette[colour])
		}
	}

	return img
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}

	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return f.Close()
}
