package decompiler

import (
	"os"
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
// over La Aventura Original. Here it is as a test, so that the next table
// change has to answer to it.

// databases are the ones the check runs over: a plain .DDB file each, so that
// nothing about disks or archives is in the way.
//
// unresolved is how many blocks are still read wrong, and it is not a target
// but a record: every one of them is a condact whose arity we have not settled.
// The number is pinned so that the check is a ratchet — a table change that
// closes one has to say so here, and one that opens a new one fails.
//
// La Aventura Original is at zero since its interpreter was read and the
// dialect taken from it. El Jabato is not: it shipped an interpreter of its
// own, nobody has opened it, and twenty-one of its blocks do not line up under
// any table we have.
var databases = []struct {
	name, path string
	version    int
	dialect    string
	unresolved int
}{
	{
		name:       "AO part 1",
		path:       "../../../work/aventuras/La_Aventura_Original/PC/PART1.DDB",
		version:    1,
		dialect:    "La Aventura Original",
		unresolved: 0,
	},
	{
		name:       "AO part 2",
		path:       "../../../work/aventuras/La_Aventura_Original/PC/PART2.DDB",
		version:    1,
		dialect:    "La Aventura Original",
		unresolved: 0,
	},
	{
		name:       "Jabato part 1",
		path:       "../../../work/aventuras/El_Jabato/PC/CGA/PART1.DDB",
		version:    1,
		dialect:    "classic",
		unresolved: 6,
	},
	{
		name:       "Jabato part 2",
		path:       "../../../work/aventuras/El_Jabato/PC/CGA/PART2.DDB",
		version:    1,
		dialect:    "classic",
		unresolved: 9,
	},
	{
		name:       "Jabato part 1, EGA",
		path:       "../../../work/aventuras/El_Jabato/PC/Ega/PART1.DDB",
		version:    1,
		dialect:    "classic",
		unresolved: 6,
	},
}

func TestEveryBlockEndsWhereTheNextBegins(t *testing.T) {
	for _, db := range databases {
		t.Run(db.name, func(t *testing.T) {
			r := openDatabase(t, db.path)

			require.Equal(t, db.version == 1, r.OldFormat(),
				"the test says this is version %d", db.version)

			require.Equal(t, db.dialect, r.dialect.name,
				"the condact table deduced for this database")

			// Either way round is worth failing for. More means a table change
			// broke something; fewer means one was settled and the number here
			// has to come down to say so, or the ratchet slips.
			broken, over := r.blockFaults()
			require.Zero(t, broken, "blocks that will not decode at all")
			require.Equal(t, db.unresolved, over,
				"blocks running over the one laid after them")
		})
	}
}

// The dialect is deduced and not declared, so the deduction is worth a test of
// its own: the table that makes a database lie flat is the one it was built
// with, and the two adventures do not pick the same one.
func TestTheDialectIsTheOneThatMakesTheBlocksLineUp(t *testing.T) {
	ao := openDatabase(t, "../../../work/aventuras/La_Aventura_Original/PC/PART1.DDB")

	require.Equal(t, "La Aventura Original", ao.dialect.name)
	broken, over := ao.blockFaults()
	require.Zero(t, broken)
	require.Zero(t, over)

	// And under the other table it does not lie flat, which is what makes the
	// choice a choice rather than a preference.
	ao.dialect = v1Dialects[0]
	_, over = ao.blockFaults()
	require.Positive(t, over,
		"the classic table leaves blocks running over their neighbour")

	jabato := openDatabase(t, "../../../work/aventuras/El_Jabato/PC/CGA/PART1.DDB")

	require.Equal(t, "classic", jabato.dialect.name,
		"its interpreter is not the one La Aventura Original shipped")
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
