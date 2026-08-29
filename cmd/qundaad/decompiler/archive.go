package decompiler

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// The archives of illustrations, which come in two families.
//
// Both are indexed by location rather than by picture. There is one slot for
// every location the interpreter allows, so slots repeat where two locations
// share an illustration and stand at zero where a location has none.
const (
	// The PC archives of the first two adventures, one per video mode: .CGA
	// for mode 4 and .EGA for mode 0Dh. Their drawings are not pictures at all
	// but a stream of drawing orders, still unread.
	pcArchiveHeader = 36
	pcArchiveSlots  = 253
	pcArchiveRecord = 10
	pcImageHeader   = 6

	// The first word of an illustration says, in its top bit, whether the
	// drawing is compressed. Where it is not, what follows is the picture as
	// it stands: one bit plane after another, whole.
	pcCompressed = 0x8000

	// The two modes the archives were built for, and how many planes each
	// takes: four colours for CGA mode 4, sixteen for EGA mode 0Dh.
	pcModeCGA = 0x04
	pcModeEGA = 0x0D

	// What a compressed picture carries before its bytes: the method, and the
	// four values the run lengths are spent on.
	pcFrequent     = 4
	pcPackedHeader = 1 + pcFrequent
	pcLongestRun   = 255

	// The .DAT archives, which hold real pictures. Every slot carries the
	// palette its picture is drawn in.
	daadArchiveSlots = 256
	daadPaletteAt    = 12
	daadImageHeader  = 6
)

// The two shapes a .DAT takes. They are the same idea built twice: the later
// adventures widened the slot by four bytes and changed how the compressed
// pixels are gathered, and nothing in the file says which of the two it is.
//
// Both numbers of each are written into the interpreter that reads it — the
// Atari one for Los Templos Sagrados multiplies by 48 and subtracts 12298 — so
// they are not guesses.
var daadLayouts = []archiveLayout{
	{header: 6, record: 44},
	{header: 10, record: 48, nibbles: true},
}

// archiveLayout is one shape of archive, in one byte order.
type archiveLayout struct {
	header int
	record int
	// nibbles marks the later scheme, which takes each pixel as a nibble of a
	// longword instead of a bit from each of four bytes.
	nibbles bool
	// little marks a PC archive. The same archives were built for both kinds of
	// machine, and only the byte order tells them apart.
	little bool
}

// firstPicture is where the table ends and the drawings begin.
func (l archiveLayout) firstPicture() int {
	return l.header + daadArchiveSlots*l.record
}

func (l archiveLayout) word(d []byte, at int) int {
	if l.little {
		return int(d[at]) | int(d[at+1])<<8
	}

	return int(d[at])<<8 | int(d[at+1])
}

func (l archiveLayout) long(d []byte, at int) int {
	if l.little {
		return l.word(d, at) | l.word(d, at+2)<<16
	}

	return l.word(d, at)<<16 | l.word(d, at+2)
}

// piece is one illustration cut out of an archive, kept exactly as it lies.
type piece struct {
	data []byte
	img  image.Image // nil where the drawing cannot be read yet
}

// splitArchive cuts an archive into its illustrations.
//
// The pieces come back in the order they lie in the file, which is not the
// order of the slots that point at them: the slots are locations, and several
// can share one picture.
func splitArchive(a asset) ([]piece, string, bool) {
	switch strings.ToUpper(filepath.Ext(a.name)) {
	case ".CGA", ".EGA":
		return pcArchive(a.data)
	case ".DAT":
		return daadArchive(a.data)
	}

	return nil, "", false
}

// slot is one entry of the table at the front of an archive.
type slot struct {
	location int
	offset   int
	palette  string // empty where the archive keeps no palette
	colours  []byte // the same, as it lies, for drawing
}

