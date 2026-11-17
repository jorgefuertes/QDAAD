package decompiler

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/version"
)

// The Amstrad CPC drawings, which are the same idea as the Spectrum's and not
// the same format. See vector.go for the shape of the thing: a table of
// pointers in the tail of the block, a strip of orders to a drawing, and
// nothing anywhere naming where the block is.
//
// Three things are different and each of them matters.
//
// The block does not end at the top of memory. It cannot: from 0xC000 up is
// video memory. It ends at 0x9E00, and the tail sits below that at the same
// distances the Spectrum's sits below 0x10000 — the CALL handler reads its
// table from (0x9DF1), which is fifteen bytes down, exactly as the Spectrum
// reads (0xFFF1).
//
// The opcode is in bits 1-3 and not 0-2. The dispatcher isolates it with
// `and 0x0E` and indexes the jump table with it directly, saving itself the
// `sla a` the Spectrum's needs. So every other field of the order byte is a
// bit over from where the Spectrum has it.
//
// And the colour is a palette, not an attribute. There are four pens in mode 1
// and each drawing's record says which of the machine's twenty-seven inks they
// are to be, so a drawing can recolour the whole screen without touching a
// pixel.
//
// Worked out by disassembling the engine of La Aventura Original, whose
// dispatcher is at 0x1EC3 and whose eight handlers are at 0x1EE7, 0x1F1D,
// 0x1F47, 0x1F9E, 0x1FC3, 0x1FDA, 0x1FF8 and 0x205A. The addresses are of that
// build and nothing here depends on them.
const (
	// cpcAnchor is where a block of drawings ends.
	cpcAnchor = 0x9E00

	cpcDrawingTableAt = cpcAnchor - 15 // base of the table of drawings
	// cpcPropertiesAt is one past the last entry of that table, which is also
	// the base of the records the interpreter reads before running a drawing.
	cpcPropertiesAt    = cpcAnchor - 13
	cpcPropertiesEndAt = cpcAnchor - 11 // one past the last of those records

	// cpcPropertySize is the eight bytes 0x1E68 indexes by drawing number: four
	// inks and the four numbers of a text window.
	cpcPropertySize = 8

	// cpcTailSize is how far back from the end of a block its first pointer
	// sits, which is as early as a block can end and still hold one.
	cpcTailSize = cpcAnchor - cpcDrawingTableAt
)

// The order lengths, by opcode. Unlike the Spectrum's these do not vary from
// one build to another: every handler ends by loading E with its own length
// and jumping to 0x1EBF, which adds it to IX, and each loads a constant.
var cpcLengths = [8]int{
	cpcOpPen:         1,
	cpcOpChar:        4,
	cpcOpCall:        2,
	cpcOpPlot:        3,
	cpcOpLine:        3,
	cpcOpFill:        3,
	cpcOpFillPattern: 4,
	cpcOpBox:         5,
}

// The opcodes, in the order of the jump table.
const (
	cpcOpPen = iota // set the pen, or return
	cpcOpChar
	cpcOpCall
	cpcOpPlot        // absolute, traced or not
	cpcOpLine        // relative, traced or not
	cpcOpFill        // in the current pen
	cpcOpFillPattern // in a pattern of two colours
	cpcOpBox
)

// The bits of an order byte that are not the opcode.
//
// Bit 7 is the ninth bit of the first operand, because the canvas is 320 wide
// and a byte does not reach; bit 6 says whether the order traces or only moves;
// bits 4 and 5 are the signs of a relative pair, and bit 0 is read by whoever
// has a use for it. The one order that does not follow this is the pen, which
// has no operands and spends bit 6 on saying it is a return instead.
const (
	cpcBitNinth = 0x80
	cpcBitTrace = 0x40
	cpcBitSignY = 0x10
	cpcBitSignX = 0x20
	cpcBitSpare = 0x01
)

// cpcDrawing is one entry of the table.
type cpcDrawing struct {
	number int
	addr   int
	at     int
	size   int
	orders []order
}

