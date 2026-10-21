package media

import "strconv"

// Just enough Spectrum BASIC to write a loader.
//
// This is not a tokeniser and does not want to be one. A loader is three lines
// — take the memory, load the blocks, jump into them — and writing those three
// directly is a page of code, where a tokeniser for the language is a program
// of its own and would be read by nobody.
//
// What it writes is the loader daadmaker carries as a byte array
// (daadmaker.pas:22-25), down to the leading space on the second line. That is
// worth matching rather than improving on, because it is what makes a tape this
// program writes comparable with one the reference writes.
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
	tokVAL       = 0xB0
	tokCODE      = 0xAF
	tokUSR       = 0xC0
	tokLOAD      = 0xEF
	tokRANDOMIZE = 0xF9
	tokCLEAR     = 0xFD
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

// LoadCode writes as many LOAD "" CODE as there are blocks to load, all on one
// line, separated by colons.
//
// Each loads the next CODE block on the tape to wherever its own header says it
// goes, so the loader never repeats an address and cannot contradict one.
func (b *BASIC) LoadCode(blocks int) {
	// The leading space is the reference's and has no effect. It is here so
	// that the two loaders are the same bytes and not merely the same program.
	body := []byte{' '}

	for i := range blocks {
		if i > 0 {
			body = append(body, ':')
		}

		body = append(body, tokLOAD, '"', '"', tokCODE)
	}

	b.statement(body)
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

// number writes a number as VAL "digits".
//
// Written the way the editor writes it, a number is its digits and then the
// same value again in five bytes, because that second copy is the one the
// interpreter evaluates and the digits are only what it lists as. VAL of a
// string skips all of that: there is nothing but the digits, and they are
// evaluated when the line runs.
//
// It is shorter, it is what every loader of the period did, and it is what the
// reference does — which is the reason it is here, since a loader has room for
// either.
func number(value int) []byte {
	out := []byte{tokVAL, '"'}
	out = append(out, strconv.Itoa(value)...)

	return append(out, '"')
}
