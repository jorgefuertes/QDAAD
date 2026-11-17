package decompiler

import (
	"image"
	"image/color"
)

// Drawing a CPC listing back into a picture, which is the only way to tell
// whether it was read right.
//
// Nothing here is the Spectrum's model turned sideways. There are no attributes:
// a pixel carries its own colour, two bits of it, and what those two bits mean
// is the palette the drawing's record set. So one plane of pens is the whole of
// the screen, and the inks are kept beside it.

// The canvas. Mode 1 is 320x200 real pixels and the firmware works in user
// coordinates of 0-639 by 0-399 whatever the mode, so every coordinate the
// engine stores is doubled on the way to the firmware and what it stores is
// pixels. The y runs upwards, as the firmware's does.
//
// It does not run up from the bottom of the screen, though, and that is the one
// number worth getting right. The interpreter moves the origin before any
// drawing runs and narrows the graphics window to match:
//
//	ld de,0 / ld hl,0x20 / call 0xbbc9     GRA SET ORIGIN  0, 32
//	ld de,0x20 / ld hl,0x18f / call 0xbbd2 GRA WIN HEIGHT  32 to 399
//
// Thirty-two user units are sixteen pixels, so the engine's y=0 is sixteen
// pixels up from the bottom and the window is the 184 rows above that. The
// fill's own address routine says the same thing in one instruction —
// `ld a,0xb7 / sub d` at 0x228E, which is 183 minus y — and the drawings agree:
// the frame every scene of La Aventura Original opens with runs from y=72 to
// y=183, which is rows 0 to 111, the fourteen character rows its record asks
// for. The bottom sixteen rows are the line the interpreter writes on.
const (
	cpcCanvasWidth   = 320
	cpcCanvasHeight  = 200
	cpcPens          = 4  // mode 1
	cpcInks          = 27 // the colours the hardware has
	cpcCellSize      = 8
	cpcPixelsPerByte = 4

	// cpcDrawRows is the height of the graphics window: every row the engine
	// can reach, and the whole of what a fill walks over.
	cpcDrawRows = 184

	cpcMaxCallDepth = 10 // the engine resets itself past ten levels
	cpcFillLimit    = cpcCanvasWidth * cpcCanvasHeight
	// cpcMaxBranches is how many runs the fill will hold waiting at once
	// (0x2139 loads thirty into A'). Reaching it is not an error: the routine
	// stops branching and leaves the rest of the region unshaded.
	cpcMaxBranches = 30

	// cpcCharsetSize is the character set the patterned fills come out of: 256
	// glyphs of eight rows, which is also what the interpreter loads at 0x9E00.
	cpcPatternSize = 8
	cpcCharsetSize = 256 * cpcPatternSize
)

// cpcBasePalette is what the four pens stand for before any record speaks.
//
// None of it is invented and none of it is quite first-hand either, because the
// BASIC loader that sets the mode is not in the payload — the disk is formatted
// for its own loader and what we read are the binaries it loads.
//
//   - Pen 1 is bright white because the interpreter says so: it calls SCR SET
//     INK with A=1 and BC=0x1A1A before anything else.
//   - Pen 0 is black because the Spectrum edition's own records say PAPER 0 for
//     every drawing but two, and this is the same art ported.
//   - Pens 2 and 3 are the firmware's defaults for mode 1, which is what a
//     MODE 1 leaves behind. Almost every record sets them, so they rarely show.
var cpcBasePalette = [cpcPens]int{0, 26, 20, 6}

// cpcScreen is the plane of pens, the palette they are read through, and the
// state the engine keeps: where the pen is and which one it is.
type cpcScreen struct {
	pens [cpcCanvasWidth * cpcCanvasHeight]byte

	x, y int
	pen  int
	inks [cpcPens]int

	// edges are the columns and rows a fill may reach, which is the window the
	// drawing's own record declares. The engine has no such bound; the machine
	// does not need one, because outside the window is the picture that was
	// there before and a fill stops the moment the colour changes. Here there
	// is nothing outside, so a fill that gets through the frame — and the frame
	// is drawn in pen 0 over a screen that is pen 0 — would run to the foot of
	// the canvas. See the note on cpcScreen.bounded.
	left, right, bottom, top int
}

