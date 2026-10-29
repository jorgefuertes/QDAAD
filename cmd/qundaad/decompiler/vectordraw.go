package decompiler

import (
	"image"
	"image/color"
)

// Drawing a listing back into a picture, which is the only way to tell whether
// it was read right: a strip that decodes without falling over can still be
// nonsense, and a picture of a house cannot.
//
// Colour has to be done the way the machine does it. Many drawings trace in
// INK 0 over a coloured PAPER, so rendering the ink alone would leave them
// invisible. So the pixels go in one plane and the attributes in another, one
// per 8x8 cell, and they are only combined at the end.

// The Spectrum palette, normal and bright.
var (
	normalInks = [8]color.NRGBA{
		{0x00, 0x00, 0x00, 0xFF},
		{0x00, 0x00, 0xCD, 0xFF},
		{0xCD, 0x00, 0x00, 0xFF},
		{0xCD, 0x00, 0xCD, 0xFF},
		{0x00, 0xCD, 0x00, 0xFF},
		{0x00, 0xCD, 0xCD, 0xFF},
		{0xCD, 0xCD, 0x00, 0xFF},
		{0xCD, 0xCD, 0xCD, 0xFF},
	}
	brightInks = [8]color.NRGBA{
		{0x00, 0x00, 0x00, 0xFF},
		{0x00, 0x00, 0xFF, 0xFF},
		{0xFF, 0x00, 0x00, 0xFF},
		{0xFF, 0x00, 0xFF, 0xFF},
		{0x00, 0xFF, 0x00, 0xFF},
		{0x00, 0xFF, 0xFF, 0xFF},
		{0xFF, 0xFF, 0x00, 0xFF},
		{0xFF, 0xFF, 0xFF, 0xFF},
	}
)

const (
	canvasWidth  = 256
	cellSize     = 8
	cellsAcross  = canvasWidth / cellSize
	cellsDown    = canvasHeight / cellSize
	maxCallDepth = 10 // the engine errors past ten levels
	// fillLimit is a backstop against a loop, nothing more. Once the pen stopped
	// being moved by a fill, no fill in any of these games comes near it: every
	// region turns out to be properly enclosed, as it has to be on the real
	// machine.
	fillLimit = canvasWidth * canvasHeight
)

// attribute is the colour of one character cell.
type attribute struct {
	ink, paper byte
	bright     bool
	set        bool
}

// screen is the pixel plane and the attribute plane, plus the pen state the
// engine keeps: where it is, what colour it is, and the transform a CALL is
// running under.
type screen struct {
	pixels [canvasWidth * canvasHeight]bool
	cells  [cellsAcross * cellsDown]attribute

	x, y   int
	ink    byte
	paper  byte
	bright bool

	// What a cell shows when the drawing never touches it. The engine does not
	// clear the screen; the interpreter has already put the drawing's own
	// attribute on it (§9.6), so that is what shows through.
	background attribute
}

func (s *screen) cellAt(x, y int) *attribute {
	return &s.cells[(y/cellSize)*cellsAcross+x/cellSize]
}

// The pen modes PLOT and DRAW carry in bits 3 and 4. 0x7CA8 folds them into
// P-FLAG as (mode+2) and 5, which sets bit 0 for OVER and bit 2 for INVERSE —
// so mode 1 is OVER, mode 2 is INVERSE and mode 3 is both. For a DRAW, mode 3
// never reaches here: it is intercepted earlier as a MOVE.
const (
	penNormal = iota
	penOver
	penInverse
	penOverInverse
)

// paint sets a pixel and colours the cell it falls in. Coordinates come in as
// the engine has them, with y running upwards.
func (s *screen) paint(x, y int) { s.mark(x, y, penNormal) }

// mark puts a pixel down the way the pen mode says: laid on, exclusive-ored, or
// wiped out. The ROM's PLOT-SUB tests OVER before INVERSE, so a pen that is
// both exclusive-ors. The attribute of the cell is written either way.
func (s *screen) mark(x, y, mode int) {
	if x < 0 || x >= canvasWidth || y < 0 || y >= canvasHeight {
		return
	}

	at := (canvasHeight-1-y)*canvasWidth + x

	switch mode {
	case penOver, penOverInverse:
		s.pixels[at] = !s.pixels[at]
	case penInverse:
		s.pixels[at] = false
	default:
		s.pixels[at] = true
	}

	s.colour(x, y)
}

