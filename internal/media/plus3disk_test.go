package media

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// Writing a +3 disk and then reading it with our own reader.
//
// This is the test worth having, and it is worth it because the two halves
// were written years apart and for opposite reasons: the reader exists to find
// databases inside the disks of 1989, and knew nothing about a disk we would
// write. If the directory this lays down is wrong in any of the ways a CP/M
// directory can be wrong — a block number off by the reserved track, a record
// count that truncates the last sector, a name padded with nulls instead of
// spaces — the reader gives back the wrong bytes or no files at all.
func TestADiskWeWriteIsADiskWeCanRead(t *testing.T) {
	// Sizes chosen to catch the arithmetic: one well under a block, one
	// exactly a block, one that spills a few bytes into a second.
	files := []File{
		{Name: "DISK", Data: bytes.Repeat([]byte{1}, 202)},
		{Name: "DAAD.BIN", Data: bytes.Repeat([]byte{2}, plus3BlockSize)},
		{Name: "PART1.DDB", Data: bytes.Repeat([]byte{3}, plus3BlockSize+7)},
	}

	image, err := Plus3Disk(files)
	require.NoError(t, err)

	disk, err := NewDSK(image)
	require.NoError(t, err, "our own reader will not take the image we just wrote")

	back, err := disk.Files()
	require.NoError(t, err)
	require.Len(t, back, len(files), "the directory does not list what was put on it")

	for i, want := range files {
		require.Equal(t, want.Name, back[i].Name, "file %d came back under another name", i)

		// CP/M stores a length in records of 128 bytes, so what comes back is
		// the file rounded up to one. What matters is that every byte of the
		// file is there and in order; the padding after it is the filesystem's.
		require.GreaterOrEqual(t, len(back[i].Data), len(want.Data),
			"%s came back short", want.Name)
		require.Equal(t, want.Data, back[i].Data[:len(want.Data)],
			"%s came back with its bytes changed", want.Name)
	}
}

// The shape of the image, which is what an emulator reads before it reads
// anything else.
func TestTheImageIsADiskAnEmulatorRecognises(t *testing.T) {
	image, err := Plus3Disk([]File{{Name: "X", Data: []byte("x")}})
	require.NoError(t, err)

	require.Equal(t, dskSignature, string(image[:len(dskSignature)]))
	require.Equal(t, byte(plus3Tracks), image[0x30])
	require.Equal(t, byte(plus3Sides), image[0x31])

	trackSize := dskTrackHeader + plus3SectorsTrack*plus3SectorSize
	require.Equal(t, trackSize, int(image[0x32])|int(image[0x33])<<8)
	require.Len(t, image, dskHeaderSize+plus3Tracks*trackSize,
		"180K is 40 tracks of nine 512-byte sectors, and the headers on top")

	// Every track says what it holds, and the sector numbering starts where
	// the +3 expects it to. A disk numbered from 1 instead of 0xC1 formats
	// fine and is read as unformatted.
	track := image[dskHeaderSize:]
	require.Equal(t, trackSignature, string(track[:len(trackSignature)]))
	require.Equal(t, byte(plus3SectorsTrack), track[0x15])
	require.Equal(t, byte(plus3FirstSectorID), track[0x18+2])
	require.Equal(t, byte(plus3FirstSectorID+plus3SectorsTrack-1), track[0x18+8*8+2])
}

// The directory is in the first sector of track 0, and that is the one thing
// about this format worth a test of its own.
//
// Nothing in CP/M makes it obvious: a PCW reserves a track before its data
// area and an AMSDOS disk reserves two, so one reserved track is the natural
// guess and it is wrong. A disk written that way formats perfectly, passes
// every structural check, is read back correctly by our own reader — which
// looks for the directory on tracks 0, 1 and 2 — and CATs as empty on a +3,
// because the machine looks at track 0, finds the 0xE5 of a freshly formatted
// disk, and believes it.
//
// The number comes off a disk known to work: DAAD's own +3 disk has
// "PART1   SDG" 512 bytes into the image, which is track 0's first sector.
func TestTheDirectoryIsInTheFirstSectorOfTrackZero(t *testing.T) {
	image, err := Plus3Disk([]File{{Name: "DISK", Data: []byte("loader")}})
	require.NoError(t, err)

	entry := image[dskHeaderSize+dskTrackHeader:]

	require.Equal(t, byte(0), entry[0], "user zero")
	require.Equal(t, "DISK    ", string(entry[1:9]))
	require.Equal(t, "   ", string(entry[9:12]))

	// Block 2, because the directory itself takes blocks 0 and 1.
	require.Equal(t, byte(plus3DirBlocks), entry[16],
		"the first file was put somewhere other than straight after the directory")
}

// Names a CP/M directory cannot hold are refused rather than cut down.
//
// Quietly truncating would be the worst of the options: DISK is the one name
// the +3's boot menu looks for, so a release whose loader was silently renamed
// is a release that does not start and says nothing about why.
func TestNamesADiskCannotHold(t *testing.T) {
	for _, name := range []string{
		"DEMASIADOLARGO.BIN", "X.CUATRO", "", "CON ESPACIO.BIN", "RARO?.BIN",
	} {
		_, err := Plus3Disk([]File{{Name: name, Data: []byte("x")}})
		require.Error(t, err, "name %q was accepted", name)
	}
}

// And a release too big for the disk says so with the numbers.
func TestMoreThanFitsOnADiskIsRefused(t *testing.T) {
	_, err := Plus3Disk([]File{{Name: "ENORME.BIN", Data: make([]byte, 200*1024)}})
	require.ErrorContains(t, err, "does not fit")
	require.ErrorContains(t, err, "blocks")
}
