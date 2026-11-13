// Package zx0 is the compression the Spectrum builds carry their pictures in.
//
// It is not a choice. The interpreter that goes on a 48K tape has Einar
// Saukas' ZX0 decoder built into it and runs it over every image block it
// loads, so a block that is not a ZX0 stream is not a picture: it is whatever
// those 68 bytes of Z80 make of it. The reference pipeline does the same thing
// from outside — ZXsplitter.php cuts the screen up and calls the zx0 tool on
// each piece — and the tool it calls is an x86 binary, so the coder has to
// live here.
//
// Everything here is derived from the decoder the interpreter runs,
// work/ZXDAAD128/src/asm/dzx0_standard.asm. Decompress is that decoder line
// for line.
//
// # How this is known to be the format
//
// A round trip proves that the coder and the decoder agree, and nothing more.
// It was not enough here, and the offset is why: its two halves are built in
// different registers, the low bit of the low byte is shifted out to become
// the first bit of the length, and whether the carry riding into that rotate
// is set decides what the whole thing means. A shared misreading of that
// passes every round trip there is and still produces a stream that walks the
// interpreter off into memory.
//
// So the check is against Saukas' own coder, whose C source sits beside the
// interpreter in work/ZXDAAD128/DCP: it compresses every screen the examples
// carry, and Decompress is required to read back what it wrote. That makes the
// decoder the format rather than our reading of it, and the coder follows,
// because what it writes goes back through that same decoder. See
// script/compare-zx0.sh, and make compare-zx0 to run it.
//
// It also settled which ZX0 this is. compress() takes an invert_mode, and the
// two modes are two formats; DCP.c passes the classic one, and that is the one
// Decompress reads. The other it refuses, which is the answer that was wanted.
//
// # How well it packs
//
// As well as his does, which was not true at first and is worth saying because
// of how it was found out. The coder chose greedily to begin with — the longest
// copy available at each step — and the same comparison that settled the format
// measured what that cost: a tenth more than his over the screens of La
// Aventura Original, a third more on the worst block. Both were correct ZX0 and
// neither was more correct; one was simply bigger, and on a 48K tape bigger is
// counted in pictures.
//
// So the choosing was rewritten as the shortest path he uses, and the two now
// tie to the byte over those 1450 blocks. That is what compare-zx0 reports, and
// the number stays in the report rather than becoming an assertion: two coders
// for one format are two programs, and a tie is a result, not a contract.
//
// It costs what his costs. The search looks at every offset at every position,
// so a 6912-byte screen takes about a quarter of a second here against about a
// fifth in his C — see optimal.go for the free list that got it there from six
// tenths of a second and a gigabyte and a quarter of garbage per screen.
package zx0

import (
	"errors"
	"fmt"
)

// The shape of a stream, as the decoder reads it:
//
//   - A block of literals: a length, then that many bytes.
//   - Then one bit: 0 to copy from the offset already in hand, 1 for a new one.
//   - A copy from the last offset: just a length.
//   - A copy from a new offset: the offset, then a length.
//   - After a copy, one bit again: 0 back to literals, 1 for a new offset.
//
// Lengths are interlaced Elias gamma: a control bit saying "another bit
// follows", then that bit, until a control bit of 1 ends it. The value starts
// at 1, and each data bit doubles it and adds itself.
//
// endMarkerBits is how the stream says it is over. The decoder builds the high
// half of a new offset by counting up from 0xFE and then does "inc c; ret z",
// so eight data bits of 1 leave it at 0xFF and the routine returns.
const endMarkerBits = 8

// The limits of what a match can say.
const (
	// maxOffset is how far back a copy may reach: 255 high values of 128
	// positions each, which is what the two halves of an offset hold between
	// them.
	maxOffset = 32640
	// minMatch is the shortest copy there is. The format cannot say one: the
	// length that rides out inside the offset's low bit starts at two.
	minMatch = 2
)

// Compress encodes data as a ZX0 stream.
//
// Nothing at all has no stream and comes back as nothing. The decoder's first
// act is to read a length and copy that many literals, and the shortest length
// Elias gamma can say is one, so "no literals here" is not a thing the format
// can express. Nobody asks for it either: the blocks that come through here
// are cut out of a screen and always hold bytes.
//
// Which literals and which copies is not decided here: optimalParse works that
// out as a shortest path over the whole block, and this walks what it chose and
// writes it down. The two halves are worth keeping apart — what the format can
// say is one problem and which of the things it can say is cheapest is another,
// and only the first of them is the machine's.
func Compress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	w := &writer{}

	lastOffset := initialOffset
	at := 0

	for i, m := range optimalParse(data) {
		switch m.offset {
		case 0:
			// Literals. The bit that says so is skipped the first time round,
			// because there the decoder is already reading a literal length:
			// its very first act is to read one, which is why no stream can
			// begin with a copy.
			if i > 0 {
				w.bit(0)
			}

			w.elias(m.length)
			w.bytes(data[at : at+m.length])

		case lastOffset:
			// A repeat says its length straight, with no adjustment: the
			// "inc bc" that makes a fresh offset's length one more than it
			// says is on the other path only.
			w.bit(0)
			w.elias(m.length)

		default:
			w.bit(1)
			w.match(m.offset, m.length)
			lastOffset = m.offset
		}

		at += m.length
	}

	w.end()

	return w.out
}

