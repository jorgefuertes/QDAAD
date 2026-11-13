package back

import (
	"fmt"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
)

// The .XMB, which is where the texts that do not live in the database go.
//
// An xmessage is a message kept outside the 64 KiB the database has to fit in,
// read off disk when a condact asks for it. That buys two things: room, which
// is the obvious one, and a list with no limit of 255 entries, which is the one
// that matters to an adventure with a lot to say.
//
// The format is almost provocatively simple — no header, no index, no lengths.
// The messages are concatenated, every byte is obfuscated like every other text
// in a database, and one byte ends each of them. The position of each message
// is not written down anywhere: the compiler resolves it and puts it in the
// condact, which is why laying this out has to happen before the processes are
// emitted.
//
// It is all derived from generateXMessages, drb.php:448-523.

const (
	// xmesEndOfText is what ends a message. A line feed, obfuscated like
	// everything else, so 0x0A ^ 0xFF.
	//
	// It works as a terminator for a reason worth knowing: a line break inside
	// a text is stored as 0x0D and never as 0x0A, which leaves this byte free
	// to mean something. The interpreter needs it because it is never told how
	// long a message is — it reads a fixed block from the offset and lets the
	// ordinary printing routine stop here.
	xmesEndOfText = endOfText ^ obfuscate
	// xmesGap is the hole left at the front of a +3 file.
	//
	// The +3 interpreter reads the first 16 KiB of this into page 1 and keeps
	// it, and anything past that it has to fetch from disk — into these 512
	// bytes, which are its buffer. So the first real message starts at 512.
	// See LoadXMessagesInRAM in PLUS3.ASM.
	xmesGap = 512
	// xmesSwap is the room the +3 needs under the top of a slot.
	//
	// Printing a message means swapping 512 bytes between pages — the charset
	// lives where the message would have to be read from, so the interpreter
	// trades the two and trades them back. A message that began inside the
	// last 512 bytes of a slot would be swapped in short, so none does.
	xmesSwap = 512
	// xmesLimit is the hard ceiling, and it is not a choice either: the offset
	// travels in two bytes of a condact.
	xmesLimit = 0xFFFF
)

// xmessages is a laid-out .XMB: the bytes of it, and where each message landed.
type xmessages struct {
	body []byte
	// offsets is where each message starts, as the condact will carry it. It
	// is a position in the concatenation of every slot and not in a file: an
	// interpreter divides by the slot size to work out which file to open and
	// takes the remainder as the position in it.
	offsets []int
	// size is what goes in the header where the database's own length would
	// be, on a +3. The interpreter reads it to decide whether it has 16 KiB to
	// load or less than that.
	size int
}

// xmesSlot is how big one file, or one slot of a single file, may get.
//
// It is per machine because it is the interpreter's own arithmetic: the offset
// a condact carries is global, so the interpreter recovers the file from
// offset/slot and the position from offset%slot. Change the number and every
// offset already compiled means something else — this is an interface and not
// a tuning knob. From getXMessageFileSizeByTarget, drb.php:420-445.
func xmesSlot(machine arch.Machine, subtarget string) int {
	const k = 1024

	switch machine {
	case arch.ZX:
		// The two 128K machines page this through a 16 KiB window, so that is
		// what a slot is. A 48K has no filesystem to read an .XMB from at all,
		// which the packager is what says: here it is only a size.
		if xmesPagedWindow(subtarget) {
			return 16 * k
		}

		return 64 * k

	case arch.MSX2:
		return 16 * k

	case arch.CPC, arch.C64, arch.CP4:
		return 2 * k

	case arch.ZX81, arch.MSX, arch.PCW, arch.CPM, arch.HTML, arch.AMIGA, arch.ST:
		return 64 * k

	case arch.PC:
		if strings.EqualFold(subtarget, "vga256") {
			return 64 * k
		}
	}

	// Nought means the machine has nowhere to put one.
	return 0
}

// A machine that keeps every xmessage in one file pads each slot out instead of
// opening the next, so that there is always a whole slot to read from whatever
// offset — some of them choke on a read that runs past the end of a file.
func xmesOneFile(machine arch.Machine, subtarget string) bool {
	if machine == arch.MSX2 {
		return true
	}

	return machine == arch.ZX && xmesPagedWindow(subtarget)
}