// cpcVectors is the set of drawings of one loaded block.
type cpcVectors struct {
	load  int // address = payload offset + load
	table int
	// properties is the eight-byte record per drawing that 0x1E68 indexes.
	properties    []byte
	propertyTable int
	// patterns is the character set a patterned FILL paints with. The engine
	// does not take it out of the block: the handler has 0x9E00 in it as a
	// constant, which is where the interpreter's own font is loaded, and the
	// shading patterns are its glyphs from 128 up.
	patterns []byte
	drawings []cpcDrawing
}

// isEmpty says whether a drawing does nothing but return.
func (d cpcDrawing) isEmpty() bool {
	return len(d.orders) == 1 && d.orders[0].code&cpcBitTrace != 0
}

func (v cpcVectors) offsetOf(addr int) int { return addr - v.load }

func (v cpcVectors) wordAtAddress(data []byte, addr int) (int, bool) {
	return wordAt(data, v.offsetOf(addr))
}

// dataAt is where the drawings themselves begin, which is what says which
// database a set belongs to.
func (v cpcVectors) dataAt() int {
	if len(v.drawings) == 0 {
		return 0
	}

	return v.drawings[0].at
}

// pattern returns the eight rows of one shading glyph, or nil past the end.
func (v cpcVectors) pattern(n int) []byte {
	at := n * cpcPatternSize
	if n < 0 || at+cpcPatternSize > len(v.patterns) {
		return nil
	}

	return v.patterns[at : at+cpcPatternSize]
}

// palette is the four inks the interpreter sets before running a drawing, and
// whether the record says to set them at all.
//
// Bit 7 of the first ink is not an ink: 0x1E59 tests it and skips the whole
// preparation when it is set — no palette, no window, no clearing and no reset
// of the pen. Those are the drawings meant to be stamped into a scene, which
// take the screen as they find it.
//
// An ink of 27 or more is past the machine's twenty-seven colours and means
// that pen is to be left as it stands. 0x22D2 compares against 27 and skips.
func (v cpcVectors) palette(number int) (inks [cpcPens]int, prepares bool) {
	at := number * cpcPropertySize
	if number < 0 || at+cpcPropertySize > len(v.properties) {
		return inks, false
	}

	if v.properties[at]&0x80 != 0 {
		return inks, false
	}

	for pen := range cpcPens {
		inks[pen] = int(v.properties[at+pen])
	}

	return inks, true
}

// window is the text window the record declares: top row, left column, rows and
// columns, in characters. 0x1E74 hands it to TXT WIN ENABLE.
//
// As on the Spectrum this is not a clipping rectangle. It says where the
// interpreter writes afterwards, and the drawings paint outside it freely — the
// engine hands every coordinate to the firmware and the firmware clips to the
// graphics window, which nothing here ever narrows.
func (v cpcVectors) window(number int) (top, left, rows, columns int, ok bool) {
	at := number * cpcPropertySize
	if number < 0 || at+cpcPropertySize > len(v.properties) {
		return 0, 0, 0, 0, false
	}

	r := v.properties[at:]

	return int(r[4]), int(r[5]), int(r[6]), int(r[7]), true
}

// callers returns, for each drawing, the ones that invoke it.
func (v cpcVectors) callers() map[int][]int {
	found := map[int][]int{}

	for _, d := range v.drawings {
		for _, o := range d.orders {
			if o.name != "CALL" {
				continue
			}

			if to := o.args[0]; !slices.Contains(found[to], d.number) {
				found[to] = append(found[to], d.number)
			}
		}
	}

	return found
}

// machineCPC is the number the high nibble of header byte 1 carries for an
// Amstrad CPC build.
const machineCPC = 0x03

// isCPC says whether a set of databases found in one blob was built for the
// Amstrad, which is what decides the format of the drawings beside them.
func isCPC(databases []database) bool {
	for _, db := range databases {
		if db.reader.Machine() == machineCPC {
			return true
		}
	}

	return false
}

