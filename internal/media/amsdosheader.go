package media

import "fmt"

// The 128-byte header an AMSDOS file carries, which is the CPC's answer to the
// same question the +3 asks.
//
// A disk file has nowhere to say where it goes or where it starts, so AMSDOS
// puts that in front of it and hides it from whatever reads the file. The
// layout is the one MCRF writes — work/DRC/src/mcrf.pas, SetHeader — and what
// it sets is all a binary needs:
//
//	0       user number
//	1..8    name, space-padded, upper case
//	9..11   extension, the same
//	18      file type: 2 is a binary
//	21..22  load address
//	24..25  length
//	26..27  entry address
//	64..66  the length again, over three bytes
//	67..68  the sum of bytes 0 to 66
//
// Everything else is zero. The checksum is what AMSDOS looks at to decide
// whether there is a header here at all: a file whose first 128 bytes do not
// add up is a file without one, which is how headerless files are told apart
// from the rest.
const AMSDOSHeaderSize = 128

// amsdosBinary is the file type for machine code, as opposed to BASIC or a
// protected program.
const amsdosBinary = 2

// AMSDOSHeader builds the header for a binary that loads at load and is
// entered at run.
func AMSDOSHeader(name string, load, run, length int) ([]byte, error) {
	stem, ext, err := dataName(name)
	if err != nil {
		return nil, err
	}

	for what, value := range map[string]int{"load address": load, "entry address": run} {
		if value < 0 || value > 0xFFFF {
			return nil, fmt.Errorf("0x%X is not a CPC %s", value, what)
		}
	}

	if length < 0 || length > 0xFFFF {
		return nil, fmt.Errorf("%d bytes is more than a CPC file header can declare", length)
	}

	out := make([]byte, AMSDOSHeaderSize)

	copy(out[1:9], stem)
	copy(out[9:12], ext)

	out[18] = amsdosBinary

	put := func(at, value int) {
		out[at] = byte(value)
		out[at+1] = byte(value >> 8)
	}

	put(21, load)
	put(24, length)
	put(26, run)
	put(64, length)
	out[66] = 0

	sum := 0
	for _, b := range out[:67] {
		sum += int(b)
	}

	put(67, sum)

	return out, nil
}

// WithoutAMSDOSHeader gives back a file's contents with its AMSDOS header
// taken off, and the file unchanged when it has none.
//
// The checksum decides. A CPC file is as likely to be headerless as not — the
// interpreters DAAD ships are raw Z80 and the graphics blocks are not — so
// guessing from the length or the first byte would be guessing.
func WithoutAMSDOSHeader(data []byte) []byte {
	if !hasAMSDOSHeader(data) {
		return data
	}

	body := data[AMSDOSHeaderSize:]

	// The declared length is used here, unlike on the +3, because an AMSDOS
	// file is padded to a whole record and the header is the only thing that
	// says where the padding starts. Nothing in the corpus declares a length
	// past its own data, which is the mistake that stopped us trimming there.
	if length := int(data[24]) | int(data[25])<<8; length <= len(body) {
		return body[:length]
	}

	return body
}

// hasAMSDOSHeader reports whether the first 128 bytes are one.
func hasAMSDOSHeader(data []byte) bool {
	if len(data) < AMSDOSHeaderSize {
		return false
	}

	sum := 0
	for _, b := range data[:67] {
		sum += int(b)
	}

	// A header of all zeros adds up to zero and would pass, so the type byte
	// has to mean something too.
	return sum != 0 && sum == int(data[67])|int(data[68])<<8
}
