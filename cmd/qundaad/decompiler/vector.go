package decompiler

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strconv"
	"strings"
)

// The eight-bit Spanish adventures did not ship bitmaps. A drawing there is not
// a picture but a program: a strip of orders the interpreter runs to trace it,
// with a flat table of pointers saying where each one starts.
//
// The block holding them always ends at the top of memory, and the pointers the
// engine reads live in its own tail. That is what makes it findable without
// knowing anything about the loader, and without the drawings having to travel
// with the interpreter: some games put both in one block and others do not.
//
// The whole format is worked out in docs/DAAD_v2_v3_Research/09-graficos.md,
// section 9.

// Where the engine reads its tables from. All three sit at the top of memory,
// in the tail of the block that carries the drawings.
const (
	drawingTableAt = 0xFFF1 // base of the table of drawings
	// propertiesAt is one past the last entry of that table, so it gives the
	// count — and it is also the base of a second table, five bytes to the
	// drawing, holding what the interpreter sets up before running one.
	propertiesAt    = 0xFFF3
	propertiesEndAt = 0xFFF5 // one past the last of those records
	udgTableAt      = 0xFFF7 // base of the table of user-defined graphics

	// propertySize is the five bytes 0x7C18 indexes by drawing number.
	propertySize = 5

	topOfMemory = 0x10000

	// How far back from the end of a block its first pointer sits, which is as
	// early as a block can end and still hold one.
	tailSize = topOfMemory - drawingTableAt
)

// The opcode is the low three bits of the order byte.
const (
	opPlot = iota
	opDraw
	opFill
	opCall
	opUDG
	opPaper
	opInk
	opRet
)

// attrFillOrder is the one order byte that is a special case rather than a set
// of flags: it paints a rectangle of attributes instead of filling with a
// pattern, and it is five bytes long.
const attrFillOrder = 0x12

// The canvas. The y wraps modulo its height and runs upwards, as the ROM's PLOT
// does, so no legitimate PLOT ever names a row above the last one. A cell is
// addressed over the whole screen and not just the drawing window, which is
// what the ROM's AT accepts.
const (
	canvasHeight = 176
	charRows     = 24
	// The interpreter prints in a font six pixels wide, not eight: 0x7B1A adds
	// three bytes for every four columns, and 0x770F holds the count to 42.
	// Attributes are still eight pixels to the cell, because the hardware says
	// so, which is why an ATTR counts in 32s and a UDG in 42s.
	charColumns     = 42
	charColumnWidth = 6
)

// dialect is the set of order lengths one build of the engine uses. They are
// not the same in every game: La Aventura Original and El Jabato never set bit
// 5 of a DRAW and their operand routine reads two whole bytes, while Cozumel,
// which is later, does have the two-byte short form.
//
// Rather than pick one by the game, every dialect is tried and the one that
// decodes the whole table is the one used. A length that is wrong by a byte
// desynchronises the strip, so a dialect that reads every drawing to its RET
// exactly where the next begins is the dialect that build was written in.
type dialect struct {
	call      int  // bytes a CALL occupies, counting the one the RET adds
	drawShort bool // bit 5 makes DRAW a two-byte order, a nibble to the axis
	fillSizes bool // FILL has three sizes rather than always three bytes
}

// The order matters only for determinism: where more than one dialect decodes a
// table they do so identically, because they differ only in bits that table
// never sets.
var dialects = []dialect{
	{call: 2, drawShort: true, fillSizes: true},
	{call: 2, drawShort: false, fillSizes: true},
	{call: 2, drawShort: true, fillSizes: false},
	{call: 2, drawShort: false, fillSizes: false},
	{call: 3, drawShort: true, fillSizes: true},
	{call: 3, drawShort: false, fillSizes: true},
	{call: 3, drawShort: true, fillSizes: false},
	{call: 3, drawShort: false, fillSizes: false},
}

// describe says what a dialect is, for the header of a listing.
func (d dialect) describe() string {
	draw := "3 bytes"
	if d.drawShort {
		draw = "3 bytes, or 2 with bit 5 set"
	}

	fill := "3 bytes"
	if d.fillSizes {
		fill = "3, 4 or 5 bytes"
	}

	return fmt.Sprintf("CALL %d bytes; DRAW %s; FILL %s", d.call, draw, fill)
}

