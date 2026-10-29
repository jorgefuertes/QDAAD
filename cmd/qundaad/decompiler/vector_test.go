package decompiler

import (
	"math/bits"
	"os"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

const spectrumDisk = "../../../work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original.dsk"

func spectrumPayload(t *testing.T) []byte {
	t.Helper()

	if _, err := os.Stat(spectrumDisk); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	volume, err := media.Open(spectrumDisk)
	require.NoError(t, err)

	img, ok := volume.(media.Image)
	require.True(t, ok, "the disk should read as an image")

	return img.Payload()
}

// laOriginal is the dialect La Aventura Original and El Jabato are written in.
var laOriginal = dialect{call: 2, drawShort: false, fillSizes: true}

// The lengths are what desynchronise a strip when they are wrong, and they are
// not the ones of the 1991 module, where a CALL takes three bytes.
func TestOrderLengthsAreThoseOfTheDiskBuild(t *testing.T) {
	for _, tc := range []struct {
		name string
		code byte
		want int
	}{
		{"PLOT", 0x00, 3},
		{"LINE", 0x01, 3},
		{"LINE with the sign bits set", 0xC1, 3},
		{"LINE with bit 5, which is not a short form in this dialect", 0x21, 3},
		{"MOVE", 0x19, 3},
		{"FILL", 0x02, 3},
		{"FILL with a UDG for a pattern", 0x22, 4},
		{"FILL of attributes", attrFillOrder, 5},
		{"CALL", 0x03, 2},
		{"UDG", 0x04, 4},
		{"PAPER", 0x2D, 1},
		{"INK", 0x46, 1},
		{"RET", 0x07, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, orderLength(tc.code, laOriginal))
		})
	}
}

// Cozumel is later and its build does have the short form, which is why the
// dialect is settled by decoding rather than assumed.
func TestTheShortDrawIsTwoBytesWhereTheDialectHasIt(t *testing.T) {
	short := dialect{call: 2, drawShort: true, fillSizes: true}

	require.Equal(t, 2, orderLength(0x21, short))
	require.Equal(t, 3, orderLength(0x01, short), "without bit 5 it is still three")

	text, _, ok := orderText([]byte{0x21, 0x74}, 0, 64, short)
	require.True(t, ok)
	require.Equal(t, "LINE 7,4", text, "the high nibble is x and the low one y")
}

func TestOrdersAreWrittenAsTheyLie(t *testing.T) {
	for _, tc := range []struct {
		name string
		code []byte
		want string
		note string
	}{
		{name: "PLOT is absolute", code: []byte{0x00, 0x37, 0x46}, want: "PLOT 55,70"},
		{name: "a line is relative", code: []byte{0x81, 0x04, 0x06}, want: "LINE 4,-6"},
		{name: "both sign bits", code: []byte{0xC1, 0x04, 0x06}, want: "LINE -4,-6"},
		{name: "bits 3 and 4 make it a MOVE", code: []byte{0x19, 0x13, 0x00}, want: "MOVE 19,0"},
		{name: "the trace mode is noted", code: []byte{0x18, 0x04, 0x06}, want: "PLOT 4,6", note: "over inverse"},
		{name: "a call names its drawing", code: []byte{0x03, 0x1E}, want: "CALL 30,0,0"},
		{name: "a call can scale", code: []byte{0x23, 0x0C}, want: "CALL 12,4,0"},
		{name: "a call can mirror", code: []byte{0x43, 0x0C}, want: "CALL 12,0,1"},
		{name: "a fill is relative too", code: []byte{0x02, 0x10, 0x20}, want: "FILL 16,32"},
		{name: "bit 5 takes the pattern from a UDG", code: []byte{0x22, 0x10, 0x20, 0x84}, want: "FILL 16,32,132"},
		{name: "0x12 paints attributes", code: []byte{attrFillOrder, 0x01, 0x02, 0x03, 0x04}, want: "ATTR 1,2,3,4"},
		{name: "a UDG is stamped on a cell", code: []byte{0x04, 0xC4, 0x0A, 0x07}, want: "UDG 196,10,7"},
		{name: "PAPER", code: []byte{0x2D}, want: "PAPER 5"},
		{name: "the top bit makes it BRIGHT", code: []byte{0xAD}, want: "BRIGHT 5"},
		{name: "INK", code: []byte{0x46}, want: "INK 8"},
		{name: "the top bit makes it FLASH", code: []byte{0xC6}, want: "FLASH 8"},
		{name: "RET", code: []byte{0x07}, want: "RET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text, note, ok := orderText(tc.code, 0, 64, laOriginal)
			require.True(t, ok)
			require.Equal(t, tc.want, text)
			require.Equal(t, tc.note, note)
		})
	}
}

