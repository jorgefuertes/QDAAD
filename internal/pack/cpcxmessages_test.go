package pack

import (
	"fmt"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/stretchr/testify/require"
)

// The size the names and the cut are built on is the linker's own, because the
// offsets in the condacts were laid out on it. Two numbers that have to agree
// and live in two places are two numbers until something says so.
func TestTheXMessageSlotIsTheOneTheLinkerLaidThemOut(t *testing.T) {
	require.Equal(t, 2048, back.XMessageSlot(arch.CPC, ""),
		"the interpreter divides the offset by this: 0BAE shifts the high byte right three times")

	// And it is the record the firmware reads into the buffer below the game,
	// which is the reason it is two kilobytes and not some other number.
	require.Equal(t, cpcBufferSize, back.XMessageSlot(arch.CPC, ""))
}

// An adventure with no texts outside its database gets no files, which is every
// adventure that ships here.
func TestAnAdventureWithoutXMessagesGetsNoFiles(t *testing.T) {
	files, err := cpcXMessageFiles(Options{})
	require.NoError(t, err)
	require.Empty(t, files)
}

// The name is the file number in two digits, and the cut is every slot bytes,
// so that a message lands at the position its offset says: the interpreter
// opens offset/2048 and reads from offset%2048.
func TestTheXMessageFilesAreCutWhereTheOffsetsSayTheyAre(t *testing.T) {
	const slot = 2048

	body := make([]byte, 2*slot+100)
	for i := range body {
		body[i] = byte(i)
	}

	files, err := cpcXMessageFiles(Options{XMessages: body})
	require.NoError(t, err)
	require.Len(t, files, 3)

	for i, f := range files {
		require.Equal(t, fmt.Sprintf("%02d.XMB", i), f.Name)

		at := i * slot
		require.Equal(t, body[at:min(at+slot, len(body))], f.Body,
			"file %d does not hold the bytes that offsets %d upwards name", i, at)
	}

	require.Len(t, files[0].Body, slot)
	require.Len(t, files[1].Body, slot)
	require.Len(t, files[2].Body, 100)
}

// Two digits is what the name holds and no more.
func TestTooManyXMessageFilesAreRefused(t *testing.T) {
	_, err := cpcXMessageFiles(Options{XMessages: make([]byte, 100*2048+1)})
	require.ErrorContains(t, err, "the interpreter can name")
}

// A disk carries them, under the names the interpreter builds.
func TestACPCDiskCarriesItsXMessages(t *testing.T) {
	opts := cpcBankOptions(2)
	opts.XMessages = make([]byte, 3000)

	release, err := CPCRelease(nil, opts)
	require.NoError(t, err)

	var named []string
	for _, f := range release.Files() {
		named = append(named, f.Name)
	}

	require.Contains(t, named, "00.XMB")
	require.Contains(t, named, "01.XMB")
}

// And a tape says it cannot, rather than writing one that stops being readable
// at the first XMESSAGE.
func TestACPCTapeRefusesXMessages(t *testing.T) {
	opts := cpcBankOptions(1)
	opts.XMessages = make([]byte, 40)

	_, err := CPCTape(nil, opts)
	require.ErrorContains(t, err, "no disk to read xmessages from")
}
