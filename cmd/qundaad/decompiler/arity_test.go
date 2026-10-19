package decompiler

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// The arity of the condacts is the one thing in a database that the database
// does not say. A block is a run of bytes where an opcode is followed by as
// many operands as the table claims, and nothing marks where one condact ends
// and the next begins: read one operand too few or too many and everything
// after it shifts, silently, into something that still decodes.
//
// What catches that is the layout. Every entry reaches its block through a
// pointer of its own, and the blocks are laid one after another, so a block
// that runs past where the next one starts has been read with the wrong arity.
// It cannot be anything else — the two addresses come from the binary and the
// walk between them comes from the table.
//
// This is the check the note in oldcondacts.go describes having done by hand
// over La Aventura Original. Here it is as a test, over every database we can
// open, so that the next table change has to answer to it.

// databases are the ones the check runs over: a plain .DDB file each, so that
// nothing about disks or archives is in the way.
//
// unresolved is how many blocks are still read wrong, and it is not a target
// but a record: every one of them is a condact whose arity we have not settled.
// The number is pinned so that the check is a ratchet — a table change that
// closes one has to say so here, and one that opens a new one fails.
//
// The two in La Aventura Original are the fourteen bytes of issue #8. The
// twenty-one in El Jabato are a separate matter and were found by widening
// this list: its interpreter is not the one shipped with La Aventura Original
// and its condacts do not all take what ours say they take.
var databases = []struct {
	name, path string
	version    int
	unresolved int
}{
	{
		name:       "AO part 1",
		path:       "../../../work/aventuras/La_Aventura_Original/PC/PART1.DDB",
		version:    1,
		unresolved: 1,
	},
	{
		name:       "AO part 2",
		path:       "../../../work/aventuras/La_Aventura_Original/PC/PART2.DDB",
		version:    1,
		unresolved: 1,
	},
	{
		name:       "Jabato part 1",
		path:       "../../../work/aventuras/El_Jabato/PC/CGA/PART1.DDB",
		version:    1,
		unresolved: 6,
	},
	{
		name:       "Jabato part 2",
		path:       "../../../work/aventuras/El_Jabato/PC/CGA/PART2.DDB",
		version:    1,
		unresolved: 9,
	},
	{
		name:       "Jabato part 1, EGA",
		path:       "../../../work/aventuras/El_Jabato/PC/Ega/PART1.DDB",
		version:    1,
		unresolved: 6,
	},
}

func TestEveryBlockEndsWhereTheNextBegins(t *testing.T) {
	for _, db := range databases {
		t.Run(db.name, func(t *testing.T) {
			r := openDatabase(t, db.path)

			require.Equal(t, db.version == 1, r.OldFormat(),
				"the test says this is version %d", db.version)

			found := overruns(t, r)

			if len(found) == db.unresolved {
				return
			}

			// Either way round is worth failing for. More means a table change
			// broke something; fewer means one was settled and the number here
			// has to come down to say so, or the ratchet slips.
			t.Errorf("%d blocks run over the next one, and %d is what this "+
				"database is recorded as having", len(found), db.unresolved)

			for _, overrun := range found {
				t.Logf("  %s", overrun)
			}
		})
	}
}

func openDatabase(t *testing.T, path string) *Reader {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("%s is not in the checkout: %v", path, err)
	}

	r, ok := readerAt(data, 0)
	require.True(t, ok, "%s does not read as a database", path)

	return r
}

// overruns walks every block and reports the ones that run past the start of
// the block after them.
//
// The process table is walked here rather than through Processes, because what
// is being checked is where each block sits and that is the one thing the
// decoded entry does not carry.
func overruns(t *testing.T, r *Reader) map[int]string {
	t.Helper()

	const (
		entrySize    = 4
		endOfProcess = 0x00
	)

	type span struct {
		process, entry int
		start, end     int
	}

	var spans []span

	table := r.Pointer(ptrProcesses)

	for i := range r.NumProcesses() {
		p := r.address(table + 2*i)
		if !r.inRange(p, entrySize) {
			break
		}

		for entry := 0; r.inRange(p, entrySize) && r.data[p] != endOfProcess; entry++ {
			at := r.address(p + 2)

			condacts, err := r.condactsAt(at)
			require.NoError(t, err, "process %d entry %d", i, entry)

			spans = append(spans, span{
				process: i,
				entry:   entry,
				start:   at,
				end:     at + r.blockBytes(condacts),
			})

			p += entrySize
		}
	}

	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	found := make(map[int]string)

	for i := range spans {
		// The next block that starts anywhere after this one. Two entries may
		// share a block — the same pointer twice — and that is not an overrun.
		next := -1

		for j := i + 1; j < len(spans); j++ {
			if spans[j].start > spans[i].start {
				next = spans[j].start

				break
			}
		}

		if next < 0 || spans[i].end <= next {
			continue
		}

		found[spans[i].start] = fmt.Sprintf(
			"process %d entry %d: its block runs from %d to %d, over the block at %d, "+
				"so something in it was read with the wrong number of operands",
			spans[i].process, spans[i].entry, spans[i].start, spans[i].end, next)
	}

	return found
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
