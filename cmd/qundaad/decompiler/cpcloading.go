package decompiler

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
)

// The picture the Amstrad shows while the rest loads.
//
// It is not in the database, it is not among the drawings, and no filesystem
// names it: the disks of 1989 are formatted for their own loader and the tapes
// are a custom turbo format. What there is, is a dump of video memory sitting in
// the payload, and a loader that drops it on the screen with
//
//	ld hl,<where> / ld de,0xc000 / ld bc,0x4000 / ldir
//
// So it is found the way the drawings are: by what it has to look like rather
// than by anything declaring it.
//
// A dump of a CPC display is sixteen kilobytes and only 16000 of them are
// shown. The display is eight interleaved bands of 0x800, and a band holds
// twenty-five rows of eighty bytes — 2000 — and then **48 bytes nothing reads**.
// Those 48 are what give it away: a screen that was dumped out of a machine has
// them all zero, eight times over, which is 384 bytes of agreement no stretch of
// code or text is going to manage by accident. Across La Aventura Original it
// finds two on the disk, one for each part, and one on each side of the tape,
// and nothing else.
const (
	cpcLoadingSize = 8 * 0x800 // the whole of video memory
	cpcBandShown   = 2000      // of each band, and the rest is the gap
)

// cpcLoadingScreen is one of them: the bytes, the colours they are read
// through, and where it was found.
type cpcLoadingScreen struct {
	at     int
	screen []byte
	// inks are the sixteen pens as the firmware numbers its colours, or nil
	// where the loader's table was not found. border is the seventeenth, and
	// is -1 when it is not known.
	inks   []int
	border int
	// found says which shape the palette turned up in, for the listing.
	found string
}

// mode is which screen mode the picture was drawn in.
//
// Nothing in a dump declares it — DAAD ships both an M0LOAD and an M1LOAD for
// exactly that reason — so it is taken from the palette: mode 1 has four pens
// and a loader that sets more than four is not loading a mode 1 picture. With no
// palette at all the answer is mode 0, which is what these loading screens are
// and what a sixteen-colour table implies anyway.
func (s cpcLoadingScreen) mode() int {
	if len(s.inks) == 0 {
		return 0
	}

	distinct := map[int]bool{}
	for _, ink := range s.inks {
		distinct[ink] = true
	}

	if len(distinct) <= 4 {
		return 1
	}

	return 0
}

// cpcHardwareInk turns one of the gate array's colour numbers into the one the
// firmware uses, which is the one everything else here speaks.
//
// The table is the hardware's and has duplicates in it: 32 numbers for 27
// colours. Taken from SCRMAKER.PAS's GetCPCFirmwarePalette, which is the only
// place in the kit where it is written down.
var cpcHardwareInk = map[byte]int{
	0x14: 0, 0x04: 1, 0x15: 2, 0x1C: 3, 0x18: 4, 0x1D: 5, 0x0C: 6, 0x05: 7, 0x0D: 8,
	0x16: 9, 0x06: 10, 0x17: 11, 0x1E: 12, 0x00: 13, 0x1F: 14, 0x0E: 15, 0x07: 16,
	0x0F: 17, 0x12: 18, 0x02: 19, 0x13: 20, 0x1A: 21, 0x19: 22, 0x1B: 23, 0x0A: 24,
	0x03: 25, 0x0B: 26,
	// The five the hardware repeats, which a palette may still carry.
	0x01: 13, 0x08: 7, 0x09: 24, 0x10: 1, 0x11: 19,
}

// findCPCLoadingScreens returns every dump of a display the data holds, in the
// order they lie.
func findCPCLoadingScreens(data []byte) []cpcLoadingScreen {
	var found []cpcLoadingScreen

	for at := 0; at+cpcLoadingSize <= len(data); {
		if !looksLikeCPCScreen(data[at : at+cpcLoadingSize]) {
			at++

			continue
		}

		screen := cpcLoadingScreen{at: at, screen: data[at : at+cpcLoadingSize], border: -1}
		screen.readPalette(data)

		found = append(found, screen)

		// A screen is one screen: the window a byte along is the same picture
		// read crooked, so the search carries on past it.
		at += cpcLoadingSize
	}

	return found
}

