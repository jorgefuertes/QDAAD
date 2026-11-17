package media_test

import (
	"os"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

const cpcTape = "../../work/aventuras/La_Aventura_Original/AmstradCPC/" +
	"La aventura original (Cara A).cdt"

// A tape we write has to come out shaped like one the machine wrote. The
// original's first record is the one held against: a header record of 263
// bytes — the sync, one segment of 256, two of checksum and four to close —
// carrying the name at the front.
func TestCDTRecordsLookLikeTheOriginals(t *testing.T) {
	tape, err := media.CDT([]media.TapeFile{{
		Name: "AO1", Data: make([]byte, 100), Load: 0x0840, Entry: 0x0840,
	}})
	require.NoError(t, err)

	require.Equal(t, "ZXTape!\x1a", string(tape[:8]))

	// The first block, past the ten bytes of container header and the one of
	// block identifier.
	at := 10
	require.Equal(t, byte(0x11), tape[at])

	header := tape[at+1 : at+0x13]
	length := int(header[0x0F]) | int(header[0x10])<<8 | int(header[0x11])<<16
	require.Equal(t, 263, length)

	record := tape[at+0x13 : at+0x13+length]
	require.Equal(t, byte(0x2C), record[0], "a header record opens with 0x2C")
	require.Equal(t, "AO1", string(record[1:4]))
	require.Equal(t, []byte{0xFF, 0xFF, 0xFF, 0xFF}, record[259:263])

	// And the fields the firmware reads: one block, the last and the first,
	// a binary, a hundred bytes at 0x0840.
	require.Equal(t, byte(1), record[17], "block number")
	require.Equal(t, byte(0xFF), record[18], "the last block")
	require.Equal(t, byte(0x02), record[19], "a binary")
	require.Equal(t, byte(0xFF), record[24], "the first block")
	require.Equal(t, 100, int(record[20])|int(record[21])<<8, "bytes in this block")
	require.Equal(t, 0x0840, int(record[22])|int(record[23])<<8, "where they go")
	require.Equal(t, 100, int(record[25])|int(record[26])<<8, "bytes in the file")
	require.Equal(t, 0x0840, int(record[27])|int(record[28])<<8, "where it is entered")
}

// The pulses are the original tape's, so a block we write has to describe
// itself the way its blocks do. Worth a test because the fields of a turbo
// block are easy to put in the wrong order and the result still looks like a
// tape: what gives it away is the one bit being exactly twice the zero.
func TestCDTPulsesAreTheOriginalTapes(t *testing.T) {
	if _, err := os.Stat(cpcTape); err != nil {
		t.Skipf("the tape is not in the tree: %v", err)
	}

	original, err := os.ReadFile(cpcTape)
	require.NoError(t, err)

	// Past the container header and the pause block the rip opens with.
	require.Equal(t, byte(0x20), original[10])
	require.Equal(t, byte(0x11), original[13])

	theirs := original[14 : 14+0x0D]

	ours, err := media.CDT([]media.TapeFile{{
		Name: "AO1", Data: make([]byte, 100), Load: 0x0840, Entry: 0x0840,
	}})
	require.NoError(t, err)

	require.Equal(t, theirs, []byte(ours[11:11+0x0D]))
}

// A file longer than a block is written as several, and the firmware finds its
// way through them by the three fields that change.
func TestCDTCutsAFileIntoBlocks(t *testing.T) {
	tape, err := media.CDT([]media.TapeFile{{
		Name: "AO1", Data: make([]byte, 2048+100), Load: 0x0840, Entry: 0x0840,
	}})
	require.NoError(t, err)

	headers := cdtHeaderRecords(t, tape)
	require.Len(t, headers, 2)

	require.Equal(t, byte(1), headers[0][17])
	require.Equal(t, byte(0xFF), headers[0][24], "the first block says so")
	require.Equal(t, byte(0), headers[0][18], "and is not the last")
	require.Equal(t, 2048, int(headers[0][20])|int(headers[0][21])<<8)
	require.Equal(t, 0x0840, int(headers[0][22])|int(headers[0][23])<<8)

	require.Equal(t, byte(2), headers[1][17])
	require.Equal(t, byte(0), headers[1][24])
	require.Equal(t, byte(0xFF), headers[1][18], "the second block is the last")
	require.Equal(t, 100, int(headers[1][20])|int(headers[1][21])<<8)
	require.Equal(t, 0x0840+2048, int(headers[1][22])|int(headers[1][23])<<8,
		"and goes on after the first")
}

// cdtHeaderRecords pulls the header records out of a tape, for the tests that
// read what the firmware would.
func cdtHeaderRecords(t *testing.T, tape []byte) [][]byte {
	t.Helper()

	var found [][]byte

	for at := 10; at < len(tape); {
		require.Equal(t, byte(0x11), tape[at], "an Amstrad tape is turbo blocks")
		at++

		length := int(tape[at+0x0F]) | int(tape[at+0x10])<<8 | int(tape[at+0x11])<<16
		record := tape[at+0x12 : at+0x12+length]

		if record[0] == 0x2C {
			found = append(found, record)
		}

		at += 0x12 + length
	}

	return found
}