// order is one decoded drawing order.
type order struct {
	at   int    // where it lies in the payload
	code byte   // the order byte, as it lies
	size int    // how many bytes it occupies
	name string // the mnemonic it is written as
	args []int  // its operands, in the order they are written
	note string // what the mnemonic does not carry, or empty
}

// text is the order as it goes into a listing.
func (o order) text() string { return orderLine(o.name, o.args) }

// drawing is one entry of the table: a strip of orders, and where it came from.
type drawing struct {
	number int
	addr   int // address in the loaded block
	at     int // offset in the payload
	size   int // bytes, up to the start of the next drawing
	orders []order
}

// vectors is the set of drawings of one loaded block.
type vectors struct {
	load     int // address = payload offset + load
	table    int // address of the table of drawings
	udg      int // address of the table of UDGs
	graphics []byte
	// properties is the five-byte record per drawing that 0x7BBC reads before
	// running one: its first byte, masked with 0x3F, is the attribute the
	// interpreter puts into ATTR-P and ATTR-T. propertyTable is where it lies.
	properties    []byte
	propertyTable int
	dialect       dialect
	drawings      []drawing
}

// udgSize is a user-defined graphic: eight rows of eight pixels, a byte to the
// row, which is how the Spectrum has kept them since 1982.
const udgSize = 8

// count says how many graphics the table holds.
func (v vectors) udgCount() int { return len(v.graphics) / udgSize }

// graphic returns the eight rows of one entry, or nil past the end.
func (v vectors) graphic(n int) []byte {
	at := n * udgSize
	if n < 0 || at+udgSize > len(v.graphics) {
		return nil
	}

	return v.graphics[at : at+udgSize]
}

// startsWith is the ink and paper a drawing is entered with, and whether the
// table said anything at all.
//
// The interpreter reads the drawing's record at 0x7BBC, masks its first byte
// with 0x3F and writes that to both ATTR-P and ATTR-T (0x7BE4, 0x7BE7). The
// mask is what leaves bright and flash off. It then clears COORDS (0x7BEB), so
// the pen starts at the origin.
func (v vectors) startsWith(number int) (ink, paper byte, ok bool) {
	at := number * propertySize
	if number < 0 || at >= len(v.properties) {
		return 0, 0, false
	}

	attribute := v.properties[at] & 0x3F

	return attribute & 7, (attribute >> 3) & 7, true
}

// windowRows is how many character rows the drawing declares for itself in its
// record (§9.6). Not every drawing uses the same: in La Aventura Original 26 of
// the 49 ask for 14 rows and 23 for 12, and part 2 has three that take the
// whole screen.
//
// It is not a clipping rectangle, and the pictures are not cut to it. The title
// screen of the first part says 14 rows and paints all 22 — its frame runs from
// screen row 24 to row 151, which the capture confirms — because the engine
// hands every coordinate to the ROM's PLOT and the ROM clips to the screen and
// nothing else. What the record governs is where the interpreter then puts the
// text, so this is reported in the listing and left out of the drawing.
func (v vectors) windowRows(number int) int {
	at := number*propertySize + 3
	if number < 0 || at >= len(v.properties) {
		return canvasHeight / cellSize
	}

	rows := int(v.properties[at])
	if rows <= 0 || rows > canvasHeight/cellSize {
		return canvasHeight / cellSize
	}

	return rows
}

// offsetOf turns an address of the loaded block into an offset in the payload.
func (v vectors) offsetOf(addr int) int { return addr - v.load }

// isEmpty says whether a drawing does nothing at all. Plenty of the table's
// entries hold a lone RET; they are worth counting but not worth a listing.
func (d drawing) isEmpty() bool {
	return len(d.orders) == 1 && d.orders[0].code&7 == opRet
}

// callers returns, for each drawing, the ones that invoke it, in the order they
// lie. A drawing nobody calls is a scene; one that is called is a piece meant
// to be stamped into others, and cannot be read on its own.
func (v vectors) callers() map[int][]int {
	found := map[int][]int{}

	for _, d := range v.drawings {
		for _, o := range d.orders {
			if o.name != "CALL" {
				continue
			}

			// Once is enough: what matters is who, not how many times.
			if to := o.args[0]; !slices.Contains(found[to], d.number) {
				found[to] = append(found[to], d.number)
			}
		}
	}

	return found
}