// writer puts bits and bytes down the way the decoder takes them off.
//
// The decoder reads bits out of a byte it loads when it runs out, most
// significant first, and finds out that it has run out by shifting a marker
// bit off the top. So the bits of a group go into a byte reserved in the
// stream ahead of the bytes that follow it, and are filled in there.
type writer struct {
	out []byte
	// at is where the byte being filled with bits lives, and left is how many
	// of its bits are still free.
	at   int
	left int
}

func (w *writer) bit(b int) {
	if w.left == 0 {
		w.at = len(w.out)
		w.out = append(w.out, 0)
		w.left = 8
	}

	w.left--

	if b != 0 {
		w.out[w.at] |= 1 << w.left
	}
}

func (w *writer) bytes(b []byte) { w.out = append(w.out, b...) }

// elias writes a value of one or more as interlaced Elias gamma.
//
// The value's leading bit is implied — the decoder starts its counter at one —
// so what goes out is the rest of the value, each bit behind a 0 that says it
// is coming, and a 1 to finish.
func (w *writer) elias(value int) {
	if value < 1 {
		panic(fmt.Sprintf("zx0: elias of %d, which has no coding", value))
	}

	top := 0
	for 1<<(top+1) <= value {
		top++
	}

	for i := top - 1; i >= 0; i-- {
		w.bit(0)
		w.bit(value >> i & 1)
	}

	w.bit(1)
}

// match writes a copy from a new offset, and the length that goes with it.
//
// The two are tangled, and not by us. The decoder holds the offset negated,
// builds its high half by counting up from 0xFE with the Elias bits and reads
// its low half as a byte, then rotates the pair right — so the bit that falls
// out of the bottom of that byte is the first thing the length says, and the
// pair has to be written already doubled with that bit in it.
//
// Which bit it is says everything the length has to say when the length is
// two: a 1 there and there is nothing more to write. Anything longer sets it
// to 0 and the rest of the length follows, continued from a value of one —
// which is what dzx0s_elias_backtrack is entered to do.
func (w *writer) match(offset, length int) {
	if offset < 1 || offset > maxOffset {
		panic(fmt.Sprintf("zx0: offset %d is outside 1..%d", offset, maxOffset))
	}

	if length < minMatch {
		panic(fmt.Sprintf("zx0: a match of %d is shorter than the format can say", length))
	}

	negated := 0x10000 - offset

	lengthBit := 0
	if length == minMatch {
		lengthBit = 1
	}

	pair := (negated&0x7FFF)<<1 | lengthBit

	// The counter is read one higher than the Elias run leaves it, so the run
	// has to stop one short of the high byte wanted.
	w.eliasBits(runTo(byte(pair>>8) - 1))
	w.out = append(w.out, byte(pair))

	if lengthBit == 0 {
		w.lengthTail(length - 1)
	}
}

// eliasBits writes a run of data bits with no leading value: when the decoder
// reads an offset its counter is already loaded, so what it wants is the bits.
func (w *writer) eliasBits(bits []int) {
	for _, b := range bits {
		w.bit(0)
		w.bit(b)
	}

	w.bit(1)
}

// lengthTail writes a value whose first control bit has already gone out, as
// the 0 in the offset's low byte. The decoder takes a data bit first and a
// control bit after it, the other way round from an ordinary length.
func (w *writer) lengthTail(value int) {
	top := 0
	for 1<<(top+1) <= value {
		top++
	}

	for i := top - 1; i >= 0; i-- {
		w.bit(value >> i & 1)
		w.bit(boolBit(i == 0))
	}
}

func boolBit(b bool) int {
	if b {
		return 1
	}

	return 0
}

// runTo works out which data bits take the decoder's counter from 0xFE to the
// byte wanted. There are never more than eight: past that the counter holds
// nothing but the bits themselves.
func runTo(want byte) []int {
	for n := range endMarkerBits + 1 {
		value := 0xFE

		bits := make([]int, n)
		for i := range bits {
			bits[i] = int(want>>(n-1-i)) & 1
			value = value<<1 | bits[i]
		}

		if byte(value) == want {
			return bits
		}
	}

	panic(fmt.Sprintf("zx0: no Elias run reaches %#02x", want))
}

// end writes the marker that stops the decoder: a new offset whose high half
// is eight ones, which is the value that turns its counter over.
func (w *writer) end() {
	w.bit(1)

	ones := make([]int, endMarkerBits)
	for i := range ones {
		ones[i] = 1
	}

	w.eliasBits(ones)
}