// findCPCVectors returns every set of drawings the data holds.
//
// Same method as the Spectrum's, and for the same reason: nothing marks a block,
// so every offset is tried as its end and the arrangement that implies is what
// accepts or rejects it. On the two blocks of La Aventura Original every one of
// the 157 drawings runs to its return on the byte the next one starts at, and
// the candidates that are not blocks fail on hundreds of them.
func findCPCVectors(data []byte) []cpcVectors {
	var found []cpcVectors

	for end := cpcTailSize; end <= len(data); end++ {
		if v, ok := cpcVectorsEndingAt(data, end); ok {
			found = append(found, v)
		}
	}

	return found
}

// attachCPCVectors gives each set of drawings to the database it belongs to:
// the last one that starts before it.
func attachCPCVectors(databases []database, sets []cpcVectors) {
	for i := range sets {
		owner := -1

		for j := range databases {
			if databases[j].reader.Offset() <= sets[i].dataAt() {
				owner = j
			}
		}

		if owner >= 0 && databases[owner].cpcVectors == nil {
			databases[owner].cpcVectors = &sets[i]
		}
	}
}

// cpcVectorsEndingAt reads the set of drawings of a block supposed to end at
// the given offset, and says whether that supposition holds.
func cpcVectorsEndingAt(data []byte, end int) (cpcVectors, bool) {
	v := cpcVectors{load: cpcAnchor - end}

	table, ok := v.wordAtAddress(data, cpcDrawingTableAt)
	if !ok {
		return cpcVectors{}, false
	}

	tableEnd, ok := v.wordAtAddress(data, cpcPropertiesAt)
	if !ok {
		return cpcVectors{}, false
	}

	v.table = table

	pointers, ok := v.readTable(data, table, tableEnd)
	if !ok {
		return cpcVectors{}, false
	}

	count := len(pointers) - 2

	// The eight-byte record per drawing, bounded by the pointer that follows
	// it. That the two agree to the byte is a good part of why this arrangement
	// can be trusted: 392 bytes for the 49 drawings of the first part and 864
	// for the 108 of the second.
	if at, ok := v.wordAtAddress(data, cpcPropertiesEndAt); ok && at-tableEnd == cpcPropertySize*count {
		if from, to := v.offsetOf(tableEnd), v.offsetOf(at); 0 <= from && from < to && to <= len(data) {
			v.properties, v.propertyTable = data[from:to], tableEnd
		}
	}

	// The shading patterns are the character set at 0x9E00, which is where this
	// block ends, so they are the bytes immediately after it. That they really
	// are there is not assumed: the two parts hold the same nineteen patterns at
	// the same places, glyphs 128 to 146, and they are the dithers you would
	// expect — 0xAA55 chequers, 0x8040201008040201 diagonals.
	if to := end + cpcCharsetSize; to <= len(data) {
		v.patterns = data[end:to]
	}

	drawings, ok := v.readDrawings(data, pointers)
	if !ok {
		return cpcVectors{}, false
	}

	v.drawings = drawings

	return v, true
}

// readTable reads the pointers of the table of drawings and checks the shape
// the engine's own arrangement demands, which is the Spectrum's: distinct and
// strictly increasing, the last but one pointing at the table itself to bound
// the last drawing, and 0xFFFF closing it.
func (v cpcVectors) readTable(data []byte, table, end int) ([]int, bool) {
	if end <= table || (end-table)%2 != 0 {
		return nil, false
	}

	entries := (end - table) / 2

	const maxEntries = 4096
	if entries < 3 || entries > maxEntries {
		return nil, false
	}

	last, ok := v.wordAtAddress(data, table+2*(entries-1))
	if !ok || last != 0xFFFF {
		return nil, false
	}

	bound, ok := v.wordAtAddress(data, table+2*(entries-2))
	if !ok || bound != table {
		return nil, false
	}

	pointers := make([]int, 0, entries)

	for i := range entries {
		p, ok := v.wordAtAddress(data, table+2*i)
		if !ok {
			return nil, false
		}

		pointers = append(pointers, p)
	}

	real := pointers[:len(pointers)-1]
	for i := 1; i < len(real); i++ {
		if real[i] <= real[i-1] {
			return nil, false
		}
	}

	if real[len(real)-1] != table || v.offsetOf(real[0]) < 0 {
		return nil, false
	}

	return pointers, true
}