// cut turns the slots into pieces, working out the size of each illustration
// from where the next one starts.
func cut(data []byte, slots []slot, header string,
	geometry func([]byte) string, drawPiece func([]byte, []byte) (image.Image, bool),
) ([]piece, string, bool) {
	starts := map[int]bool{}
	for _, s := range slots {
		starts[s.offset] = true
	}

	if len(starts) == 0 {
		return nil, "", false
	}

	order := make([]int, 0, len(starts))
	for at := range starts {
		order = append(order, at)
	}

	sort.Ints(order)

	// Which locations point at each illustration.
	users := map[int][]int{}
	for _, s := range slots {
		users[s.offset] = append(users[s.offset], s.location)
	}

	palettes := map[int]string{}
	colours := map[int][]byte{}

	for _, s := range slots {
		if s.palette != "" {
			palettes[s.offset] = s.palette
			colours[s.offset] = s.colours
		}
	}

	var (
		pieces []piece
		index  strings.Builder
	)

	index.WriteString(header)
	index.WriteString("\n; image  offset    size  geometry   used by locations\n")

	for i, at := range order {
		end := len(data)
		if i+1 < len(order) {
			end = order[i+1]
		}

		if at >= end || end > len(data) {
			return nil, "", false
		}

		body := data[at:end]

		// Numbered from one, to match the files they are written as.
		fmt.Fprintf(&index, "; %-5d  %-8d %5d  %-9s %v\n",
			i+1, at, len(body), geometry(body), users[at])

		if p := palettes[at]; p != "" {
			fmt.Fprintf(&index, ";        palette %s\n", p)
		}

		cut := piece{data: body}
		if drawPiece != nil {
			if img, drawn := drawPiece(body, colours[at]); drawn {
				cut.img = img
			}
		}

		pieces = append(pieces, cut)
	}

	return pieces, index.String(), true
}

// pcArchive reads a .CGA or .EGA. Its slots carry no palette: the picture is
// drawn in the colours of the video mode, which the archive names in its
// header.
func pcArchive(data []byte) ([]piece, string, bool) {
	if len(data) < pcArchiveHeader+pcArchiveSlots*pcArchiveRecord {
		return nil, "", false
	}

	le16 := func(at int) int { return int(data[at]) | int(data[at+1])<<8 }
	le32 := func(at int) int { return le16(at) | le16(at+2)<<16 }

	var slots []slot

	for i := range pcArchiveSlots {
		at := pcArchiveHeader + i*pcArchiveRecord

		offset := le32(at)
		if offset == 0 || offset >= len(data) {
			continue
		}

		slots = append(slots, slot{location: i, offset: offset})
	}

	// How many bit planes a picture has, which the mode decides.
	planes := 4
	if le16(2) == pcModeCGA {
		planes = 2
	}

	header := fmt.Sprintf(
		"; Illustrations taken out of a PC archive, video mode %#02x, %d of them,\n"+
			"; %d bit planes each. Every body opens with three words: a flag and the\n"+
			"; width, the height, and the length.\n"+
			"; Where the top bit of the first word is clear the picture is stored as it\n"+
			"; stands and is drawn here. Where it is set the drawing is compressed by a\n"+
			"; scheme still unread, and only the bytes are written.",
		le16(2), le16(4), planes)

	return cut(data, slots, header,
		func(body []byte) string {
			if len(body) < pcImageHeader {
				return "?"
			}

			width, height := int(body[0])|int(body[1])<<8, int(body[2])|int(body[3])<<8

			return fmt.Sprintf("%dx%d", width&^pcCompressed, height)
		},
		func(body, _ []byte) (image.Image, bool) {
			return pcPicture(body, planes)
		})
}

// daadArchive reads a .DAT, the archive that holds real pictures.
//
// Which of the two shapes it is, and which byte order, is not written anywhere,
// so each is tried and the one that adds up is kept: an archive lays its first
// picture exactly where its table of slots ends, and only the right reading
// puts it there.
func daadArchive(data []byte) ([]piece, string, bool) {
	for _, layout := range daadLayouts {
		for _, little := range []bool{false, true} {
			layout.little = little

			if pieces, index, ok := daadArchiveAs(data, layout); ok {
				return pieces, index, true
			}
		}
	}

	return nil, "", false
}

