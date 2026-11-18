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

// What the interpreter reads: a byte saying how many bands come in each record,
// the palette, the length of a band, and then the bands themselves at a record
// boundary apiece.
func TestWhatIsInACPCPicture(t *testing.T) {
	palette := []byte{0, 11, 25, 9}

	body, err := cpcPicture(cpcScreenOf(12), palette, 0)
	require.NoError(t, err)

	// Twelve rows are 960 bytes a band, and two of those fit in a record
	// alongside the nineteen bytes of header. That is the whole of why DAAD's
	// own IMGLINES defaults to 96.
	require.Equal(t, byte(2), body[0], "bands to a record")
	require.Equal(t, palette, body[1:5])
	require.Equal(t, 960, int(body[17])|int(body[18])<<8, "bytes to a band")

	// Four records for the eight bands, and nothing over.
	require.Equal(t, 4*cpcRecordSize, len(body))
	require.Zero(t, len(body)%cpcRecordSize)

	// The first band starts right after the header and the second right after
	// it; the third is at the top of the next record, not where it happens to
	// fall.
	require.Equal(t, byte(0xFF), body[cpcPictureHeader])
	require.Equal(t, byte(0xFF), body[cpcPictureHeader+960])
	require.Equal(t, byte(0xFF), body[cpcRecordSize])
	require.Equal(t, byte(0), body[cpcPictureHeader+2*960], "the rest of a record is padding")
}

// A picture with no inks beside it is refused rather than drawn in whatever the
// last one left behind.
func TestACPCPictureWithoutItsInksIsRefused(t *testing.T) {
	_, err := cpcPicture(cpcScreenOf(12), nil, 0)
	require.ErrorContains(t, err, "inks")

	_, err = cpcPicture(make([]byte, 100), []byte{0, 1, 2, 3}, 0)
	require.ErrorContains(t, err, "16384 bytes")
}

// The disk takes the pictures in number order and says which ones did not fit.
// Saying so is the point: a disk that silently carries nine of twenty-eight
// looks like a disk that carries them all until somebody plays it.
func TestTheCPCDiskNamesThePicturesItLeavesOut(t *testing.T) {
	images := map[int][]byte{}
	palettes := map[int][]byte{}

	for n := range 40 {
		images[n] = cpcScreenOf(12)
		palettes[n] = []byte{0, 11, 25, 9}
	}

	var said string

	opts := Options{
		Language: language.LanguageSpanish, Name: "AO",
		Images: images, Palettes: palettes,
		Note: func(_ Notice, s string) { said = s },
	}

	image, err := CPCDisk(make([]byte, 12000), opts)
	require.NoError(t, err)

	disk, err := media.NewDSK(image)
	require.NoError(t, err)

	files, err := disk.Files()
	require.NoError(t, err)

	require.Equal(t, "AO.BIN", files[0].Name)
	require.Equal(t, "000.CPC", files[1].Name, "and the pictures in number order")
	require.Less(t, len(files), 1+len(images), "forty pictures do not fit on a 180K disk")

	require.Contains(t, said, "would not fit")
	require.Contains(t, said, "039.CPC", "the last ones are the ones left out")
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

	record := tape[at+0x13 : at+0x13+cpcRecordSize]
	require.Equal(t, byte(0x16), record[0], "a data record opens with 0x16")
	require.Equal(t, body[:16], record[1:17])
}
