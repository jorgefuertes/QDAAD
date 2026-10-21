package pack

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/stretchr/testify/require"
)

// The tape is a loader and four blocks, and every one of the four has to land
// where the interpreter expects it. Three of the addresses are not ours to
// pick: the database's was written into its own pointers by the linker, and the
// stub and the index are pinned to the top of memory.
func TestTheTapeLandsEveryPieceWhereItBelongs(t *testing.T) {
	tape, err := ZX48(make([]byte, 2000), language.LanguageSpanish, "prueba")
	require.NoError(t, err)

	var at []int

	var length []int

	for _, block := range split(t, tape) {
		if block[0] != 0x00 || block[1] != 3 {
			continue // not a CODE header
		}

		length = append(length, int(block[12])|int(block[13])<<8)
		at = append(at, int(block[14])|int(block[15])<<8)
	}

	require.Len(t, at, 4, "the interpreter, the database, the index and the stub")

	require.Equal(t, 0x6000, at[0], "the interpreter is not relocatable")
	require.Equal(t, 0x8400, at[1], "and the database's pointers were written from here")
	require.Equal(t, 2000, length[1])

	// The last byte of the stub is the last byte of RAM. The interpreter reads
	// fixed addresses inside those final bytes, so this is not a convention
	// that could be changed.
	require.Equal(t, 0x10000, at[3]+length[3])

	// And the index sits immediately under it, because it is read downwards
	// from its own last byte.
	require.Equal(t, at[3], at[2]+length[2])
}

// The loader has to load every block it was given and no more. One LOAD too
// few leaves a block on the tape and a hole in memory; one too many stops the
// machine at a tape that has already ended.
func TestTheLoaderLoadsAsManyBlocksAsThereAre(t *testing.T) {
	tape, err := ZX48(make([]byte, 2000), language.LanguageSpanish, "prueba")
	require.NoError(t, err)

	blocks := split(t, tape)

	program := blocks[1]
	loads := 0

	for i := range len(program) - 1 {
		if program[i] == tokenLOAD && program[i+1] == '"' {
			loads++
		}
	}

	codeBlocks := 0

	for _, block := range blocks {
		if block[0] == 0x00 && block[1] == 3 {
			codeBlocks++
		}
	}

	require.Equal(t, codeBlocks, loads)
	require.Equal(t, 4, loads)
}

// tokenLOAD, repeated here rather than exported from media, because what this
// test is checking is the loader's shape and not the token's value.
const tokenLOAD = 0xEF

// A database too big to fit is the one failure a real adventure will meet, and
// it has to be said rather than written out as a tape that loads over its own
// index and dies without explaining itself.
func TestADatabaseThatDoesNotFitIsRefused(t *testing.T) {
	_, err := ZX48(make([]byte, 0x8000), language.LanguageSpanish, "prueba")

	require.ErrorContains(t, err, "is all there is")
}

// The index is read downwards from its last byte, which is why it is written
// backwards. A build with no pictures still carries one: without it the
// interpreter reads uninitialised memory the first time a PICTURE runs, and the
// official template runs one before printing anything.
func TestTheEmptyIndexStillSaysThereAreNoPictures(t *testing.T) {
	index := imageIndex(0x8400 + 2000)

	// Read the way the interpreter reads it: from the end, backwards.
	reversed := make([]byte, len(index))
	for i, b := range index {
		reversed[len(index)-1-i] = b
	}

	require.Equal(t, []byte{
		0xC0, 0x00, // picture data would go at 0xC000
		0x00, 0x00, // and there is none of it
		0xFF, // no more pages
		0xFF, // no more pictures
	}, reversed)

	// Once the database runs past 0xC000 the pictures go after it instead, so
	// that the two never overlap.
	index = imageIndex(0xD000)
	require.Equal(t, []byte{0xD0, 0x01}, []byte{index[5], index[4]},
		"0xD001, high byte first, as the table is read")
}

func split(t *testing.T, tape []byte) [][]byte {
	t.Helper()

	var blocks [][]byte

	for at := 0; at < len(tape); {
		require.Less(t, at+2, len(tape))

		length := int(tape[at]) | int(tape[at+1])<<8
		at += 2

		require.LessOrEqual(t, at+length, len(tape))

		blocks = append(blocks, tape[at:at+length])
		at += length
	}

	return blocks
}
