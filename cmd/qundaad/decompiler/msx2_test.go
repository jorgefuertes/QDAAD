package decompiler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Byte 2 of the header held the null word the /CTL section declares, which is
// why every database of the period has 95 in it. The MSX2 interpreter needed
// somewhere to keep the screen mode it was built for and took that byte, so on
// that one machine it is not a character at all. Reading it as one put an
// unprintable byte where every entry wanted an underscore. Issue #6.
//
// NONE OF THIS IS CHECKED AGAINST A REAL DATABASE. There is no MSX2 DDB in the
// corpus — the only MSX edition we have is machine 4 — so the headers below are
// built by hand from what drb.php writes, at :1844 and :1248-1261, and that is
// the whole of the evidence. A real one would turn these into a round trip.
func header(machine, byteTwo byte) *Reader {
	// Byte 0 is the version, byte 1 the machine in its high nibble and the
	// language in its low one. Nothing past byte 2 is read here.
	return &Reader{data: []byte{2, machine<<4 | 1, byteTwo, 0, 0, 0, 0, 0}}
}

func TestTheNullWordIsNotReadFromTheMSX2SubtargetByte(t *testing.T) {
	t.Run("a classic machine keeps reading the byte", func(t *testing.T) {
		r := header(1, '_') // a Spectrum, with the usual underscore
		require.Equal(t, byte('_'), r.NullWord())
		require.Empty(t, r.VideoMode(), "no other machine has a video mode there")
	})

	t.Run("a classic machine with an unusual null word is believed", func(t *testing.T) {
		require.Equal(t, byte('*'), header(0, '*').NullWord())
	})

	t.Run("MSX2 gets the underscore and not the subtarget", func(t *testing.T) {
		// 0x03 is SCREEN 8, and as a character it does not print.
		r := header(machineMSX2, 0x03)

		require.Equal(t, byte('_'), r.NullWord(),
			"the byte is the subtarget, so there is no null word to read")
	})
}

func TestTheMSX2VideoModeIsReadFromThatByte(t *testing.T) {
	for _, tc := range []struct {
		byteTwo byte
		want    string
	}{
		{0x00, "SCREEN 5, 6-pixel characters"},
		{0x03, "SCREEN 8, 6-pixel characters"},
		{0x07, "SCREEN 12, 6-pixel characters"},
		{0x80, "SCREEN 5, 8-pixel characters"},
		{0x87, "SCREEN 12, 8-pixel characters"},
	} {
		t.Run(fmt.Sprintf("%02x", tc.byteTwo), func(t *testing.T) {
			require.Equal(t, tc.want, header(machineMSX2, tc.byteTwo).VideoMode())
		})
	}
}
