package pack

import (
	"fmt"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/zx0"
	"github.com/stretchr/testify/require"
)

// What will not fit in memory goes on the disk, and the index says so.
//
// This is the whole of the arrangement as the Z80 sees it: find walks the index
// and comes back with a configuration, and a configuration of 0xFF is not a
// bank at all but a file to be opened. What this asks is that every picture is
// in one place or the other, that the file is called what the interpreter will
// ask for, and that the bytes in it are the entry the bank would have held.
func TestWhatDoesNotFitInMemoryGoesOnTheDisc(t *testing.T) {
	opts := cpcBankOptions(40)
	opts.Memory = 64

	bank, err := cpcPictureBank(opts, 0x6000, true)
	require.NoError(t, err)

	require.NotEmpty(t, bank.carried, "some of them fit in memory")
	require.NotEmpty(t, bank.onDisc, "and the rest went on the disk")
	require.Empty(t, bank.left, "so none of them is simply missing")

	index := entries(bank)
	require.Len(t, index, len(opts.Images), "every picture is named")

	// The ones on the disk are named with the configuration that means "not in
	// memory", which is not one the gate array has.
	onDisc := map[string]File{}
	for _, f := range bank.onDisc {
		onDisc[f.Name] = f
	}

	discs := 0

	for number := range opts.Images {
		if index[number]>>16 != cpcDiscPicture {
			continue
		}

		discs++

		name := fmt.Sprintf("%03d.CPC", number)
		file, ok := onDisc[name]
		require.True(t, ok, "picture %03d is said to be on the disk and is not there", number)

		// And what is in it is the entry, behind an AMSDOS header: the number,
		// the length, the inks and the eight streams.
		entry := media.WithoutAMSDOSHeader(file.Body)
		require.Equal(t, byte(number), entry[0], "the entry says which picture it is")
		require.Equal(t, len(entry), int(entry[1])|int(entry[2])<<8, "and how long it is")

		at := cpcBankHeader
		for band := range cpcPackBands {
			size := int(entry[at]) | int(entry[at+1])<<8
			at += 2

			_, err := zx0.Decompress(entry[at : at+size])
			require.NoError(t, err, "picture %03d, band %d does not unpack", number, band)

			at += size
		}

		require.Equal(t, len(entry), at, "the streams do not come to the length it says")
	}

	require.Equal(t, len(bank.onDisc), discs, "the index names every file it put on the disk")
}

// A tape has no disk to go and get them from, so there what does not fit is
// what did not fit: left, and pointed at the stand-in.
func TestATapeHasNoDiscToFallBackOn(t *testing.T) {
	opts := cpcBankOptions(40)
	opts.Memory = 64
	opts.Generic = 2

	bank, err := cpcPictureBank(opts, 0x6000, false)
	require.NoError(t, err)

	require.Empty(t, bank.onDisc)
	require.NotEmpty(t, bank.left)
	require.Equal(t, 2, bank.standIn)
}

// The buffer a picture is read into sits between the game and the code that
// reads it, and a game that leaves no room for it is told rather than left to
// unpack over the index that found it.
func TestAPictureReadFromTheDiscNeedsSomewhereToLand(t *testing.T) {
	opts := cpcBankOptions(40)
	opts.Memory = 128

	// Far enough up that the banks still hold what they hold and the buffer has
	// nowhere to be, which is the one thing the banks cannot help with.
	_, err := cpcPictureBank(opts, cpcResidentAt()-0x100, true)
	require.ErrorContains(t, err, "read from the disk")

	// And with room for it, the same build is fine.
	_, err = cpcPictureBank(opts, 0x6000, true)
	require.NoError(t, err)
}