// colour writes the attribute of a cell without touching the pixels in it. The
// fill does this on every pixel it passes, whether the pattern lights it or
// not: 0x7F29 calls the attribute routine unconditionally.
func (s *screen) colour(x, y int) {
	if x < 0 || x >= canvasWidth || y < 0 || y >= canvasHeight {
		return
	}

	*s.cellAt(x, canvasHeight-1-y) = attribute{
		ink: s.ink, paper: s.paper, bright: s.bright, set: true,
	}
}

func (s *screen) lit(x, y int) bool {
	if x < 0 || x >= canvasWidth || y < 0 || y >= canvasHeight {
		return true // off the canvas counts as a wall, so a fill stops there
	}

	return s.pixels[(canvasHeight-1-y)*canvasWidth+x]
}

// line traces from one point to another the way the ROM does, because that is
// what the engine calls: the DRAW handler hands it the two magnitudes and their
// signs and jumps into DRAW-LINE at 0x24BA (0x7C9B).
//
// It is a run along the longer axis — one point for every step of it — with a
// step of the shorter axis folded into the same move whenever the error runs
// over. So a diagonal comes out as a diagonal, one pixel to the row, and the
// whole line is max(|dx|,|dy|) points long.
//
// That is not what a plain Bresenham stepping one axis at a time draws. It lays
// an extra pixel at every corner and comes out as a staircase, and a staircase
// is watertight where a diagonal is not — so the fills these drawings are made
// of stop in altogether different places. Reading the line the wrong way left
// 744 pixels disagreeing with the captures of La Aventura Original; reading it
// as the ROM does leaves 33, and the whole of the second part exact.
//
// The starting point is drawn here, which the ROM does not do: there it is on
// the screen already, from the PLOT or the DRAW before.
func (s *screen) line(x0, y0, x1, y1, mode int) {
	s.mark(x0, y0, mode)

	dx, dy := abs(x1-x0), abs(y1-y0)
	sx, sy := step(x0, x1), step(y0, y1)

	major, minor := dx, dy

	along := dx >= dy
	if !along {
		major, minor = dy, dx
	}

	// The error opens at half the longer axis: 0x24CC is `ld a,b / rra`.
	err := major / 2

	for range major {
		if err += minor; err >= major {
			err -= major

			if along {
				y0 += sy
			} else {
				x0 += sx
			}
		}

		if along {
			x0 += sx
		} else {
			y0 += sy
		}

		s.mark(x0, y0, mode)
	}
}

// maxBranches is how many runs the engine will hold waiting at once (0x7EC3
// loads thirty into A'). Reaching it is not an error there: the routine simply
// stops branching and leaves the rest of the region unshaded, which is why the
// originals have fills that peter out.
const maxBranches = 30

