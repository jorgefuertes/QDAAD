package decompiler

import (
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

const charset = "../../../work/aventuras/La_Aventura_Original/PC/PART1.CHR"

func readAsset(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("the file is not in the tree: %v", err)
	}

	return data
}

// The character sets carry the 128-byte header the Amstrad tooling put on its
// binaries, and it is recognised by its own checksum rather than by its size.
func TestAmsdosHeaderIsRecognised(t *testing.T) {
	data := readAsset(t, charset)
	require.Len(t, data, amsdosHeaderSize+fontSize)

	body := withoutAmsdosHeader(data)
	require.Len(t, body, fontSize, "the header comes off, leaving 256 glyphs of 8 bytes")

	// Nothing is taken off something that does not add up.
	broken := append([]byte(nil), data...)
	broken[amsdosChecksumAt] ^= 0xFF
	require.Len(t, withoutAmsdosHeader(broken), len(broken))
}

// TestGlyphsAreTheDaadCharset reads the shapes rather than trusting the tables
// of other people's tools. Codes 16 to 31 are where DAAD keeps the letters
// Spanish needs and ASCII has not.
func TestGlyphsAreTheDaadCharset(t *testing.T) {
	font := withoutAmsdosHeader(readAsset(t, charset))

	glyph := func(code int) []byte { return font[code*glyphHeight : (code+1)*glyphHeight] }

	// An inverted exclamation mark: the dot on top, the stroke below it.
	require.Equal(t,
		[]byte{0b00100000, 0, 0b00100000, 0b00100000, 0b00100000, 0b00100000, 0b00100000, 0},
		glyph(17), "code 17 is the inverted exclamation mark")

	// A and á differ by the accent, and by nothing else below it.
	a, accented := glyph('a'), glyph(21)
	require.Equal(t, byte(0b00010000), accented[0], "the accent sits above the letter")
	require.Equal(t, a[2:], accented[2:], "and the rest of the two is the same shape")

	// Every letter is drawn six pixels wide inside its eight, which is what
	// the name of one of these files, AO6X8.BIN, says. That holds for the
	// accented codes as well as for ASCII.
	for code := 16; code < 127; code++ {
		for _, row := range glyph(code) {
			require.Zero(t, row&0x03, "glyph %d uses more than six columns", code)
		}
	}

	// The top half of the set is not letters. It holds fills for shading, and
	// those do use the full eight columns, edge to edge so they tile.
	patterns := 0

	for code := 128; code < numGlyphs; code++ {
		for _, row := range glyph(code) {
			if row&0x03 != 0 {
				patterns++

				break
			}
		}
	}

	require.Equal(t, 26, patterns, "the shading patterns of the upper half")
}

// nrgba is what comes back out of the images, which are NRGBA. The palettes are
// written as RGBA because that is how the modes are usually set down.
func nrgba(c color.RGBA) color.NRGBA {
	return color.NRGBA(c)
}

func TestFontAtlasSize(t *testing.T) {
	img := fontAtlas(withoutAmsdosHeader(readAsset(t, charset)))

	bounds := img.Bounds()
	require.Equal(t, atlasAcross*glyphHeight, bounds.Dx())
	require.Equal(t, (numGlyphs/atlasAcross)*glyphHeight, bounds.Dy())
}

// TestCgaScreenIsDeinterlaced guards the part that is easy to get wrong. CGA
// keeps the even scanlines in one half of the buffer and the odd ones in the
// other, so reading it straight through gives a picture squashed into the top
// half and repeated.
func TestCgaScreenIsDeinterlaced(t *testing.T) {
	data := make([]byte, cgaScreenSize)

	// Colour 3 across the whole of row 1, which lives in the second bank.
	for x := range cgaBytesPerRow {
		data[cgaBankSize+x] = 0xFF
	}

	img := cgaScreen(data)

	require.Equal(t, nrgba(cgaPalette[3]), img.At(0, 1), "row 1 comes from the second bank")
	require.Equal(t, nrgba(cgaPalette[0]), img.At(0, 0), "row 0 comes from the first")
	require.Equal(t, nrgba(cgaPalette[0]), img.At(0, 3), "and no other row was painted")
}

// TestEgaScreenCombinesThePlanes checks the other easy mistake: a pixel's
// colour is one bit from each of the four planes, not four bits from one.
func TestEgaScreenCombinesThePlanes(t *testing.T) {
	data := make([]byte, 4*egaPlaneSize)

	// The leftmost pixel: set in planes 0 and 2, which makes colour 5.
	data[0*egaPlaneSize] = 0x80
	data[2*egaPlaneSize] = 0x80

	img := egaScreen(data)

	require.Equal(t, nrgba(egaPalette[5]), img.At(0, 0))
	require.Equal(t, nrgba(egaPalette[0]), img.At(1, 0))
}

