package media

import (
	"fmt"
)

// The TAP container, which is what a 48K Spectrum loads and the simplest thing
// this repository writes.
//
// It is not the TZX read above and should not be confused with it. TZX
// describes the *sound* on a tape — pulse lengths, pauses, custom loaders — and
// opens with a signature so that a reader knows what it has. TAP describes none
// of that: it is a bare list of blocks, each a 16-bit length followed by that
// many bytes, with no header of its own and nothing to identify it by. What it
// can hold is exactly what the ROM loader can read, which for our purposes is
// all we need.
//
// Each block is a flag byte, the body, and a parity byte that is every
// preceding byte of the block exclusive-ored together. Flag 0x00 marks a
// seventeen-byte header saying what the block after it is; flag 0xFF marks that
// block. The ROM pairs them, which is why a header is never written alone.

// TAP collects blocks for a ZX Spectrum tape image.
//
// The zero value is ready to use and writes an empty tape, which is a valid
// thing to write and a useless thing to load.
type TAP struct {
	out []byte
}

// Spectrum header kinds, from the byte the ROM puts first in a header block.
const (
	tapProgram = 0 // a BASIC program
	tapCode    = 3 // a block of bytes for an address
)

// tapNameLength is how much room a header gives a name: ten characters, padded
// with spaces, and anything longer is cut. The ROM shows it while loading and
// matches against it, and does neither beyond ten.
const tapNameLength = 10

// Code appends a CODE block: a header saying where the bytes go, and the bytes.
func (t *TAP) Code(name string, at int, body []byte) error {
	if at < 0 || at+len(body) > 0x10000 {
		return fmt.Errorf("%s: %d bytes at 0x%04X runs past the top of memory",
			name, len(body), at)
	}

	t.header(tapCode, name, len(body), at, 0x8000)
	t.block(0xFF, body)

	return nil
}

// Program appends a BASIC program block.
//
// line is the line the program starts at when it loads, which is what makes a
// loader a loader: a program saved with one runs by itself. Saving with no
// autostart takes a line past any the program has, and 0x8000 is the value the
// ROM itself writes for that.
func (t *TAP) Program(name string, body []byte, line int) {
	t.header(tapProgram, name, len(body), line, len(body))
	t.block(0xFF, body)
}

// Bytes is the tape.
func (t *TAP) Bytes() []byte {
	return t.out
}

// header writes the seventeen bytes the ROM reads before a block: what kind of
// thing follows, its name, its length, and two parameters whose meaning depends
// on the kind. For CODE they are the address and an unused word; for a program,
// the autostart line and the length of the variables, which for a program with
// none is the length of the program itself.
func (t *TAP) header(kind byte, name string, length, first, second int) {
	body := make([]byte, 0, 17)
	body = append(body, kind)

	padded := name
	if len(padded) > tapNameLength {
		padded = padded[:tapNameLength]
	}

	body = append(body, padded...)
	for range tapNameLength - len(padded) {
		body = append(body, ' ')
	}

	body = word(body, length)
	body = word(body, first)
	body = word(body, second)

	t.block(0x00, body)
}

// block wraps a body in the flag and parity byte the ROM expects, and writes it
// with its length in front.
func (t *TAP) block(flag byte, body []byte) {
	parity := flag
	for _, b := range body {
		parity ^= b
	}

	t.out = word(t.out, len(body)+2)
	t.out = append(t.out, flag)
	t.out = append(t.out, body...)
	t.out = append(t.out, parity)
}

func word(to []byte, value int) []byte {
	return append(to, byte(value), byte(value>>8))
}
