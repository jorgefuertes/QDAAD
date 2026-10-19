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
// known is where a block is still read wrong and why, by its address. Two are
// left and they are the same fourteen bytes in both parts of the adventure:
//
//	4F 22 63 0E 1D 7F 1A 0F 1D 80 31 1D 80 FF
//
// Read as NOTEQ 34 99 / GT 29 127 / LOAD 15 / CLS / AT @49 / CLS / AT @255,
// which eats the FF. Two other readings end exactly on it — 0x80 as an opcode
// of its own rather than an indirection, or opcode 29 taking an operand in the
// old format — and both then give PLUS 29 128, which pairs with the MINUS 29
// 128 in the block laid next to it. Nothing here says which is right, and
// guessing would be worse than the hole: see issue #8.
var databases = []struct {
	name, path string
	version    int
	known      map[int]string
}{
	{
		name:    "AO part 1",
		path:    "../../../work/aventuras/La_Aventura_Original/PC/PART1.DDB",
		version: 1,
		known:   map[int]string{11274: "issue #8"},
	},
	{
		name:    "AO part 2",
		path:    "../../../work/aventuras/La_Aventura_Original/PC/PART2.DDB",
		version: 1,
		known:   map[int]string{16245: "issue #8"},
	},
}

func TestEveryBlockEndsWhereTheNextBegins(t *testing.T) {
	for _, db := range databases {
		t.Run(db.name, func(t *testing.T) {
			r := openDatabase(t, db.path)

			require.Equal(t, db.version == 1, r.OldFormat(),
				"the test says this is version %d", db.version)

			left := make(map[int]string, len(db.known))
			for at, why := range db.known {
				left[at] = why
			}

			for at, overrun := range overruns(t, r) {
				if _, expected := left[at]; expected {
					delete(left, at)

					continue
				}

				t.Errorf("%s", overrun)
			}

			// A hole that closes has to be taken off the list, or the list
			// slowly stops meaning anything.
			for at, why := range left {
				t.Errorf("the block at %d no longer runs over the next one: "+
					"take it off the known list (%s)", at, why)
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
