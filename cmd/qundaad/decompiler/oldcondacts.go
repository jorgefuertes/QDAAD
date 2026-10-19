package decompiler

import (
	"fmt"

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
	const (
		opDesc  = 19
		opMouse = 86
	)

	if !r.OldFormat() || c.Indirect {
		return "", false
	}

	switch {
	case c.Opcode == opDesc && len(c.Params) == 0:
		return fmt.Sprintf("DESC @%d   ; was a bare DESC: the old one means here", fPlayer), true

	case c.Opcode == opMouse && len(c.Params) == 1:
		return fmt.Sprintf("MOUSE %d 0   ; was MOUSE %d: the 0 is padding, not in the database",
			c.Params[0], c.Params[0]), true
	}

	return "", false
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