// A picture is only written for what can be drawn. The archives of
// illustrations cannot be yet, and are saved as they came.
func TestOnlyWhatCanBeDrawnIsDrawn(t *testing.T) {
	_, drawn := draw(asset{name: "PART1.CHR", data: readAsset(t, charset)})
	require.True(t, drawn)

	_, drawn = draw(asset{name: "PART1.CGA", data: make([]byte, 78840)})
	require.False(t, drawn, "the archives are not a single picture")
}

func TestAssetsAreMatchedByName(t *testing.T) {
	candidates := []asset{
		{name: "PART1.CHR"},
		{name: "PART1.CGA"},
		{name: "PART1.SCR"},
		{name: "PART2.CHR"},   // another part
		{name: "PART1.DDB"},   // the database itself
		{name: "AD.EXE"},      // the interpreter
		{name: "c/PART1.CGS"}, // in a subdirectory of the disk
	}

	var names []string
	for _, a := range assetsFor("PART1.DDB", candidates) {
		names = append(names, a.name)
	}

	require.Equal(t, []string{"PART1.CHR", "PART1.CGA", "PART1.SCR", "PART1.CGS"}, names)
}

// TestDegasLayoutFollowsTheMachine covers the difference that scrambles a
// picture if it is got wrong. The Atari interleaves the four bit planes a word
// at a time along each row; the Amiga keeps each plane whole. Nothing in the
// file says which, so it is taken from the disk it came off.
func TestDegasLayoutFollowsTheMachine(t *testing.T) {
	build := func(amiga bool) []byte {
		d := make([]byte, degasScreenSize)

		// Colour 15: every plane set, for the leftmost eight pixels of row 1.
		for plane := range 4 {
			at := degasHeaderSize
			if amiga {
				at += plane*egaPlaneSize + egaBytesPerRow
			} else {
				at += degasBytesPerRow + plane*2
			}

			d[at] = 0xFF
		}

		// A palette of three-bit white in the last entry.
		d[2+15*2], d[2+15*2+1] = 0x07, 0x77

		return d
	}

	white := color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}

	for _, tc := range []struct {
		name  string
		amiga bool
	}{
		{"atari", false}, {"amiga", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := degasScreen(build(tc.amiga), tc.amiga)

			require.Equal(t, white, img.At(0, 1), "the pixels that were set")
			require.Equal(t, white, img.At(7, 1))
			require.NotEqual(t, white, img.At(8, 1), "and no others")
			require.NotEqual(t, white, img.At(0, 0))

			// Read with the other machine's layout, it lands somewhere else.
			require.NotEqual(t, white, degasScreen(build(tc.amiga), !tc.amiga).At(0, 1))
		})
	}
}

// The palette is three bits a component on the Atari and four on the Amiga and
// the STE, and which it is has to be read off the values.
func TestDegasPaletteDepth(t *testing.T) {
	var threeBit, fourBit [2 + 2*degasColours]byte

	threeBit[2], threeBit[3] = 0x07, 0x77 // white, if seven is the top
	fourBit[2], fourBit[3] = 0x0F, 0xFF   // white, if fifteen is

	require.Equal(t, color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}, degasPalette(threeBit[2:])[0])
	require.Equal(t, color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}, degasPalette(fourBit[2:])[0])

	// Seven read as a four-bit value would come out halfway, not white.
	fourBit[4], fourBit[5] = 0x07, 0x77
	require.Equal(t, uint8(0x77), degasPalette(fourBit[2:])[1].R)
}

// The three archives the two shapes appear in. Two of them live inside a disk
// image rather than beside it, which is the usual way round for these.
const (
	archive68000   = "../../../work/aventuras/La_Aventura_Original/AtariST/ORIGINAL.ST"
	archiveLater   = "../../../work/aventuras/Los_templos_sagrados/AtariST/TEMPLOS1.ST"
	archiveLaterPC = "../../../work/aventuras/Los_templos_sagrados/PC/PART1.DAT"
)

// readArchive reads PART1.DAT, out of a disk image when that is where it is.
func readArchive(t *testing.T, path string) []byte {
	t.Helper()

	data := readAsset(t, path)
	if strings.HasSuffix(strings.ToUpper(path), ".DAT") {
		return data
	}

	volume, err := media.Open(path)
	require.NoError(t, err)

	files, err := volume.Files()
	require.NoError(t, err)

	for _, f := range files {
		if strings.EqualFold(f.Name, "PART1.DAT") {
			return f.Data
		}
	}

	t.Fatalf("%s holds no PART1.DAT", path)

	return nil
}