// cpcInkColour is the colour of one of the machine's twenty-seven inks.
//
// They are not a list to be copied: each component is one of three levels and
// the number counts through them, green slowest and blue fastest. Ink 11 comes
// out (0,128,255), which is the sky blue it is called, and ink 26 (255,255,255).
func cpcInkColour(ink int) color.NRGBA {
	if ink < 0 || ink >= cpcInks {
		return color.NRGBA{A: 0xFF}
	}

	levels := [3]uint8{0x00, 0x80, 0xFF}

	return color.NRGBA{
		R: levels[(ink/3)%3],
		G: levels[ink/9],
		B: levels[ink%3],
		A: 0xFF,
	}
}

// cpcAt is where a pixel lives in the plane. The engine's y runs upwards from
// sixteen pixels above the foot of the screen, and the plane is stored the way a
// screen reads, so y=183 is the top row and y=0 is row 183.
func cpcAt(x, y int) int { return (cpcDrawRows-1-y)*cpcCanvasWidth + x }

func cpcOnCanvas(x, y int) bool {
	return x >= 0 && x < cpcCanvasWidth && y >= 0 && y < cpcDrawRows
}

// plot puts the pen down. The firmware clips to the graphics window and the
// engine never narrows it, so what falls off the screen is simply lost — it
// does not wrap, which is the one place the Spectrum's ROM behaves differently.
func (s *cpcScreen) plot(x, y int) {
	if !cpcOnCanvas(x, y) {
		return
	}

	s.pens[cpcAt(x, y)] = byte(s.pen)
}

func (s *cpcScreen) penAt(x, y int) int {
	if !cpcOnCanvas(x, y) {
		return -1 // off the canvas is a wall, so a fill stops there
	}

	return int(s.pens[cpcAt(x, y)])
}

// bounded is penAt for a fill, which may not reach outside the window the
// drawing declares.
//
// The engine does not clip and does not have to. Seven of the 157 drawings
// outline that window in the pen the background already is — 026 is the whole
// of it, a rectangle, and 027 and 028 draw it, fill inside it and draw it again
// in pen 0 — so on the machine the line is a wall only because what lies beyond
// it is the picture the player was looking at a moment ago. Drawn here, on a
// screen with nothing on it, that same line is the colour the fill is spreading
// over and the fill walks straight out along it.
//
// The record is the drawing's own word for where it lives, the packager already
// cuts to it, and in these drawings it is the rectangle the frame traces to the
// pixel. So it is what the fill is held to.
func (s *cpcScreen) bounded(x, y int) int {
	if x < s.left || x > s.right || y < s.bottom || y > s.top {
		return -1
	}

	return s.penAt(x, y)
}

// line traces from one point to another: a run along the longer axis, one point
// to each step of it, with a step of the shorter axis folded in whenever the
// error runs over.
//
// It runs in pixels, with the ends halved out of the cursor before they get
// here. Stepping it in the firmware's own coordinates and halving each point as
// it is laid down is the other thing it could mean, and it is not what the
// machine draws: tried, it turns every diagonal into a staircase and leaves
// eight of the 157 drawings leaking instead of seven. The firmware takes the
// two ends, works out the pixels they fall on, and steps between those.
func (s *cpcScreen) line(x0, y0, x1, y1 int) {
	ux0, uy0, ux1, uy1 := x0, y0, x1, y1

	s.plot(ux0, uy0)

	dx, dy := abs(ux1-ux0), abs(uy1-uy0)
	sx, sy := step(ux0, ux1), step(uy0, uy1)

	major, minor := dx, dy

	along := dx >= dy
	if !along {
		major, minor = dy, dx
	}

	err := major / 2

	for range major {
		if err += minor; err >= major {
			err -= major

			if along {
				uy0 += sy
			} else {
				ux0 += sx
			}
		}

		if along {
			ux0 += sx
		} else {
			uy0 += sy
		}

		s.plot(ux0, uy0)
	}
}