// readDrawings decodes the whole table, and fails as soon as any drawing does.
func (v cpcVectors) readDrawings(data []byte, pointers []int) ([]cpcDrawing, bool) {
	count := len(pointers) - 2

	drawings := make([]cpcDrawing, 0, count)

	for i := range count {
		one, ok := v.readDrawing(data, i, pointers[i], pointers[i+1], count)
		if !ok {
			return nil, false
		}

		drawings = append(drawings, one)
	}

	return drawings, true
}

// readDrawing decodes one drawing and insists it ends where the next begins.
func (v cpcVectors) readDrawing(data []byte, number, from, to, count int) (cpcDrawing, bool) {
	at, stop := v.offsetOf(from), v.offsetOf(to)
	if at < 0 || stop > len(data) || stop <= at {
		return cpcDrawing{}, false
	}

	d := cpcDrawing{number: number, addr: from, at: at, size: stop - at}

	for o := at; ; {
		if o >= stop {
			return cpcDrawing{}, false
		}

		code := data[o]
		opcode := int(code>>1) & 7
		size := cpcLengths[opcode]

		if o+size > stop {
			return cpcDrawing{}, false
		}

		name, args, note, ok := decodeCPCOrder(data, o, count)
		if !ok {
			return cpcDrawing{}, false
		}

		d.orders = append(d.orders,
			order{at: o, code: code, size: size, name: name, args: args, note: note})

		o += size

		if opcode == cpcOpPen && code&cpcBitTrace != 0 {
			if o != stop {
				return cpcDrawing{}, false
			}

			return d, true
		}
	}
}

// decodeCPCOrder reads one order into the mnemonic and operands it is written
// as. Drawing a listing and writing one go through here alike, so a picture can
// never disagree with the text beside it.
//
// It fails on an operand that cannot be right, which is what tells a drawing
// from a stretch of bytes that happens to decode.
func decodeCPCOrder(data []byte, at, count int) (name string, args []int, note string, ok bool) {
	code := data[at]

	switch int(code>>1) & 7 {
	case cpcOpPen:
		// Bit 6 is not a pen but a return. Where it is clear the handler
		// rotates the byte left and hands it to GRA SET PEN, which masks it to
		// the four pens of mode 1 — so the pen is bit 7 and bit 0, in that
		// order, and nothing else of the byte reaches it.
		if code&cpcBitTrace != 0 {
			return "RET", nil, "", true
		}

		pen := 0
		if code&cpcBitNinth != 0 {
			pen |= 1
		}

		if code&cpcBitSpare != 0 {
			pen |= 2
		}

		return "PEN", []int{pen}, "", true

	case cpcOpChar:
		x, y := cpcAbsolute(data, at, code)
		if y >= cpcDrawRows {
			return "", nil, "", false
		}

		return "CHAR", []int{int(data[at+3]), x, y}, "", true

	case cpcOpCall:
		n := int(data[at+1])
		if n >= count {
			return "", nil, "", false
		}

		return "CALL", []int{n, cpcScaleOf(code), cpcMirrorOf(code)}, "", true

	case cpcOpPlot:
		x, y := cpcAbsolute(data, at, code)
		if y >= cpcDrawRows {
			return "", nil, "", false
		}

		if code&cpcBitTrace != 0 {
			return "PLOT", []int{x, y}, "", true
		}

		return "MOVE", []int{x, y}, "", true

	case cpcOpLine:
		dx, dy := cpcRelative(data, at, code)

		if code&cpcBitTrace != 0 {
			return "LINE", []int{dx, dy}, "", true
		}

		return "STEP", []int{dx, dy}, "", true

	case cpcOpFill:
		dx, dy := cpcRelative(data, at, code)

		return "FILL", []int{dx, dy}, "", true

	case cpcOpFillPattern:
		dx, dy := cpcRelative(data, at, code)

		// The pattern's set bits take the pen in force and its clear bits the
		// colour the order carries, which the handler builds out of bit 6 and
		// bit 0 at 0x200F.
		ground := 0
		if code&cpcBitTrace != 0 {
			ground |= 1
		}

		if code&cpcBitSpare != 0 {
			ground |= 2
		}

		return "FILL", []int{dx, dy, int(data[at+3]), ground}, "", true

	default:
		// A rectangle flooded straight into video memory: SCR FLOOD BOX takes
		// the address of its top left corner, which is why this one carries an
		// address and not a coordinate.
		return "BOX", []int{
			int(data[at+1]) | int(data[at+2])<<8, int(data[at+3]), int(data[at+4]),
		}, "", true
	}
}