func daadArchiveAs(data []byte, layout archiveLayout) ([]piece, string, bool) {
	if len(data) < layout.firstPicture() {
		return nil, "", false
	}

	var slots []slot

	for i := range daadArchiveSlots {
		at := layout.header + i*layout.record

		offset := layout.long(data, at)
		if offset == 0 || offset >= len(data) {
			continue
		}

		colours := make([]string, 0, degasColours)
		for c := range degasColours {
			colours = append(colours, fmt.Sprintf("%04x", layout.word(data, at+daadPaletteAt+c*2)))
		}

		slots = append(slots, slot{
			location: i,
			offset:   offset,
			palette:  strings.Join(colours, " "),
			colours:  data[at+daadPaletteAt : at+daadPaletteAt+2*degasColours],
		})
	}

	if len(slots) == 0 {
		return nil, "", false
	}

	earliest := slots[0].offset
	for _, s := range slots {
		earliest = min(earliest, s.offset)
	}

	if earliest != layout.firstPicture() {
		return nil, "", false
	}

	machine := "68000"
	if layout.little {
		machine = "PC"
	}

	header := fmt.Sprintf(
		"; Illustrations taken out of a %s archive: %d bit planes, %d of them.\n"+
			"; The bodies are compressed by the interpreter's own scheme, so they are\n"+
			"; written as they lie, with the drawing beside them. Each opens with three\n"+
			"; words: a flag and the width, the height, and the length; then the mask\n"+
			"; that says which colours carry a run length.\n"+
			"; The palettes below are four bits to a component.",
		machine, layout.word(data, 0), layout.word(data, 4))

	return cut(data, slots, header,
		func(body []byte) string {
			w, h, _, ok := unpackDAAD(body, layout)
			if !ok {
				return "?"
			}

			return fmt.Sprintf("%dx%d", w, h)
		},
		func(body, palette []byte) (image.Image, bool) {
			if len(palette) < 2*degasColours {
				return nil, false
			}

			return daadPicture(body, degasPalette(palette), layout)
		})
}

// The header on each 68000 illustration: three words holding a flag and the
// width, the height, and the mask that drives the run lengths.
const (
	daadCompressed = 0x8000
	daadGroupBytes = 4 // one byte of each bit plane, holding eight pixels
	daadGroupBits  = 8
)

