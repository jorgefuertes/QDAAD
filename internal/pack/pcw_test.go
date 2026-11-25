package pack

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
	"github.com/stretchr/testify/require"
)

func pcwOptions() Options {
	return Options{
		Language: language.LanguageSpanish,
		Name:     "AO1",
		Note:     func(Notice, string) {},
	}
}

// The four files DAAD READY's own PCW.BAT ends up with, under its names.
//
// They are not names anybody is free to choose: the interpreter builds them
// itself — PARTE001 and the extensions it carries in a table at 0x267C — and
// DAAD.COM is what the player types once CP/M is up.
func TestAPCWReleaseIsTheFourFilesPCWBATWrites(t *testing.T) {
	release, err := PCWRelease(make([]byte, 5000), pcwOptions())
	require.NoError(t, err)

	named := map[string][]byte{}
	for _, f := range release.Files() {
		named[f.Name] = f.Body
	}

	require.Len(t, named, 4)

	for _, name := range []string{"DAAD.COM", "PARTE001.DDB", "PARTE001.CHR", "PARTE001.DAT"} {
		require.Contains(t, named, name)
		require.NotEmpty(t, named[name], "%s is there and empty", name)
	}

	require.Len(t, named["PARTE001.DDB"], 5000, "the database goes on as it came")
}

// The character set is 2048 glyphs behind a 128-byte AMSDOS header, which is
// what SPECFORM leaves and what the one on Cozumel's own 1989 disk carries:
// type 2, loading nowhere, declaring 0x800, and a checksum of the 67 bytes in
// front of it.
func TestThePCWCharacterSetCarriesTheHeaderSPECFORMPutsOnIt(t *testing.T) {
	release, err := PCWRelease(make([]byte, 5000), pcwOptions())
	require.NoError(t, err)

	var charset []byte

	for _, f := range release.Files() {
		if f.Name == "PARTE001.CHR" {
			charset = f.Body
		}
	}

	require.Len(t, charset, media.AMSDOSHeaderSize+charsetSize)

	require.Equal(t, byte(2), charset[18], "a binary")
	require.Equal(t, charsetSize, int(charset[24])|int(charset[25])<<8, "declaring its length")

	sum := 0
	for _, b := range charset[:67] {
		sum += int(b)
	}

	require.Equal(t, sum&0xFFFF, int(charset[67])|int(charset[68])<<8, "and a checksum that adds up")
}

// An adventure with texts outside its database carries them under the name the
// interpreter holds as a file control block at 0x05EA, which is the +3's name
// as well.
func TestAPCWReleaseCarriesItsXMessages(t *testing.T) {
	opts := pcwOptions()
	opts.XMessages = make([]byte, 300)

	release, err := PCWRelease(make([]byte, 5000), opts)
	require.NoError(t, err)

	var found bool

	for _, f := range release.Files() {
		if f.Name == "0.XMB" {
			found = true

			require.Len(t, f.Body, 300)
		}
	}

	require.True(t, found, "the texts are not on the disk")
}

// An adventure with drawings puts them on the disk, and one without still
// carries the file: PICTURE is a condition in version 3, so an interpreter that
// cannot open the archive fails every one of them and takes the rest of each
// entry with it.
func TestAPCWReleaseCarriesItsPictures(t *testing.T) {
	flat := append([]byte{512 & 0xFF, 512 >> 8, 112, 0}, make([]byte, 512/8*112)...)

	opts := pcwOptions()
	opts.Images = map[int][]byte{3: flat, 27: flat}

	release, err := PCWRelease(make([]byte, 5000), opts)
	require.NoError(t, err)

	var archive []byte

	for _, f := range release.Files() {
		if f.Name == pcwPart+pcwPictures {
			archive = f.Body
		}
	}

	require.NotEmpty(t, archive, "there is no archive of illustrations on the disk")
	require.Equal(t, byte(2), archive[4], "the archive does not count two pictures")

	// And without any, the empty one the interpreter still opens.
	release, err = PCWRelease(make([]byte, 5000), pcwOptions())
	require.NoError(t, err)

	for _, f := range release.Files() {
		if f.Name == pcwPart+pcwPictures {
			require.Equal(t, byte(0), f.Body[4], "an adventure with no drawings counts none")
		}
	}
}

// The disk is the CPC's DATA format, and what goes on it comes back off it.
//
// Not a formality: the PCW editions of 1989 and the PCW.DSK every DAAD READY
// release starts from are both 40 tracks of 9 sectors with their IDs from
// 0xC1, which is why this needs no format of its own.
func TestAPCWDiskGivesBackWhatWentOnIt(t *testing.T) {
	want, err := PCWRelease(make([]byte, 5000), pcwOptions())
	require.NoError(t, err)

	image, err := PCWDisk(make([]byte, 5000), pcwOptions())
	require.NoError(t, err)

	disk, err := media.NewDSK(image)
	require.NoError(t, err)

	files, err := disk.Files()
	require.NoError(t, err)

	off := map[string][]byte{}
	for _, f := range files {
		off[f.Name] = f.Data
	}

	for _, f := range want.Files() {
		back, ok := off[f.Name]
		require.True(t, ok, "%s did not come back off the disk", f.Name)
		require.Equal(t, f.Body, back[:len(f.Body)], "%s came back changed", f.Name)
	}
}

// A build with nothing in it is refused rather than written as a disk with an
// interpreter and no adventure.
func TestAPCWReleaseWantsADatabase(t *testing.T) {
	_, err := PCWRelease(nil, pcwOptions())
	require.ErrorContains(t, err, "no database")
}

// A disk asked to boot carries CP/M, our chain program and the loader track.
//
// The loader is the part worth checking byte for byte. It is Amstrad's, copied
// off the CP/M disk DAAD READY ships, and the first sixteen bytes of it are the
// specification that tells the machine the disk's shape — so a disk of ours
// that wrote its own shape under somebody else's specification would be a disk
// that lies about itself, and the machine believes the specification.
func TestAPCWDiskThatStartsOnItsOwn(t *testing.T) {
	opts := pcwOptions()
	opts.Boot = true

	image, err := PCWDisk(make([]byte, 5000), opts)
	require.NoError(t, err)

	disk, err := media.NewDSK(image)
	require.NoError(t, err)

	files, err := disk.Files()
	require.NoError(t, err)

	names := map[string]bool{}
	for _, f := range files {
		names[f.Name] = true
	}

	for _, want := range []string{pcwInterpreter, pcwSystem, pcwSubmit, pcwProfile} {
		require.True(t, names[want], "%s is not on a disk that has to start on its own", want)
	}

	// Track 0 is the loader, verbatim and in sector order.
	beside, err := runtime.Companions(runtime.PCW)
	require.NoError(t, err)

	boot := beside["CPMBOOT.BIN"]
	require.NotEmpty(t, boot)

	require.Equal(t, boot, disk.Payload()[:len(boot)],
		"the boot track is not the one that was read off Amstrad's disk")
}

// And without it, nothing of CP/M is on the disk: a release is still the four
// files of 1989 unless it is asked for the other thing.
func TestAPCWDiskDoesNotCarryCPMUnlessAsked(t *testing.T) {
	image, err := PCWDisk(make([]byte, 5000), pcwOptions())
	require.NoError(t, err)

	disk, err := media.NewDSK(image)
	require.NoError(t, err)

	files, err := disk.Files()
	require.NoError(t, err)

	for _, f := range files {
		require.NotEqual(t, pcwSystem, f.Name)
		require.NotEqual(t, pcwSubmit, f.Name)
	}
}
