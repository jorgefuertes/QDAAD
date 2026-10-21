package media

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A line is its number as a big-endian word — the one place this machine
// writes a word that way round — then the length of the rest as a little-endian
// one, then the statement and a carriage return.
func TestALineIsNumberedBackwardsFromEverythingElse(t *testing.T) {
	var b BASIC

	b.LoadCode()

	out := b.Bytes()

	require.Equal(t, []byte{0x00, FirstLine}, out[0:2], "the number, high byte first")
	require.Equal(t, 5, int(out[2])|int(out[3])<<8, "LOAD, two quotes, CODE, and the return")
	require.Equal(t, []byte{tokLOAD, '"', '"', tokCODE, 0x0D}, out[4:])
}

// Lines come out in the order they were added and numbered upwards from the
// first, which is what makes the autostart line enough to run all of them.
func TestLinesAreNumberedInOrder(t *testing.T) {
	var b BASIC

	b.Clear(0x5FFF)
	b.LoadCode()
	b.LoadCode()
	b.RandomizeUSR(0x6000)

	var numbers []int

	for at := 0; at < len(b.Bytes()); {
		out := b.Bytes()
		numbers = append(numbers, int(out[at])<<8|int(out[at+1]))
		at += 4 + int(out[at+2]) | int(out[at+3])<<8
	}

	require.Equal(t, []int{10, 20, 30, 40}, numbers)
	require.Equal(t, FirstLine, numbers[0], "and the first is where it starts")
}

// A number is written twice: the digits a person reads and the five bytes the
// interpreter evaluates. Leaving the second out is the interesting mistake,
// because the program still lists correctly and every number in it is zero.
func TestANumberIsWrittenTwiceOverAndTheSecondIsTheRealOne(t *testing.T) {
	var b BASIC

	b.Clear(24575)

	out := b.Bytes()
	statement := out[4 : len(out)-1]

	require.Equal(t, byte(tokCLEAR), statement[0])
	require.Equal(t, "24575", string(statement[1:6]), "what it lists as")
	require.Equal(t, []byte{0x0E, 0x00, 0x00, 0xFF, 0x5F, 0x00}, statement[6:],
		"and what it evaluates to: 0x5FFF, little-endian, in the small-integer form")
}