// unpackDAAD undoes the compression of one illustration and returns it as four
// bit planes interleaved a byte at a time, which is how the interpreter builds
// it.
//
// The scheme is the interpreter's own and matches nothing standard, which is
// why it does not yield to trying known ones. A pixel is four bits, and the
// header carries a mask saying which of the sixteen colours are followed by a
// run length — so whether the next four bits are a repeat count or simply the
// next pixel depends on the colour just read.
//
// Where the pixels are gathered from is the one thing the two generations do
// differently, and it is what layout.nibbles decides. Read from the routines at
// 0x20aa of the Atari interpreter of La Aventura Original and 0x269a of the one
// for Los Templos Sagrados.
func unpackDAAD(body []byte, layout archiveLayout) (int, int, []byte, bool) {
	if len(body) < daadImageHeader+2 {
		return 0, 0, nil, false
	}

	first := layout.word(body, 0)
	if first&daadCompressed == 0 {
		return 0, 0, nil, false
	}

	width, height := first&^daadCompressed, layout.word(body, 2)

	// A picture no bigger than the screen it was drawn for. Without this a
	// header read out of something that is not one asks for a buffer of
	// hundreds of megabytes.
	if width == 0 || width > screenWidth || height == 0 || height > screenHeight {
		return 0, 0, nil, false
	}

	// The interpreter works the size out the same way: half the width, because
	// two pixels share a byte, times the height.
	size := width / 2 * height
	if size%daadGroupBytes != 0 {
		return 0, 0, nil, false
	}

	var (
		out    = make([]byte, size)
		src    = body[daadImageHeader+2:]
		read   func() (int, bool)
		at     int // where in the source
		bit    = 7 // which bit of it, for the older scheme
		buffer int // the longword being taken apart, for the later one
		left   int // and how many nibbles of it are still to come
		to     int // the group of four bytes being written
		into   = 7
	)

	if layout.nibbles {
		// A longword at a time, lowest nibble first.
		read = func() (int, bool) {
			if left == 0 {
				if at+4 > len(src) {
					return 0, false
				}

				buffer = layout.long(src, at)
				at += 4
				left = 8
			}

			value := buffer & 0x0F
			buffer >>= 4
			left--

			return value, true
		}
	} else {
		// One bit from each of four bytes, which are the four planes.
		read = func() (int, bool) {
			if at+daadGroupBytes > len(src) {
				return 0, false
			}

			value := 0
			for plane := daadGroupBytes - 1; plane >= 0; plane-- {
				value = value<<1 | int(src[at+plane]>>bit)&1
			}

			if bit--; bit < 0 {
				bit = 7
				at += daadGroupBytes
			}

			return value, true
		}
	}

	write := func(colour int) {
		for plane := range daadGroupBytes {
			if colour&(1<<plane) != 0 {
				out[to+plane] |= 1 << into
			}
		}

		if into--; into < 0 {
			into = 7
			to += daadGroupBytes
		}
	}

	for to < size {
		colour, more := read()
		if !more {
			break
		}

		write(colour)

		if layout.word(body, daadImageHeader)&(1<<colour) == 0 {
			continue
		}

		run, more := read()
		if !more {
			break
		}

		for ; run > 0 && to < size; run-- {
			write(colour)
		}
	}

	return width, height, out, true
}