// looksLikeCPCScreen is the gap test, and a screen nobody drew on is not a
// picture: a blank one would match every stretch of zeros in the payload.
func looksLikeCPCScreen(d []byte) bool {
	drawn := false

	for band := range 8 {
		shown := d[band*0x800 : band*0x800+cpcBandShown]
		gap := d[band*0x800+cpcBandShown : (band+1)*0x800]

		if bytes.Count(gap, []byte{0}) != len(gap) {
			return false
		}

		if bytes.Count(shown, []byte{0}) != len(shown) {
			drawn = true
		}
	}

	return drawn
}

// readPalette looks for the colours the loader sets before it paints.
//
// There are two shapes and both are first-hand, from the two loaders of La
// Aventura Original. They are tried in that order because the first is exact
// and the second is a search.
func (s *cpcLoadingScreen) readPalette(data []byte) {
	_ = s.fromHardwareTable(data) || s.fromFirmwareTable(data)
}

// fromHardwareTable is the disk loader's: seventeen of the gate array's own
// colour numbers lying immediately before the picture, which it copies twice
// into the firmware's ink table. Two copies because the firmware keeps two
// colours per pen, for flashing, and these never flash.
//
// The border comes first and the sixteen pens after it. Nothing says so — both
// readings are seventeen bytes — and it was settled by looking: read the other
// way round La Aventura Original's cave comes out magenta where it should be
// orange, every colour one pen along.
//
//	ld hl,<palette> / ld c,0x11 / ldir   (twice)
//	ld hl,<screen>  / ld de,0xc000 / ld bc,0x4000 / ldir
func (s *cpcLoadingScreen) fromHardwareTable(data []byte) bool {
	const entries = 17

	at := s.at - entries
	if at < 0 {
		return false
	}

	table := data[at:s.at]

	distinct := map[byte]bool{}

	for _, b := range table {
		if _, known := cpcHardwareInk[b]; !known {
			return false
		}

		distinct[b] = true
	}

	// A run of one colour repeated is not a palette, and the payload has
	// plenty of those.
	const leastColours = 8
	if len(distinct) < leastColours {
		return false
	}

	s.border = cpcHardwareInk[table[0]]

	s.inks = make([]int, 0, entries-1)
	for _, b := range table[1:] {
		s.inks = append(s.inks, cpcHardwareInk[b])
	}

	s.found = fmt.Sprintf("seventeen of the gate array's colour numbers at %#x, "+
		"the border and then the pens", at)

	return true
}

// fromFirmwareTable is the tape loader's, and it is the weaker of the two: a
// run of sixteen firmware colours that it hands to SCR SET INK one by one. It
// does not lie beside the picture — on the tape of La Aventura Original the
// table is at 0x318 and the picture at 0x51F — so what is taken is the last one
// before the picture, and there is no border in it.
func (s *cpcLoadingScreen) fromFirmwareTable(data []byte) bool {
	const (
		entries      = 16
		leastColours = 12
	)

	best := -1

	for at := range s.at - entries {
		table := data[at : at+entries]

		distinct := map[byte]bool{}

		fits := true

		for _, b := range table {
			if int(b) >= cpcInks {
				fits = false

				break
			}

			distinct[b] = true
		}

		if fits && len(distinct) >= leastColours {
			best = at
		}
	}

	if best < 0 {
		return false
	}

	s.inks = make([]int, 0, entries)
	for _, b := range data[best : best+entries] {
		s.inks = append(s.inks, int(b))
	}

	s.found = fmt.Sprintf("sixteen firmware colours at %#x, no border", best)

	return true
}

// attachCPCLoading gives each screen to the database it loads for: the first
// one that starts after it.
//
// After, and not before. A loading screen is shown while the part goes in, so
// it is on the medium ahead of what it is announcing — and the disk of La
// Aventura Original proves the point, with its second screen lying between the
// two databases and belonging to the second.
func attachCPCLoading(databases []database, screens []cpcLoadingScreen) {
	for i := range screens {
		for j := range databases {
			if databases[j].reader.Offset() <= screens[i].at {
				continue
			}

			if databases[j].cpcLoading == nil {
				databases[j].cpcLoading = &screens[i]
			}

			break
		}
	}
}

