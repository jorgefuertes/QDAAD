package pack

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// entries reads the bank's index the way the Z80 walks it: four bytes a
// picture until the number is 0xFF.
func entries(bank cpcBank) map[int]int {
	out := map[int]int{}

	index := bank.resident[bank.entriesAt:]
	for at := 0; index[at] != cpcIndexEnd; at += cpcIndexRecord {
		out[int(index[at])] = int(index[at+1])<<16 | int(index[at+2]) | int(index[at+3])<<8
	}

	return out
}

// Every picture the adventure can ask for is in the index, whether its own
// drawing fitted or not.
//
// This is the whole of what the stand-in buys. PICTURE is a condition in
// version 3: one that is not in the index fails it, and a failed condition
// abandons the rest of the entry — which in La Aventura Original is the text of
// the location and the WINDOW that puts the text back where the text goes.
func TestEveryPictureIsInTheIndexEvenWhenItDoesNotFit(t *testing.T) {
	opts := cpcBankOptions(40)
	opts.Memory = 64
	opts.Generic = 2

	bank, err := cpcPictureBank(opts, 0x6000, false)
	require.NoError(t, err)
	require.NotEmpty(t, bank.left, "forty pictures do not fit in a 64K machine")
	require.Equal(t, 2, bank.standIn)

	index := entries(bank)
	require.Len(t, index, len(opts.Images), "every picture is named")

	// And the ones that did not fit are pointed at the stand-in's own bytes.
	for _, number := range bank.left {
		require.Equal(t, index[2], index[number],
			"picture %03d does not fall back on the stand-in", number)
	}

	for _, number := range bank.carried {
		if number != 2 {
			require.NotEqual(t, index[2], index[number],
				"picture %03d fitted and should be its own drawing", number)
		}
	}
}

// The stand-in is laid out first, so that it is never the one that does not
// fit. Asked for one that is not among the pictures, there is simply none.
func TestTheStandInIsNeverTheOneLeftOut(t *testing.T) {
	opts := cpcBankOptions(40)
	opts.Memory = 64
	opts.Generic = 39

	bank, err := cpcPictureBank(opts, 0x6000, false)
	require.NoError(t, err)
	require.Contains(t, bank.carried, 39, "the stand-in travels whatever its number")
	require.NotContains(t, bank.left, 39)

	opts.Generic = 0

	bank, err = cpcPictureBank(opts, 0x6000, false)
	require.NoError(t, err)
	require.Zero(t, bank.standIn)
	require.Len(t, entries(bank), len(bank.carried), "without one, what did not fit is not named")
}

// Asking for a stand-in that the adventure does not have leaves things as they
// were rather than pointing every miss at nothing.
func TestAStandInThatIsNotThereIsNoStandIn(t *testing.T) {
	opts := cpcBankOptions(40)
	opts.Memory = 64
	opts.Generic = 200

	bank, err := cpcPictureBank(opts, 0x6000, false)
	require.NoError(t, err)
	require.Zero(t, bank.standIn)
	require.Len(t, entries(bank), len(bank.carried))
}