// These are what tell a drawing from a stretch of bytes that happens to decode.
func TestAnImpossibleCoordinateIsNotADrawing(t *testing.T) {
	t.Run("the canvas is 176 tall, so no PLOT names a row above it", func(t *testing.T) {
		_, _, ok := orderText([]byte{0x00, 0x37, canvasHeight}, 0, 64, laOriginal)
		require.False(t, ok)
	})

	t.Run("a call cannot name a drawing the table does not hold", func(t *testing.T) {
		_, _, ok := orderText([]byte{0x03, 0x40}, 0, 64, laOriginal)
		require.False(t, ok)
	})

	t.Run("a UDG cannot be stamped off the screen", func(t *testing.T) {
		_, _, ok := orderText([]byte{0x04, 0xC4, charColumns, 0x07}, 0, 64, laOriginal)
		require.False(t, ok)
	})
}

// The candidate that fooled an earlier reading of this disk was a run of filler
// bytes: a byte followed by 0xFF reads as a pointer into the top of memory, and
// there are thousands of those. Nothing but decoding the drawings tells them
// apart, so this is the regression that matters.
func TestFillerIsNotADrawingTable(t *testing.T) {
	t.Run("filler holds no table at any offset", func(t *testing.T) {
		require.Empty(t, findVectors(filled(0xFF, 0x10000)))
	})

	t.Run("zeroes hold none either", func(t *testing.T) {
		require.Empty(t, findVectors(make([]byte, 0x10000)))
	})

	t.Run("a table that ends where it should but holds rubbish is rejected", func(t *testing.T) {
		// The shape alone is not enough: a block whose tail says all the right
		// things still has to decode, and filler does not.
		const size = 0x800

		data := filled(0xFF, size)
		end := size
		put := func(addr, value int) {
			at := addr - (topOfMemory - end)
			data[at] = byte(value)
			data[at+1] = byte(value >> 8)
		}

		table := 0xFF00
		put(drawingTableAt, table)
		put(propertiesAt, table+2*4) // four entries: two drawings, a bound, a sentinel
		put(udgTableAt, 0xF7E4)
		put(table, 0xFE00)
		put(table+2, 0xFE80)
		put(table+4, table)
		put(table+6, 0xFFFF)

		require.Empty(t, findVectors(data), "the pointers are shaped right but the drawings are filler")
	})
}

func filled(with byte, n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = with
	}

	return data
}

func TestTheDrawingsOfLaAventuraOriginalDecodeWhole(t *testing.T) {
	sets := findVectors(spectrumPayload(t))
	require.Len(t, sets, 2, "one set per part, each with its own copy of the engine")

	for i, want := range []int{49, 108} {
		v := sets[i]

		t.Run("every drawing runs to its RET", func(t *testing.T) {
			require.Len(t, v.drawings, want)

			for _, d := range v.drawings {
				require.NotEmpty(t, d.orders, "drawing %d", d.number)

				last := d.orders[len(d.orders)-1]
				require.Equal(t, opRet, int(last.code&7), "drawing %d should end in a RET", d.number)

				// The orders have to account for every byte between one drawing
				// and the next, with none left over and none borrowed.
				size := 0
				for _, o := range d.orders {
					size += o.size
				}

				require.Equal(t, d.size, size, "drawing %d should fill its space exactly", d.number)
			}
		})
	}

	t.Run("the first part holds the 49 drawings of its 49 locations", func(t *testing.T) {
		require.Len(t, sets[0].drawings, 49)
	})
}

// Cozumel is the case that keeps the reader honest twice over: it is written in
// a later dialect of the engine, and it is the only one of these that shipped
// its parts as files, so its drawings have to be matched to the database found
// in the same file rather than to whichever came first.
func TestCozumelIsReadInItsOwnDialect(t *testing.T) {
	const disk = "../../../work/aventuras/Cozumel/ZXSpectrum/Cozumel.dsk"

	if _, err := os.Stat(disk); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	databases, err := databasesIn(disk)
	require.NoError(t, err)
	require.Len(t, databases, 2)

	for i, want := range []int{30, 33} {
		require.NotNil(t, databases[i].vectors, "part %d should have its own drawings", i+1)
		require.Len(t, databases[i].vectors.drawings, want)
		require.True(t, databases[i].vectors.dialect.drawShort,
			"this build has the two-byte DRAW that La Aventura Original does not")
	}
}

func TestEachPartKeepsTheDrawingsFoundBesideIt(t *testing.T) {
	databases, err := databasesIn(spectrumDiskOrSkip(t))
	require.NoError(t, err)
	require.Len(t, databases, 2)

	for i, want := range []int{49, 108} {
		require.NotNil(t, databases[i].vectors)
		require.Len(t, databases[i].vectors.drawings, want)

		// The drawings of a part lie after its database and before the next.
		require.Greater(t, databases[i].vectors.dataAt(), databases[i].reader.Offset())
	}
}