// cpcCursor and cpcPixel are the two coordinate spaces the engine lives in.
//
// The firmware works in 0-639 by 0-399 whatever the mode, so every coordinate
// the engine hands it is doubled — 0x1F31 and 0x1F33 for an absolute order,
// 0x208F and 0x20A0 for an increment — and mode 1 halves them again on the way
// to the screen. The cursor is kept in the firmware's, which is why it can sit
// on a half pixel; only the fill ever asks for a pixel back, at 0x20D3, and it
// is that single halving at the end rather than one at every step that decides
// which side of an outline a fill starts on.
func cpcCursor(pixel int) int { return pixel * 2 }
func cpcPixel(cursor int) int { return cursor >> 1 }

// cpcScaleBy is the engine's transform: the increment times the factor, shifted
// down three. Scale 0 is natural size and 1 to 7 only ever reduce.
//
// The shift is arithmetic and the Spectrum's is not, which is a pixel and
// matters: 0x20B8 is `sra h / rr l` three times over a signed pair, so a
// negative increment rounds away from zero where the Spectrum's handler rounds
// towards it.
func cpcScaleBy(value, scale int) int {
	if scale == 0 {
		return value
	}

	return (value * scale) >> 3
}

// cpcPattern is the eight rows a fill paints with, read as mode 1 reads them.
//
// A row is one byte and a byte is four pixels, so a pattern tiles every four
// pixels across and every eight rows down — a quarter of the Spectrum's width,
// which is why the same dither looks finer here.
//
// The holes are not transparent and they are not paper either. The handler at
// 0x2030 builds each row as (glyph AND pen) OR (NOT glyph AND ground), and it
// does that on the bytes, which in mode 1 means the two bits of a pixel are
// chosen separately: the glyph's own pixel value says, bit by bit, which of the
// two colours that bit comes from.
type cpcPattern struct {
	rows   []byte
	pen    int
	ground int
}

// pixel is the pen a pattern puts at one place.
func (p cpcPattern) pixel(x, y int) int {
	row := p.rows[y%cpcPatternSize]
	at := x % cpcPixelsPerByte

	// Each bit of the glyph's own pixel chooses, for the bit of the pen in the
	// same place, between the pen and the ground. Where the glyph's pixel is 3
	// that is the pen and where it is 0 the ground; the two in between take one
	// bit from each, which is what makes these dithers look like three colours.
	out := p.ground & 1
	if row&(0x80>>at) != 0 {
		out = p.pen & 1
	}

	if row&(0x08>>at) != 0 {
		out |= p.pen & 2
	} else {
		out |= p.ground & 2
	}

	return out
}

// solidCPCPattern is what a FILL without a pattern paints with: the handler at
// 0x1FDA builds eight rows of the current pen and nothing else.
func solidCPCPattern(pen int) cpcPattern {
	rows := make([]byte, cpcPatternSize)
	for i := range rows {
		rows[i] = 0xFF
	}

	return cpcPattern{rows: rows, pen: pen, ground: pen}
}

// fill lays a pattern over a region, following the engine's routine at 0x20E2.
//
// It is the Spectrum's fill, order for order: a scanline sweep that goes one
// vertical way only, run by run, seeding the next line where it finds a gap,
// and then a second sweep the other way starting one line further. Thirty runs
// may be waiting at once and the thirty-first is not taken, which is why some of
// the originals have fills that peter out.
//
// What is different is what counts as a wall. The Spectrum's stops where a
// pixel is lit; this one stops where the colour is not the colour it started on
// (0x2132 exclusive-ors the screen byte against the seed's own colour), so it
// spreads over a region of one pen rather than over unlit paper.
func (s *cpcScreen) fill(x, y int, pattern cpcPattern) {
	// Everything above the graphics window is dropped before the routine is
	// even called: 0x2022 compares the row against 184 and gives up.
	if x < 0 || x >= cpcCanvasWidth || y < 0 || y >= cpcDrawRows {
		return
	}

	ground := s.bounded(x, y)
	if ground < 0 {
		return
	}

	// How far right each scanline was filled. It is set up once and carried
	// across both passes, which is what stops the second redoing the first.
	var edge [cpcDrawRows]int

	// So is the count of runs left to branch into: 0x2139 loads A' before the
	// first pass and the second starts at 0x212E, past it, so what the first
	// pass did not hand back the second never gets.
	branches := cpcMaxBranches
	painted := 0

	for _, upwards := range [2]bool{false, true} {
		from := y
		if upwards {
			// The second pass starts one line the other way and is skipped
			// altogether when that falls outside the window.
			if from = y + 1; from >= cpcDrawRows {
				break
			}
		}

		// A starting point that is no longer the colour it was ends the
		// routine, which for the second pass means keeping what the first did.
		if s.bounded(x, from) != ground {
			break
		}

		painted = s.fillPass(x, from, ground, upwards, &edge, &branches, painted, pattern)
	}
}

