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

// Taking the header off again, which is what +3DOS does for whatever reads a
// file and so what a reader of ours has to do too.
//
// A database with a header in front of it is a database whose every pointer is
// 128 bytes out, and what that looks like is not a clear failure but an
// adventure with the wrong words in it.
func TestTakingThePlus3HeaderBackOff(t *testing.T) {
	body := []byte("the file itself")

	with, err := Plus3Code(0x8400, body)
	require.NoError(t, err)
	require.Equal(t, body, WithoutPlus3Header(with))

	// A file that has none is left alone, which is every file on a CPC disk.
	require.Equal(t, body, WithoutPlus3Header(body))

	// And neither is a file that merely starts with the right eight letters.
	// Taking 128 bytes off on that evidence alone would be worse than leaving
	// it: the checksum is what makes it a header.
	fake := append([]byte("PLUS3DOS"), make([]byte, 200)...)
	fake[Plus3HeaderSize-1] ^= 0xFF
	require.Equal(t, fake, WithoutPlus3Header(fake))
}

// The end of the file is not trimmed, and this is the measurement that says so.
//
// CP/M pads the last record of a file to 128 bytes, so the declared length
// looks like the right place to cut. On Cozumel's Spectrum disk it is not:
// D1.BIN declares 40443 bytes and its graphics block runs to 40448, so the
// drawings are in what the header calls padding. Cutting to the declared
// length finds no drawings at all.
//
// This test exists so that trimming — which reads as an obvious improvement —
// cannot be put back without the reason it was left out.
func TestTheDeclaredLengthIsNotWhereAFileIsCut(t *testing.T) {
	body := []byte("the file itself")

	with, err := Plus3Code(0x8400, body)
	require.NoError(t, err)

	// Padded the way CP/M pads it, to a whole number of 128-byte records.
	padded := append(with, make([]byte, 5)...)

	require.Len(t, WithoutPlus3Header(padded), len(body)+5,
		"the padding was trimmed off, and on at least one real disk the drawings are in it")
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