// The two that need the gap at the front, and the two that need the room under
// the top of a slot. They are nearly the same pair and not quite: the +3 wants
// both, the Amiga wants the gap because on a real machine the first xmessage
// came out corrupted now and then, and the 128K wants the room because it does
// the same page swap the +3 does.
func xmesNeedsGap(machine arch.Machine, subtarget string) bool {
	return machine == arch.AMIGA ||
		(machine == arch.ZX && strings.EqualFold(subtarget, "plus3"))
}

func xmesNeedsSwapRoom(machine arch.Machine, subtarget string) bool {
	return machine == arch.ZX && xmesPagedWindow(subtarget)
}

// xmesPagedWindow is the pair of Spectrums that read this through a 16 KiB
// window: the +3 and the 128K, which are the same memory layout.
func xmesPagedWindow(subtarget string) bool {
	return strings.EqualFold(subtarget, "plus3") || strings.EqualFold(subtarget, "128k")
}

// xmesCondact turns an XMES the author wrote into the one the interpreter
// runs.
//
// What changes is everything: the opcode becomes the slot reserved for it, and
// the operand stops being which message it is and becomes where that message
// sits in the .XMB, in two bytes. Version 2 writes the same thing as an EXTERN
// with Maluva's function 3 wedged between the halves of the offset — see
// Condact in internal/ddb — and this does not do version 2.
func (l *linker) xmesCondact(c ddb.Condact) (ddb.Condact, error) {
	if c.XMessage.Int() >= len(l.xmes.offsets) {
		return c, fmt.Errorf("a process prints xmessage %d and there are %d",
			c.XMessage.Int(), len(l.xmes.offsets))
	}

	offset := l.xmes.offsets[c.XMessage.Int()]

	return ddb.Condact{
		Opcode: ddb.OpXMesFinal,
		Params: []ddb.Param{
			{Value: ddb.ID(offset & 0xFF)},
			{Value: ddb.ID(offset >> 8 & 0xFF)},
		},
	}, nil
}

// layoutXMessages places the texts and says where each one went.
//
// bodies are encoded and deliberately not tokenised. The texts are counted
// when the token table is chosen — so they change the database — and then
// written plain, because the reference writes this file before it substitutes
// anything. See prepare, which keeps the two apart and says why.
func layoutXMessages(bodies []string, machine arch.Machine, subtarget string) (xmessages, error) {
	if len(bodies) == 0 {
		return xmessages{}, nil
	}

	slot := xmesSlot(machine, subtarget)
	if slot == 0 {
		return xmessages{}, fmt.Errorf(
			"the %s has nowhere to read xmessages from, so it cannot have any", machine)
	}

	var (
		out    xmessages
		offset int
		file   int
	)

	oneFile := xmesOneFile(machine, subtarget)
	swapRoom := 0

	if xmesNeedsSwapRoom(machine, subtarget) {
		swapRoom = xmesSwap
	}

	if xmesNeedsGap(machine, subtarget) {
		out.body = append(out.body, make([]byte, xmesGap)...)
		offset += xmesGap
	}

	out.offsets = make([]int, len(bodies))

	for i, body := range bodies {
		// Whether this one still fits where we are. The second test is the
		// page swap's and does not look at the message at all: what it asks is
		// whether the swap would reach past the end of the slot.
		full := offset+len(body)+1 > slot
		tooNearTheTop := swapRoom > 0 && offset+swapRoom > slot

		if full || tooNearTheTop {
			if oneFile {
				// Pad this slot out and carry on in the same file.
				out.body = append(out.body, make([]byte, slot-offset)...)
			}

			file++
			offset = 0
		}

		out.offsets[i] = offset + file*slot

		if out.offsets[i] > xmesLimit {
			return xmessages{}, fmt.Errorf(
				"xmessage %d would sit at %d, and the offset a condact carries reaches %d",
				i, out.offsets[i], xmesLimit)
		}

		if len(body)+1 > slot {
			return xmessages{}, fmt.Errorf(
				"xmessage %d is %d bytes and a slot of this machine holds %d",
				i, len(body), slot-1)
		}

		for j := range len(body) {
			out.body = append(out.body, body[j]^obfuscate)
		}

		out.body = append(out.body, xmesEndOfText)
		offset += len(body) + 1
	}

	out.size = file*slot + offset

	return out, nil
}
