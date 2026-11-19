package pack

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

// cpcScreenOf is a mode 1 screen with the top rows filled, for the tests that
// care how tall a picture is rather than what is in it.
func cpcScreenOf(rows int) []byte {
	screen := make([]byte, cpcPackScreen)

	for band := range cpcPackBands {
		at := band * cpcPackBandSize
		for i := range rows * cpcPackBytesWide {
			screen[at+i] = 0xFF
		}
	}

	return screen
}

// A picture is carried down to its own last row with a pixel in it, which is
// what keeps the drawings of 1989 whole: they are not all the same height and
// DAAD's one height for all of them takes the bottom off the tall ones.
func TestACPCPictureIsAsTallAsItNeeds(t *testing.T) {
	require.Equal(t, 14, cpcPictureRows(cpcScreenOf(14), 0))
	require.Equal(t, 1, cpcPictureRows(cpcScreenOf(0), 0))
	require.Equal(t, 25, cpcPictureRows(cpcScreenOf(25), 0))

	// And a height asked for wins, rounded up to the character row it falls
	// in: a band is copied in whole rows and there is no saying half of one.
	require.Equal(t, 12, cpcPictureRows(cpcScreenOf(25), 96))
	require.Equal(t, 13, cpcPictureRows(cpcScreenOf(25), 97))
	require.Equal(t, 25, cpcPictureRows(cpcScreenOf(25), 400))
}

// A disk is a 6128 disk, and one asked for without the memory to hold it is
// refused rather than written as the thing it cannot be.
//
// It used to be written: the pictures went on as files of their own,
// uncompressed, and nine of twenty-eight fit. What the other nineteen cost was
// not nine-twenty-eighths of the drawings but the text of their locations as
// well, because PICTURE is a condition in version 3 and a failed one abandons
// the rest of its entry. See CPCDisk.
func TestACPCDiskWantsTheMemoryItNeeds(t *testing.T) {
	opts := Options{
		Language: language.LanguageSpanish, Name: "AO",
		Note: func(Notice, string) {},
	}

	_, err := CPCDisk(make([]byte, 12000), opts)
	require.ErrorContains(t, err, "--ram 128")
	require.ErrorContains(t, err, "cdt", "and says where a 464 goes instead")

	opts.Memory = 128

	_, err = CPCDisk(make([]byte, 12000), opts)
	require.NoError(t, err)
}

// The tape is the binary and nothing else, and the AMSDOS header comes off on
// the way: a tape carries the load address in the record the firmware reads,
// and a file with both would land 128 bytes adrift.
func TestTheCPCTapeIsTheBinaryWithoutItsHeader(t *testing.T) {
	opts := Options{Language: language.LanguageSpanish, Name: "AO"}

	tape, err := CPCTape(make([]byte, 5000), opts)
	require.NoError(t, err)

	require.Equal(t, "ZXTape!\x1a", string(tape[:8]))

	binary, err := cpcBinary(make([]byte, 5000), opts)
	require.NoError(t, err)

	body := media.WithoutAMSDOSHeader(binary)

	// The first data record holds the first block of it, which starts with the
	// interpreter's own first bytes.
	at := 10
	length := int(tape[at+0x10]) | int(tape[at+0x11])<<8 | int(tape[at+0x12])<<16
	at += 1 + 0x12 + length

	require.Equal(t, byte(0x11), tape[at])

	// A tape is cut into blocks of this, which is the same two kilobytes a
	// record is on a disk and for the same reason: it is what the firmware
	// reads at a time.
	const block = 2048

	record := tape[at+0x13 : at+0x13+block]
	require.Equal(t, byte(0x16), record[0], "a data record opens with 0x16")
	require.Equal(t, body[:16], record[1:17])
}