// cpcAbsolute reads the coordinate pair of an absolute order. The x reaches 320
// and a byte does not, so its ninth bit travels in the order byte.
func cpcAbsolute(data []byte, at int, code byte) (x, y int) {
	x = int(data[at+1])
	if code&cpcBitNinth != 0 {
		x += 0x100
	}

	return x, int(data[at+2])
}

// cpcRelative reads the pair of a relative order: two magnitudes, their signs in
// the order byte rather than in the operands.
func cpcRelative(data []byte, at int, code byte) (dx, dy int) {
	dx, dy = int(data[at+1]), int(data[at+2])

	if code&cpcBitNinth != 0 {
		dx += 0x100
	}

	if code&cpcBitSignX != 0 {
		dx = -dx
	}

	if code&cpcBitSignY != 0 {
		dy = -dy
	}

	return dx, dy
}

// cpcScaleOf is the scale a CALL runs its drawing at: 0 is natural size and 1 to
// 7 are eighths. The handler gathers it with two rotations and a mask, which
// leaves it spread over bits 6, 7 and 0 of the order byte.
func cpcScaleOf(code byte) int {
	scale := 0

	if code&cpcBitTrace != 0 {
		scale |= 1
	}

	if code&cpcBitNinth != 0 {
		scale |= 2
	}

	if code&cpcBitSpare != 0 {
		scale |= 4
	}

	return scale
}

// cpcMirrorOf is the mirror a CALL applies, in the two bits the handler
// exclusive-ors onto the one already in force: bit 0 flips x and bit 1 flips y.
func cpcMirrorOf(code byte) int {
	mirror := 0

	if code&cpcBitSignX != 0 {
		mirror |= 1
	}

	if code&cpcBitSignY != 0 {
		mirror |= 2
	}

	return mirror
}

// writeCPCVectors writes one listing, one picture and one screen per entry of
// the table, numbered as the table numbers them so that a CALL names the file it
// invokes.
//
// The entries holding nothing but a return get no file, the same as on the
// Spectrum: they are real entries and the table counts them, but a listing of
// one order and a picture of nothing are no use to anybody.
func writeCPCVectors(v cpcVectors, gfx string, opts Options) (int, error) {
	if len(v.drawings) == 0 {
		return 0, nil
	}

	callers := v.callers()
	written := 0

	for _, d := range v.drawings {
		if d.isEmpty() {
			continue
		}

		body := cpcPlotFile(v, d, callers[d.number], v.inheritedFor(d.number, callers))
		if err := writeInto(gfx, listingsDir, numbered(d.number, ".plot"), []byte(body)); err != nil {
			return 0, err
		}

		drawn := v.render(d.number, callers)

		picture := drawn.picture()
		if err := writePNGInto(gfx, picturesDir, numbered(d.number, ".png"), picture); err != nil {
			return 0, err
		}

		// And the same drawing as the machine held it: the mode 1 screen the
		// tracing filled in, and beside it the inks its four pens stand for.
		// The picture is for a person and these two are for a packager.
		if err := writeInto(gfx, screensDir, numbered(d.number, ".scr"), drawn.scr()); err != nil {
			return 0, err
		}

		if err := writeInto(gfx, screensDir, numbered(d.number, ".pal"), drawn.pal()); err != nil {
			return 0, err
		}

		if opts.Scale > 1 {
			name := fmt.Sprintf("%03dx%d.png", d.number, opts.Scale)
			if err := writePNGInto(gfx, picturesDir, name, enlarge(picture, opts.Scale)); err != nil {
				return 0, err
			}
		}

		written++
	}

	if err := writeCPCPatterns(v, gfx); err != nil {
		return 0, err
	}

	return written, nil
}