// fill lays a graphic over a region, following the engine's routine at 0x7E6B
// rather than approximating it.
//
// The shape of it: a scanline fill that goes in one vertical direction only,
// run by run, seeding the next line where it finds a gap — and then, having
// exhausted that, starts again one line the other way and does it all again.
// Two one-way passes, not one spread in every direction.
//
// Three details decide what these pictures look like, and none of them is what
// a modern fill would do:
//
//   - It gives up. Thirty runs may be waiting at once; the thirty-first is not
//     taken and the pass ends there, part-filled.
//   - It colours every cell it walks over, whether the pattern lights that
//     pixel or not (0x7F29 calls the attribute routine unconditionally).
//   - It only ever turns pixels on (`or (hl)` at 0x7F27), so a pattern's holes
//     leave whatever was underneath.
func (s *screen) fill(x, y int, pattern []byte) {
	// A byte a scanline, holding how far right that line was filled. It is set
	// up once and carried across both passes (0x7FAC re-enters after the
	// initialisation at 0x7EA5), which is what stops the second pass redoing
	// the first.
	var (
		edge   [canvasHeight]int
		region []image.Point
	)

	// Both of the ways out below are `break` and not `return`, and the
	// difference is not cosmetic. The engine paints as it goes, so giving up on
	// the second pass leaves whatever the first one already put on the screen.
	// Gathering the region to paint at the end means a return here would throw
	// that away — and it did: every fill seeded on the top row lost its whole
	// first pass, because the second starts a line beyond the canvas.
	for _, upwards := range [2]bool{false, true} {
		from := y
		if upwards {
			// The second pass starts one line the other way (0x7FA8), and is
			// skipped altogether when that falls off the canvas (0x7FAA).
			if from = y + 1; from >= canvasHeight {
				break
			}
		}

		// A starting point already under ink ends the routine (0x7EC0 jumps to
		// the exit), which for the second pass means keeping what the first did.
		if s.lit(x, from) {
			break
		}

		region = s.fillPass(x, from, upwards, &edge, region)
	}

	for _, p := range region {
		// Which row of the pattern applies is the engine's y and not the screen
		// row: 0x7EE8 does `and 7` on the y the ROM's PLOT takes, counting from
		// the bottom, and writes it into its own `ld d,(ix+NN)`.
		//
		// And that displacement reaches the rows the other way up, because of how
		// they were stored. 0x7E98 builds them with `push af / inc sp`, a byte at
		// a time down the stack, while ix was set to the bottom of the eight
		// before the loop ran (0x7E90). So (ix+0) is the last row of the graphic
		// and (ix+7) the first, and displacement d picks row 7-d. Turned over
		// like this it comes out upright on the screen, y running the other way.
		if pattern[cellSize-1-p.Y%cellSize]&(0x80>>(p.X%cellSize)) != 0 {
			s.paint(p.X, p.Y)
		} else {
			s.colour(p.X, p.Y)
		}
	}
}

// fillPass is one of the two sweeps, seeding only in the direction given. It
// gathers what it covers rather than painting as it goes, so the caller can
// weigh the whole region before committing any of it.
func (s *screen) fillPass(
	x, y int, upwards bool, edge *[canvasHeight]int, region []image.Point,
) []image.Point {
	type run struct{ x, y int }

	seeds := []run{{x: x, y: y}}
	branches := maxBranches

	for len(seeds) > 0 && len(region) < fillLimit {
		here := seeds[len(seeds)-1]
		seeds = seeds[:len(seeds)-1]

		next := here.y - 1
		if upwards {
			next = here.y + 1
		}

		// First run right to the edge of the region, and remember how far it
		// reached (0x7F07 to 0x7F19).
		column := here.x
		for column < canvasWidth-1 && !s.lit(column+1, here.y) {
			column++
		}

		edge[here.y] = column

		// Then work leftwards from there, seeding the next line on the way.
		// `seeded` is bit 0 of the engine's state and `done` is bit 1.
		seeded, done := false, false

		for ; column >= 0 && !s.lit(column, here.y); column-- {
			region = append(region, image.Point{X: column, Y: here.y})

			if done || next < 0 || next >= canvasHeight {
				continue
			}

			// A line already filled at least this far needs no more seeds, and
			// this run stops looking altogether (0x7F40 sets both bits).
			if e := edge[next]; e != 0 && e >= column {
				seeded, done = true, true

				continue
			}

			// Otherwise seed on the edge where the next line opens up.
			switch open := !s.lit(column, next); {
			case seeded:
				seeded = open
			case open:
				if branches--; branches == 0 {
					return region
				}

				seeds = append(seeds, run{x: column, y: next})
				seeded = true
			}
		}

		// A closed run hands its branch back (0x7F95).
		branches++
	}

	return region
}

// solidPattern is what a FILL without a graphic paints with: the seven 0xFF
// bytes the handler points at in 0x7CDC.
var solidPattern = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

// fillPattern builds the eight rows the fill paints with, the way 0x7E98 does:
// each row of the graphic exclusive-ored with 0x00, or with 0xFF when bit 4 of
// the order asked for the pattern turned over.
func fillPattern(graphic []byte, inverse bool) []byte {
	var over byte
	if inverse {
		over = 0xFF
	}

	rows := make([]byte, cellSize)
	for i := range rows {
		rows[i] = graphic[i%len(graphic)] ^ over
	}

	return rows
}

