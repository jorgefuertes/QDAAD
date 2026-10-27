package media

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The header a +3 file carries, which is what tells the machine where a file
// goes.
//
// A tape block says that in the bytes before it; a disk file has nowhere to
// say it, so +3DOS puts the same eight bytes a tape header held into a header
// of its own and hides them from the program. Getting any of it wrong is not a
// readable failure: LOAD "x"CODE puts the bytes somewhere else, or refuses the
// file as not a file, and the game does neither.
func TestThePlus3HeaderSaysWhereAFileGoes(t *testing.T) {
	body := []byte("machine code, pretend")

	out, err := Plus3Code(0x6000, body)
	require.NoError(t, err)
	require.Len(t, out, Plus3HeaderSize+len(body))

	require.Equal(t, "PLUS3DOS", string(out[:8]))
	require.Equal(t, byte(0x1A), out[8], "the soft end-of-file, so that TYPE stops here")
	require.Equal(t, byte(1), out[9], "issue")

	// The length counts the header as well as the body, over four bytes.
	require.Equal(t, Plus3HeaderSize+len(body),
		int(out[11])|int(out[12])<<8|int(out[13])<<16|int(out[14])<<24)

	// The eight bytes a tape header would have held.
	require.Equal(t, byte(3), out[15], "type 3 is CODE")
	require.Equal(t, len(body), int(out[16])|int(out[17])<<8, "the body's own length")
	require.Equal(t, 0x6000, int(out[18])|int(out[19])<<8, "the address it loads at")

	require.Equal(t, body, out[Plus3HeaderSize:], "the body did not come through")
}

// A program carries the line to start at, and that line matters more than it
// looks: saved to start anywhere but its first, a loader sits at a LOAD it was
// supposed to have set up for.
func TestThePlus3HeaderOfAProgramCarriesItsStartLine(t *testing.T) {
	out, err := Plus3Program(FirstLine, []byte("tokens"))
	require.NoError(t, err)

	require.Equal(t, byte(0), out[15], "type 0 is a BASIC program")
	require.Equal(t, FirstLine, int(out[18])|int(out[19])<<8, "the autostart line")
}

// And the checksum, which +3DOS checks before it will read the file at all.
func TestThePlus3HeaderChecksums(t *testing.T) {
	out, err := Plus3Code(0x8400, []byte("x"))
	require.NoError(t, err)

	sum := 0
	for _, b := range out[:Plus3HeaderSize-1] {
		sum += int(b)
	}

	require.Equal(t, byte(sum), out[Plus3HeaderSize-1])
}

func TestAnAddressOutsideASpectrumIsRefused(t *testing.T) {
	_, err := Plus3Code(0x10000, []byte("x"))
	require.ErrorContains(t, err, "not an address")
}

// The disk machines load by name, which is the whole difference from a tape.
//
// A tape has no names and no choice: the blocks come in the order they were
// written and LOAD ""CODE takes the next one. Naming them is what lets a disk
// hold two parts of an adventure without one loading over the other.
func TestLoadingByName(t *testing.T) {
	var b BASIC

	b.LoadNamed("DAAD.BIN", "PART1.DDB")

	// Line number big-endian, then the length little-endian, then the body.
	require.Equal(t, []byte{0x00, 0x0A}, b.Bytes()[:2])

	body := b.Bytes()[4:]
	require.Equal(t, []byte{
		tokLOAD, '"', 'D', 'A', 'A', 'D', '.', 'B', 'I', 'N', '"', tokCODE,
		':',
		tokLOAD, '"', 'P', 'A', 'R', 'T', '1', '.', 'D', 'D', 'B', '"', tokCODE,
		0x0D,
	}, body)

	// No address anywhere in it: where a file goes is the file's own business,
	// so the loader cannot contradict the linker.
	require.NotContains(t, string(body), "CODE 2", "the loader named an address")
}