// fillPass is one of the two sweeps, seeding only in the direction given. It
// paints as it walks, which is what the engine does and what keeps a run from
// being walked twice.
func (s *cpcScreen) fillPass(
	x, y, ground int, upwards bool, edge *[cpcDrawRows]int, branches *int,
	painted int, pattern cpcPattern,
) int {
	type run struct{ x, y int }

	seeds := []run{{x: x, y: y}}

	for len(seeds) > 0 && painted < cpcFillLimit {
		here := seeds[len(seeds)-1]
		seeds = seeds[:len(seeds)-1]

		next := here.y - 1
		if upwards {
			next = here.y + 1
		}

		// First run right to the edge of the region, and remember how far it
		// reached.
		column := here.x
		for column < s.right && s.bounded(column+1, here.y) == ground {
			column++
		}

		edge[here.y] = column

		// Then work leftwards from there, seeding the next line on the way.
		seeded, done := false, false

		for ; column >= s.left && s.bounded(column, here.y) == ground; column-- {
			s.pens[cpcAt(column, here.y)] = byte(pattern.pixel(column, here.y))
			painted++

			// The leftmost pixel of a run is painted and then the run ends,
			// before anything is seeded from it: 0x21D4 tests for column zero
			// between the painting and the seeding.
			if column == s.left || done || next < s.bottom || next > s.top {
				continue
			}

			// A line already filled at least this far needs no more seeds, and
			// this run stops looking altogether.
			if e := edge[next]; e != 0 && e >= column {
				seeded, done = true, true

				continue
			}

			switch open := s.bounded(column, next) == ground; {
			case seeded:
				seeded = open
			case open:
				if *branches--; *branches == 0 {
					return painted
				}

				seeds = append(seeds, run{x: column, y: next})
				seeded = true
			}
		}

		// A closed run hands its branch back.
		*branches++
	}

	return painted
}

// trace runs one drawing onto the screen, following its calls.
func (v cpcVectors) trace(s *cpcScreen, number, scale, mirror, depth int, stop *halt) {
	if depth > cpcMaxCallDepth || number < 0 || number >= len(v.drawings) {
		return
	}

	for _, o := range v.drawings[number].orders {
		if stop != nil && stop.reached {
			return
		}

		switch o.name {
		case "PEN":
			s.pen = o.args[0]

		case "PLOT", "MOVE":
			// The absolute orders carry no scale and no mirror. The handler at
			// 0x1F9E doubles both operands and hands them to the firmware, so
			// the pixel is the operand and the cursor is twice it.
			s.x, s.y = cpcCursor(o.args[0]), cpcCursor(o.args[1])

			if o.name == "PLOT" {
				s.plot(o.args[0], o.args[1])
			}

		case "LINE", "STEP", "FILL":
			// The increments are negated before they are doubled and scaled,
			// which is not the same as negating what comes out: the shift at
			// 0x20B8 is arithmetic, so it rounds down and not towards zero.
			ax, ay := o.args[0], o.args[1]

			if mirror&1 != 0 {
				ax = -ax
			}

			if mirror&2 != 0 {
				ay = -ay
			}

			dx, dy := cpcScaleBy(cpcCursor(ax), scale), cpcScaleBy(cpcCursor(ay), scale)

			x1, y1 := s.x+dx, s.y+dy

			switch o.name {
			case "LINE":
				s.line(cpcPixel(s.x), cpcPixel(s.y), cpcPixel(x1), cpcPixel(y1))
			case "FILL":
				s.fill(cpcPixel(x1), cpcPixel(y1), v.patternFor(o, s.pen))
			}

			// A line and a step take the pen with them; a fill does not. The
			// handlers differ in exactly that: 0x20CA reads the cursor to work
			// out where to fill from and never writes it back, so a run of
			// fills all measure from the one place.
			if o.name != "FILL" {
				s.x, s.y = x1, y1
			}

		case "CALL":
			if stop != nil && o.args[0] == stop.at {
				stop.reached = true

				return
			}

			// The handler writes the scale over whatever was there but
			// exclusive-ors the mirror onto it, so one replaces and the other
			// accumulates.
			v.trace(s, o.args[0], o.args[1], mirror^o.args[2], depth+1, stop)

		case "CHAR":
			v.stamp(s, o.args[0], o.args[1], o.args[2])

		case "BOX":
			s.box(o.args[0], o.args[1], o.args[2])
		}
	}
}