// dataAt is where the drawings themselves begin, which is what says which
// database a set belongs to when a payload carries several parts.
func (v vectors) dataAt() int {
	if len(v.drawings) == 0 {
		return 0
	}

	return v.drawings[0].at
}

// findVectors returns every set of drawings the data holds, in the order they
// lie. A multi-part adventure keeps one set per part.
//
// There is nothing to search for, because nothing marks a graphics block: what
// is known is only that it ends at the top of memory. So every offset is tried
// as that end, and the arrangement it implies is what accepts or rejects it. It
// sounds indiscriminate and is not — across the six Spectrum editions we have,
// only the real blocks get as far as being decoded.
func findVectors(data []byte) []vectors {
	var found []vectors

	for end := tailSize; end <= len(data); end++ {
		if v, ok := vectorsEndingAt(data, end); ok {
			found = append(found, v)
		}
	}

	return found
}

// attachVectors gives each set of drawings to the database it belongs to: the
// last one that starts before it.
//
// A part carries its database, then its engine, then its drawings, and only
// then does the next part begin, so where a set lies is enough to say whose it
// is. That beats pairing them off in order, which would put the drawings of the
// second part with the first if a part ever turned up without any.
func attachVectors(databases []database, sets []vectors) {
	for i := range sets {
		owner := -1

		for j := range databases {
			if databases[j].reader.Offset() <= sets[i].dataAt() {
				owner = j
			}
		}

		if owner >= 0 && databases[owner].vectors == nil {
			databases[owner].vectors = &sets[i]
		}
	}
}

// vectorsEndingAt reads the set of drawings of a block supposed to end at the
// given offset, and says whether that supposition holds.
//
// Nothing here is taken on trust. A run of filler bytes will happily read as a
// pair of pointers — that is how an earlier candidate for this block fooled us —
// so every step has to be confirmed against the next: the table of drawings has
// to be ordered and self-terminating, and every drawing has to run to its RET
// exactly where the next one starts.
func vectorsEndingAt(data []byte, end int) (vectors, bool) {
	// If this offset is the top of memory, every address of the block follows.
	v := vectors{load: topOfMemory - end}

	table, ok := v.wordAtAddress(data, drawingTableAt)
	if !ok {
		return vectors{}, false
	}

	tableEnd, ok := v.wordAtAddress(data, propertiesAt)
	if !ok {
		return vectors{}, false
	}

	udg, ok := v.wordAtAddress(data, udgTableAt)
	if !ok {
		return vectors{}, false
	}

	v.table, v.udg = table, udg

	pointers, ok := v.readTable(data, table, tableEnd)
	if !ok {
		return vectors{}, false
	}

	// The five-byte record per drawing, bounded by the pointer that follows it.
	// That the two agree to the byte across every edition we have is a good
	// part of why this arrangement can be trusted at all.
	if end, ok := v.wordAtAddress(data, propertiesEndAt); ok && end-tableEnd == propertySize*(len(pointers)-2) {
		if from, to := v.offsetOf(tableEnd), v.offsetOf(end); 0 <= from && from < to && to <= len(data) {
			v.properties, v.propertyTable = data[from:to], tableEnd
		}
	}

	// The graphics run from their base to where the block's own pointers begin,
	// which is the only bound there is: nothing declares how many there are.
	if from, to := v.offsetOf(udg), v.offsetOf(drawingTableAt); 0 <= from && from < to && to <= len(data) {
		v.graphics = data[from : from+(to-from)/udgSize*udgSize]
	}

	// Which build of the engine wrote this is not declared anywhere, so it is
	// settled by reading: the dialect that takes every drawing to its RET on the
	// byte the next one starts at is the one it was written in.
	for _, d := range dialects {
		if drawings, ok := v.readDrawings(data, pointers, d); ok {
			v.dialect, v.drawings = d, drawings

			return v, true
		}
	}

	return vectors{}, false
}

// readDrawings decodes the whole table in one dialect, and fails as soon as any
// drawing does. Nothing partial is worth keeping: a table that reads most of the
// way and then falls over was read in the wrong dialect, and the drawings it did
// produce are as likely to be wrong as the one that failed.
func (v vectors) readDrawings(data []byte, pointers []int, d dialect) ([]drawing, bool) {
	// The last two entries are not drawings: one bounds the last drawing and the
	// other terminates the table.
	count := len(pointers) - 2

	drawings := make([]drawing, 0, count)

	for i := range count {
		one, ok := v.readDrawing(data, i, pointers[i], pointers[i+1], count, d)
		if !ok {
			return nil, false
		}

		drawings = append(drawings, one)
	}

	return drawings, true
}

