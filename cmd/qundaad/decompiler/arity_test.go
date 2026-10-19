package decompiler

import (
	"os"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
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
// over La Aventura Original. Here it is as a test, over both version 1
// adventures, so that the next table change has to answer to it.
//
// It is also the check that decided every arity in that file. The interpreter
// of La Aventura Original was read to find them and suggested eight more than
// are there: those eight make its own two databases lie just as flat and make
// El Jabato stop reading, so the layout threw them out. Two tables agreeing
// about one adventure prove nothing; five databases from two adventures,
// shipped with different interpreters, are a different matter.
var databases = []struct {
	name, path string
	version    int
}{
	{
		name:    "AO part 1",
		path:    "../../../work/aventuras/La_Aventura_Original/PC/PART1.DDB",
		version: 1,
	},
	{
		name:    "AO part 2",
		path:    "../../../work/aventuras/La_Aventura_Original/PC/PART2.DDB",
		version: 1,
	},
	{
		name:    "Jabato part 1",
		path:    "../../../work/aventuras/El_Jabato/PC/CGA/PART1.DDB",
		version: 1,
	},
	{
		name:    "Jabato part 2",
		path:    "../../../work/aventuras/El_Jabato/PC/CGA/PART2.DDB",
		version: 1,
	},
	{
		name:    "Jabato part 1, EGA",
		path:    "../../../work/aventuras/El_Jabato/PC/Ega/PART1.DDB",
		version: 1,
	},
}

func TestEveryBlockEndsWhereTheNextBegins(t *testing.T) {
	for _, db := range databases {
		t.Run(db.name, func(t *testing.T) {
			r := openDatabase(t, db.path)

			require.Equal(t, db.version == 1, r.OldFormat(),
				"the test says this is version %d", db.version)

			broken, over := r.blockFaults()

			require.Zero(t, broken, "blocks that will not decode at all")
			require.Zero(t, over, "blocks running over the one laid after them")
		})
	}
}

// And the two arities the layout settled, held the other way round: with the
// reading version 2 has, the blocks stop lining up.
//
// Without this the test above would pass just as well on a table that never
// knew the difference, and the point of those two entries would be invisible.
func TestTheOldArityIsWhatMakesThemLineUp(t *testing.T) {
	for _, op := range []byte{26, 73} { // LOAD and PARSE
		t.Run(oldCondacts[op].Name, func(t *testing.T) {
			was := oldCondacts[op]
			defer func() { oldCondacts[op] = was }()

			// The version 2 reading: one operand where the old format has none.
			modern, found := ddb.LookupCondact(ddb.Opcode(op))
			require.True(t, found)
			require.Len(t, modern.Params, 1)

			oldCondacts[op] = modern

			faults := 0

			for _, db := range databases {
				r := openDatabase(t, db.path)
				broken, over := r.blockFaults()
				faults += broken + over
			}

			require.Positive(t, faults,
				"with an operand this condact leaves blocks that do not line up")
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