func spectrumDiskOrSkip(t *testing.T) string {
	t.Helper()

	if _, err := os.Stat(spectrumDisk); err != nil {
		t.Skipf("the image is not in the tree: %v", err)
	}

	return spectrumDisk
}

// Los Templos Sagrados is why the block is found by where it ends rather than
// by the engine beside it: its loader puts the interpreter and the drawings in
// separate headerless blocks, so there is no dispatcher in the same stretch of
// bytes to take a load address from. Chichén Itzá is the same arrangement.
func TestTheDrawingsAreFoundWithNoEngineBesideThem(t *testing.T) {
	for _, tc := range []struct {
		tape string
		want int
	}{
		{"Los_templos_sagrados/ZXSpectrum/Los Templos Sagrados - Side 1.tzx", 60},
		{"Los_templos_sagrados/ZXSpectrum/Los Templos Sagrados - Side 2.tzx", 58},
		{"Chichén_Itzá/ZXSpectrum/Chichen Itza - Side 1.tzx", 21},
		{"Chichén_Itzá/ZXSpectrum/Chichen Itza - Side 2.tzx", 38},
	} {
		t.Run(tc.tape, func(t *testing.T) {
			path := "../../../work/aventuras/" + tc.tape

			if _, err := os.Stat(path); err != nil {
				t.Skipf("the tape is not in the tree: %v", err)
			}

			volume, err := media.Open(path)
			require.NoError(t, err)

			img, ok := volume.(media.Image)
			require.True(t, ok)

			sets := findVectors(img.Payload())
			require.Len(t, sets, 1)
			require.Len(t, sets[0].drawings, tc.want)
		})
	}
}

// A piece stamped into a scene cannot be read on its own: its coordinates and
// its colour are whatever the caller had reached. INHERIT is what puts that in
// the file, so drawing 2 — the AD logo, which only drawing 27 ever calls — comes
// out where it belongs rather than from the corner of the canvas.
func TestACalledDrawingInheritsTheStateOfItsCaller(t *testing.T) {
	sets := findVectors(spectrumPayload(t))
	require.NotEmpty(t, sets)

	v := sets[0]
	callers := v.callers()

	require.Equal(t, []int{27}, callers[2], "only drawing 27 stamps the logo")

	from, ok := v.inheritedFor(2, callers).get()
	require.True(t, ok)
	require.Equal(t, 27, from.parent)

	// Its caller does PLOT 136,135 immediately before the call, so that is
	// where the pen stands.
	require.Equal(t, 136, from.x)
	require.Equal(t, 135, from.y)

	body := plotFile(v, v.drawings[2], callers[2], v.inheritedFor(2, callers))
	require.Contains(t, body, "CALLED FROM    : 027")
	require.Contains(t, body, "INHERIT 027,")

	t.Run("a drawing nobody calls has no INHERIT", func(t *testing.T) {
		scene := plotFile(v, v.drawings[3], callers[3], v.inheritedFor(3, callers))
		require.NotContains(t, scene, "INHERIT")
		require.NotContains(t, scene, "CALLED FROM")
	})
}

// Entries holding a lone RET are real entries but say nothing, so they get no
// file and leave a gap in the numbering.
func TestEmptyDrawingsGetNoListing(t *testing.T) {
	sets := findVectors(spectrumPayload(t))
	require.NotEmpty(t, sets)

	empty := 0

	for _, d := range sets[0].drawings {
		if d.isEmpty() {
			empty++
		}
	}

	require.Equal(t, 8, empty, "eight of the 49 hold nothing but a RET")

	written, err := writeVectors(sets[0], t.TempDir(), Options{})
	require.NoError(t, err)
	require.Equal(t, len(sets[0].drawings)-empty, written)
}

// The second pointer in the tail was taken for the end of the drawing table and
// nothing more. It is also the base of a table of five-byte records, one per
// drawing, and the interpreter reads the drawing's own record before running it
// (0x7BBC) to set the ink and paper it starts under. That the records run
// exactly from that pointer to the next is what confirms the reading.
func TestADrawingIsEnteredWithTheAttributeOfItsRecord(t *testing.T) {
	sets := findVectors(spectrumPayload(t))
	require.Len(t, sets, 2)

	for i, v := range sets {
		t.Run("the records fill the space between the pointers", func(t *testing.T) {
			require.Len(t, v.properties, propertySize*len(v.drawings),
				"part %d: five bytes a drawing, exactly", i+1)
		})
	}

	// White on black and white on blue are the only two the first part uses.
	seen := map[[2]byte]int{}

	for n := range sets[0].drawings {
		ink, paper, ok := sets[0].startsWith(n)
		require.True(t, ok)

		seen[[2]byte{ink, paper}]++
	}

	require.Equal(t, map[[2]byte]int{{7, 0}: 40, {7, 1}: 9}, seen)
}

