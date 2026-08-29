package decompiler

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
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
	name string
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
		name := fmt.Sprintf("%03d.bin", i)

		fmt.Fprintf(&index, "; %-5d  %-8d %5d  %-9s %v\n",
			i, at, len(body), geometry(body), users[at])

		if p := palettes[at]; p != "" {
			fmt.Fprintf(&index, ";        palette %s\n", p)
		}

		cut := piece{name: name, data: body}
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

	header := fmt.Sprintf(
		"; Illustrations taken out of a PC archive, video mode %#02x, %d of them.\n"+
			"; The bodies are written as they lie: how the drawing inside one is\n"+
			"; encoded is not known yet. It is not a bitmap and it is not compressed —\n"+
			"; the bytes run to about four and a half bits of entropy over some seventy\n"+
			"; distinct values — so it reads as a stream of drawing orders.\n"+
			"; Each body opens with six bytes whose last two hold its length.",
		le16(2), le16(4))

	return cut(data, slots, header, func(body []byte) string {
		if len(body) < pcImageHeader {
			return "?"
		}

		// The length the body declares for itself, which is its size less the
		// six bytes of its own header.
		return fmt.Sprintf("len %d", int(body[4])|int(body[5])<<8)
	}, nil)
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
