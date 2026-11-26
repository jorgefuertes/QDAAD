// Package cgapic reads the PC edition's illustration archive.
//
// A .CGA is laid out like the PCW's .DAT — six bytes of header, 256 slots of
// ten, the pictures behind them — and packed like it too: up to four markers
// chosen per picture, each followed by how many times it repeats. The markers
// are the dither bytes, which is why they are the ones worth naming: 0xAA and
// 0x55 are a pixel on and a pixel off all the way across.
//
// The odd rows are written backwards, and that is part of the encoding rather
// than part of the format: writing a picture back and forth lets a run carry on
// across the end of a row instead of stopping there. So it applies to the
// pictures that are run-encoded and to no others.
//
// Which is a distinction worth stating, because getting it wrong either way
// produces a coherent picture rather than a broken one — flat fills where there
// should be stipple, which reads as a different drawing style and not as
// damage. Two things pin it. The one illustration in the two archives that is
// stored uncompressed, picture 106 of part two, comes byte for byte out of the
// decompiler's own renderer unreversed. And reversing the odd rows of a picture
// that was not written that way leaves every pair of neighbouring rows
// disagreeing where they should nearly match: measured over both archives, the
// uncompressed one is a third more coherent unreversed and every encoded one is
// more coherent reversed.
//
// It is shared by the two scripts that convert those drawings for a machine La
// Aventura Original was never released on: pcwpic, which dithers them down to
// the PCW's one bit, and pcpcx, which puts them back on a PC in the mode a
// version 3 interpreter draws. Shared rather than copied because the archive is
// one format and a second reader of it would be a second place to get the
// run encoding wrong.
//
// It is a tool of this repository and not a part of the compiler: nothing in
// internal/ knows it exists.
package cgapic

const (
	archiveHeader = 6
	archiveSlots  = 256
	slotSize      = 10
	pictureHeader = 6
	compressed    = 0x8000
	maxMarkers    = 4

	// Pixels is four to the byte: CGA mode 4 is two bits each.
	PixelsPerByte = 4
	// Colours is how many a mode 4 picture has.
	Colours = 4
)

// Slots is how many pictures an archive can name.
const Slots = archiveSlots

// Picture reads one out of the archive, as one four-colour pixel per byte.
//
// ok is false where the slot is empty, which most of the 256 are.
func Picture(data []byte, number int) (pixels []byte, width, height int, ok bool) {
	at := archiveHeader + number*slotSize
	if at+4 > len(data) {
		return nil, 0, 0, false
	}

	offset := int(data[at]) | int(data[at+1])<<8 | int(data[at+2])<<16 | int(data[at+3])<<24
	if offset == 0 || offset+pictureHeader > len(data) {
		return nil, 0, 0, false
	}

	word := func(i int) int { return int(data[offset+i]) | int(data[offset+i+1])<<8 }

	flags := word(0)
	width, height = flags&0x0FFF, word(2)
	body := data[offset+pictureHeader:]

	if length := word(4); length < len(body) {
		body = body[:length]
	}

	stride := width / PixelsPerByte

	if flags&compressed != 0 {
		body = expand(body)

		if len(body) < stride*height {
			return nil, 0, 0, false
		}

		body = unweave(body, stride, height)
	}

	if len(body) < stride*height {
		return nil, 0, 0, false
	}

	return indices(body, width, height), width, height, true
}

// Packed reads one out of the archive as the bytes it is stored in, four pixels
// to each, with the run encoding undone and nothing else done to them.
//
// It is what pcwpic wants: it reads the two bits of a pixel out of the byte
// itself while it dithers.
func Packed(data []byte, number int) (body []byte, width, height int, ok bool) {
	at := archiveHeader + number*slotSize
	if at+4 > len(data) {
		return nil, 0, 0, false
	}

	offset := int(data[at]) | int(data[at+1])<<8 | int(data[at+2])<<16 | int(data[at+3])<<24
	if offset == 0 || offset+pictureHeader > len(data) {
		return nil, 0, 0, false
	}

	word := func(i int) int { return int(data[offset+i]) | int(data[offset+i+1])<<8 }

	flags := word(0)
	width, height = flags&0x0FFF, word(2)
	rest := data[offset+pictureHeader:]

	if length := word(4); length < len(rest) {
		rest = rest[:length]
	}

	stride := width / PixelsPerByte

	if flags&compressed != 0 {
		rest = expand(rest)

		if len(rest) < stride*height {
			return nil, 0, 0, false
		}

		rest = unweave(rest, stride, height)
	}

	if len(rest) < stride*height {
		return nil, 0, 0, false
	}

	return rest, width, height, true
}

// indices unpacks the two bits of each pixel into a byte of its own.
func indices(body []byte, width, height int) []byte {
	stride := width / PixelsPerByte
	out := make([]byte, width*height)

	for y := range height {
		for x := range width {
			b := body[y*stride+x/PixelsPerByte]
			out[y*width+x] = (b >> (6 - 2*(x%PixelsPerByte))) & 0x03
		}
	}

	return out
}

// expand undoes the run-length scheme: five bytes of preamble naming up to
// four markers, and a marker is followed by how many times it repeats, with
// zero meaning 256.
func expand(body []byte) []byte {
	if len(body) < 1+maxMarkers {
		return body
	}

	marker := [256]bool{}
	for _, m := range body[1 : 1+int(body[0])] {
		marker[m] = true
	}

	out := make([]byte, 0, len(body)*2)

	for i := 1 + maxMarkers; i < len(body); {
		b := body[i]
		i++

		if !marker[b] || i >= len(body) {
			out = append(out, b)

			continue
		}

		run := int(body[i])
		if run == 0 {
			run = 256
		}

		i++

		for range run {
			out = append(out, b)
		}
	}

	return out
}

// unweave puts the rows of an encoded picture the right way round: the odd ones
// are stored backwards so that a run can carry across the end of a row.
func unweave(body []byte, stride, height int) []byte {
	out := make([]byte, 0, stride*height)

	for y := range height {
		row := body[y*stride : (y+1)*stride]
		if y%2 == 0 {
			out = append(out, row...)

			continue
		}

		for i := len(row) - 1; i >= 0; i-- {
			out = append(out, row[i])
		}
	}

	return out
}
