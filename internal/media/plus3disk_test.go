package media

import (
	"bytes"
	"fmt"
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
		{Name: "DAAD.BIN", Data: bytes.Repeat([]byte{2}, 1024)},
		{Name: "PART1.DDB", Data: bytes.Repeat([]byte{3}, 1024+7)},
	}

	// Both formats, because the reader has to find the directory wherever the
	// geometry put it, and the two put it a track apart.
	for name, write := range map[string]func([]File) ([]byte, error){
		"DATA": DataDisk, "+3": Plus3Disk,
	} {
		t.Run(name, func(t *testing.T) {
			image, err := write(files)
			require.NoError(t, err)

			disk, err := NewDSK(image)
			require.NoError(t, err, "our own reader will not take the image we just wrote")

			back, err := disk.Files()
			require.NoError(t, err)
			require.Len(t, back, len(files), "the directory does not list what was put on it")

			for i, want := range files {
				require.Equal(t, want.Name, back[i].Name,
					"file %d came back under another name", i)

				// CP/M stores a length in records of 128 bytes, so what comes
				// back is the file rounded up to one. What matters is that every
				// byte of the file is there and in order; the padding after it
				// is the filesystem's.
				require.GreaterOrEqual(t, len(back[i].Data), len(want.Data),
					"%s came back short", want.Name)
				require.Equal(t, want.Data, back[i].Data[:len(want.Data)],
					"%s came back with its bytes changed", want.Name)
			}
		})
	}
}

// The shape of the image, which is what an emulator reads before it reads
// anything else.
func TestTheImageIsADiskAnEmulatorRecognises(t *testing.T) {
	image, err := DataDisk([]File{{Name: "X", Data: []byte("x")}})
	require.NoError(t, err)

	g := dataGeometry

	require.Equal(t, dskSignature, string(image[:len(dskSignature)]))
	require.Equal(t, byte(g.tracks), image[0x30])
	require.Equal(t, byte(dataSides), image[0x31])

	trackSize := dskTrackHeader + g.sectorsTrack*dataSectorSize
	require.Equal(t, trackSize, int(image[0x32])|int(image[0x33])<<8)
	require.Len(t, image, dskHeaderSize+g.tracks*trackSize,
		"180K is 40 tracks of nine 512-byte sectors, and the headers on top")

	// Every track says what it holds, and the sector numbering starts where
	// the machine expects it to. A DATA disk numbered from 1 instead of 0xC1
	// formats fine and is read as unformatted.
	track := image[dskHeaderSize:]
	require.Equal(t, trackSignature, string(track[:len(trackSignature)]))
	require.Equal(t, byte(g.sectorsTrack), track[0x15])
	require.Equal(t, g.firstSectorID, track[0x18+2])
	require.Equal(t, g.firstSectorID+byte(g.sectorsTrack)-1, track[0x18+8*8+2])
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
	image, err := DataDisk([]File{{Name: "DISK", Data: []byte("loader")}})
	require.NoError(t, err)

	entry := image[dskHeaderSize+dskTrackHeader:]

	require.Equal(t, byte(0), entry[0], "user zero")
	require.Equal(t, "DISK    ", string(entry[1:9]))
	require.Equal(t, "   ", string(entry[9:12]))

	// Block 2, because the directory itself takes blocks 0 and 1.
	require.Equal(t, byte(dataGeometry.dirBlocks), entry[16],
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
		_, err := DataDisk([]File{{Name: name, Data: []byte("x")}})
		require.Error(t, err, "name %q was accepted", name)
	}
}

// And a release too big for the disk says so with the numbers.
func TestMoreThanFitsOnADiskIsRefused(t *testing.T) {
	_, err := DataDisk([]File{{Name: "ENORME.BIN", Data: make([]byte, 200*1024)}})
	require.ErrorContains(t, err, "does not fit")
	require.ErrorContains(t, err, "blocks")
}

