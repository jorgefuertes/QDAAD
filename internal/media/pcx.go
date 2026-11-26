package media

import "fmt"

// PCX, which is the only picture format on this side of the project that is not
// a machine's own.
//
// Every other picture here is a dump of some hardware's video memory — a
// Spectrum screen, a mode 1 Amstrad screen, a PCW bitmap — and carries its
// meaning in its layout. A PC drawing does not: DAAD's DOS tooling takes PCX
// files, because in 1991 that was what a paint program on a PC wrote, and the
// interpreter's own format is built from one.
//
// So this is a reader and not a writer, and it reads the one shape that matters
// here: 256 colours, one plane, eight bits to the pixel. The reference accepts
// fewer colours as well (SCRMAKER.PAS:595-665 handles 1, 2 and 4 bits and
// several planes), and for the PC target that would be no use anyway — the
// palette there is 256 entries shifted up by sixteen, and a sixteen-colour file
// has nothing to put in the other 240.
type PCX struct {
	Width, Height int
	// Pixels is one palette index per pixel, row by row, with the scanline
	// padding taken off.
	Pixels []byte
	// Palette is 256 entries of red, green and blue, as the file holds them:
	// 0 to 255 each. The VGA wants them 0 to 63, and dividing by four is the
	// reader's business and not ours — PCDAAD does it on the way in
	// (pic.pas:400).
	Palette [768]byte
}

// The fields this reads, by where they sit in the 128-byte header.
const (
	pcxManufacturer = 0  // 0x0A, and the only thing identifying the format
	pcxEncoding     = 2  // zero for raw, anything else for the run encoding
	pcxBitsPerPixel = 3  // per plane
	pcxWindow       = 4  // xmin, ymin, xmax, ymax, four words
	pcxPlanes       = 65 // colour planes
	pcxBytesPerLine = 66 // a word, and at least as wide as the picture needs
	pcxHeaderSize   = 128
	// pcxPaletteTail is where a 256-colour palette lives: the last 769 bytes
	// of the file, a marker and then the colours. It is at the end rather than
	// in the header because the header was fixed before anybody wanted 256 of
	// anything.
	pcxPaletteTail   = 769
	pcxPaletteMarker = 12
)

// ReadPCX reads a 256-colour PCX.
func ReadPCX(file []byte) (*PCX, error) {
	if len(file) < pcxHeaderSize+pcxPaletteTail {
		return nil, fmt.Errorf("a PCX is at least %d bytes and this one is %d",
			pcxHeaderSize+pcxPaletteTail, len(file))
	}

	if file[pcxManufacturer] != 0x0A {
		return nil, fmt.Errorf("this is not a PCX: it starts 0x%02X and a PCX starts 0x0A",
			file[pcxManufacturer])
	}

	bits, planes := int(file[pcxBitsPerPixel]), int(file[pcxPlanes])
	if bits != 8 || planes != 1 {
		return nil, fmt.Errorf(
			"a picture for the PC is 256 colours, which is 8 bits and one plane, "+
				"and this one is %d bits and %d planes", bits, planes)
	}

	word := func(at int) int { return int(file[at]) | int(file[at+1])<<8 }

	out := &PCX{
		Width:  word(pcxWindow+4) - word(pcxWindow) + 1,
		Height: word(pcxWindow+6) - word(pcxWindow+2) + 1,
	}

	if out.Width <= 0 || out.Height <= 0 {
		return nil, fmt.Errorf("this PCX says it is %dx%d", out.Width, out.Height)
	}

	// A scanline is padded to a whole number of bytes and may be padded
	// further, so it is read as what the header says and cut to the width
	// afterwards.
	stride := word(pcxBytesPerLine)
	if stride*8 < out.Width*bits {
		return nil, fmt.Errorf("this PCX says %d bytes a line, which will not hold %d pixels",
			stride, out.Width)
	}

	if file[len(file)-pcxPaletteTail] != pcxPaletteMarker {
		return nil, fmt.Errorf(
			"this PCX has no 256-colour palette: the byte before the last 768 is 0x%02X and not 0x%02X",
			file[len(file)-pcxPaletteTail], pcxPaletteMarker)
	}

	copy(out.Palette[:], file[len(file)-768:])

	scans, err := pcxUnpack(file[pcxHeaderSize:len(file)-pcxPaletteTail],
		stride*out.Height, file[pcxEncoding] != 0)
	if err != nil {
		return nil, err
	}

	out.Pixels = make([]byte, out.Width*out.Height)
	for y := range out.Height {
		copy(out.Pixels[y*out.Width:], scans[y*stride:y*stride+out.Width])
	}

	return out, nil
}

// pcxUnpack undoes the run encoding: a byte with its top two bits set is a
// count in the low six and the value in the byte after, and anything else is
// one pixel of itself.
//
// Which is why a PCX of a photograph can be bigger than the photograph: a value
// of its own that happens to have both top bits set costs two bytes. Nothing
// here cares, since nothing here writes one.
func pcxUnpack(data []byte, want int, encoded bool) ([]byte, error) {
	out := make([]byte, 0, want)

	for at := 0; len(out) < want; {
		if at >= len(data) {
			return nil, fmt.Errorf("this PCX stops %d bytes short of its %d pixels",
				want-len(out), want)
		}

		value := data[at]
		at++

		times := 1

		if encoded && value&0xC0 == 0xC0 {
			times = int(value & 0x3F)

			if at >= len(data) {
				return nil, fmt.Errorf("this PCX ends on a run with no value after it")
			}

			value = data[at]
			at++
		}

		for range times {
			if len(out) == want {
				break
			}

			out = append(out, value)
		}
	}

	return out, nil
}
