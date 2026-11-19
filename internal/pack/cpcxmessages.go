package pack

// The .XMB files a CPC release carries: the texts the processes read off the
// disk instead of out of the database.
//
// The interpreter opens them itself and never asks anybody where they are, so
// what matters here is only that they are on the disk under the names it builds
// and cut where it expects them cut. Both were read out of the interpreter
// rather than guessed, and this is where they are written down.
//
// # The name
//
// At 0x0BA8 it takes the offset the condact carries, shifts the high byte right
// three times — which is the offset divided by 2048 — and turns the result into
// two decimal digits:
//
//	0BAE: 7c cb 3f cb 3f cb 3f   ld a,h / srl a / srl a / srl a   the file
//	0BCD: cd 40 0c               call 0x0C40                      a = a mod 10
//	0BD0: c6 30 32 fd 0c         add a,'0' / ld (0x0CFD),a        the units
//	0BD5: 7a c6 30 32 fc 0c      ld a,d / add a,'0' / ld (0x0CFC),a   the tens
//	0BE1: 06 06 21 fc 0c         ld b,6 / ld hl,0x0CFC            six characters
//	0BE6: 11 40 00 cd 77 bc      ld de,0x0040 / call CAS IN OPEN
//
// 0x0C40 is a divide by ten: it leaves the remainder in A and the quotient in D.
// So the name is the file number in two digits and then ".XMB", and 0x0CFC is
// the template "00.XMB" sitting next to "000.CPC", which is the pictures' and
// has three digits filled in the same way.
//
// # Where it is cut
//
// Those three shifts are the whole of it: a file is 2048 bytes and the position
// in it is the offset's low eleven bits, which 0x0BF7 takes with an "and 7" on
// the high byte. internal/back lays the texts out on the same number — see
// back.XMessageSlot, which is where it comes from — and pads each slot out, so
// cutting the body every slot bytes puts every message where its offset says.
//
// # And no AMSDOS header
//
// For the reason the pictures have one and this does not: a picture is loaded
// to an address and this is read as a stream from a position the compiler
// worked out. A header would put every message 128 bytes out.

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
)

// cpcXMessageSuffix is what the interpreter calls them, after the two digits.
const cpcXMessageSuffix = ".XMB"

// cpcXMessageFiles cuts an .XMB into the files the interpreter opens.
//
// None of them if the adventure has no texts outside its database, which is
// every adventure that ships here.
func cpcXMessageFiles(opts Options) ([]File, error) {
	if len(opts.XMessages) == 0 {
		return nil, nil
	}

	slot := back.XMessageSlot(arch.CPC, "")
	if slot <= 0 {
		return nil, fmt.Errorf("the CPC has no size for an xmessage file")
	}

	// Two digits is what the name holds, so this is as many as the machine can
	// be asked for. The offset in a condact runs out first — it is two bytes,
	// which is 32 files of 2048 — but that limit belongs to the linker and is
	// checked there; this one belongs to the name.
	const mostFiles = 100

	files := make([]File, 0, (len(opts.XMessages)+slot-1)/slot)

	for at := 0; at < len(opts.XMessages); at += slot {
		number := at / slot
		if number >= mostFiles {
			return nil, fmt.Errorf(
				"the texts outside the database come to %d bytes, which is more than the "+
					"%d files of %d the interpreter can name",
				len(opts.XMessages), mostFiles, slot)
		}

		files = append(files, File{
			Name: fmt.Sprintf("%02d%s", number, cpcXMessageSuffix),
			Body: opts.XMessages[at:min(at+slot, len(opts.XMessages))],
			Note: fmt.Sprintf("texts the processes read from disk, offsets %d to %d",
				at, min(at+slot, len(opts.XMessages))-1),
		})
	}

	return files, nil
}
