package pack

import (
	"encoding/binary"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/stretchr/testify/require"
)

// A PC release is a directory of loose files and the whole of the packaging is
// giving them the names the interpreter opens. Those names are not a convention
// anybody is free with: they are read out of PCDAAD's own source, which is in
// the checkout, so these tests are about the names and the shapes and not about
// guesses.
//
// What the pictures are held against is script/compare-scrmaker.sh, which
// converts the same PCX with the reference tool and compares the bytes.

func TestThePCReleaseIsTheFilesTheInterpreterOpens(t *testing.T) {
	release, err := PCRelease(someDatabase(), "vga256", Options{
		Language: language.LanguageSpanish,
		Name:     "prueba",
		Images:   map[int][]byte{3: pcxFixture(t, 7), 44: pcxFixture(t, 9)},
		Loading:  pcxFixture(t, 11),
	})
	require.NoError(t, err)

	require.Equal(t,
		[]string{
			"003.MSD", "044.MSD", "DAAD.DDB", "DAAD.FNT", "DAAD.MSD",
			"PCDAAD.EXE", "PRUEBA.BAT",
		},
		release.sortedNames())
}

// The interpreter is the thing to run and the batch file is what says so, named
// after the adventure because that is what a person at a DOS prompt reads.
// Eight characters and nothing a DOS filesystem will not take.
func TestTheBatchFileIsNamedAfterTheAdventure(t *testing.T) {
	for name, want := range map[string]string{
		"prueba":            "PRUEBA.BAT",
		"La Aventura":       "LAAVENTU.BAT",
		"AO1":               "AO1.BAT",
		"":                  "GAME.BAT",
		"¡!¿?":              "GAME.BAT",
		"part-1_of_the_two": "PART1OFT.BAT",
	} {
		require.Equal(t, want, pcStarter(name), "from %q", name)
	}
}

// The font is two formats and the name is what picks between them: the
// interpreter tries DAAD.FNT with SINTAC's layout and then DAAD.CHR with the
// plain one, giving every glyph a width of six. Handing a 2048-byte charset
// over as a .FNT would have it read from the wrong end.
func TestTheFontGoesInUnderTheNameThatSaysHowToReadIt(t *testing.T) {
	// Nothing declared: the built-in proportional one.
	release, err := PCRelease(someDatabase(), "", Options{Language: language.LanguageSpanish})
	require.NoError(t, err)
	require.Contains(t, release.sortedNames(), "DAAD.FNT")

	// An eight-bit character set, which is what every other target here takes.
	release, err = PCRelease(someDatabase(), "", Options{
		Language: language.LanguageSpanish,
		Charset:  make([]byte, charsetSize),
	})
	require.NoError(t, err)
	require.Contains(t, release.sortedNames(), "DAAD.CHR")
	require.NotContains(t, release.sortedNames(), "DAAD.FNT")

	// And something that is neither.
	_, err = PCRelease(someDatabase(), "", Options{
		Language: language.LanguageSpanish,
		Charset:  make([]byte, 500),
	})
	require.ErrorContains(t, err, "SINTAC")
}

// VGA256 is the one PC DAAD READY builds. The others are a different screen and
// so a different conversion of every picture, which is not a flag.
func TestOnlyTheVGA256PCIsWritten(t *testing.T) {
	for _, subtarget := range []string{"text", "cga", "ega", "vga"} {
		_, err := PCRelease(someDatabase(), subtarget,
			Options{Language: language.LanguageSpanish})
		require.ErrorContains(t, err, "vga256")
	}
}

// The header is sixteen bytes and every field in it is read by the interpreter
// at a fixed offset (pic.pas:365-382). This is that, in the order it reads them.
func TestTheMSDHeaderSaysWhereThePictureGoes(t *testing.T) {
	picture, err := pcPicture(pcxFixture(t, 3), imageLines)
	require.NoError(t, err)

	require.Equal(t, []byte{0xDA, 0xAD}, picture[:2], "the signature")
	require.Equal(t, pcCanvasWidth, int(binary.LittleEndian.Uint16(picture[2:])))
	require.Equal(t, imageLines, int(binary.LittleEndian.Uint16(picture[4:])))
	require.Equal(t, []byte{0, 255}, picture[6:8], "the whole palette may be faded")
	require.Equal(t, []byte{0, 0, 0, 0}, picture[8:12], "pinned to the top left")
	require.Equal(t, []byte{0, 0, 0, 0}, picture[12:16], "no compression, and three spare")

	require.Len(t, picture, pcHeaderSize+pcPaletteSize+pcCanvasWidth*imageLines)
}

// The palette is the whole of the difference between a picture that draws right
// and one that draws in the text colours: DAAD keeps the bottom sixteen entries
// for its own inks, so the drawing's palette moves up to make room and every
// pixel moves with it.
func TestThePaletteLeavesDAADItsOwnSixteenColours(t *testing.T) {
	picture, err := pcPicture(pcxFixture(t, 3), imageLines)
	require.NoError(t, err)

	palette := picture[pcHeaderSize : pcHeaderSize+pcPaletteSize]

	require.Equal(t, pcStandardPalette[:], palette[:pcReservedColours*3],
		"the Spectrum's sixteen, which is what the text is drawn in")

	// Bright blue is 0xF0 and not 0xFF, which is the one entry a reconstructed
	// palette gets wrong. See pcStandardPalette.
	require.Equal(t, []byte{0x00, 0x00, 0xF0}, palette[9*3:9*3+3])

	// And the drawing's own colours start right after them.
	require.Equal(t, []byte{0, 0, 0, 5, 11, 23}, palette[pcReservedColours*3:pcReservedColours*3+6])

	// Every pixel moved up by the same sixteen.
	pixels := picture[pcHeaderSize+pcPaletteSize:]
	require.Equal(t, byte(3+pcReservedColours), pixels[0])
}