// patternFor is what a FILL paints with: the current pen alone, or a glyph of
// the character set in two colours.
func (v cpcVectors) patternFor(o order, pen int) cpcPattern {
	if len(o.args) < 4 {
		return solidCPCPattern(pen)
	}

	rows := v.pattern(o.args[2])
	if rows == nil {
		return solidCPCPattern(pen)
	}

	return cpcPattern{rows: rows, pen: pen, ground: o.args[3]}
}

// stamp writes one character of the set at the graphics cursor, which is where
// GRA WR CHAR puts it: at a pixel, not at a character cell, with its top left
// corner on the cursor.
//
// Nothing in La Aventura Original uses this order — not one of its 157 drawings
// — so what is here follows the firmware's own description and has not been held
// against a picture.
func (v cpcVectors) stamp(s *cpcScreen, glyph, x, y int) {
	rows := v.pattern(glyph)
	if rows == nil {
		return
	}

	for row := range cpcPatternSize {
		for column := range cpcCellSize {
			if rows[row]&(0x80>>column) == 0 {
				continue
			}

			s.plot(x+column, y-row)
		}
	}
}

// box floods a rectangle in the current pen, which is what SCR FLOOD BOX does.
//
// Its operands are not a coordinate: they are the address of the top left corner
// in video memory, a count of scan lines and a width in bytes, because that is
// what the firmware takes. So this is the one order that has to be turned back
// from an address into a place on the screen.
//
// Unused in La Aventura Original too, and unverified for the same reason.
func (s *cpcScreen) box(address, lines, width int) {
	const base = 0xC000

	at := address - base
	if at < 0 || lines <= 0 || width <= 0 {
		return
	}

	// Video memory is eight interleaved bands: the scan line within the
	// character row picks the band, and the character row picks the offset in
	// it.
	band := at / 0x800
	within := at % 0x800
	row := within / (cpcCanvasWidth / cpcPixelsPerByte)
	column := within % (cpcCanvasWidth / cpcPixelsPerByte)

	for line := range lines {
		screenRow := (row+(line+band)/cpcCellSize)*cpcCellSize + (line+band)%cpcCellSize

		for b := range width {
			for p := range cpcPixelsPerByte {
				x := (column+b)*cpcPixelsPerByte + p
				s.plot(x, cpcCanvasHeight-1-screenRow)
			}
		}
	}
}

