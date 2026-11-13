package media

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// A TZX of ours is a TAP with a signature and a pause after each block, so the
// one thing worth proving is that it is still the same tape: unwrap it and the
// blocks have to come back byte for byte, or the two files do not load alike.
func TestATZXIsTheSameTapeWrapped(t *testing.T) {
	var tape TAP

	tape.Program("loader", []byte{0x00, 0x0A}, FirstLine)
	require.NoError(t, tape.Code("screen", 0x4000, make([]byte, 6912)))
	require.NoError(t, tape.Code("code", 0x8000, []byte{1, 2, 3}))

	tap := tape.Bytes()

	out, err := TZX(tap)
	require.NoError(t, err)

	require.Equal(t, tzxSignature, string(out[:8]))
	require.Equal(t, []byte{tzxMajor, tzxMinor}, out[8:10])

	var unwrapped []byte

	blocks := 0

	for at := 10; at < len(out); blocks++ {
		require.Equal(t, byte(tzxStandardBlock), out[at], "block %d is not a standard one", blocks)
		require.Equal(t, uint16(tzxPause), binary.LittleEndian.Uint16(out[at+1:]))

		length := int(binary.LittleEndian.Uint16(out[at+3:]))
		at += 5

		unwrapped = word(unwrapped, length)
		unwrapped = append(unwrapped, out[at:at+length]...)
		at += length
	}

	require.Equal(t, 6, blocks, "three things saved is six blocks, header and body each")
	require.Equal(t, tap, unwrapped)
}

// A tape that ends in the middle of a block is not a tape, and saying so beats
// writing a TZX that claims a length nobody can honour.
func TestATruncatedTapeIsRefused(t *testing.T) {
	var tape TAP

	require.NoError(t, tape.Code("code", 0x8000, []byte{1, 2, 3}))

	full := tape.Bytes()

	for _, cut := range []int{1, len(full) - 1} {
		_, err := TZX(full[:cut])
		require.Error(t, err, "cut to %d bytes", cut)
	}
}

// An empty tape is a valid thing to write: a signature and nothing after it.
func TestAnEmptyTapeIsJustTheSignature(t *testing.T) {
	out, err := TZX(nil)
	require.NoError(t, err)
	require.Len(t, out, 10)
}