// TestUnpackIllustration decodes a picture the way the interpreter does. The
// scheme is the interpreter's own — a mask in the picture saying which of the
// sixteen colours carry a run length — so nothing but the real thing tests it.
func TestUnpackIllustration(t *testing.T) {
	// The first location of La Aventura Original, which is of the earlier of
	// the two archive shapes: big-endian, and its pixels gathered a bit from
	// each of four bytes.
	layout := daadLayouts[0]

	data := readArchive(t, archive68000)

	at := layout.long(data, layout.header+3*layout.record)
	require.Equal(t, layout.firstPicture(), at,
		"the first picture begins where the table ends")

	width, height, planes, ok := unpackDAAD(data[at:], layout)
	require.True(t, ok)
	require.Equal(t, 240, width)
	require.Equal(t, 96, height)

	// Half a byte to the pixel, which is how the interpreter works the size out.
	require.Len(t, planes, width/2*height)

	// Every colour of the palette is used somewhere in it, which a picture
	// decoded wrong would not manage.
	seen := map[int]bool{}

	for pixel := range width * height {
		base := pixel / daadGroupBits * daadGroupBytes
		bit := 7 - pixel%daadGroupBits

		colour := 0

		for plane := range daadGroupBytes {
			if planes[base+plane]&(1<<bit) != 0 {
				colour |= 1 << plane
			}
		}

		seen[colour] = true
	}

	require.Len(t, seen, degasColours, "all sixteen colours appear")
}

// TestBothArchiveShapesRead covers the difference between the generations. The
// later adventures widened the slot and changed how the pixels are gathered,
// and the PC editions of those hold the same archive the other way round.
func TestBothArchiveShapesRead(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		nibbles bool
		little  bool
	}{
		{name: "the earlier shape", path: archive68000},
		{name: "the later shape", path: archiveLater, nibbles: true},
		{name: "the later shape on a PC", path: archiveLaterPC, nibbles: true, little: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pieces, index, ok := daadArchive(readArchive(t, tc.path))
			require.True(t, ok, "the archive reads")
			require.NotEmpty(t, pieces)

			drawn := 0

			for _, p := range pieces {
				if p.img != nil {
					drawn++
				}
			}

			require.Equal(t, len(pieces), drawn, "every picture in it draws")
			require.Contains(t, index, "bit planes")
		})
	}
}

// An archive of one shape must not read as the other. Both put their first
// picture where their own table ends, and only one reading can be right.
func TestArchiveShapesDoNotCross(t *testing.T) {
	data := readArchive(t, archiveLater)

	_, _, ok := daadArchiveAs(data, daadLayouts[0])
	require.False(t, ok, "the later archive is not of the earlier shape")

	later := daadLayouts[1]
	later.little = true

	_, _, ok = daadArchiveAs(data, later)
	require.False(t, ok, "nor is it little-endian")
}

// A picture whose header does not say it is compressed, or whose body ends
// before the mask, is not one we can read.
func TestUnpackRejectsRubbish(t *testing.T) {
	layout := daadLayouts[0]

	_, _, _, ok := unpackDAAD([]byte{0x00, 0xF0, 0x00, 0x60, 0x00, 0x00, 0x00, 0x00}, layout)
	require.False(t, ok, "the flag says it is not compressed")

	_, _, _, ok = unpackDAAD([]byte{0x80, 0xF0, 0x00, 0x60}, layout)
	require.False(t, ok, "too short to hold the mask")
}

// TestPCPictureLayoutsDiffer covers the mistake that was made and corrected: a
// CGA picture is not planed. Read as planes it still looks like something —
// which is why it went unnoticed — so the test is that the two readings differ
// and that each puts the pixels where its own mode says.
func TestPCPictureLayoutsDiffer(t *testing.T) {
	const width, height = 8, 2

	// A CGA picture: four pixels to a byte, rows one after another. The first
	// byte paints colours 3, 2, 1, 0 across the top left.
	cga := make([]byte, pcImageHeader+width*height*2/8)
	cga[0], cga[2] = width, height
	cga[pcImageHeader] = 0b11100100

	img, ok := pcPicture(cga, 2)
	require.True(t, ok)
	require.Equal(t, nrgba(cgaPalette[3]), img.At(0, 0))
	require.Equal(t, nrgba(cgaPalette[2]), img.At(1, 0))
	require.Equal(t, nrgba(cgaPalette[1]), img.At(2, 0))
	require.Equal(t, nrgba(cgaPalette[0]), img.At(3, 0))

	// An EGA picture: four planes, each whole. A pixel takes one bit from each,
	// so setting the top bit of the first byte of planes 0 and 2 makes colour 5.
	ega := make([]byte, pcImageHeader+width*height*4/8)
	ega[0], ega[2] = width, height

	perPlane := width * height / 8
	ega[pcImageHeader+0*perPlane] = 0x80
	ega[pcImageHeader+2*perPlane] = 0x80

	img, ok = pcPicture(ega, 4)
	require.True(t, ok)
	require.Equal(t, nrgba(egaPalette[5]), img.At(0, 0))
	require.Equal(t, nrgba(egaPalette[0]), img.At(1, 0))
}