// The +3's own format, which exists for one byte of it.
//
// A DATA disk holds 64 names and there is no arguing with it: nothing written
// on the disk says how big the directory is, the machine works it out from the
// sector numbering. This format says, in the byte at +7 of the first sector of
// track 0, and +3DOS believes it — which is the difference between an adventure
// of 69 pictures-as-files fitting on a disk and not.
//
// The rest of the specification is La Aventura Original's own disk, read out of
// it, so a failure here is this drifting away from a disk known to run on real
// hardware rather than a judgement about what ought to work.
func TestThePlus3SpecificationSaysWhatTheOriginalDiskSays(t *testing.T) {
	image, err := Plus3Disk([]File{{Name: "DISK", Data: []byte("loader")}})
	require.NoError(t, err)

	spec := image[dskHeaderSize+dskTrackHeader:][:dataSectorSize]

	//	00 00 28 09 02 01 03 04 2A 52
	//	type, one side, 40 tracks, 9 sectors of 512, one reserved track,
	//	blocks of 1 KiB, and four of them for the directory where the
	//	original says two.
	require.Equal(t,
		[]byte{0x00, 0x00, 0x28, 0x09, 0x02, 0x01, 0x03, 0x04, 0x2A, 0x52},
		spec[:10], "the specification is not the one a +3 reads")

	require.Equal(t, 128, Plus3Names(), "four blocks of 32-byte entries is 128 names")

	// And the sector must not add up to the boot signature. If it does, the +3
	// menu does not load a disk, it runs these 512 bytes.
	sum := 0
	for _, b := range spec {
		sum += int(b)
	}

	require.NotEqual(t, plus3Bootable, sum%256,
		"the +3 would take this for a boot sector and execute it")
}

// The specification costs a track and the arithmetic has to come out.
//
// Nine 512-byte sectors is four and a half blocks a track, so 39 tracks is
// 175.5 of them and the half is unusable. Rounding that up instead of down
// would hand a file a block whose second half is off the end of the disk.
func TestThePlus3DiskIsOneTrackShorterAndWholeBlocks(t *testing.T) {
	g := plus3Geometry

	require.Equal(t, 1, g.reserved, "the specification needs a track to live in")

	image, err := Plus3Disk([]File{{Name: "DISK", Data: []byte("loader")}})
	require.NoError(t, err)

	trackSize := dskTrackHeader + g.sectorsTrack*dataSectorSize
	require.Len(t, image, dskHeaderSize+g.tracks*trackSize,
		"a reserved track is still written out: the drive reads it even if the filesystem does not")

	// The first file after a four-block directory, and on the track after the
	// reserved one.
	entry := image[dskHeaderSize+trackSize+dskTrackHeader:]
	require.Equal(t, "DISK    ", string(entry[1:9]))
	require.Equal(t, byte(g.dirBlocks), entry[16])

	// 175 blocks and not 176: the last file that fits ends inside the disk.
	_, err = Plus3Disk([]File{
		{Name: "FILLER.BIN", Data: make([]byte, (175-g.dirBlocks)*g.blockSize)},
	})
	require.NoError(t, err, "175 blocks less the directory should fit exactly")

	_, err = Plus3Disk([]File{
		{Name: "FILLER.BIN", Data: make([]byte, (175-g.dirBlocks)*g.blockSize+1)},
	})
	require.ErrorContains(t, err, "does not fit",
		"the half block at the end of 39 tracks was handed to a file")
}

// 128 names, and the 129th is refused rather than written over the 97th.
func TestThePlus3DirectoryHoldsWhatTheSpecificationSays(t *testing.T) {
	var files []File

	for i := range Plus3Names() {
		files = append(files, File{Name: fmt.Sprintf("F%03d.SCR", i), Data: []byte("x")})
	}

	image, err := Plus3Disk(files)
	require.NoError(t, err, "%d names is what the specification asks for", len(files))

	disk, err := NewDSK(image)
	require.NoError(t, err)

	back, err := disk.Files()
	require.NoError(t, err)
	require.Len(t, back, Plus3Names())

	_, err = Plus3Disk(append(files, File{Name: "ONEMORE.SCR", Data: []byte("x")}))
	require.ErrorContains(t, err, "the directory is full")
}

// The CPC's disk did not change, and this is the test that says so.
//
// DataDisk is the CPC's as well as being what the +3 release used to use, and
// the +3 moving off it must not take the CPC with it: a CPC reads a disk by its
// sector numbering, there is no specification to tell it anything, and a disk
// with a reserved track it is not expecting is a disk it CATs as empty.
func TestTheCPCKeepsTheDataFormat(t *testing.T) {
	require.Equal(t, geometry{
		tracks: 40, sectorsTrack: 9, firstSectorID: 0xC1,
		reserved: 0, blockSize: 1024, dirBlocks: 2,
	}, dataGeometry, "the DATA format is the CPC's and is not ours to change")

	require.False(t, dataGeometry.spec,
		"a DATA disk has nowhere to put a specification: that sector is directory")
	require.Equal(t, 64, dataGeometry.dirEntries())
}
