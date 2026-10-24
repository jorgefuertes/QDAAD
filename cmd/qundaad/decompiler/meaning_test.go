package decompiler

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/stretchr/testify/require"
)

// The arity test next door proves how many operands each condact takes. It
// does not prove what any of them means, and the two are not the same
// question: a table that called opcode 54 something other than SYSMESS would
// lay every block just as flat, because the blocks only care about lengths.
//
// What this adds is the operands themselves. Most of them are not free bytes —
// an object number has to be inside the number of objects, a message inside
// the number of messages, a percentage under a hundred — so a condact filed
// under the wrong name usually points its operands somewhere impossible.
//
// It is weaker than the layout check and worth saying how. It cannot see a
// mistake between two condacts that take the same kinds of operand, and it
// says nothing at all about the ones whose operands are flags or free values —
// which is half the operands in these databases and most of the common
// condacts: DONE, LET, SET, CLEAR, WINDOW, COPYFF, PRINTAT.
//
// Those stay on two other things. One is the layout check next door. The other
// is the DAAD manual of 1991, in work/, which describes every condact in this
// table and misses exactly two: INDIR and SETAT, which are the two DAAD added
// for version 3. So the vocabulary is not in question — only which number
// names which condact, and the five that version 1 numbers differently are in
// oldcondacts.go.
//
// One idea that did not survive contact with the data, recorded so that nobody
// spends an afternoon on it again: conditions do not come first. It would be a
// fine extra check if they did, but the response table of La Aventura Original
// has "SYSMESS 7, EQ 33 13, SYSMESS 3" — a condition after an action, building
// a sentence out of pieces — and that shape is everywhere.
func TestEveryOperandFitsWhatItIsSupposedToBe(t *testing.T) {
	// Looked at and skipped across every database, so that the test can say
	// how much of the question it actually answered.
	var looked, blind int

	for _, db := range databases {
		t.Run(db.name, func(t *testing.T) {
			r := openDatabase(t, db.path)

			processes, err := r.Processes()
			require.NoError(t, err)

			for p, entries := range processes {
				for e, entry := range entries {
					for _, c := range entry.Condacts {
						def, found := r.Condact(c.Opcode)
						require.True(t, found, "opcode %d is in no table", c.Opcode)

						for i, value := range c.Params {
							if i >= len(def.Params) {
								continue
							}

							// Indirection is bit 7 of the opcode and reaches
							// the first operand only: it names a flag, and
							// what that flag holds at run time is nobody's
							// business here.
							if i == 0 && c.Indirect {
								blind++

								continue
							}

							bound, discriminating := boundOf(r, def.Params[i])
							if !discriminating {
								blind++

								continue
							}

							looked++

							// The markers live at the top of the byte and are
							// not locations at all: 252 is "not created", 253
							// "worn", 254 "carried" and 255 "here", as the 1991
							// manual spells out and ddb.LOC_* already say. A
							// condact that takes them accepts either.
							if def.Params[i] == ddb.ParamLocationOrMarker && int(value) >= firstMarker {
								continue
							}

							require.Less(t, int(value), bound,
								"process %d, entry %d: %s operand %d is %d, and there are only %d",
								p, e, def.Name, i+1, value, bound)
						}
					}
				}
			}
		})
	}

	// And the trap this test can fall into itself: a walk that looks at
	// nothing passes just as green as one that looks at everything. So the
	// reach is part of the result rather than a detail — about half of the
	// operands in these five databases are of a kind that can be judged, and
	// the other half are flags and free bytes that any value fits.
	//
	// The floor is where the corpus stands today, less a little room. It is a
	// floor and not a target: another database, or another kind made
	// checkable, should raise it.
	t.Logf("%d operands checked, %d blind", looked, blind)
	require.Greater(t, looked, 4000,
		"the oracle has stopped looking: %d checked, %d blind", looked, blind)
}

// firstMarker is where the location markers start. Below it a locno_ operand
// is a real location; from it up it is one of ddb.LOC_NOT_CREATED, LOC_WORN,
// LOC_CARRIED or LOC_HERE.
const firstMarker = int(ddb.LOC_NOT_CREATED)

// boundOf is how large an operand of this kind may be in this database, and
// whether that bound says anything at all. A flag or a free value is any byte,
// so those two answer no and are counted as blind rather than passed.
func boundOf(r *Reader, kind ddb.ParamKind) (bound int, discriminating bool) {
	switch kind {
	case ddb.ParamObject:
		return r.NumObjects(), true
	// Both are locations; the kind that also takes a marker is let through
	// above when the value is one.
	case ddb.ParamLocation, ddb.ParamLocationOrMarker:
		return r.NumLocations(), true
	case ddb.ParamMessage:
		return r.NumMessages(), true
	case ddb.ParamSysMessage:
		return r.NumSysMessages(), true
	case ddb.ParamProcess:
		return r.NumProcesses(), true
	case ddb.ParamPercent:
		return 100, true
	case ddb.ParamWindow:
		return 8, true
	default:
		return 0, false
	}
}
