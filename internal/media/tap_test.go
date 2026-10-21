package media

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A tape block is its length, a flag, the bytes, and every byte before the last
// one exclusive-ored together. The ROM checks that last byte and refuses the
// block if it disagrees, which is the whole of the format's error detection and
// the one thing here that cannot be eyeballed.
func TestEveryBlockCarriesItsOwnParity(t *testing.T) {
	var tape TAP

	require.NoError(t, tape.Code("x", 0x8000, []byte{0x01, 0x02}))

	for _, block := range split(t, tape.Bytes()) {
		parity := byte(0)
		for _, b := range block[:len(block)-1] {
			parity ^= b
		}

		require.Equal(t, block[len(block)-1], parity)
	}
}

// What the ROM reads before a block: what it is, what it is called, how long it
// is, and where it goes. Ten characters of name, padded, because that is all
// the room there is.
func TestACodeHeaderSaysWhereTheBytesGo(t *testing.T) {
	var tape TAP

	require.NoError(t, tape.Code("interp", 0x6000, make([]byte, 7155)))

	header := split(t, tape.Bytes())[0]

	require.Equal(t, byte(0x00), header[0], "a header block")
	require.Equal(t, byte(tapCode), header[1], "of a CODE block")
	require.Equal(t, "interp    ", string(header[2:12]), "padded to ten")
	require.Equal(t, 7155, int(header[12])|int(header[13])<<8)
	require.Equal(t, 0x6000, int(header[14])|int(header[15])<<8)

	// A name longer than the header has room for is cut rather than refused.
	tape = TAP{}
	require.NoError(t, tape.Code("una aventura larga", 0, nil))
	require.Equal(t, "una avent", string(split(t, tape.Bytes())[0][2:11]))
}

// Nothing can be written past the top of a 64K address space, and asking to is
// a mistake worth refusing: the ROM would wrap it round to zero and load the
// rest of the block over the interrupt vectors.
func TestABlockThatWouldRunPastTheTopIsRefused(t *testing.T) {
	var tape TAP

	require.Error(t, tape.Code("x", 0xFF00, make([]byte, 0x200)))
	require.Error(t, tape.Code("x", -1, nil))
	require.NoError(t, tape.Code("x", 0xFF00, make([]byte, 0x100)), "ending on the last byte is fine")
}

// A loader is a program that runs itself, and that is the autostart line rather
// than anything in the program. Saved to start at a line the program does not
// have, it loads and sits there; saved to start at the wrong one, it skips the
// statements before it — which on a loader means loading blocks into memory
// BASIC still thinks is its own.
func TestALoaderStartsAtItsFirstLine(t *testing.T) {
	var basic BASIC

	basic.Clear(0x5FFF)
	basic.LoadCode(3)
	basic.RandomizeUSR(0x6000)

	var tape TAP

	tape.Program("loader", basic.Bytes(), FirstLine)

	header := split(t, tape.Bytes())[0]

	require.Equal(t, byte(tapProgram), header[1])
	require.Equal(t, FirstLine, int(header[14])|int(header[15])<<8, "the autostart line")
	require.Equal(t, len(basic.Bytes()), int(header[16])|int(header[17])<<8,
		"a program with no variables: they would start where it ends")
}

// split walks a tape back into its blocks, so that a test reads what the ROM
// would read rather than an offset counted by hand.
func split(t *testing.T, tape []byte) [][]byte {
	t.Helper()

	var blocks [][]byte

	for at := 0; at < len(tape); {
		require.Less(t, at+2, len(tape), "a length with no block after it")

		length := int(tape[at]) | int(tape[at+1])<<8
		at += 2

		require.LessOrEqual(t, at+length, len(tape), "a block shorter than its length")

		blocks = append(blocks, tape[at:at+length])
		at += length
	}

	return blocks
}