// readTable reads the pointers of the table of drawings and checks the shape
// the engine's own arrangement demands: distinct and strictly increasing, the
// last but one pointing at the table itself to bound the last drawing, and a
// sentinel closing it.
func (v vectors) readTable(data []byte, table, end int) ([]int, bool) {
	if end <= table || (end-table)%2 != 0 {
		return nil, false
	}

	entries := (end - table) / 2

	// Three is the least that can hold a drawing, its bound and the sentinel.
	const maxEntries = 4096
	if entries < 3 || entries > maxEntries {
		return nil, false
	}

	// The two entries that close the table, checked before anything is read in
	// bulk. Every offset of the data is tried as the end of a block, so all but
	// a handful of calls are rejected here and it is worth their being cheap.
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

	if pointers[len(pointers)-1] != 0xFFFF {
		return nil, false
	}

	// Everything but the sentinel has to climb, and stop at the table.
	real := pointers[:len(pointers)-1]
	for i := 1; i < len(real); i++ {
		if real[i] <= real[i-1] {
			return nil, false
		}
	}

	if real[len(real)-1] != table {
		return nil, false
	}

	if v.offsetOf(real[0]) < 0 {
		return nil, false
	}

	return pointers, true
}

// readDrawing decodes one drawing and insists it ends where the next begins.
// An order length that is wrong by one byte desynchronises the strip and shows
// up here, which is what makes this check worth making.
func (v vectors) readDrawing(data []byte, number, from, to, count int, dl dialect) (drawing, bool) {
	at, stop := v.offsetOf(from), v.offsetOf(to)
	if at < 0 || stop > len(data) || stop <= at {
		return drawing{}, false
	}

	d := drawing{number: number, addr: from, at: at, size: stop - at}

	for o := at; ; {
		if o >= stop {
			return drawing{}, false
		}

		code := data[o]

		size := orderLength(code, dl)
		if o+size > stop {
			return drawing{}, false
		}

		name, args, note, ok := decodeOrder(data, o, count, dl)
		if !ok {
			return drawing{}, false
		}

		d.orders = append(d.orders,
			order{at: o, code: code, size: size, name: name, args: args, note: note})

		o += size

		if code&7 == opRet {
			// A drawing that stops short of the next one leaves bytes nothing
			// reaches, which means the strip was misread.
			if o != stop {
				return drawing{}, false
			}

			return d, true
		}
	}
}

// orderLength is how many bytes an order occupies. Only the three the dialect
// names vary; the rest are the same in every build. CALL is the odd one: its
// handler jumps straight to the dispatcher without touching IX, so what it
// occupies is what the matching RET adds on the way back.
func orderLength(code byte, d dialect) int {
	switch code & 7 {
	case opPaper, opInk, opRet:
		return 1
	case opCall:
		return d.call
	case opUDG:
		return 4
	case opFill:
		switch {
		case !d.fillSizes:
			return 3
		case code == attrFillOrder:
			return 5
		case code&0x20 != 0:
			return 4 // the pattern is a UDG, named by the fourth byte
		default:
			return 3
		}
	case opDraw:
		if d.drawShort && code&0x20 != 0 {
			return 2
		}

		return 3
	default:
		return 3
	}
}

// traceModes names what the top bits of PLOT and DRAW do to the pen. The engine
// folds them into P-FLAG, so they are OVER and INVERSE as BASIC means them.
var traceModes = [...]string{"", "over", "inverse", "over inverse"}

// orderText writes one order as the line it becomes, and whatever the mnemonic
// cannot carry as a note.
func orderText(data []byte, at, count int, d dialect) (text, note string, ok bool) {
	name, args, note, ok := decodeOrder(data, at, count, d)
	if !ok {
		return "", "", false
	}

	return orderLine(name, args), note, true
}

// orderLine is the mnemonic and its operands separated by commas, which is the
// whole of the format's grammar.
func orderLine(name string, args []int) string {
	if len(args) == 0 {
		return name
	}

	operands := make([]string, len(args))
	for i, a := range args {
		operands[i] = strconv.Itoa(a)
	}

	return name + " " + strings.Join(operands, ",")
}