// halt asks a trace to stop the moment it is about to call a given drawing,
// keeping the pen state as it stood right then. That state is what the called
// drawing would have inherited, and without it a piece meant to be stamped into
// a scene traces from nowhere, in whatever colour we guessed.
type halt struct {
	at      int  // the drawing whose call to stop at
	reached bool // whether the call turned up
}

// trace runs one drawing onto the screen, following its calls.
func (v vectors) trace(s *screen, number, scale, mirror, depth int, stop *halt) {
	if depth > maxCallDepth || number < 0 || number >= len(v.drawings) {
		return
	}

	for _, o := range v.drawings[number].orders {
		if stop != nil && stop.reached {
			return
		}

		switch o.name {
		case "PLOT":
			s.x, s.y = o.args[0], o.args[1]%canvasHeight
			s.mark(s.x, s.y, int(o.code>>3)&3)

		case "LINE", "MOVE", "FILL":
			dx, dy := scaleBy(o.args[0], scale), scaleBy(o.args[1], scale)

			if mirror&1 != 0 {
				dx = -dx
			}

			if mirror&2 != 0 {
				dy = -dy
			}

			x1, y1 := (s.x+dx)&0xFF, mod(s.y+dy, canvasHeight)

			switch o.name {
			case "LINE":
				s.line(s.x, s.y, x1, y1, int(o.code>>3)&3)
			case "FILL":
				graphic := solidPattern
				if len(o.args) > 2 {
					if g := v.graphic(o.args[2]); g != nil {
						graphic = g
					}
				}

				s.fill(x1, y1, fillPattern(graphic, o.note == "inverse"))
			}

			// A line and a move take the pen with them; a fill does not. The
			// handlers differ by one instruction: 0x7C75 writes the new
			// position to COORDS, and 0x7CC5 works out where to fill from the
			// same operands and leaves COORDS alone. So a run of fills all
			// measure from the one place, rather than walking away from it.
			if o.name != "FILL" {
				s.x, s.y = x1, y1
			}

		case "CALL":
			// Stopping here rather than descending is what captures the state
			// the called drawing starts from.
			if stop != nil && o.args[0] == stop.at {
				stop.reached = true

				return
			}

			// The handler writes the scale over whatever was there but XORs the
			// mirror onto it, so one replaces and the other accumulates.
			v.trace(s, o.args[0], o.args[1], mirror^o.args[2], depth+1, stop)

		case "UDG":
			v.stamp(s, o.args[0], o.args[1], o.args[2])

		case "INK", "FLASH":
			s.ink = byte(o.args[0])

		case "PAPER":
			s.paper = byte(o.args[0])

		case "BRIGHT":
			s.bright = o.args[0] != 0

		case "ATTR":
			// A rectangle painted straight into the attribute file: rows,
			// columns, starting column and starting row, the counts being one
			// less than what gets painted. This is how a drawing clears its
			// window.
			s.rectangle(o.args[3], o.args[2], o.args[0]+1, o.args[1]+1)
		}
	}
}

// stamp puts a graphic where the UDG order asks for it, on a whole character
// cell: column times eight.
//
// It is tempting to reach for the interpreter's own six-pixel font here — it
// has one, and its text really is 42 columns wide (§9.6) — but the UDG handler
// prints through RST 0x10, which lands in the ordinary routine of 32 columns.
// Comparing against a capture settles it: drawing 40 of part 2 puts a letter at
// column 8 and it appears at x=64, not x=48.
func (v vectors) stamp(s *screen, graphic, column, row int) {
	rows := v.graphic(graphic)
	if rows == nil || column >= charColumns || row >= cellsDown {
		return
	}

	left := column * cellSize
	top := row * cellSize

	for y := range cellSize {
		for x := range cellSize {
			if rows[y]&(0x80>>x) == 0 {
				continue
			}

			if at := left + x; at < canvasWidth {
				s.pixels[(top+y)*canvasWidth+at] = true
			}
		}
	}

	// The colour goes on whichever cells the graphic actually fell across.
	for _, at := range [2]int{left, left + cellSize - 1} {
		if at < canvasWidth {
			s.cells[row*cellsAcross+at/cellSize] = attribute{
				ink: s.ink, paper: s.paper, bright: s.bright, set: true,
			}
		}
	}
}