// A drawing using the top sixteen colours has nowhere to move to. It is refused
// rather than wrapped, because a wrap would paint those pixels in DAAD's inks
// and look very nearly right.
func TestAPictureWithNoRoomToShiftIsRefused(t *testing.T) {
	_, err := pcPicture(pcxFixture(t, 240), imageLines)

	require.ErrorContains(t, err, "colour 240")
	require.ErrorContains(t, err, "move up")
}

// A drawing need not be as wide as the screen, and these are not: La Aventura
// Original's PC illustrations are 256 across a screen of 320. The header is
// what places them, so a narrow one is centred rather than left against the
// edge with sixty-four columns of black beside it.
func TestANarrowPictureIsCentred(t *testing.T) {
	picture, err := pcPicture(pcx(t, 256, 112, 3), 0)
	require.NoError(t, err)

	require.Equal(t, 256, int(binary.LittleEndian.Uint16(picture[2:])))
	require.Equal(t, 112, int(binary.LittleEndian.Uint16(picture[4:])))
	require.Equal(t, (320-256)/2, int(binary.LittleEndian.Uint16(picture[8:])), "fixed X")
	require.Equal(t, 0, int(binary.LittleEndian.Uint16(picture[10:])), "fixed Y")

	// And one as wide as the screen sits at zero, which is what the reference
	// writes and what compare-scrmaker holds this against.
	picture, err = pcPicture(pcxFixture(t, 3), imageLines)
	require.NoError(t, err)
	require.Equal(t, 0, int(binary.LittleEndian.Uint16(picture[8:])))
}

// Wider than the screen, though, has nowhere to go.
func TestAPictureWiderThanTheScreenIsRefused(t *testing.T) {
	_, err := pcPicture(pcx(t, 640, 100, 3), 0)

	require.ErrorContains(t, err, "320 pixels wide and this picture is 640")
}

// A drawing taller than the screen has nowhere to go.
func TestAPictureTallerThanTheScreenIsRefused(t *testing.T) {
	_, err := pcPicture(pcx(t, pcCanvasWidth, 240, 3), 0)

	require.ErrorContains(t, err, "200 lines and this picture is 240")
}

// With no height given, a picture is as tall as it is — which is the answer
// DAAD's own IMGLINES gets wrong for an adventure whose drawings are not all
// one size. See pcPicture.
func TestAPictureIsAsTallAsItIs(t *testing.T) {
	for _, height := range []int{112, 176, pcCanvasHeight} {
		picture, err := pcPicture(pcx(t, pcCanvasWidth, height, 3), 0)
		require.NoError(t, err)

		require.Equal(t, height, int(binary.LittleEndian.Uint16(picture[4:])))
		require.Len(t, picture, pcHeaderSize+pcPaletteSize+pcCanvasWidth*height)
	}
}

// And a height the picture has not got is an error rather than a short read.
func TestAHeightThePictureHasNotGot(t *testing.T) {
	_, err := pcPicture(pcx(t, pcCanvasWidth, 112, 3), 176)

	require.ErrorContains(t, err, "176 lines of a picture that has 112")
}

// pcxFixture is a whole VGA screen of one colour, which is the shortest valid
// picture and the one every test here starts from.
func pcxFixture(t *testing.T, colour byte) []byte {
	t.Helper()

	return pcx(t, pcCanvasWidth, pcCanvasHeight, colour)
}

// pcx writes an 8-bit PCX by hand.
//
// Written rather than taken from a fixture file so that the tests say what is
// in the picture they are about. The encoding is the format's own: a byte with
// its top two bits set is a count and the value follows, which is why a run of
// one still has to be encoded when the value itself has those bits set.
func pcx(t *testing.T, width, height int, colour byte) []byte {
	t.Helper()

	out := make([]byte, 128)

	out[0], out[1], out[2], out[3] = 0x0A, 5, 1, 8 // PCX, version 5, encoded, 8 bits

	binary.LittleEndian.PutUint16(out[4:], 0)
	binary.LittleEndian.PutUint16(out[6:], 0)
	binary.LittleEndian.PutUint16(out[8:], uint16(width-1))
	binary.LittleEndian.PutUint16(out[10:], uint16(height-1))

	out[65] = 1 // one plane

	binary.LittleEndian.PutUint16(out[66:], uint16(width))

	for range height {
		left := width

		for left > 0 {
			run := min(left, 63)
			out = append(out, 0xC0|byte(run), colour)
			left -= run
		}
	}

	out = append(out, 12) // the marker that says a 256-colour palette follows

	for i := range 256 {
		out = append(out, byte(i*5), byte(i*11), byte(i*23))
	}

	return out
}
