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

	b.LoadCode(1)

	out := b.Bytes()

	require.Equal(t, []byte{0x00, FirstLine}, out[0:2], "the number, high byte first")
	require.Equal(t, 6, int(out[2])|int(out[3])<<8, "a space, four tokens and the return")
	require.Equal(t, []byte{' ', tokLOAD, '"', '"', tokCODE, 0x0D}, out[4:])
}

// Several LOADs go on one line, separated by colons. One per line would work
// just as well and is not what the reference writes.
func TestEveryBlockIsLoadedFromTheSameLine(t *testing.T) {
	var b BASIC

	b.LoadCode(3)

	require.Equal(t,
		[]byte{
			' ',
			tokLOAD, '"', '"', tokCODE, ':',
			tokLOAD, '"', '"', tokCODE, ':',
			tokLOAD, '"', '"', tokCODE,
			0x0D,
		},
		b.Bytes()[4:])
}

// Lines come out in the order they were added and numbered upwards from the
// first, which is what makes the autostart line enough to run all of them.
func TestLinesAreNumberedInOrder(t *testing.T) {
	var b BASIC

	b.Clear(0x5FFF)
	b.LoadCode(3)
	b.RandomizeUSR(0x6000)

	out := b.Bytes()

	var numbers []int

	for at := 0; at < len(out); {
		numbers = append(numbers, int(out[at])<<8|int(out[at+1]))
		at += 4 + int(out[at+2]) | int(out[at+3])<<8
	}

	require.Equal(t, []int{10, 20, 30}, numbers)
	require.Equal(t, FirstLine, numbers[0], "and the first is where it starts")
}

// A number is VAL of a string rather than a number.
//
// Written the way the editor writes it, a number is its digits followed by the
// same value again in five bytes, and it is that second copy the interpreter
// evaluates. VAL skips it: nothing but the digits, evaluated when the line
// runs. Both work; this is the one the reference writes, and a loader has room
// for either.
func TestANumberIsValOfAString(t *testing.T) {
	var b BASIC

	b.Clear(24575)

	out := b.Bytes()

	require.Equal(t,
		append([]byte{tokCLEAR, tokVAL, '"'}, []byte("24575\"\r")...),
		out[4:])
}

// And the whole loader, which is the thing daadmaker carries as a byte array
// and the reason the two tapes come out the same.
func TestTheLoaderIsTheOneTheReferenceWrites(t *testing.T) {
	var b BASIC

	b.Clear(24575)
	b.LoadCode(3)
	b.RandomizeUSR(24576)

	// daadmaker.pas:22-25, with the tape wrapping taken off: the program
	// starts at the twenty-sixth byte of that array and runs to the checksum.
	want := []byte{
		0, 10, 10, 0, 253, 176, 34, 50, 52, 53, 55, 53, 34, 13,
		0, 20, 16, 0, 32, 239, 34, 34, 175, 58, 239, 34, 34, 175, 58, 239, 34, 34, 175, 13,
		0, 30, 11, 0, 249, 192, 176, 34, 50, 52, 53, 55, 54, 34, 13,
	}

	require.Equal(t, want, b.Bytes())
}