// Decompress is the decoder the interpreter runs, in Go.
//
// It is a translation of dzx0_standard.asm and deliberately nothing more: the
// flow follows its labels and the comments name them, because what makes it
// worth having is that the two can be read side by side. Nothing plays
// anything with it — it is what the tests hold the coder against.
func Decompress(src []byte) ([]byte, error) {
	r := &reader{src: src, a: 0x80}

	var (
		out    []byte
		offset = 1 // the default the decoder pushes before it starts
	)

	for {
		// dzx0s_literals.
		length, err := r.elias()
		if err != nil {
			return nil, err
		}

		// ldir, with BC holding what the Elias run gave: that many bytes, not
		// one fewer. Writing it with an adjustment here and the matching
		// adjustment in the coder is how a stream that this read back
		// perfectly came out a byte long for the Z80 — which copies what the
		// length says and then reads the rest as a stream it is no longer in
		// step with.
		for range length {
			b, err := r.byteAt()
			if err != nil {
				return nil, err
			}

			out = append(out, b)
		}

		fresh, err := r.bitAt()
		if err != nil {
			return nil, err
		}

		if fresh == 0 {
			if length, err = r.elias(); err != nil {
				return nil, err
			}
		}

		// dzx0s_copy, and dzx0s_new_offset above it.
		for {
			if fresh == 1 {
				var done bool

				offset, length, done, err = r.offset()
				if err != nil {
					return nil, err
				}

				if done {
					return out, nil
				}
			}

			if offset > len(out) {
				return nil, fmt.Errorf("zx0: a copy reaches %d bytes back into %d", offset, len(out))
			}

			for range length {
				out = append(out, out[len(out)-offset])
			}

			if fresh, err = r.bitAt(); err != nil {
				return nil, err
			}

			if fresh == 0 {
				break
			}
		}
	}
}

// reader is the decoder's source: the stream, and the byte of bits it is
// working through.
type reader struct {
	src []byte
	i   int
	a   byte
}

var errTruncated = errors.New("zx0: the stream ends in the middle of something")

func (r *reader) byteAt() (byte, error) {
	if r.i >= len(r.src) {
		return 0, errTruncated
	}

	b := r.src[r.i]
	r.i++

	return b, nil
}

// bitAt is "add a, a", with the reload the decoder does when the marker bit
// falls off the top.
func (r *reader) bitAt() (int, error) {
	carry := r.a&0x80 != 0
	r.a <<= 1

	if r.a == 0 {
		next, err := r.byteAt()
		if err != nil {
			return 0, err
		}

		// rla: the new byte's top bit becomes the carry, and the old carry
		// goes in at the bottom as the next marker.
		newCarry := next&0x80 != 0
		r.a = next << 1

		if carry {
			r.a |= 1
		}

		carry = newCarry
	}

	if carry {
		return 1, nil
	}

	return 0, nil
}

// elias reads a value that starts counting at one.
func (r *reader) elias() (int, error) {
	value := 1

	for {
		control, err := r.bitAt()
		if err != nil {
			return 0, err
		}

		if control == 1 {
			return value, nil
		}

		data, err := r.bitAt()
		if err != nil {
			return 0, err
		}

		value = value<<1 | data

		if value > 0xFFFF {
			return 0, errors.New("zx0: a length runs past what sixteen bits hold")
		}
	}
}

// offset reads a new offset and the length that goes with it, and reports the
// end marker.
//
// This half is the one the coder does not write; see the package comment. It
// is translated anyway, because a decoder that stops at the first match is no
// use as a check on anything.
func (r *reader) offset() (offset, length int, done bool, err error) {
	// ld c, $fe: the counter the Elias bits count up from.
	counter := 0x00FE

	for {
		control, err := r.bitAt()
		if err != nil {
			return 0, 0, false, err
		}

		if control == 1 {
			break
		}

		data, err := r.bitAt()
		if err != nil {
			return 0, 0, false, err
		}

		counter = (counter<<1 | data) & 0xFFFF
	}

	// inc c; ret z.
	high := byte(counter) + 1
	if high == 0 {
		return 0, 0, true, nil
	}

	low, err := r.byteAt()
	if err != nil {
		return 0, 0, false, err
	}

	// ld b, c; ld c, (hl); rr b; rr c — one bit right through the carry, which
	// the Elias run left set, and the bit that falls out is the length's.
	//
	// The "ld b, c" takes C as "inc c" left it, one higher than the Elias run
	// built, and that one is not a detail: with it, a run of no bits at all
	// means the offsets from 1 to 128, which is where a compressor spends most
	// of its time. Reading it as the un-incremented value puts the shortest
	// offsets out of reach and makes the whole table look wrong.
	pair := int(high)<<8 | int(low)
	lengthBit := pair & 1
	offset = 0x10000 - (pair>>1 | 0x8000)

	length = 1

	if lengthBit == 0 {
		if length, err = r.eliasFrom(1); err != nil {
			return 0, 0, false, err
		}
	}

	// ld bc, 1 ... inc bc.
	return offset, length + 1, false, nil
}

// eliasFrom continues a value whose first control bit has already been read,
// which is what dzx0s_elias_backtrack is entered to do.
func (r *reader) eliasFrom(value int) (int, error) {
	for {
		data, err := r.bitAt()
		if err != nil {
			return 0, err
		}

		value = value<<1 | data

		control, err := r.bitAt()
		if err != nil {
			return 0, err
		}

		if control == 1 {
			return value, nil
		}
	}
}
