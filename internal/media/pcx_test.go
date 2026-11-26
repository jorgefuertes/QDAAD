package media

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// PCX is the only picture format here that is not some machine's video memory,
// and the only one read rather than written. What it is held against is
// script/compare-scrmaker.sh, which converts the same files with the reference
// tool; these check the reading itself, and in particular the two things a
// hand-rolled reader gets wrong — the run encoding and the scanline padding.

func TestAPCXReadsBackWhatWasPutIn(t *testing.T) {
	// A gradient, so that a reader which lost a row or slipped a byte shows it.
	want := make([]byte, 64*32)
	for i := range want {
		want[i] = byte(i % 251)
	}

	image, err := ReadPCX(buildPCX(t, 64, 32, 64, want))
	require.NoError(t, err)

	require.Equal(t, 64, image.Width)
	require.Equal(t, 32, image.Height)
	require.Equal(t, want, image.Pixels)

	// The palette comes off the end of the file, not the header: the header was
	// fixed before anybody wanted 256 colours.
	require.Equal(t, []byte{0, 0, 0, 5, 11, 23}, image.Palette[:6])
}

// A scanline is padded to whatever the header says, which may be wider than the
// picture. Reading the padding as pixels shifts every row after the first.
func TestThePaddingAtTheEndOfAScanlineIsNotPixels(t *testing.T) {
	const width, height, stride = 10, 4, 16

	want := make([]byte, width*height)
	for i := range want {
		want[i] = byte(i + 1)
	}

	image, err := ReadPCX(buildPCX(t, width, height, stride, want))
	require.NoError(t, err)

	require.Equal(t, want, image.Pixels)
}

// A value of its own with both top bits set has to be written as a run of one,
// or a reader takes it for a count. It is the case that a picture of a
// photograph meets and a picture of a gradient does not.
func TestAPixelThatLooksLikeARunCount(t *testing.T) {
	want := []byte{0xC0, 0xFF, 0x01, 0xC3, 0xC3, 0xC3, 0x00, 0xFE}

	image, err := ReadPCX(buildPCX(t, len(want), 1, len(want), want))
	require.NoError(t, err)

	require.Equal(t, want, image.Pixels)
}

// Raw PCX files exist — encoding byte zero — and the reference reads them
// (SCRMAKER.PAS:633 tests that byte before treating anything as a run).
func TestAnUnencodedPCXIsStillAPCX(t *testing.T) {
	want := []byte{0xC0, 1, 2, 0xFF}

	file := buildPCX(t, len(want), 1, len(want), want)
	file[pcxEncoding] = 0

	// Rewrite the body raw, since buildPCX encoded it.
	raw := append(file[:pcxHeaderSize:pcxHeaderSize], want...)
	raw = append(raw, file[len(file)-pcxPaletteTail:]...)

	image, err := ReadPCX(raw)
	require.NoError(t, err)
	require.Equal(t, want, image.Pixels)
}

// The ones that have to be refused rather than half-read.
func TestAPCXThatIsNotOneSaysSo(t *testing.T) {
	good := buildPCX(t, 8, 2, 8, make([]byte, 16))

	t.Run("too short", func(t *testing.T) {
		_, err := ReadPCX(good[:100])
		require.ErrorContains(t, err, "at least")
	})

	t.Run("not a PCX at all", func(t *testing.T) {
		bad := append([]byte{}, good...)
		bad[pcxManufacturer] = 0x42

		_, err := ReadPCX(bad)
		require.ErrorContains(t, err, "starts 0x42")
	})

	t.Run("sixteen colours", func(t *testing.T) {
		bad := append([]byte{}, good...)
		bad[pcxBitsPerPixel] = 4

		_, err := ReadPCX(bad)
		require.ErrorContains(t, err, "256 colours")
	})

	t.Run("no palette at the end", func(t *testing.T) {
		bad := append([]byte{}, good...)
		bad[len(bad)-pcxPaletteTail] = 0

		_, err := ReadPCX(bad)
		require.ErrorContains(t, err, "no 256-colour palette")
	})

	t.Run("a line too narrow for the picture", func(t *testing.T) {
		bad := append([]byte{}, good...)
		binary.LittleEndian.PutUint16(bad[pcxBytesPerLine:], 2)

		_, err := ReadPCX(bad)
		require.ErrorContains(t, err, "will not hold")
	})

	t.Run("cut short", func(t *testing.T) {
		bad := append([]byte{}, good[:pcxHeaderSize+2]...)
		bad = append(bad, good[len(good)-pcxPaletteTail:]...)

		_, err := ReadPCX(bad)
		require.ErrorContains(t, err, "stops")
	})
}

// buildPCX writes an 8-bit PCX of the pixels given, run-encoding each scanline
// and padding it out to the stride.
func buildPCX(t *testing.T, width, height, stride int, pixels []byte) []byte {
	t.Helper()

	require.Len(t, pixels, width*height)
	require.GreaterOrEqual(t, stride, width)

	out := make([]byte, pcxHeaderSize)

	out[pcxManufacturer] = 0x0A
	out[1] = 5
	out[pcxEncoding] = 1
	out[pcxBitsPerPixel] = 8
	out[pcxPlanes] = 1

	binary.LittleEndian.PutUint16(out[pcxWindow+4:], uint16(width-1))
	binary.LittleEndian.PutUint16(out[pcxWindow+6:], uint16(height-1))
	binary.LittleEndian.PutUint16(out[pcxBytesPerLine:], uint16(stride))

	for y := range height {
		row := make([]byte, stride)
		copy(row, pixels[y*width:(y+1)*width])

		for at := 0; at < len(row); {
			run := 1
			for at+run < len(row) && row[at+run] == row[at] && run < 63 {
				run++
			}

			if run > 1 || row[at] >= 0xC0 {
				out = append(out, 0xC0|byte(run), row[at])
			} else {
				out = append(out, row[at])
			}

			at += run
		}
	}

	out = append(out, pcxPaletteMarker)

	for i := range 256 {
		out = append(out, byte(i*5), byte(i*11), byte(i*23))
	}

	return out
}
