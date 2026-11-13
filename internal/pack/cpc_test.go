package pack

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

// The CPC build is one file with three things glued into it, and where each
// one lands is not ours to choose.
//
// The database's address is the linker's: every pointer inside it was written
// as an absolute address from 0x2880, so a database that lands anywhere else
// is a database every pointer of which is wrong — and what that looks like is
// not a crash but an adventure with the wrong words in it. The interpreter's
// is its own, because it is not relocatable. So the padding between them is
// load-bearing.
func TestWhereTheCPCBuildPutsEachPiece(t *testing.T) {
	database := make([]byte, 10000)
	for i := range database {
		database[i] = 0x5A
	}

	r, err := CPCRelease(database, Options{Language: language.LanguageSpanish, Name: "AO"})
	require.NoError(t, err)

	one, err := r.Single()
	require.NoError(t, err, "a CPC release is one file: the gluing happens before the disk")
	require.Equal(t, "AO.BIN", one.Name)

	body := one.Body[media.AMSDOSHeaderSize:]

	// The database starts where the linker wrote its pointers for.
	at := int(arch.CPC.BaseAddress("")) - cpcInterpreterOrigin
	require.Equal(t, database, body[at:at+len(database)],
		"the database is not where the linker expects it")

	// And the interpreter is told how much graphics follows, in two bytes of
	// its own. A zero there is an interpreter that draws nothing.
	graphics := int(body[cpcGraphicsLengthAt]) | int(body[cpcGraphicsLengthAt+1])<<8
	require.Positive(t, graphics, "the interpreter was not told the graphics length")
	require.Equal(t, len(body), at+len(database)+graphics,
		"the length it was told is not the length that is there")
}

// The header AMSDOS reads before it reads anything else.
func TestTheCPCBuildCarriesAnAMSDOSHeader(t *testing.T) {
	r, err := CPCRelease(make([]byte, 1000),
		Options{Language: language.LanguageSpanish, Name: "AO"})
	require.NoError(t, err)

	one, err := r.Single()
	require.NoError(t, err)

	header := one.Body[:media.AMSDOSHeaderSize]

	require.Equal(t, "AO      ", string(header[1:9]))
	require.Equal(t, "BIN", string(header[9:12]))
	require.Equal(t, byte(2), header[18], "type 2 is a binary")
	require.Equal(t, cpcInterpreterOrigin, int(header[21])|int(header[22])<<8, "load address")
	require.Equal(t, cpcInterpreterOrigin, int(header[26])|int(header[27])<<8, "entry address")
	require.Equal(t, len(one.Body)-media.AMSDOSHeaderSize,
		int(header[24])|int(header[25])<<8, "declared length")

	// Taking it off again gives back exactly the body, which is what AMSDOS
	// does for whatever reads the file.
	require.Equal(t, one.Body[media.AMSDOSHeaderSize:], media.WithoutAMSDOSHeader(one.Body))
}

// A build that would reach the firmware's variables is refused.
//
// Above them is the video memory, and a build that runs into either does not
// run. MCRF checks the same number, and saying so with the numbers beats
// letting somebody find out on the machine.
func TestACPCBuildTooBigIsRefused(t *testing.T) {
	_, err := CPCRelease(make([]byte, 40000),
		Options{Language: language.LanguageSpanish, Name: "AO"})

	require.ErrorContains(t, err, "firmware")
	require.ErrorContains(t, err, "A6FC")
}

// And the disk is the binary on a disk, which is all a CPC release is.
func TestTheCPCDiskHoldsTheBuild(t *testing.T) {
	opts := Options{Language: language.LanguageSpanish, Name: "AO"}

	image, err := CPCDisk(make([]byte, 5000), opts)
	require.NoError(t, err)

	disk, err := media.NewDSK(image)
	require.NoError(t, err)

	files, err := disk.Files()
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "AO.BIN", files[0].Name)

	r, err := CPCRelease(make([]byte, 5000), opts)
	require.NoError(t, err)

	one, err := r.Single()
	require.NoError(t, err)
	require.Equal(t, one.Body, files[0].Data[:len(one.Body)],
		"what came off the disk is not what was put on it")
}
