package media

import (
	"encoding/binary"
	"fmt"
)

// Writing the TZX container, which tape.go reads.
//
// TZX describes the sound on a tape rather than the bytes in it, which is what
// lets it carry the custom loaders TAP cannot. None of that is needed here:
// what this writes is a tape the ROM loader reads, so every block goes in as
// the one TZX block that means "the ROM's own timings", and the result is a
// TAP with a signature and a pause after each block.
//
// That is worth doing even so. A TAP has nothing to identify it by — it is a
// bare list of lengths — so a tool given one has to guess, and an emulator
// given a TZX does not. The pauses are the other half: a TAP leaves them to
// whatever is replaying it, and a tape of several parts wants them written
// down.
const (
	// tzxSignature opens the file, and the 0x1A after it is an end-of-file for
	// anything that tries to read the tape as text.
	tzxSignature = "ZXTape!\x1a"
	// tzxVersion is the revision of the format this writes. 1.20 is what every
	// reader of the last twenty years takes, and nothing here needs more.
	tzxMajor = 1
	tzxMinor = 20
	// tzxStandardBlock is the block ID for "data at the ROM's own timings":
	// a pause in milliseconds, a length, and that many bytes.
	tzxStandardBlock = 0x10
	// tzxPause is the silence after each block, in milliseconds. One second is
	// what the ROM leaves between a SAVE and the next, and what every tool
	// writes when it has no reason to write anything else.
	tzxPause = 1000
)

// TZX wraps the blocks of a TAP into a TZX tape.
//
// It takes the bytes rather than the TAP so that both formats come off the one
// assembly: a tape is built once, and what it is written as is a choice made
// afterwards. The blocks are not touched — same flags, same parity — so the
// two files load identically.
func TZX(tap []byte) ([]byte, error) {
	out := append([]byte(tzxSignature), tzxMajor, tzxMinor)

	for at := 0; at < len(tap); {
		if at+2 > len(tap) {
			return nil, fmt.Errorf("the tape ends inside a block length, at byte %d", at)
		}

		length := int(binary.LittleEndian.Uint16(tap[at:]))
		at += 2

		if at+length > len(tap) {
			return nil, fmt.Errorf(
				"a block says it is %d bytes and only %d are left, at byte %d",
				length, len(tap)-at, at)
		}

		out = append(out, tzxStandardBlock)
		out = word(out, tzxPause)
		out = word(out, length)
		out = append(out, tap[at:at+length]...)

		at += length
	}

	return out, nil
}