// writeCPCPatterns writes the character set the patterned fills paint with: the
// table as it lies, and a sheet of all of it so it can be looked at.
//
// It is a character set and not a table of shading patterns — the engine takes
// it from where the interpreter's font is loaded — so it goes among the fonts,
// and the 2048 bytes can be handed straight back to the compiler as one.
func writeCPCPatterns(v cpcVectors, gfx string) error {
	if len(v.patterns) < cpcCharsetSize {
		return nil
	}

	if err := writeInto(gfx, fontsDir, "udg.chr", v.patterns[:cpcCharsetSize]); err != nil {
		return err
	}

	return writePNGInto(gfx, fontsDir, "udg.png", fontAtlas(v.patterns[:cpcCharsetSize]))
}

// cpcPlotFile writes one drawing: a header saying what was read and from where,
// then a line an order.
func cpcPlotFile(v cpcVectors, d cpcDrawing, callers []int, start cpcMaybeInherited) string {
	var sb strings.Builder

	line := strings.Repeat("-", 74)
	add := func(f string, a ...any) { fmt.Fprintf(&sb, "; "+f+"\n", a...) }

	add("%s", line)
	add("Decompiled by QUnDAAD - CPC vector drawing %d of %d", d.number, len(v.drawings))
	add("Queru's DAAD decompiler %s", version.Version)
	add("%s", line)
	add("Read from      : payload %#05x..%#05x, %d bytes", d.at, d.at+d.size, d.size)
	add("Laid at        : %#04x   (block loaded so that it ends at %#04x)", d.addr, cpcAnchor)
	add("Table entry    : %d of %d, at %#04x", d.number, len(v.drawings), v.table+2*d.number)
	add("Orders         : %d", len(d.orders))

	if inks, prepares := v.palette(d.number); prepares {
		add("Palette        : pens %s   (record %d of the table at %#04x)",
			cpcInkList(inks), d.number, v.propertyTable)
	} else if len(v.properties) > 0 {
		add("Palette        : inherited   (bit 7 of its record is set, so the")
		add("                 interpreter prepares nothing: no palette, no window,")
		add("                 no clearing, and the pen is left where it was)")
	}

	if top, left, rows, columns, ok := v.window(d.number); ok {
		add("Declared window: %d rows of %d at row %d, column %d   (where the text goes "+
			"afterwards, not a limit: the engine does not clip)", rows, columns, top, left)
	}

	if len(callers) > 0 {
		names := make([]string, len(callers))
		for i, c := range callers {
			names[i] = fmt.Sprintf("%03d", c)
		}

		add("CALLED FROM    : %s", strings.Join(names, ", "))
	}

	add("%s", line)
	add("IMAGE names the drawing, which is what a CALL elsewhere refers to.")
	add("PLOT and MOVE are absolute; LINE, STEP and FILL are relative, each")
	add("starting where the last one ended. PLOT and LINE trace, MOVE and STEP")
	add("only carry the pen. The canvas is %dx%d and its y runs upwards from",
		cpcCanvasWidth, cpcDrawRows)
	add("sixteen pixels above the foot of the screen, where the interpreter puts")
	add("the origin.")
	add("PEN takes one of the four pens of mode 1, whose inks the record sets.")
	add("FILL takes an offset from the pen, and with two more operands a pattern")
	add("out of the character set and the colour its holes are painted in.")
	add("CALL takes a drawing number, a scale (0 natural, 1-7 eighths) and a")
	add("mirror (bit 0 flips x, bit 1 flips y).")

	if from, ok := start.get(); ok {
		add("INHERIT is not an order of the engine's. It records the state this")
		add("drawing is entered in — caller, pen, x and y — so that the listing")
		add("stands on its own. Use it when drawing this one by itself; ignore it")
		add("when drawing it as part of %03d, which sets that state as it runs.", from.parent)
	}

	add("%s", line)

	fmt.Fprintf(&sb, "\nIMAGE %03d\n", d.number)

	if from, ok := start.get(); ok {
		fmt.Fprintf(&sb, "INHERIT %03d,%d,%d,%d\n", from.parent, from.pen, from.x, from.y)
	}

	sb.WriteString("\n")

	for _, o := range d.orders {
		sb.WriteString(o.text())
		sb.WriteString("\n")
	}

	return sb.String()
}

