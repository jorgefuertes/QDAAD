package legacy

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// A label names an entry of a process so that a SKIP can say where it jumps
// without counting entries by hand. It is the one name in the language that
// never reaches the database: what a database holds is how far to jump, and
// working that out is all a label is for.
//
// Three of the five sources DAAD READY ships use them, which is why they are
// here. None of our own decompilations does — the decompiler writes the number
// it read — so nothing in the corpus exercised this until the templates were
// added to it.

// label is where a name was defined. The process is kept because a jump cannot
// cross from one process to another: the entry number a SKIP adds to is an
// index within its own process.
type label struct {
	process ddb.ID
	entry   int
	where   Token
}

// place is where in the processes the parser is reading. It is set as each
// condact is read and used by the one operand that needs it: a jump is
// measured in entries, so resolving a label takes both ends of the distance.
type place struct {
	process ddb.ID
	entry   int
	condact int
}

// pendingSkip is a SKIP that named a label the parser had not reached yet.
//
// A forward reference cannot be worked out where it is written, so the operand
// is left at zero and the condact noted. Everything needed to fill it in later
// is here, the block included: the heads of one entry group share their block,
// so patching it through this slice patches it for all of them, which is what
// the compiler of reference does as well.
type pendingSkip struct {
	name    string
	where   Token
	process ddb.ID
	from    int
	block   []ddb.Condact
	condact int
}

// labels is the whole source's labels and the references still waiting.
//
// One table for the whole source and not one per process, which is what makes
// "defined in another process" something that can be said at all: the name is
// found, and then it is the process that is wrong. It also means a name can be
// used once in a source and not once per process. That is the rule of the
// compiler of reference, at ULabelList.pas:33-45, and a source written for it
// would break under a laxer one.
type labelTable struct {
	defined map[string]label
	pending []pendingSkip
}

func newLabelTable() labelTable {
	return labelTable{defined: make(map[string]label)}
}

// defineLabel records where a name points.
//
// Redefining is an error rather than a move, which is the compiler of
// reference's rule and the safe one: the second of two SKIPs to the same name
// would otherwise jump somewhere the author did not write.
func (p *parser) defineLabel(t Token, process ddb.ID, entry int) error {
	if was, already := p.labels.defined[t.Text]; already {
		return fmt.Errorf("%s: the label $%s is already defined, at %s",
			t.Where(), t.Text, was.where.Where())
	}

	p.labels.defined[t.Text] = label{process: process, entry: entry, where: t}

	return nil
}

// skipToLabel works out the operand of a SKIP that named a label.
//
// A label already defined is resolved here. One that is not yet is noted and
// left at zero for resolvePending, which runs when the whole source has been
// read: a label may be defined anywhere after the SKIP that uses it, and until
// the end there is no telling whether it ever will be.
func (p *parser) skipToLabel(t Token) (ddb.ID, error) {
	known, defined := p.labels.defined[t.Text]
	if !defined {
		p.labels.pending = append(p.labels.pending, pendingSkip{
			name:    t.Text,
			where:   t,
			process: p.place.process,
			from:    p.place.entry,
			condact: p.place.condact,
		})

		return 0, nil
	}

	return jumpTo(t, p.place.process, p.place.entry, known)
}

// jumpTo is the arithmetic, and it is the same whichever pass gets there.
//
// The jump is how many entries to move on from the one after this: the
// interpreter adds it to the entry number and carries on, so landing on entry
// E from entry F means E - F - 1. A label on the entry right after a SKIP is
// therefore a jump of zero.
//
// The range is a signed byte and the two ends are checked where the compiler of
// reference checks them, which is why they are spelled differently: it tests
// the value against -128 going back and the distance against 128 going forward.
// The two meet at the same place — a jump of 128 forward is a value of 127 — so
// what this allows is exactly what fits in the byte.
func jumpTo(t Token, process ddb.ID, from int, to label) (ddb.ID, error) {
	if to.process != process {
		return 0, fmt.Errorf("%s: the label $%s is in process %d and this is process %d, "+
			"and a jump cannot leave its process", t.Where(), t.Text, to.process, process)
	}

	if to.entry-from > 128 {
		return 0, fmt.Errorf("%s: the label $%s is %d entries ahead, and a jump reaches 128",
			t.Where(), t.Text, to.entry-from)
	}

	jump := to.entry - from - 1

	if jump < -128 {
		return 0, fmt.Errorf("%s: the label $%s is %d entries back, and a jump reaches 128",
			t.Where(), t.Text, from-to.entry)
	}

	if jump < 0 {
		jump += 256
	}

	return ddb.ID(jump), nil
}

// resolvePending fills in the SKIPs that named a label before it existed.
//
// It runs once the whole source is read, because that is the first moment a
// name can be called undefined.
//
// Every pending reference is resolved and not only the first of an entry. The
// compiler of reference stops at the first one it fixes in each entry —
// USintactic.pas:84, a break inside the condact loop — so an entry with two
// forward SKIPs leaves the second one jumping to zero. That is a bug and not a
// rule, and nothing in the corpus has two.
func (p *parser) resolvePending() error {
	for _, waiting := range p.labels.pending {
		known, defined := p.labels.defined[waiting.name]
		if !defined {
			return fmt.Errorf("%s: the label $%s is used here and defined nowhere",
				waiting.where.Where(), waiting.name)
		}

		jump, err := jumpTo(waiting.where, waiting.process, waiting.from, known)
		if err != nil {
			return err
		}

		waiting.block[waiting.condact].Params[0].Value = jump
	}

	p.labels.pending = nil

	return nil
}
