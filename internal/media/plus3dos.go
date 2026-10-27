package media

import "fmt"

// The 128-byte header a +3 file carries, which is the disk's answer to a tape
// block header.
//
// A tape block says what it is in the bytes that come before it, so LOAD ""CODE
// knows where to put what it finds. A disk file has nowhere to say that, so
// +3DOS puts the same information in a header at the front of the file and
// subtracts it from what the program sees. A file without one is "headerless"
// and can only be read with LOAD ... CODE at an explicit address; one with it
// loads where it belongs with nothing said.
//
// The layout is +3DOS's own, from the +3 manual's chapter 8:
//
//	0..7    "PLUS3DOS"
//	8       0x1A, a soft end-of-file so that TYPE stops here
//	9       issue, 1
//	10      version, 0
//	11..14  the length of the file, header and all, little-endian
//	15..22  the eight bytes a tape header would have held
//	23..126 reserved, zero
//	127     the sum of the first 127 bytes, modulo 256
//
// Bytes 15 to 22 are the tape header verbatim: the type, the length of the data
// without the header, and two parameters whose meaning depends on the type. For
// CODE the first parameter is the address; for a BASIC program it is the line to
// start at.
const Plus3HeaderSize = 128

// The file types, as both a tape header and a +3DOS header number them.
const (
	plus3Program = 0
	plus3Code    = 3
)

// Plus3Code wraps a block of machine code so that LOAD "name"CODE puts it at
// the address given.
func Plus3Code(at int, body []byte) ([]byte, error) {
	if at < 0 || at > 0xFFFF {
		return nil, fmt.Errorf("0x%X is not an address a Spectrum can load to", at)
	}

	// The second parameter is unused for CODE and the convention is 32768,
	// which is what the ROM writes when it saves one.
	const unused = 32768

	return plus3Header(plus3Code, len(body), at, unused, body)
}

// Plus3Program wraps a tokenised BASIC program so that LOAD "name" runs it from
// the line given.
//
// The line matters more than it looks. A program saved to start at a line other
// than its first runs only from there, which for a loader means sitting at a
// LOAD it was supposed to have set up for — so the caller passes FirstLine and
// not a number it made up.
func Plus3Program(line int, body []byte) ([]byte, error) {
	// The first parameter is the autostart line and the second is where the
	// variables begin, which for a program with none is its whole length.
	return plus3Header(plus3Program, len(body), line, len(body), body)
}

// WithoutPlus3Header gives back a file's contents with its +3DOS header taken
// off, and the file unchanged when it has none.
//
// +3DOS hides the header from whatever reads the file, so anything that reads
// one and keeps it is reading 128 bytes that were never part of the file. It
// matters most where it is least visible: a database with a header in front of
// it is a database whose every pointer is 128 bytes out, and what that looks
// like is not a clear failure but an adventure with the wrong words in it.
//
// What it does not do is trim the end, and that is a decision with a
// measurement behind it rather than an omission.
//
// CP/M stores a length in records of 128 bytes, so the last record of a file is
// padded and the header is the only place the true length is written down. It
// would follow that the declared length is where to cut — except that on
// Cozumel's Spectrum disk it is not where the file ends. D1.BIN declares 40443
// bytes and its graphics block runs to 40448: the drawings are in what the
// header calls padding. Cutting to the declared length finds no drawings at
// all, and leaving the padding on finds them.
//
// So the header's presence is trusted and its length is not. Trailing bytes
// cost a reader nothing — everything downstream looks for structures rather
// than measuring from the end — while five missing ones cost a whole edition
// its illustrations.
func WithoutPlus3Header(data []byte) []byte {
	if len(data) < Plus3HeaderSize || string(data[:8]) != "PLUS3DOS" {
		return data
	}

	sum := 0
	for _, b := range data[:Plus3HeaderSize-1] {
		sum += int(b)
	}

	// A header that does not add up is not a header. Taking 128 bytes off a
	// file on the strength of eight that happen to spell PLUS3DOS would be
	// worse than leaving it alone.
	if byte(sum) != data[Plus3HeaderSize-1] {
		return data
	}

	return data[Plus3HeaderSize:]
}

func plus3Header(kind, length, param1, param2 int, body []byte) ([]byte, error) {
	out := make([]byte, Plus3HeaderSize, Plus3HeaderSize+len(body))

	copy(out, "PLUS3DOS")
	out[8] = 0x1A
	out[9] = 1 // issue
	out[10] = 0

	total := Plus3HeaderSize + len(body)
	out[11] = byte(total)
	out[12] = byte(total >> 8)
	out[13] = byte(total >> 16)
	out[14] = byte(total >> 24)

	// The eight bytes of a tape header, in their tape order.
	out[15] = byte(kind)
	out[16] = byte(length)
	out[17] = byte(length >> 8)
	out[18] = byte(param1)
	out[19] = byte(param1 >> 8)
	out[20] = byte(param2)
	out[21] = byte(param2 >> 8)

	sum := 0
	for _, b := range out[:Plus3HeaderSize-1] {
		sum += int(b)
	}

	out[Plus3HeaderSize-1] = byte(sum)

	return append(out, body...), nil
}