// writeCPCLoading writes the picture beside the source: the bytes as video
// memory held them, the colours they are read through, and a PNG of the two
// together.
//
// It goes at the top of gfx/ rather than among the numbered pictures, because
// that is what it is not: nothing asks for it by number, and a loading.scr
// among the 000.scr would be a picture the compiler could not name.
func writeCPCLoading(s cpcLoadingScreen, gfx string, opts Options) error {
	if err := writeInto(gfx, "", "loading.scr", s.screen); err != nil {
		return err
	}

	if len(s.inks) > 0 {
		if err := writeInto(gfx, "", "loading.pal", s.pal()); err != nil {
			return err
		}
	}

	img := s.picture()
	if err := writePNGInto(gfx, picturesDir, "loading.png", img); err != nil {
		return err
	}

	if opts.Scale > 1 {
		name := fmt.Sprintf("loadingx%d.png", opts.Scale)
		if err := writePNGInto(gfx, picturesDir, name, enlarge(img, opts.Scale)); err != nil {
			return err
		}
	}

	return nil
}

// pal is the colours as a file: the sixteen pens and then the border, one
// firmware number each. Seventeen and not sixteen, because a loading screen is
// the one picture whose border is part of it.
//
// A border nobody told us about is written as 0, which is black and is what the
// tape loaders set theirs to.
func (s cpcLoadingScreen) pal() []byte {
	out := make([]byte, 17)
	for pen, ink := range s.inks {
		if pen < 16 {
			out[pen] = byte(ink)
		}
	}

	if s.border >= 0 {
		out[16] = byte(s.border)
	}

	return out
}

// picture draws the dump at the size the machine held it: 160 pixels across in
// mode 0 and 320 in mode 1, which is what is in the bytes. The hardware shows a
// mode 0 pixel twice as wide as a mode 1 one and this does not, for the same
// reason the character sets are not scaled up: it would be kinder to the eye and
// a lie about what is in there.
func (s cpcLoadingScreen) picture() image.Image {
	width, perByte := cpcCanvasWidth, cpcPixelsPerByte
	if s.mode() == 0 {
		width, perByte = cpcCanvasWidth/2, cpcPixelsPerByte/2
	}

	var shades [16]color.NRGBA

	for pen := range shades {
		if pen < len(s.inks) {
			shades[pen] = cpcInkColour(s.inks[pen])

			continue
		}

		// With no palette the pens are drawn as a ramp of greys, which says
		// "these are the pens" and does not pretend to be the colours.
		level := uint8(pen * 0xFF / 15)
		shades[pen] = color.NRGBA{R: level, G: level, B: level, A: 0xFF}
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, cpcCanvasHeight))

	for row := range cpcCanvasHeight {
		line := (row%cpcCellSize)*0x800 + (row/cpcCellSize)*cpcBytesWide

		for x := range width {
			img.SetNRGBA(x, row, shades[cpcPenAt(s.screen, line+x/perByte, x%perByte, s.mode())])
		}
	}

	return img
}

// cpcPenAt reads one pixel out of a byte of video memory.
//
// Neither mode puts a pixel's bits anywhere obvious. A pixel's bits lie at 7-p,
// 3-p, 5-p and 1-p of the byte, lowest first; mode 1 uses the first two of
// those and mode 0 all four, which is why one holds four pixels to the byte and
// the other two.
func cpcPenAt(screen []byte, at, pixel, mode int) int {
	if at >= len(screen) {
		return 0
	}

	bits := [...]int{7 - pixel, 3 - pixel, 5 - pixel, 1 - pixel}

	used := 2
	if mode == 0 {
		used = 4
	}

	pen := 0

	for weight, bit := range bits[:used] {
		if bit >= 0 && screen[at]&(1<<bit) != 0 {
			pen |= 1 << weight
		}
	}

	return pen
}