// Only the pictures stored as they stand are drawn. The top bit of the first
// word says the drawing is compressed, and that scheme is not read yet.
func TestPCCompressedPicturesAreNotDrawn(t *testing.T) {
	body := make([]byte, pcImageHeader+64)
	body[0], body[1] = 8, 0x80 // the width, with the compressed bit set
	body[2] = 8

	_, ok := pcPicture(body, 4)
	require.False(t, ok)
}

// TestPCUnpack covers the scheme DMG.EXE writes: a byte, and after it a count
// when that byte is one of the few the compressor picked out.
func TestPCUnpack(t *testing.T) {
	// Method 2, so only the first two of the four values carry a count.
	packed := []byte{
		2, 0xAA, 0x55, 0x00, 0xFF,
		0xAA, 3, // three of AA
		0x54,    // a value with no count after it
		0x55, 2, // two of 55
		0x00, // one of the four, but beyond the method
	}

	out, ok := pcUnpack(packed, 7)
	require.True(t, ok)
	require.Equal(t, []byte{0xAA, 0xAA, 0xAA, 0x54, 0x55, 0x55, 0x00}, out)
}

// The count takes in the byte already written, so a run of one repeats nothing.
func TestPCUnpackCountsTheFirstByte(t *testing.T) {
	out, ok := pcUnpack([]byte{1, 0x11, 0, 0, 0, 0x11, 1, 0x22}, 2)
	require.True(t, ok)
	require.Equal(t, []byte{0x11, 0x22}, out)
}

func TestPCUnpackRejectsRubbish(t *testing.T) {
	_, ok := pcUnpack([]byte{0, 1, 2, 3, 4, 5}, 4)
	require.False(t, ok, "there is no method zero")

	_, ok = pcUnpack([]byte{5, 1, 2, 3, 4, 5}, 4)
	require.False(t, ok, "nor a fifth")

	_, ok = pcUnpack([]byte{1, 0x11, 0, 0, 0, 0x22}, 100)
	require.False(t, ok, "the stream runs out before the picture is filled")
}

// TestPCPicturesAreByteExact is the check that says the scheme is right rather
// than merely plausible: across the archives, a compressed picture unpacks to
// exactly the size its own header implies, having eaten its whole stream.
func TestPCPicturesAreByteExact(t *testing.T) {
	data := readAsset(t, "../../../work/aventuras/La_Aventura_Original/PC/PART1.CGA")

	pieces, _, ok := pcArchive(data)
	require.True(t, ok)
	require.NotEmpty(t, pieces)

	for i, p := range pieces {
		require.NotNil(t, p.img, "picture %d did not draw", i+1)
	}
}

// TestUnserpentine covers the last thing the PC pictures needed, and the one
// that is invisible without something to compare against: every other row of a
// compressed picture is written backwards.
func TestUnserpentine(t *testing.T) {
	buffer := []byte{
		1, 2, 3, 4, // row 0, left to right
		8, 7, 6, 5, // row 1, backwards
		9, 10, 11, 12, // row 2
	}

	unserpentine(buffer, 4)

	require.Equal(t, []byte{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
	}, buffer)
}

// A row that runs off the end is left alone rather than half turned.
func TestUnserpentineLeavesAPartialRow(t *testing.T) {
	buffer := []byte{1, 2, 3, 4, 8, 7}
	unserpentine(buffer, 4)
	require.Equal(t, []byte{1, 2, 3, 4, 8, 7}, buffer)

	unserpentine(nil, 0) // and asks for nothing when there is no row
}

// A CGA row packs four pixels to a byte, an EGA one holds a single plane.
func TestRowBytes(t *testing.T) {
	require.Equal(t, 64, rowBytes(256, 2))
	require.Equal(t, 32, rowBytes(256, 4))
}