// rectangle colours a block of cells without touching the pixels under it.
func (s *screen) rectangle(row, column, rows, columns int) {
	for y := row; y < row+rows && y < cellsDown; y++ {
		for x := column; x < column+columns && x < cellsAcross; x++ {
			if y >= 0 && x >= 0 {
				s.cells[y*cellsAcross+x] = attribute{
					ink: s.ink, paper: s.paper, bright: s.bright, set: true,
				}
			}
		}
	}
}

// picture combines the two planes into an image the size of the canvas the
// engine traces on. Where the adventure later prints its text over the lower
// rows is the interpreter's business and not the drawing's, so the whole of
// what was traced comes out.
func (s *screen) picture() image.Image {
	const height = canvasHeight

	img := image.NewNRGBA(image.Rect(0, 0, canvasWidth, height))

	for row := range height {
		for column := range canvasWidth {
			cell := s.cells[(row/cellSize)*cellsAcross+column/cellSize]
			if !cell.set {
				cell = s.background
			}

			inks := &normalInks
			if cell.bright {
				inks = &brightInks
			}

			shade := inks[cell.paper&7]
			if s.pixels[row*canvasWidth+column] {
				shade = inks[cell.ink&7]
			}

			img.SetNRGBA(column, row, shade)
		}
	}

	return img
}

// The screen as the machine held it: 6144 bytes of pixels and 768 of
// attributes, which is what a .SCR is and what every tool that takes a
// Spectrum picture expects.
const (
	scrPixels     = 6144
	scrAttributes = 768
	scrSize       = scrPixels + scrAttributes
	scrRows       = 24 // character rows in a full screen, against our 22
	scrBytesWide  = canvasWidth / cellSize
)

// scr writes the two planes out in the machine's own order.
//
// A .SCR is a dump of video memory and video memory is not laid out the way
// the screen reads. The display is three banks of 64 lines; within a bank the
// pixel row of the character comes first and the character row second, so
// consecutive lines of a drawing are 256 bytes apart and the eighth of them
// jumps back. Attributes are the easy half: one byte per cell, left to right
// and top to bottom.
//
// The canvas is 176 lines and a screen is 192, and the difference is not
// padding to be trimmed later: the kit's tools take 6912 bytes exactly and the
// adventure draws its text over the lower rows anyway. So the drawing goes at
// the top and the rest is left as the background attribute over empty pixels,
// which is what the machine showed there.
func (s *screen) scr() []byte {
	out := make([]byte, scrSize)

	for y := range canvasHeight {
		bank, pixelRow, charRow := y>>6, y&7, (y>>3)&7
		line := bank*2048 + pixelRow*256 + charRow*scrBytesWide

		for x := range canvasWidth {
			if s.pixels[y*canvasWidth+x] {
				out[line+x/cellSize] |= 0x80 >> (x % cellSize)
			}
		}
	}

	for row := range scrRows {
		for column := range cellsAcross {
			cell := s.background
			if row < cellsDown {
				if c := s.cells[row*cellsAcross+column]; c.set {
					cell = c
				}
			}

			out[scrPixels+row*cellsAcross+column] = cell.byte()
		}
	}

	return out
}

// byte is the attribute as the hardware stores it: ink in the low three bits,
// paper in the next three, bright above them. FLASH is not something any of
// these drawings sets.
func (a attribute) byte() byte {
	const bright = 0x40

	out := a.ink&7 | (a.paper&7)<<3
	if a.bright {
		out |= bright
	}

	return out
}

// inherited is the state a called drawing starts from: whose call it came out
// of, and the pen as that caller had left it. It is written into the listing as
// an INHERIT order so the file says where it stands without anyone having to go
// and read the caller.
type inherited struct {
	parent     int
	x, y       int
	ink, paper byte
	bright     bool
}