// daadPicture draws an unpacked illustration in the palette its slot carries.
func daadPicture(body []byte, palette [degasColours]color.NRGBA, layout archiveLayout) (image.Image, bool) {
	width, height, planes, ok := unpackDAAD(body, layout)
	if !ok {
		return nil, false
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for y := range height {
		for x := range width {
			pixel := y*width + x
			base := pixel / daadGroupBits * daadGroupBytes
			bit := 7 - pixel%daadGroupBits

			colour := 0

			for plane := range daadGroupBytes {
				if planes[base+plane]&(1<<bit) != 0 {
					colour |= 1 << plane
				}
			}

			img.Set(x, y, palette[colour])
		}
	}

	return img, true
}

// pcPicture draws an illustration of a PC archive, if it is one of those stored
// as it stands.
//
// The picture is bit planes, each held whole and one after another rather than
// interleaved as the 68000 archives keep them. That is the layout EGA reads
// from its own memory, and it is what a full screen of Cozumel comes out as.
//
// Nothing is drawn for the compressed ones. Their scheme is neither of the two
// the 68000 interpreters use — both were tried against them — and reading it
// wants the drawing routine of the PC interpreter.
func pcPicture(body []byte, planes int) (image.Image, bool) {
	if len(body) < pcImageHeader {
		return nil, false
	}

	le16 := func(at int) int { return int(body[at]) | int(body[at+1])<<8 }

	first := le16(0)
	width, height := first&^pcCompressed, le16(2)

	if width == 0 || width > screenWidth || height == 0 || height > screenHeight {
		return nil, false
	}

	// Two bits to a pixel or four, which is the same as saying four colours or
	// sixteen.
	size := width * height * planes / 8

	pixels := body[pcImageHeader:]
	if first&pcCompressed != 0 {
		unpacked, ok := pcUnpack(pixels, size)
		if !ok {
			return nil, false
		}

		unserpentine(unpacked, rowBytes(width, planes))

		pixels = unpacked
	}

	if len(pixels) < size {
		return nil, false
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	if planes == 2 {
		drawCGAPicture(img, pixels, width, height)
	} else {
		drawEGAPicture(img, pixels, width, height)
	}

	return img, true
}

// drawCGAPicture reads a four-colour picture: four pixels to a byte, rows one
// after another.
func drawCGAPicture(img *image.NRGBA, pixels []byte, width, height int) {
	const perByte = 4

	bytesPerRow := width / perByte

	for y := range height {
		row := y * bytesPerRow

		for x := range width {
			// The leftmost pixel of a byte lives in its top bits.
			shift := 6 - 2*(x%perByte)
			img.Set(x, y, cgaPalette[(pixels[row+x/perByte]>>shift)&0x03])
		}
	}
}

// drawEGAPicture reads a sixteen-colour picture: four bit planes, each whole,
// one after another.
func drawEGAPicture(img *image.NRGBA, pixels []byte, width, height int) {
	const planes = 4

	perPlane := width * height / 8

	for y := range height {
		for x := range width {
			pixel := y*width + x
			at := pixel / 8
			bit := byte(7 - pixel%8)

			colour := 0

			for plane := range planes {
				if pixels[plane*perPlane+at]&(1<<bit) != 0 {
					colour |= 1 << plane
				}
			}

			img.Set(x, y, egaPalette[colour])
		}
	}
}

// pcUnpack undoes the compression of a PC illustration.
//
// The scheme is the plainest of the three DAAD uses, and the only one whose
// compressor survives: DMG.EXE, the graphics manager that built these archives,
// still ships with the 1991 release, and the routine at 0x3b4f of it does this.
//
// It counts how often each byte value occurs in the picture, keeps the four
// commonest, and then writes the picture out a byte at a time — except that
// when a byte is one of those four it is followed by a count of how many times
// it repeats. How many of the four are treated that way is the method, one to
// four; the compressor tries all four and keeps whichever came out smallest,
// which is why it varies from picture to picture.
//
// The picture says which method was used and which four values, in the five
// bytes before the stream.
func pcUnpack(packed []byte, size int) ([]byte, bool) {
	if len(packed) < pcPackedHeader {
		return nil, false
	}

	method := int(packed[0])
	if method < 1 || method > pcFrequent {
		return nil, false
	}

	frequent := packed[1:pcPackedHeader]
	stream := packed[pcPackedHeader:]

	out := make([]byte, 0, size)

	for at := 0; len(out) < size && at < len(stream); {
		value := stream[at]
		at++

		out = append(out, value)

		// Only the values the compressor chose carry a count after them.
		if !slices.Contains(frequent[:method], value) {
			continue
		}

		if at >= len(stream) {
			break
		}

		// The count takes in the byte already written.
		run := int(stream[at])
		at++

		for range min(run-1, size-len(out)) {
			out = append(out, value)
		}
	}

	if len(out) < size {
		return nil, false
	}

	return out, true
}

// rowBytes is how many bytes of the buffer one row takes, which is not the same
// on the two machines: a CGA picture packs both bits of a pixel into the same
// byte, so a row is a quarter of its width, while an EGA one keeps its planes
// apart and a row of one plane is an eighth.
func rowBytes(width, planes int) int {
	if planes == 2 {
		return width / 4
	}

	return width / 8
}

// unserpentine puts back the rows the compressor wrote backwards.
//
// Every other row is stored right to left. It is a trick worth the trouble: the
// last pixel of one row and the first of the next are neighbours on the screen,
// so laying the rows out in alternating directions makes the runs longer than a
// plain left-to-right scan would. Only compressed pictures carry it, the plain
// ones being written as they stand.
//
// It came out of comparing what we decode against what the game itself draws:
// the runs of an odd row read in exactly the reverse order of the real ones. On
// twelve screens of La Aventura Original, undoing it gives the picture back to
// the pixel.
func unserpentine(buffer []byte, row int) {
	if row <= 0 {
		return
	}

	for at := row; at+row <= len(buffer); at += 2 * row {
		slices.Reverse(buffer[at : at+row])
	}
}
