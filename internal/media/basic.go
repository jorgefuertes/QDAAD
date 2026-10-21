package media

import "strconv"

// Just enough Spectrum BASIC to write a loader.
//
// This is not a tokeniser and does not want to be one. A loader is three kinds
// of statement — take the memory, load the blocks, jump into them — and writing
// those three directly is a page of code, where a tokeniser for the language is
// a program of its own and would be read by nobody.
//
// A line is its number as a *big-endian* word, which is the one place the
// machine writes a word the other way round, then the length of what follows as
// a little-endian word, then the statement, then a carriage return.
type BASIC struct {
	out  []byte
	line int
}

// The keywords a loader needs. Every token is one byte with the top bit set,
// which is why a BASIC program cannot hold those bytes as text.
const (
	tokCLEAR     = 0xFD
	tokLOAD      = 0xEF
	tokCODE      = 0xAF
	tokRANDOMIZE = 0xF9
	tokUSR       = 0xC0
)

// lineStep is the gap between line numbers, and 10 only because that is what
// anyone typing this in by hand would have used.
const lineStep = 10

// FirstLine is where the program starts running. Lines are numbered from here
// upwards in the order they are added, so a loader saved to start at this line
// runs all of itself and in order — which is not true of a loader saved to
// start at any other, and is the kind of mistake that leaves a tape sitting at
// the first LOAD forever.
const FirstLine = lineStep

// Clear moves RAMTOP below the address given, reserving everything above it.
//
// It has to come first and it has to be right: the loader itself lives in the
// memory under RAMTOP, and a block loaded over the BASIC area takes the loader
// with it halfway through loading.
func (b *BASIC) Clear(top int) {
	b.statement(append([]byte{tokCLEAR}, number(top)...))
}

// LoadCode is LOAD "" CODE: the next CODE block on the tape, to wherever its
// own header says it goes. The address is in the block, so the loader does not
// repeat it and cannot contradict it.
func (b *BASIC) LoadCode() {
	b.statement([]byte{tokLOAD, '"', '"', tokCODE})
}

// RandomizeUSR calls machine code at an address, which is how BASIC hands over
// and, for a loader, the last thing it does.
func (b *BASIC) RandomizeUSR(at int) {
	b.statement(append([]byte{tokRANDOMIZE, tokUSR}, number(at)...))
}

// Bytes is the program.
func (b *BASIC) Bytes() []byte {
	return b.out
}

func (b *BASIC) statement(body []byte) {
	b.line += lineStep

	b.out = append(b.out, byte(b.line>>8), byte(b.line)) // big-endian, once
	b.out = word(b.out, len(body)+1)
	b.out = append(b.out, body...)
	b.out = append(b.out, 0x0D)
}

// number writes a number the way the editor does: the digits a person reads,
// and then the value again in the five bytes the interpreter actually
// evaluates. Both are needed. Without the digits the line lists as a gap;
// without the five bytes it evaluates as zero, and a loader that CLEARs zero
// and calls zero does nothing visible before it dies.
//
// The five bytes are the small-integer form — a zero exponent, a sign, the
// value as a word, and a byte that is not used — which covers everything an
// address can be.
func number(value int) []byte {
	out := append([]byte(nil), strconv.Itoa(value)...)

	return append(out, 0x0E, 0x00, 0x00, byte(value), byte(value>>8), 0x00)
}