// cpcInkList writes the four inks of a record as the firmware numbers them,
// saying so where one of them asks to be left alone.
func cpcInkList(inks [cpcPens]int) string {
	out := make([]string, 0, cpcPens)

	for _, ink := range inks {
		if ink >= cpcInks {
			out = append(out, "-")

			continue
		}

		out = append(out, fmt.Sprintf("%d", ink))
	}

	return strings.Join(out, ", ")
}

// cpcInherited is the state a called drawing starts from: whose call it came out
// of, and the pen as that caller had left it.
type cpcInherited struct {
	parent int
	x, y   int
	pen    int
	inks   [cpcPens]int
}

type cpcMaybeInherited struct {
	state cpcInherited
	found bool
}

func (m cpcMaybeInherited) get() (cpcInherited, bool) { return m.state, m.found }

func (v cpcVectors) inheritedFor(number int, callers map[int][]int) cpcMaybeInherited {
	state, found := v.inheritedBy(number, callers, 0)

	return cpcMaybeInherited{state: state, found: found}
}

// inheritedBy works out that state, or reports that there is none because
// nothing calls this drawing.
//
// The first caller is the one taken: where a piece is stamped into several
// scenes there is no single right answer, and the first is at least always the
// same one.
func (v cpcVectors) inheritedBy(number int, callers map[int][]int, depth int) (cpcInherited, bool) {
	if depth > cpcMaxCallDepth || len(callers[number]) == 0 {
		return cpcInherited{}, false
	}

	caller := callers[number][0]
	if caller == number {
		return cpcInherited{}, false
	}

	scratch := v.blankScreen(caller)

	if from, ok := v.inheritedBy(caller, callers, depth+1); ok {
		scratch.apply(from)
	}

	stop := &halt{at: number}
	v.trace(scratch, caller, 0, 0, 0, stop)

	if !stop.reached {
		return cpcInherited{}, false
	}

	return cpcInherited{
		parent: caller,
		x:      scratch.x,
		y:      scratch.y,
		pen:    scratch.pen,
		inks:   scratch.inks,
	}, true
}

// render traces one drawing of the set, returning the screen rather than the
// picture: a picture is for looking at and the screen is what a packager needs,
// and both come out of the same tracing.
func (v cpcVectors) render(number int, callers map[int][]int) *cpcScreen {
	s := v.renderWithPen(number, callers, 0)

	// A drawing whose record prepares nothing is entered in the state the last
	// one left, and the state we hand it is a caller's — which for a piece is
	// right and for a picture the adventure asks for by number is a guess. When
	// that guess draws the whole thing in the colour of the background, which
	// is what pen 0 on a screen of pen 0 means, there is nothing to look at and
	// the guess was the wrong one. The engine's own entry state is the other
	// answer, and it is the one a machine has when a PICTURE asks directly.
	if s.isBlank() {
		if lit := v.renderWithPen(number, callers, 1); !lit.isBlank() {
			return lit
		}
	}

	return s
}

// renderWithPen is render with the pen it is entered in said outright, for the
// drawings whose record prepares nothing and whose caller holds the pen the
// background already is. A pen of zero takes whatever the caller had.
func (v cpcVectors) renderWithPen(number int, callers map[int][]int, pen int) *cpcScreen {
	s := v.blankScreen(number)

	if from, ok := v.inheritedBy(number, callers, 0); ok {
		s.apply(from)
	}

	if pen > 0 {
		s.pen = pen
	}

	v.trace(s, number, 0, 0, 0, nil)

	return s
}