// decodeOrder reads one order into the mnemonic and operands it is written as.
// Drawing a listing and writing one go through here alike, so a picture can
// never disagree with the text beside it.
//
// It fails on a coordinate that cannot be right, which is what tells a drawing
// from a stretch of bytes that decodes without ever falling over.
func decodeOrder(data []byte, at, count int, d dialect) (name string, args []int, note string, ok bool) {
	code := data[at]

	switch code & 7 {
	case opPlot:
		x, y := int(data[at+1]), int(data[at+2])
		if y >= canvasHeight {
			return "", nil, "", false
		}

		return "PLOT", []int{x, y}, traceModes[(code>>3)&3], true

	case opDraw:
		// The short form packs a nibble to the axis into a single byte, so each
		// increment runs from 0 to 15 — and the third byte is not there to read.
		var dx, dy int

		if d.drawShort && code&0x20 != 0 {
			dx, dy = signed(code, int(data[at+1])>>4, int(data[at+1])&0x0F)
		} else {
			dx, dy = relative(data, at, code)
		}

		// Both bits set is not a mode but a MOVE: the order writes the
		// coordinates without tracing, which is how the pen is lifted.
		if (code>>3)&3 == 3 {
			return "MOVE", []int{dx, dy}, "", true
		}

		return "LINE", []int{dx, dy}, traceModes[(code>>3)&3], true

	case opFill:
		return decodeFill(data, at, code, d)

	case opCall:
		n := int(data[at+1])
		if n >= count {
			return "", nil, "", false
		}

		// Scale only ever reduces: 0 is natural size and 1 to 7 are eighths.
		return "CALL", []int{n, int((code >> 3) & 7), int((code >> 6) & 3)}, "", true

	case opUDG:
		col, row := int(data[at+2]), int(data[at+3])
		if col >= charColumns || row >= charRows {
			return "", nil, "", false
		}

		return "UDG", []int{int(data[at+1]), col, row}, "", true

	case opPaper:
		// The top bit picks a different BASIC control code altogether.
		if code&0x80 != 0 {
			return "BRIGHT", []int{int((code >> 3) & 0x0F)}, "", true
		}

		return "PAPER", []int{int((code >> 3) & 0x0F)}, "", true

	case opInk:
		if code&0x80 != 0 {
			return "FLASH", []int{int((code >> 3) & 0x0F)}, "", true
		}

		return "INK", []int{int((code >> 3) & 0x0F)}, "", true

	default:
		return "RET", nil, "", true
	}
}

// fillText writes the three shapes of FILL. The odd one out is the exact order
// byte 0x12, which is not a fill at all: it paints a rectangle straight into the
// attribute file, so it is given a name of its own.
// Its four operands are written as they lie and not range-checked. In the build
// of La Aventura Original the handler at 0x7D04 reads them as a count of rows
// (0x7D17), a count of columns (0x7D1A), a starting column (0x7D0A) and a
// starting row (0x7D06) — the counts being one less than what gets painted. A
// screen-clearing routine there reads ATTR 13,31,0,0, which is 14 rows by 32
// columns and exactly the drawing window.
//
// Cozumel puts values there that cannot mean that, and its engine sits at
// different addresses, so what they mean in every build is not something we
// know. The length is another matter: five bytes is confirmed by the whole
// table chaining on it.
func decodeFill(data []byte, at int, code byte, d dialect) (name string, args []int, note string, ok bool) {
	dx, dy := relative(data, at, code)

	if !d.fillSizes {
		return "FILL", []int{dx, dy}, "", true
	}

	if code == attrFillOrder {
		return "ATTR", []int{
			int(data[at+1]), int(data[at+2]), int(data[at+3]), int(data[at+4]),
		}, "", true
	}

	// Bit 5 names a graphic to fill with; bit 4 turns the pattern over.
	if code&0x20 != 0 {
		if code&0x10 != 0 {
			note = "inverse"
		}

		return "FILL", []int{dx, dy, int(data[at+3])}, note, true
	}

	return "FILL", []int{dx, dy}, "", true
}

// relative reads the two operands of a relative order.
func relative(data []byte, at int, code byte) (dx, dy int) {
	return signed(code, int(data[at+1]), int(data[at+2]))
}

