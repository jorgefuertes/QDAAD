package decompiler

import (
	"fmt"
	"sort"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// Four opcodes meant something else before version 2 reassigned them. In the
// old format 56 to 59 hold the COPY family, each taking two parameters; the
// version 2 table this project compiles for puts SETCO, SPACE, HASAT and
// HASNAT there, with one parameter or none.
//
// The arity is the part that matters, and it is proven: with one parameter the
// walk over La Aventura Original swallows whole blocks and yields impossible
// operands such as "LT 255 49" in a game of 49 locations. With two, the 253
// blocks of part one and the 357 of part two decode without a single
// contradiction between overlapping blocks.
//
// The names come from the undaad and unDRC decompilers, which are secondary
// sources: the 1991 manual documents version 2, where these opcodes already
// mean something else, so there is no primary source naming them. A complete
// and consecutive COPY family is too neat to be a coincidence, but it is worth
// remembering where it comes from.
var oldCondacts = map[byte]ddb.CondactDef{
	// DESC describes the current location and takes nothing, as in PAWS. The
	// version 2 table gives it a location to describe.
	//
	// Every DESC in La Aventura Original and El Jabato came out as "DESC 255"
	// — 1142 of them across the editions, without one exception — and 255 is
	// the 0xFF that ends a block. So the operand was never an operand: it was
	// the terminator, eaten, after which the walk carried on into the entry
	// laid next to this one and copied its condacts into this one's block.
	// Reported as issue #1.
	19: {Name: "DESC", Params: nil},

	// MOUSE takes one option, which is how the 1991 manual writes it ("MOUSE
	// option") and how the undaad decompiler reads it. The version 2 table
	// gives it two.
	//
	// It appears four times in the two version 1 adventures and never more
	// than once in a database, so no single reading contradicts itself — which
	// is why the note below could not settle it from this evidence alone. What
	// settles it is the layout: see arity_test.go. Issue #2.
	86: {Name: "MOUSE", Params: []ddb.ParamKind{ddb.ParamValue}},

	// LOAD and PARSE take nothing in the old format. The version 2 table gives
	// each of them one operand, and that operand was the next condact.
	//
	// This is what the interpreter does. AD.EXE, shipped in the same directory
	// as the two databases of La Aventura Original, has its condact loop at
	// 0x233 of the code segment and a table of 109 handlers at DS:0x5cd:
	//
	//	0233  mov al, [bx]        the opcode byte
	//	0235  cmp al, 0xff        the end of the block
	//	0239  mov [0x4c], al      kept whole, bit 7 and all
	//	023d  and al, 0x7f        and masked off to index the table
	//	023f  cmp al, 0x6d        109 condacts, and no more
	//	025a  inc bx              past the opcode
	//	025b  mov al, [bx]        the byte after it, whatever it turns out to be
	//	025d  test [0x4c], 0x80   was the opcode marked?
	//	0264  mov di, 0x12a       then that byte is a flag number
	//	0269  mov al, [di]        and the flag's content is the operand
	//
	// So bit 7 is the indirection and nothing else, and every handler is
	// entered with the next byte already read and bx still sitting on it.
	// Whether that byte was an operand is the handler's to say, and it says so
	// by where it leaves: 0x690 does an inc bx and takes one, 0x691 takes none.
	// The handlers for these two leave at 0x691.
	//
	// LOAD is also where issue #8 was hiding. The two blocks of La Aventura
	// Original that ran over their neighbour did so because of this one
	// operand; without it the fourteen bytes read NOTEQ 34 99 / GT 29 127 /
	// LOAD / LT 29 128 / PLUS 29 128 and end exactly on the terminator, with a
	// PLUS matching the MINUS in the block laid beside it. The two readings
	// that issue weighed are both refused by the loop above: one needed bit 7
	// not to be the indirection, the other needed CLS to take an operand.
	//
	// Walking the other 107 handlers suggests eight more differences, and the
	// layout says they are wrong: with them El Jabato stops reading at all.
	// They are not here. The walk over a handler is a guess about where control
	// goes and these two are the ones the databases confirm — in both
	// adventures, across five databases, every block ends where the next
	// begins. See arity_test.go.
	26: {Name: "LOAD", Params: nil},
	73: {Name: "PARSE", Params: nil},

	56: {Name: "COPYOF", Params: []ddb.ParamKind{ddb.ParamObject, ddb.ParamFlag}},
	57: {Name: "COPYOO", Params: []ddb.ParamKind{ddb.ParamObject, ddb.ParamObject}},
	58: {Name: "COPYFO", Params: []ddb.ParamKind{ddb.ParamFlag, ddb.ParamObject}},
	59: {Name: "COPYFF", Params: []ddb.ParamKind{ddb.ParamFlag, ddb.ParamFlag}},
}

// TODO: settle the arity of CALL in old databases.
//
// The 1991 manual writes it with a single operand, "CALL address", and both
// existing decompilers read one byte. The table this project compiles for
// gives it two, and that is the reading that makes sense: an address is 16
// bits and a DAAD parameter is a byte, so it takes two of them. Read as one,
// the only CALL in La Aventura Original comes out as "CALL 3", a call to
// address 3.
//
// Nothing available decides it. It appears once per database, in a block
// nothing else shares and which ends where the next one starts under either
// reading, so the layout check in arity_test.go is no help either.
//
// Two parameters are kept, which is what an address needs. Settling it takes
// evidence this adventure cannot give: an interpreter of the period, or a game
// that uses it more than once.
//
// MOUSE was on this list and came off it. The layout settled it — see the note
// on opcode 86 above — which is what the check over the blocks is for.
var uncertainArity = map[byte]string{
	101: "CALL: the manual writes one operand, this reads two. See oldcondacts.go",
}

// fPlayer is the flag holding the location the player is in. See
// docs/DAAD_v2_v3_Research/05-flujo-ejecucion.md §flags.
const fPlayer = 38

// oldCondactText writes a version 1 condact that has no version 3 spelling.
//
// Two of them take a different number of operands in the old format, and a
// version 3 compiler — ours included — reads neither as written. Left alone
// they are not sources at all: a bare DESC makes the compiler take whatever
// follows as its location, and a one-operand MOUSE does the same with the next
// condact's name.
//
// So they are translated, and this is the one place the emitted source says
// something the binary does not. It is the bargain the emitter already makes
// everywhere — the source is version 3 and the database is not — and keeping
// it here is what lets the walk over the binary stay literal: the reader still
// decodes DESC with no operands and MOUSE with one, which is what stops the
// entries swallowing their neighbours.
//
// The two translations are not equally sound and the comments in the source say
// so. DESC is exact: the old one describes wherever the player is, and
// "DESC @fPlayer" is the version 3 way of saying that — it is what the version
// 2 adventures of this corpus write when they mean "here". MOUSE is not: the
// documented version 3 signature is two values and nothing says what the second
// one is, so the old single option is kept and a zero put beside it. That zero
// is a guess, it is the only guess in the emitter, and it is one condact per
// database — 24 in the corpus, against 879 DESCs.
func (r *Reader) oldCondactText(c Condact) (string, bool) {
	const opDesc = 19

	if !r.OldFormat() || c.Indirect {
		return "", false
	}

	old, found := r.Condact(c.Opcode)
	if !found {
		return "", false
	}

	// Only a condact that takes fewer operands than version 3 gives it needs
	// anything doing. One that takes more, or the same, writes itself.
	modern, known := ddb.LookupCondact(ddb.Opcode(c.Opcode))
	if !known || len(old.Params) >= len(modern.Params) {
		return "", false
	}

	// DESC is the one with something to say rather than a hole to fill.
	if c.Opcode == opDesc && len(c.Params) == 0 {
		return fmt.Sprintf("DESC @%d   ; was a bare DESC: the old one means here", fPlayer), true
	}

	out := old.Name

	for _, p := range c.Params {
		out += fmt.Sprintf(" %d", p)
	}

	short := len(modern.Params) - len(old.Params)
	for range short {
		out += " 0"
	}

	return fmt.Sprintf("%s   ; the old %s takes %d, so the last %d is padding and not in the database",
		out, old.Name, len(old.Params), short), true
}

// blockFaults counts what the condact table gets wrong about the layout.
//
// broken is the blocks that will not decode at all — an opcode outside the
// table, or operands running off the end — and over is the blocks that decode
// but then run past the start of the block laid after them.
//
// A block that runs long cannot have been read right. Every entry reaches its
// block through a pointer of its own and the blocks lie one after another, so
// both addresses come from the binary while the walk between them comes from
// the table: when they disagree it is the table that is wrong. That is what
// settled the arity of DESC, of MOUSE, and of the two above, and it is the only
// measure here that the databases themselves can give.
//
// The two are counted apart because they are not the same kind of wrong. A
// database with a broken block cannot be decompiled at all; one with an overrun
// comes out with a few entries carrying their neighbour's condacts.
func (r *Reader) blockFaults() (broken, over int) {
	const (
		entrySize    = 4
		endOfProcess = 0x00
	)

	type span struct{ start, end int }

	var spans []span

	table := r.Pointer(ptrProcesses)

	for i := range r.NumProcesses() {
		p := r.address(table + 2*i)
		if !r.inRange(p, entrySize) {
			break
		}

		for r.inRange(p, entrySize) && r.data[p] != endOfProcess {
			at := r.address(p + 2)

			if condacts, err := r.condactsAt(at); err != nil {
				broken++
			} else {
				spans = append(spans, span{start: at, end: at + r.blockBytes(condacts)})
			}

			p += entrySize
		}
	}

	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	for i := range spans {
		// The next block that starts anywhere after this one. Two entries may
		// share a block — the same pointer twice — and that is not an overrun.
		for j := i + 1; j < len(spans); j++ {
			if spans[j].start <= spans[i].start {
				continue
			}

			if spans[i].end > spans[j].start {
				over++
			}

			break
		}
	}

	return broken, over
}

// blockBytes is how many bytes a decoded block took: an opcode and its operands
// each, and the 0xFF after them unless the last condact ended the block itself.
func (r *Reader) blockBytes(condacts []Condact) int {
	if len(condacts) == 0 {
		return 1
	}

	n := 0

	for _, c := range condacts {
		n += 1 + len(c.Params)
	}

	if def, found := r.Condact(condacts[len(condacts)-1].Opcode); found && def.Terminal {
		return n
	}

	return n + 1
}

// OldFormat reports whether this is a pre-version-2 database.
//
// Nothing in the file says so outright: the version byte is unreliable across
// the intermediate builds of the era. What does say so is the shape of the
// header, since version 2 inserted the pointer to the extra object attributes.
// The undaad decompiler decides the same way, by checking whether the word at
// offset 30 is the length of the file.
func (r *Reader) OldFormat() bool {
	return r.headerSize == headerSizeV1
}

// Condact resolves an opcode against the table this database belongs to.
func (r *Reader) Condact(opcode byte) (ddb.CondactDef, bool) {
	if r.OldFormat() {
		if def, found := oldCondacts[opcode]; found {
			return def, true
		}
	}

	return ddb.LookupCondact(ddb.Opcode(opcode))
}