// inheritedBy works out that state, or reports that there is none because
// nothing calls this drawing.
//
// The first caller is the one taken: where a piece is stamped into several
// scenes there is no single right answer, and the first is at least always the
// same one. Callers are followed up the chain, since the first caller may
// itself be a piece somebody else stamps.
func (v vectors) inheritedBy(number int, callers map[int][]int, depth int) (inherited, bool) {
	if depth > maxCallDepth || len(callers[number]) == 0 {
		return inherited{}, false
	}

	caller := callers[number][0]

	// A drawing that reaches itself would recurse for ever.
	if caller == number {
		return inherited{}, false
	}

	// Trace the caller onto a screen of its own, so its marks go nowhere near
	// the picture, and keep only the pen it had reached by the call.
	scratch := v.blankScreen(caller)

	if from, ok := v.inheritedBy(caller, callers, depth+1); ok {
		scratch.apply(from)
	}

	stop := &halt{at: number}
	v.trace(scratch, caller, 0, 0, 0, stop)

	if !stop.reached {
		return inherited{}, false
	}

	return inherited{
		parent: caller,
		x:      scratch.x,
		y:      scratch.y,
		ink:    scratch.ink,
		paper:  scratch.paper,
		bright: scratch.bright,
	}, true
}

// blankScreen is the screen a drawing is entered on: the pen at the origin and
// the ink and paper the interpreter took from the drawing's own record. Where
// there is no record, white on black is the least surprising guess.
func (v vectors) blankScreen(number int) *screen {
	ink, paper, ok := v.startsWith(number)
	if !ok {
		ink, paper = 7, 0
	}

	return &screen{
		ink:   ink,
		paper: paper,
		// Untouched cells keep that same attribute: it is what the interpreter
		// wrote to ATTR-P and ATTR-T before the drawing ran.
		background: attribute{ink: ink, paper: paper, set: true},
	}
}

// apply puts a screen into an inherited state.
func (s *screen) apply(from inherited) {
	s.x, s.y = from.x, from.y
	s.ink, s.paper, s.bright = from.ink, from.paper, from.bright
}

// maybeInherited carries an inherited state that may not be there, for the
// drawings nobody calls.
type maybeInherited struct {
	state inherited
	found bool
}

func (m maybeInherited) get() (inherited, bool) { return m.state, m.found }

// inheritedFor is inheritedBy packaged for a caller that only wants to pass the
// answer around.
func (v vectors) inheritedFor(number int, callers map[int][]int) maybeInherited {
	state, found := v.inheritedBy(number, callers, 0)

	return maybeInherited{state: state, found: found}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}

	return 0
}

// render traces one drawing of the set.
//
// A drawing somebody calls does not begin from nothing: it inherits the pen
// where its caller left it, and the ink and paper in force at that moment. Drawn
// on its own it starts from there. Drawn as part of a scene it does not, because
// then the caller really is running and sets the state itself — which is why the
// INHERIT order is used on a picture of one and ignored on a picture of the
// other.
//
// It returns the screen rather than the picture because a picture is only one
// of the two things worth having out of a drawing: the other is the screen
// itself, byte for byte as the machine held it, which is what a packager
// needs. Both come out of the same tracing.
func (v vectors) render(number int, callers map[int][]int) *screen {
	s := v.blankScreen(number)

	if from, ok := v.inheritedBy(number, callers, 0); ok {
		s.apply(from)
	}

	v.trace(s, number, 0, 0, 0, nil)

	return s
}

// enlarge scales an image by a whole number, pixel for pixel, so that what
// comes out is still exactly what the machine drew.
func enlarge(src image.Image, by int) image.Image {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*by, b.Dy()*by))

	for y := range b.Dy() {
		for x := range b.Dx() {
			shade := src.At(b.Min.X+x, b.Min.Y+y)

			for dy := range by {
				for dx := range by {
					out.Set(x*by+dx, y*by+dy, shade)
				}
			}
		}
	}

	return out
}

// scaleBy is the engine's transform: increment x factor / 8. Scale 0 is natural
// size, and 1 to 7 only ever reduce.
func scaleBy(value, scale int) int {
	if scale == 0 {
		return value
	}

	// The magnitude is scaled and the sign put back, rather than dividing a
	// negative number, so that it rounds the way the engine's shifts do.
	reduced := abs(value) * scale / 8
	if value < 0 {
		return -reduced
	}

	return reduced
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

func step(from, to int) int {
	if from < to {
		return 1
	}

	return -1
}

func mod(v, by int) int {
	v %= by
	if v < 0 {
		v += by
	}

	return v
}