// signed applies the signs of an order byte to a pair of magnitudes. They are
// not two's complement: they are the top two bits of the order byte, and the
// operand is always an unsigned magnitude.
func signed(code byte, dx, dy int) (int, int) {
	if code&0x40 != 0 {
		dx = -dx
	}

	if code&0x80 != 0 {
		dy = -dy
	}

	return dx, dy
}

// wordAt reads a little-endian word, if there is one to read.
func wordAt(data []byte, at int) (int, bool) {
	if at < 0 || at+1 >= len(data) {
		return 0, false
	}

	return int(data[at]) | int(data[at+1])<<8, true
}

// wordAtAddress reads a word given where it sits in the loaded block.
func (v vectors) wordAtAddress(data []byte, addr int) (int, bool) {
	return wordAt(data, v.offsetOf(addr))
}

// writeVectors writes one .plot file per entry of the table, numbered as the
// table numbers them so that a CALL names the file it invokes.
//
// The entries holding nothing but a RET get no file. They are real entries and
// the table counts them, but a listing of one order and a picture of nothing
// are no use to anybody, and the gaps they leave in the numbering say plainly
// enough that the entry was empty.
func writeVectors(v vectors, gfx string, opts Options) (int, error) {
	if len(v.drawings) == 0 {
		return 0, nil
	}

	callers := v.callers()
	written := 0

	for _, d := range v.drawings {
		if d.isEmpty() {
			continue
		}

		start := v.inheritedFor(d.number, callers)

		body := plotFile(v, d, callers[d.number], start)
		if err := writeInto(gfx, listingsDir, numbered(d.number, ".plot"), []byte(body)); err != nil {
			return 0, err
		}

		// The listing drawn back into a picture, at the size the machine had
		// it. An enlargement goes beside it when one was asked for, under a
		// name saying so, rather than in place of it.
		picture := v.draw(d.number, callers)
		if err := writePNGInto(gfx, picturesDir, numbered(d.number, ".png"), picture); err != nil {
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

	if err := writeGraphicsTable(v, gfx); err != nil {
		return 0, err
	}

	return written, nil
}

// writeGraphicsTable writes the user-defined graphics: the table as it lies,
// and a sheet of all of them so they can be looked at.
//
// These are what a FILL paints with, so most of them are shading patterns —
// a checkerboard, scattered dots — rather than pictures. They go with the
// character sets, being one: eight bytes to the glyph, laid out the same way,
// and the table kept beside its sheet rather than off among the originals.
func writeGraphicsTable(v vectors, gfx string) error {
	if len(v.graphics) == 0 {
		return nil
	}

	if err := writeInto(gfx, fontsDir, "udg.bin", v.graphics); err != nil {
		return err
	}

	// And the same thing cut to the 2048 bytes a character set is, so that it
	// can be handed to the compiler as one.
	//
	// udg.bin is a dump of the memory between two addresses the interpreter
	// holds, so it is a few bytes longer than the font: whatever came after it
	// is in there too. That is the right thing for a dump and the wrong thing
	// for a file meant to be loaded, which has to be exactly 256 glyphs.
	//
	// It is worth having. The font an adventure of 1989 shipped is not the one
	// a version 3 interpreter loads, and the difference shows: AD8x6.CHR has a
	// letter where this has the blank that a forced space needs, and italics
	// where this has the little drawings that go beside an object's name.
	const charsetSize = 256 * 8

	if len(v.graphics) >= charsetSize {
		if err := writeInto(gfx, fontsDir, "udg.chr", v.graphics[:charsetSize]); err != nil {
			return err
		}
	}

	return writePNGInto(gfx, fontsDir, "udg.png", udgAtlas(v))
}

// udgAtlas lays the graphics out in a grid, sixteen across, the way the
// character sets are drawn. A one-pixel gutter keeps one from running into the
// next, which matters when so many of them are regular patterns.
func udgAtlas(v vectors) image.Image {
	const across = 16

	var (
		ink    = color.NRGBA{0x00, 0x00, 0x00, 0xFF}
		ground = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
		gutter = color.NRGBA{0xC0, 0xC0, 0xC8, 0xFF}
	)

	count := v.udgCount()
	down := (count + across - 1) / across
	step := udgSize + 1

	img := image.NewNRGBA(image.Rect(0, 0, across*step+1, down*step+1))

	for at := 0; at < len(img.Pix); at += 4 {
		img.Pix[at], img.Pix[at+1] = gutter.R, gutter.G
		img.Pix[at+2], img.Pix[at+3] = gutter.B, 0xFF
	}

	for n := range count {
		left := (n%across)*step + 1
		top := (n/across)*step + 1
		rows := v.graphic(n)

		for row := range udgSize {
			for column := range udgSize {
				shade := ground
				if rows[row]&(0x80>>column) != 0 {
					shade = ink
				}

				img.SetNRGBA(left+column, top+row, shade)
			}
		}
	}

	return img
}

// plotFile writes one drawing: a header saying what was read and from where,
// then a line an order.
func plotFile(v vectors, d drawing, callers []int, start maybeInherited) string {
	var sb strings.Builder

	line := strings.Repeat("-", 74)
	add := func(f string, a ...any) { fmt.Fprintf(&sb, "; "+f+"\n", a...) }

	add("%s", line)
	add("Decompiled by QUnDAAD - vector drawing %d of %d", d.number, len(v.drawings))
	add("Queru's DAAD decompiler v%s", VERSION)
	add("%s", line)
	add("Read from      : payload %#05x..%#05x, %d bytes", d.at, d.at+d.size, d.size)
	add("Laid at        : %#04x   (block loaded so that %#04x is the top of memory)", d.addr, topOfMemory)
	add("Table entry    : %d of %d, at %#04x", d.number, len(v.drawings), v.table+2*d.number)
	add("Orders         : %d", len(d.orders))
	add("Order lengths  : %s   (deduced, not declared)", v.dialect.describe())

	// What the interpreter puts on the screen before running this drawing. It
	// is not a guess: 0x7BBC reads it from the drawing's own record.
	if ink, paper, ok := v.startsWith(d.number); ok {
		add("Entered with   : INK %d, PAPER %d at 0,0   (record %d of the table at %#04x)",
			ink, paper, d.number, v.propertyTable)
		add("Declared window: %d rows   (what the record asks for, not a limit: the engine "+
			"does not clip)", v.windowRows(d.number))
	}

	// Naming who calls this says what it is. A drawing nobody calls is a scene
	// and stands on its own; one that is called is a piece, and its coordinates
	// and colours only mean anything carried on from the first of these.
	if len(callers) > 0 {
		names := make([]string, len(callers))
		for i, c := range callers {
			names[i] = fmt.Sprintf("%03d", c)
		}

		add("CALLED FROM    : %s", strings.Join(names, ", "))
	}

	add("%s", line)
	add("IMAGE names the drawing, which is what a CALL elsewhere refers to.")
	add("PLOT is absolute. LINE, MOVE and FILL are relative, as the engine")
	add("stores them, each starting where the last one ended. The canvas is")
	add("256x%d and its y runs upwards.", canvasHeight)
	add("CALL takes a drawing number, a scale (0 natural, 1-7 eighths) and a")
	add("mirror (bit 0 flips x, bit 1 flips y).")

	if from, ok := start.get(); ok {
		add("INHERIT is not an order of the engine's. It records the state this")
		add("drawing is entered in — caller, ink, paper, bright, x and y — so that")
		add("the listing stands on its own. Use it when drawing this one by")
		add("itself; ignore it when drawing it as part of %03d, which sets that", from.parent)
		add("state as it runs.")
	}

	add("%s", line)

	// The listing opens by naming itself. Nothing in the data says this — the
	// number is the drawing's place in the table — but it is what a CALL
	// elsewhere refers to, and it is what makes a file stand on its own once it
	// has been moved away from the directory that numbered it.
	fmt.Fprintf(&sb, "\nIMAGE %03d\n", d.number)

	if from, ok := start.get(); ok {
		fmt.Fprintf(&sb, "INHERIT %03d,%d,%d,%d,%d,%d\n",
			from.parent, from.ink, from.paper, boolToInt(from.bright), from.x, from.y)
	}

	sb.WriteString("\n")

	// Widest line, so the notes line up in a column of their own.
	width := 0

	for _, o := range d.orders {
		if o.note != "" && len(o.text()) > width {
			width = len(o.text())
		}
	}

	for _, o := range d.orders {
		if o.note == "" {
			sb.WriteString(o.text())
			sb.WriteString("\n")

			continue
		}

		fmt.Fprintf(&sb, "%-*s  ; %s\n", width, o.text(), o.note)
	}

	return sb.String()
}