// Bits 3 and 4 of a PLOT or a DRAW pick the pen, which 0x7CA8 folds into P-FLAG
// as (mode+2) and 5. The captures barely exercise this — nearly every use is a
// lone PLOT over blank screen, where an exclusive-or looks the same as laying
// the pixel down — so it is worth pinning here.
func TestThePenModeDecidesWhatAPixelDoes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mode   int
		before bool
		want   bool
	}{
		{"normal lays the pixel down", penNormal, false, true},
		{"normal over an existing pixel leaves it", penNormal, true, true},
		{"over flips a blank pixel on", penOver, false, true},
		{"over flips a set pixel off", penOver, true, false},
		{"inverse wipes a set pixel", penInverse, true, false},
		{"inverse leaves a blank one blank", penInverse, false, false},
		{"over and inverse together exclusive-or", penOverInverse, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &screen{ink: 7}
			if tc.before {
				s.mark(10, 20, penNormal)
			}

			s.mark(10, 20, tc.mode)

			require.Equal(t, tc.want, s.lit(10, 20))
		})
	}
}

// The drawing as the machine held it, which is the only shape in which any of
// these can reach a build we make: version 3 has no vector engine.
//
// Video memory is not laid out the way the screen reads, and getting that
// wrong does not fail loudly — it draws something, sliced into thirds and
// interleaved. So the addresses are pinned one at a time: the start of the
// screen, the next line down, the next character row, and the next third.
func TestTheScreenComesOutInTheMachinesOwnOrder(t *testing.T) {
	t.Run("it is a screen's worth of bytes", func(t *testing.T) {
		require.Len(t, (&screen{}).scr(), 6912)
	})

	for _, tc := range []struct {
		name string
		x, y int
		at   int
		bit  byte
	}{
		{"the top left corner is the first byte", 0, 0, 0, 0x80},
		{"and the pixel beside it is the same byte", 1, 0, 0, 0x40},
		{"the eighth pixel across is the next byte", 8, 0, 1, 0x80},
		{"one line down is 256 bytes on", 0, 1, 256, 0x80},
		{"one character row down is 32 bytes on", 0, 8, 32, 0x80},
		{"and the next third of the screen is 2048", 0, 64, 2048, 0x80},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &screen{}
			s.pixels[tc.y*canvasWidth+tc.x] = true

			out := s.scr()
			require.Equal(t, tc.bit, out[tc.at], "the pixel is not where the hardware reads it")

			var lit int

			for _, b := range out[:scrPixels] {
				lit += bits.OnesCount8(b)
			}

			require.Equal(t, 1, lit, "one pixel in, one pixel out")
		})
	}

	t.Run("the attributes", func(t *testing.T) {
		s := &screen{background: attribute{ink: 7, paper: 0}}
		s.cells[0] = attribute{ink: 6, paper: 1, bright: true, set: true}

		out := s.scr()
		require.EqualValues(t, 6|1<<3|0x40, out[scrPixels], "ink, paper and bright of the first cell")
		require.EqualValues(t, 7, out[scrPixels+1], "a cell nothing touched shows the background")
	})

	// The canvas is 176 lines and a screen is 192. The two rows of difference
	// are not left as whatever the struct happened to hold.
	t.Run("the rows below the canvas carry the background", func(t *testing.T) {
		s := &screen{background: attribute{ink: 5, paper: 2}}

		out := s.scr()
		for row := cellsDown; row < scrRows; row++ {
			require.EqualValues(t, 5|2<<3, out[scrPixels+row*cellsAcross],
				"row %d is not the background", row)
		}
	})
}

func TestAPlotFileSaysWhereItCameFrom(t *testing.T) {
	sets := findVectors(spectrumPayload(t))
	require.NotEmpty(t, sets)

	// Drawing 3 reuses two others, so it shows a CALL as well as the geometry.
	callers := sets[0].callers()
	body := plotFile(sets[0], sets[0].drawings[3], callers[3], sets[0].inheritedFor(3, callers))

	require.Contains(t, body, "vector drawing 3 of 49")
	require.Contains(t, body, "Read from")
	require.Contains(t, body, "Orders         : ")

	orders := []string{}

	for line := range strings.SplitSeq(body, "\n") {
		if line != "" && !strings.HasPrefix(line, ";") {
			orders = append(orders, line)
		}
	}

	require.Equal(t, "IMAGE 003", orders[0], "the listing names itself first")
	require.Equal(t, "CALL 30,0,0", orders[1])
	require.Equal(t, "PLOT 55,70", orders[2])
	require.Equal(t, "RET", orders[len(orders)-1])
}
