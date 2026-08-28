package media

import (
	"bytes"
	"fmt"
)

// The TZX container, which the Amstrad world calls CDT: the same format under
// another extension, and both start with the same signature.
//
// A tape is a list of blocks, each headed by an identifier that says how long
// its description is and whether anything follows. Most describe pulses, pauses
// or text for the person loading; only a few carry the bytes that end up in
// memory, and those are the ones worth keeping.
const (
	tapeSignature  = "ZXTape!\x1a"
	tapeHeaderSize = 10 // the signature, plus a major and a minor version
)

// tapeBlock says how to step over one kind of block, and whether it holds bytes
// that were loaded or merely a description of the sound on the tape.
type tapeBlock struct {
	header   int  // bytes of description before anything else
	lengthAt int  // where in that description the length sits, -1 if fixed size
	lengthOf int  // how many bytes the length itself takes
	scale    int  // bytes per unit of that length
	loaded   bool // whether what follows is loaded data rather than pulses
}

// Every block the format defines, bar the handful deprecated before version
// 1.10. An identifier not in here stops the walk rather than guessing: getting a
// length wrong turns the rest of the tape into nonsense without saying so.
var tapeBlocks = map[byte]tapeBlock{
	0x10: {header: 0x04, lengthAt: 0x02, lengthOf: 2, scale: 1, loaded: true}, // standard speed data
	0x11: {header: 0x12, lengthAt: 0x0F, lengthOf: 3, scale: 1, loaded: true}, // turbo speed data
	0x12: {header: 0x04, lengthAt: -1},                                        // pure tone
	0x13: {header: 0x01, lengthAt: 0x00, lengthOf: 1, scale: 2},               // pulse sequence
	0x14: {header: 0x0A, lengthAt: 0x07, lengthOf: 3, scale: 1, loaded: true}, // pure data
	0x15: {header: 0x08, lengthAt: 0x05, lengthOf: 3, scale: 1},               // direct recording: samples
	0x18: {header: 0x04, lengthAt: 0x00, lengthOf: 4, scale: 1},               // CSW recording
	0x19: {header: 0x04, lengthAt: 0x00, lengthOf: 4, scale: 1},               // generalized data
	0x20: {header: 0x02, lengthAt: -1},                                        // pause or stop the tape
	0x21: {header: 0x01, lengthAt: 0x00, lengthOf: 1, scale: 1},               // group start
	0x22: {header: 0x00, lengthAt: -1},                                        // group end
	0x23: {header: 0x02, lengthAt: -1},                                        // jump to block
	0x24: {header: 0x02, lengthAt: -1},                                        // loop start
	0x25: {header: 0x00, lengthAt: -1},                                        // loop end
	0x26: {header: 0x02, lengthAt: 0x00, lengthOf: 2, scale: 2},               // call sequence
	0x27: {header: 0x00, lengthAt: -1},                                        // return from sequence
	0x28: {header: 0x02, lengthAt: 0x00, lengthOf: 2, scale: 1},               // select block
	0x2A: {header: 0x04, lengthAt: -1},                                        // stop if in 48K mode
	0x2B: {header: 0x04, lengthAt: 0x00, lengthOf: 4, scale: 1},               // set signal level
	0x30: {header: 0x01, lengthAt: 0x00, lengthOf: 1, scale: 1},               // text description
	0x31: {header: 0x02, lengthAt: 0x01, lengthOf: 1, scale: 1},               // message
	0x32: {header: 0x02, lengthAt: 0x00, lengthOf: 2, scale: 1},               // archive info
	0x33: {header: 0x01, lengthAt: 0x00, lengthOf: 1, scale: 3},               // hardware type
	0x35: {header: 0x14, lengthAt: 0x10, lengthOf: 4, scale: 1},               // custom info
	0x5A: {header: 0x09, lengthAt: -1},                                        // glue block
}

// Tape reads a ZX Spectrum .TZX or an Amstrad .CDT.
//
// Like the disks the same adventures were sold on, a tape holds no filesystem.
// The Spectrum blocks do carry names — this one names its parts ao1, AO$ and
// codigo — but a name on a tape belongs to the block that follows it, not to
// anything the tape keeps track of, so there is nothing to walk and nothing to
// list. What a caller gets is what the machine would have loaded.
type Tape struct {
	data []byte
}

func NewTape(data []byte) (*Tape, error) {
	if len(data) < tapeHeaderSize {
		return nil, fmt.Errorf("too short to hold a tape header: %d bytes", len(data))
	}

	return &Tape{data: data}, nil
}

func (t *Tape) Format() string {
	return fmt.Sprintf("TZX tape, version %d.%d", t.data[8], t.data[9])
}

// Files returns nothing: a tape is a stream of blocks, not a filesystem.
func (t *Tape) Files() ([]File, error) {
	return nil, fmt.Errorf("%s: a stream of blocks, with no filesystem to walk", t.Format())
}

// Payload returns the bytes the machine would have loaded, block after block.
// Blocks describing pulses, pauses or text contribute nothing.
func (t *Tape) Payload() []byte {
	out := make([]byte, 0, len(t.data))

	for at := tapeHeaderSize; at < len(t.data); {
		id := t.data[at]
		at++

		block, known := tapeBlocks[id]
		if !known || at+block.header > len(t.data) {
			// Guessing at a length would turn the rest of the tape into
			// nonsense without saying so. Better to stop with what is certain.
			break
		}

		length := 0
		if block.lengthAt >= 0 {
			length = t.number(at+block.lengthAt, block.lengthOf) * block.scale
		}

		body := at + block.header
		if body+length > len(t.data) {
			break
		}

		if block.loaded {
			out = append(out, t.contents(id, t.data[body:body+length])...)
		}

		at = body + length
	}

	return out
}

// contents strips the wrapping a block puts around what was loaded.
//
// A standard speed block is one the ROM itself wrote, so its shape is known: a
// flag saying whether it is a header or data, the bytes, and a checksum. The
// faster blocks belong to whatever loader the game brought with it, and there
// is no telling what is wrapping and what is not, so they are left alone.
func (t *Tape) contents(id byte, body []byte) []byte {
	const standardSpeed = 0x10

	if id != standardSpeed || len(body) < 2 {
		return body
	}

	return body[1 : len(body)-1]
}

// number reads a little-endian value of the given width.
func (t *Tape) number(at, width int) int {
	value := 0
	for i := range width {
		value |= int(t.data[at+i]) << (8 * i)
	}

	return value
}

func looksLikeTape(data []byte) bool {
	return bytes.HasPrefix(data, []byte(tapeSignature))
}
