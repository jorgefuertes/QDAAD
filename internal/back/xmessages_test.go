package back

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/stretchr/testify/require"
)

// decodeXMes reads a message back out of a laid-out file, the way an
// interpreter does: from the offset, until the byte that ends a text.
//
// Doing it this way rather than against the lengths we put in is the point. The
// interpreter is never told how long a message is — it reads a fixed block and
// prints until the terminator — so a file that only reads back correctly when
// you already know the answer is a file that does not read back.
func decodeXMes(t *testing.T, file []byte, offset int) string {
	t.Helper()

	require.Less(t, offset, len(file), "the offset is past the end of the file")

	end := bytes.IndexByte(file[offset:], xmesEndOfText)
	require.GreaterOrEqual(t, end, 0, "the message at %d never ends", offset)

	out := make([]byte, end)
	for i, b := range file[offset : offset+end] {
		out[i] = b ^ obfuscate
	}

	return string(out)
}

// An adventure with no xmessages has no file, and asks for none.
func TestNoXMessagesIsNoFile(t *testing.T) {
	out, err := layoutXMessages(nil, arch.ZX, "plus3")
	require.NoError(t, err)
	require.Empty(t, out.body)
	require.Zero(t, out.size)

	// And a machine that could not have had them is not asked about it: the
	// question only comes up when there is something to place.
	out, err = layoutXMessages(nil, arch.PC, "vga")
	require.NoError(t, err)
	require.Empty(t, out.body)
}

// A +3 file starts with 512 bytes of nothing, and the first message after it.
//
// The gap is the interpreter's buffer for the messages it has to fetch from
// disk, so the first real text is at 512 and not at 0. See xmesGap.
func TestAPlus3FileStartsWithTheGap(t *testing.T) {
	out, err := layoutXMessages([]string{"uno", "dos"}, arch.ZX, "plus3")
	require.NoError(t, err)

	require.Equal(t, make([]byte, xmesGap), out.body[:xmesGap],
		"the first 512 bytes should be the interpreter's buffer")

	require.Equal(t, []int{xmesGap, xmesGap + 4}, out.offsets,
		"the first message goes after the gap and the second after its terminator")

	require.Equal(t, "uno", decodeXMes(t, out.body, out.offsets[0]))
	require.Equal(t, "dos", decodeXMes(t, out.body, out.offsets[1]))

	require.Equal(t, xmesGap+8, out.size)
	require.Len(t, out.body, out.size)
}

// A 48K has neither the gap nor the page swap, so the first message is at zero.
//
// It still gets a size and a layout: whether the machine can read the file is
// the packager's to refuse, and it does. Here it is only arithmetic.
func TestA48KFileStartsAtTheBeginning(t *testing.T) {
	out, err := layoutXMessages([]string{"uno"}, arch.ZX, "48k")
	require.NoError(t, err)

	require.Equal(t, []int{0}, out.offsets)
	require.Equal(t, "uno", decodeXMes(t, out.body, 0))
}

// Every byte is obfuscated, and the terminator is a line feed that has been.
func TestTheBytesAreObfuscatedAndTheTerminatorIsOne(t *testing.T) {
	out, err := layoutXMessages([]string{"A"}, arch.ZX, "48k")
	require.NoError(t, err)

	require.Equal(t, []byte{'A' ^ 0xFF, 0x0A ^ 0xFF}, out.body)
}

// No message begins inside the last 512 bytes of a slot.
//
// Printing one means swapping 512 bytes between pages, so a message starting
// any later would be swapped in short. What the layout does instead is pad the
// slot out and start the next — which is why the file grows in whole slots and
// why the offsets jump.
func TestNoMessageStartsInsideTheLast512BytesOfASlot(t *testing.T) {
	const slot = 16 * 1024

	// Messages big enough to walk up to the top of the slot in a few steps,
	// and one more to be pushed over it.
	long := strings.Repeat("x", 4000)

	out, err := layoutXMessages([]string{long, long, long, long, "pushed"}, arch.ZX, "plus3")
	require.NoError(t, err)

	for i, offset := range out.offsets {
		within := offset % slot

		require.LessOrEqual(t, within+xmesSwap, slot,
			"message %d starts at %d, which is inside the last %d bytes of its slot",
			i, within, xmesSwap)
	}

	// Three of them fit in the first slot and the fourth starts the second,
	// which the offsets say by crossing it exactly: a slot boundary is where
	// an interpreter would open the next file.
	require.Equal(t, []int{xmesGap, 4513, 8514, slot, slot + 4001}, out.offsets)

	for i, want := range []string{long, long, long, long, "pushed"} {
		require.Equal(t, want, decodeXMes(t, out.body, out.offsets[i]), "message %d", i)
	}

	// Padded with zeros rather than cut short: a machine that reads a whole
	// slot from any offset has to have a whole slot there. So the first slot
	// is full length whatever it holds.
	require.Equal(t, make([]byte, slot-12515), out.body[12515:slot],
		"the tail of the first slot should be padding")

	require.Equal(t, slot+4001+len("pushed")+1, len(out.body))
	require.Equal(t, len(out.body), out.size)
}

// A message bigger than a slot has nowhere to go, and says so rather than
// being written across two of them: nothing reads it that way.
func TestAMessageTooBigForASlotIsRefused(t *testing.T) {
	_, err := layoutXMessages([]string{strings.Repeat("x", 3000)}, arch.CPC, "")
	require.ErrorContains(t, err, "a slot of this machine holds")
}

// The offset travels in two bytes of a condact, so that is the ceiling.
func TestXMessagesPastTheOffsetCeilingAreRefused(t *testing.T) {
	// Five slots of 16 KiB is past 0xFFFF, and each of these fills one.
	var bodies []string
	for range 6 {
		bodies = append(bodies, strings.Repeat("x", 16*1024-1024))
	}

	_, err := layoutXMessages(bodies, arch.ZX, "plus3")
	require.ErrorContains(t, err, "the offset a condact carries reaches")
}

// A machine with nowhere to read a file from cannot have xmessages at all.
func TestAMachineWithNoFilesystemIsToldSo(t *testing.T) {
	_, err := layoutXMessages([]string{"uno"}, arch.PC, "vga")
	require.ErrorContains(t, err, "nowhere to read xmessages from")
}

// The slot sizes are the interpreters' own arithmetic and not ours to round.
//
// They are checked as a table because that is what they are: an offset is
// divided by this to find the file and taken modulo it to find the position, so
// a number changed here changes what every compiled offset means.
func TestTheSlotSizesAreTheOnesTheInterpretersUse(t *testing.T) {
	const k = 1024

	for _, tc := range []struct {
		machine   arch.Machine
		subtarget string
		slot      int
	}{
		{arch.ZX, "plus3", 16 * k},
		{arch.ZX, "128k", 16 * k},
		{arch.ZX, "48k", 64 * k},
		{arch.MSX2, "", 16 * k},
		{arch.CPC, "", 2 * k},
		{arch.C64, "", 2 * k},
		{arch.CP4, "", 2 * k},
		{arch.AMIGA, "", 64 * k},
		{arch.ST, "", 64 * k},
		{arch.PC, "vga256", 64 * k},
		// And the ones that cannot: a PC that is not VGA256 has no Maluva to
		// read them with.
		{arch.PC, "vga", 0},
		{arch.PC, "", 0},
	} {
		require.Equal(t, tc.slot, xmesSlot(tc.machine, tc.subtarget),
			"%s %q", tc.machine, tc.subtarget)
	}
}
