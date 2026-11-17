package decompiler

import (
	"os"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

const cpcOriginal = "../../../work/aventuras/La_Aventura_Original/AmstradCPC/ORIGINAL.DSK"

func cpcPayload(t *testing.T) []byte {
	t.Helper()

	if _, err := os.Stat(cpcOriginal); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	volume, err := media.Open(cpcOriginal)
	require.NoError(t, err)

	image, ok := volume.(media.Image)
	require.True(t, ok, "the disk is formatted for its own loader and has only sectors")

	return image.Payload()
}

// The search has to find the two blocks and nothing else. Every offset of the
// payload is tried as the end of one, which sounds indiscriminate until you see
// that only two of 132 608 of them get as far as decoding: a block has to carry
// a table that is ordered, self-terminating and bounded by itself, a record per
// drawing that agrees with it to the byte, and then every drawing has to run to
// its return exactly where the next one starts.
func TestFindsTheTwoCPCBlocks(t *testing.T) {
	found := findCPCVectors(cpcPayload(t))

	require.Len(t, found, 2)
	require.Len(t, found[0].drawings, 49)
	require.Len(t, found[1].drawings, 108)

	// The same counts as the Spectrum edition, which is what you would expect
	// of the same art ported, and the first thing that said the anchor at
	// 0x9E00 was right.
	for _, v := range found {
		require.Equal(t, cpcPropertySize*len(v.drawings), len(v.properties))
		require.Len(t, v.patterns, cpcCharsetSize)
	}
}

// Every order of every drawing has to decode, and the strip has to end where
// the table says it does. A length wrong by one byte desynchronises the strip
// and shows up here, so this is what holds the order lengths of the eight
// handlers.
func TestEveryCPCDrawingChains(t *testing.T) {
	for _, v := range findCPCVectors(cpcPayload(t)) {
		for _, d := range v.drawings {
			size := 0
			for _, o := range d.orders {
				size += o.size
			}

			require.Equal(t, d.size, size, "drawing %d", d.number)

			last := d.orders[len(d.orders)-1]
			require.Equal(t, "RET", last.name, "drawing %d", d.number)
		}
	}
}

// The frame the scenes of the first part open with, which is what pinned the
// origin: a rectangle 256 by 111 whose top left corner is the top left corner
// of the fourteen character rows the records declare.
func TestTheCPCFrameIsThePictureArea(t *testing.T) {
	found := findCPCVectors(cpcPayload(t))
	require.Len(t, found, 2)

	frame := found[0].drawings[43]

	var lines []string
	for _, o := range frame.orders {
		lines = append(lines, o.text())
	}

	require.Equal(t, []string{
		"PLOT 32,72",
		"LINE 0,111",
		"LINE 255,0",
		"LINE 0,-111",
		"LINE -255,0",
		"RET",
	}, lines)

	// Which comes out as rows 0 to 111 of the screen, the window its record
	// asks for.
	_, _, rows, columns, ok := found[0].window(3)
	require.True(t, ok)
	require.Equal(t, 14, rows)
	require.Equal(t, 32, columns)
}

// A drawing traced onto a screen, measured where it matters: the pictures live
// in the rows their records declare, which is what says the canvas is anchored
// right.
//
// None of the 157 runs a fill past the window its record declares, which is
// the whole of what the packager keeps.
//
// Seven of them used to. The machine never draws onto an empty screen, and
// seven drawings outline their own window in the pen the background already is
// — 026 is that rectangle and nothing else — so on the machine the line is a
// wall only because beyond it lies the picture the player was looking at a
// moment ago. Drawn onto nothing, the fill walked out along it. Holding a fill
// to the declared window is what puts that right; see cpcScreen.bounded.
func TestTheCPCPicturesLandInTheWindow(t *testing.T) {
	found := findCPCVectors(cpcPayload(t))
	require.Len(t, found, 2)

	var leaked []int

	for _, v := range found {
		callers := v.callers()

		for _, d := range v.drawings {
			if d.isEmpty() {
				continue
			}

			if lowest(v.render(d.number, callers)) >= cpcDrawRows-1 {
				leaked = append(leaked, d.number)
			}
		}
	}

	require.Empty(t, leaked, "no drawing fills past its own window")
}

// lowest is the engine's y of the lowest row the drawing touched.
func lowest(s *cpcScreen) int {
	for row := cpcDrawRows - 1; row >= 0; row-- {
		for x := range cpcCanvasWidth {
			if s.pens[row*cpcCanvasWidth+x] != 0 {
				return row
			}
		}
	}

	return 0
}

// The screen as the machine held it, which is what a packager is handed: the
// whole of a mode 1 display, and the pens written where the hardware reads
// them.
func TestTheCPCScreenIsAModeOneDump(t *testing.T) {
	found := findCPCVectors(cpcPayload(t))
	require.Len(t, found, 2)

	v := found[0]
	screen := v.render(3, v.callers())

	require.Len(t, screen.scr(), cpcScreenSize)

	// Pen 3 is both bits of a pixel, so its byte is solid; the palette says
	// what that pen is worth, and drawing 3 asks for sky blue, pastel yellow
	// and green over a background it leaves alone.
	require.Equal(t, []byte{0, 11, 25, 9}, screen.pal()[:cpcPens])

	// What the record does not set keeps the base palette, and pen 0 is the
	// black the Spectrum edition's own records use for paper.
	require.Equal(t, cpcBasePalette[0], screen.inks[0])
}

const cpcTape = "../../../work/aventuras/La_Aventura_Original/AmstradCPC/" +
	"La aventura original (Cara A).cdt"

// The loading screen, which is the one picture of these editions that nothing
// declares: no table holds it, no filesystem names it, and what finds it is
// that a dump of a CPC display has 48 bytes nobody reads at the end of each of
// its eight bands.
func TestFindsTheCPCLoadingScreen(t *testing.T) {
	found := findCPCLoadingScreens(cpcPayload(t))

	require.Len(t, found, 2, "one for each part of the disk")

	for _, s := range found {
		require.Len(t, s.screen, cpcLoadingSize)
		require.Equal(t, 0, s.mode(), "sixteen pens is a mode 0 picture")
		require.Len(t, s.inks, 16)
		require.Contains(t, s.found, "gate array")
	}
}

// And the same picture off the tape, which is the check that says the reading
// is right rather than merely consistent: two media with nothing in common —
// a custom turbo tape against a disk formatted for its own loader — and two
// loaders that keep their palettes in different places and in different
// encodings, giving the same 16384 bytes and the same seventeen colours.
func TestTheCPCLoadingScreenIsTheSameOnTapeAndDisk(t *testing.T) {
	if _, err := os.Stat(cpcTape); err != nil {
		t.Skipf("the tape is not in the tree: %v", err)
	}

	volume, err := media.Open(cpcTape)
	require.NoError(t, err)

	tape, ok := volume.(media.Image)
	require.True(t, ok)

	fromTape := findCPCLoadingScreens(tape.Payload())
	require.Len(t, fromTape, 1)

	fromDisk := findCPCLoadingScreens(cpcPayload(t))
	require.Len(t, fromDisk, 2)

	require.Equal(t, fromDisk[0].screen, fromTape[0].screen)
	require.Equal(t, fromDisk[0].inks, fromTape[0].inks)

	// The tape's loader keeps a table of sixteen firmware colours and the
	// disk's seventeen of the gate array's, so the two are found by different
	// rules and still agree.
	require.Contains(t, fromTape[0].found, "firmware colours")
	require.Equal(t, -1, fromTape[0].border, "a tape table carries no border at all")
}
