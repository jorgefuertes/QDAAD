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
// for line, and it is worth having for one reason: there is no reference
// stream anywhere in the tree to check against, so what stands in for one is a
// decoder that is demonstrably the same program as the machine's, and a coder
// whose output it reads back byte for byte.
//
// # What this coder does not do yet
//
// It writes literals and no matches, which is a correct ZX0 stream and a
// badly compressed one: a block comes out the size it went in, plus about
// four bytes. The format can do far better — that is the whole point of it —
// but a match needs an offset, and an offset is the one part of the stream
// whose coding cannot be read off the decoder with any confidence: the two
// halves are built in different registers, the low bit of the low byte is
// shifted out to become the first bit of the length, and whether the carry
// riding into that rotate is set decides what the whole thing means. Guessing
// wrong there does not produce a worse picture, it produces a stream that
// walks the interpreter off into memory.
//
// So: literals now, because they are provably right and they fit what a 48K
// build has room for anyway, and matches when there is something to verify
// them against — the zx0 tool built for this machine, or a capture of a real
// stream.
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
// The match finder is greedy: at each position it takes the longest copy it
// can see, and prefers the offset already in hand when that one reaches as
// far, because repeating an offset costs a single bit and no offset. Saukas'
// own coder runs a shortest-path search over the same format and gets a few
// per cent more; this is simpler to be sure of, and sure is what matters when
// the reader is 68 bytes of Z80 that will not complain.
func Compress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	w := &writer{}

	at, lastOffset := 0, 1

	// Where the decoder is, which decides what a bit means. After a block of
	// literals, 0 asks for a copy at the offset already in hand and 1 for a
	// new one. After a copy, 0 goes back to literals and 1 takes a new offset
	// — there is no way back to the old offset from here, which is why a
	// repeat only ever follows literals.
	copied := false

	for at < len(data) {
		// Literals, up to the next copy worth making. The first time round
		// there is always at least one: nothing can be copied from in front
		// of the first byte.
		start := at

		var length, offset int

		for at < len(data) {
			if length, offset = longestMatch(data, at, lastOffset); worthCopying(length, offset, lastOffset) {
				break
			}

			length = 0
			at++
		}

		if at > start {
			if copied {
				w.bit(0) // back to literals
			}

			w.elias(at - start)
			w.bytes(data[start:at])

			copied = false
		}

		if length == 0 {
			break // the data ran out in literals
		}

		if offset == lastOffset && !copied {
			// A repeat says its length straight, with no adjustment: the
			// "inc bc" that makes a fresh offset's length one more than it
			// says is on the other path only.
			w.bit(0)
			w.elias(length)
		} else {
			w.bit(1)
			w.match(offset, length)
			lastOffset = offset
		}

		at += length
		copied = true
	}

	w.end()

	return w.out
}

// worthCopying says whether a match pays for what it costs to write.
//
// A repeat of the offset in hand is a bit and a length, so anything the
// format can express is worth it. A fresh one carries its two halves as well,
// which two bytes do not cover unless the offset is near enough to fit in the
// short form.
func worthCopying(length, offset, lastOffset int) bool {
	if length < minMatch {
		return false
	}

	const shortOffset = 128

	return length > minMatch || offset == lastOffset || offset <= shortOffset
}

// longestMatch is the match finder, and it is a plain backwards scan.
//
// Nothing here is big — a screen block is a few thousand bytes — so a hash
// chain would buy time nothing is waiting for.
func longestMatch(data []byte, at, lastOffset int) (length, offset int) {
	from := max(at-maxOffset, 0)

	// The offset in hand first and on equal terms, because repeating it is
	// nearly free: one bit, and no offset to write.
	if n := matchLength(data, at, at-lastOffset); n >= minMatch {
		length, offset = n, lastOffset
	}

	for back := at - 1; back >= from; back-- {
		if n := matchLength(data, at, back); n > length {
			length, offset = n, at-back
		}
	}

	return length, offset
}

// matchLength is how far the data at two places agrees.
//
// A match may run past where it started copying from, which is not an error
// and is the whole trick behind a run: the decoder copies byte by byte, so a
// length of 300 at offset 1 fills 300 bytes with one value.
func matchLength(data []byte, at, from int) int {
	if from < 0 || from >= at {
		return 0
	}

	n := 0
	for at+n < len(data) && data[from+n] == data[at+n] {
		n++
	}

	return n
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
