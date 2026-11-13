package decompiler

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
)

var uncertainArity = map[byte]string{
	101: "CALL: the manual writes one operand, this reads two. See oldcondacts.go",
}

// fPlayer is the flag holding the location the player is in. See
// docs/DAAD_v2_v3_Research/05-flujo-ejecucion.md §flags.
const fPlayer = 38

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
func oldCondactText(r *ddbfile.Reader, c ddbfile.Condact) (string, bool) {
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

	// What is missing is measured against the condact in hand and not against
	// the old table's arity, because the two stopped agreeing the moment a
	// conversion was allowed to supply an operand. A converted PARSE arrives
	// with its 1 already in it, and padding it by the old arity wrote
	// "PARSE 1 0" — an operand the author never asked for, in a condact that
	// takes one.
	if len(c.Params) >= len(modern.Params) {
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

	short := len(modern.Params) - len(c.Params)
	for range short {
		out += " 0"
	}

	return fmt.Sprintf("%s   ; the old %s takes %d, so the last %d is padding and not in the database",
		out, old.Name, len(old.Params), short), true
}