// blankScreen is the screen a drawing is entered on: the pen at the origin,
// pen 1 chosen, and the palette the record asked for over the base one.
//
// 0x1E74 moves the cursor to 0,0 and sets pen 1 before the drawing runs, and
// 0x1C52 clears the declared window — so the picture starts in pen 0, which is
// what an untouched pixel is here.
func (v cpcVectors) blankScreen(number int) *cpcScreen {
	s := &cpcScreen{inks: cpcBasePalette, pen: 1}

	s.left, s.right, s.bottom, s.top = 0, cpcCanvasWidth-1, 0, cpcDrawRows-1

	// The window the record declares, in pixels. Its rows are counted from the
	// top of the screen and the canvas counts its own from the bottom, which is
	// the one inversion in all of this.
	if top, left, rows, columns, ok := v.window(number); ok && rows > 0 && columns > 0 {
		s.left = left * cpcPixelsPerByte * 2
		s.right = min(s.left+columns*cpcPixelsPerByte*2-1, cpcCanvasWidth-1)
		s.top = max(cpcDrawRows-1-top*cpcPatternSize, 0)
		s.bottom = max(cpcDrawRows-(top+rows)*cpcPatternSize, 0)
	}

	if inks, prepares := v.palette(number); prepares {
		for pen, ink := range inks {
			// An ink past the machine's colours means that pen is left as it
			// stands, which is how a drawing recolours three pens and keeps the
			// background it was given.
			if ink < cpcInks {
				s.inks[pen] = ink
			}
		}
	}

	return s
}

// isBlank is a screen with nothing on it: every pixel still the pen it started
// as, which on this machine is pen 0.
func (s *cpcScreen) isBlank() bool {
	for _, pen := range s.pens {
		if pen != 0 {
			return false
		}
	}

	return true
}

// apply puts a screen into an inherited state.
func (s *cpcScreen) apply(from cpcInherited) {
	s.x, s.y = from.x, from.y
	s.pen = from.pen
	s.inks = from.inks
}

// picture reads the plane of pens through the palette.
func (s *cpcScreen) picture() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, cpcCanvasWidth, cpcCanvasHeight))

	var shades [cpcPens]color.NRGBA
	for pen := range cpcPens {
		shades[pen] = cpcInkColour(s.inks[pen])
	}

	for y := range cpcCanvasHeight {
		for x := range cpcCanvasWidth {
			img.SetNRGBA(x, y, shades[s.pens[y*cpcCanvasWidth+x]&(cpcPens-1)])
		}
	}

	return img
}

// The screen as the machine held it: the whole of a mode 1 display, which is
// sixteen kilobytes and not the twelve and a half the pixels take. The gaps are
// part of it — each of the eight bands holds twenty-five rows of eighty bytes
// and then forty-eight bytes nothing reads — and they are kept, because what
// this is for is to be loaded back where it came from.
const (
	cpcBytesWide  = cpcCanvasWidth / cpcPixelsPerByte // 80
	cpcBandSize   = 0x800
	cpcScreenSize = cpcCellSize * cpcBandSize // 16384
	cpcScreenRows = 25
)

// cpcScreenAddress is where a row of the display lives, as an offset from the
// start of video memory.
func cpcScreenAddress(row int) int {
	return (row%cpcCellSize)*cpcBandSize + (row/cpcCellSize)*cpcBytesWide
}

// scr writes the plane out in the machine's own order.
func (s *cpcScreen) scr() []byte {
	out := make([]byte, cpcScreenSize)

	for row := range cpcCanvasHeight {
		line := cpcScreenAddress(row)

		for x := range cpcCanvasWidth {
			pen := s.pens[row*cpcCanvasWidth+x]
			at := line + x/cpcPixelsPerByte
			in := x % cpcPixelsPerByte

			// In mode 1 the two bits of a pixel are four apart: the low bit of
			// the pen goes in the high nibble and the high bit in the low one.
			if pen&1 != 0 {
				out[at] |= 0x80 >> in
			}

			if pen&2 != 0 {
				out[at] |= 0x08 >> in
			}
		}
	}

	return out
}

// pal writes the inks the four pens stand for, one byte each as the firmware
// numbers them, in a block of sixteen so that the file is the same size whatever
// the mode. It is not OCP Art Studio's .PAL and does not pretend to be: that one
// carries a palette animation and this is four numbers.
func (s *cpcScreen) pal() []byte {
	out := make([]byte, 16)
	for pen := range cpcPens {
		out[pen] = byte(s.inks[pen])
	}

	return out
}
