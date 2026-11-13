package legacy

import (
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/stretchr/testify/require"
)

// The processes of the bare source, so that a test can put its own in their
// place and still have a whole world around them.
const bareProcesses = `/PRO 0
> COGER LLAVE   MESSAGE 0
                DONE
> _     _       SYSMESS 0
/END`

// withProcesses is the bare source with its processes swapped for these.
func withProcesses(t *testing.T, processes string) (*ddb.DDB, error) {
	t.Helper()

	source := strings.Replace(bare, bareProcesses, processes, 1)
	require.NotEqual(t, bare, source, "the processes of the bare source moved")

	return parse(t, source)
}

// skipIn digs out the operand of the first SKIP of an entry, which is the whole
// point of a label: what a database holds is the distance, never the name.
func skipIn(t *testing.T, db *ddb.DDB, process ddb.ID, entry int) ddb.ID {
	t.Helper()

	proc, found := db.Processes().Get(process)
	require.True(t, found)
	require.Greater(t, len(proc.Entries), entry)

	for _, c := range proc.Entries[entry].Condacts {
		if nameOf(t, c.Opcode) == "SKIP" {
			return c.Params[0].Value
		}
	}

	t.Fatalf("entry %d of process %d has no SKIP", entry, process)

	return 0
}

// A jump is how far to move on from the entry after the one jumping, so a label
// on the next entry is a jump of zero and one entry back is 255.
func TestALabelBecomesTheDistance(t *testing.T) {
	for _, tc := range []struct {
		name      string
		processes string
		entry     int
		want      ddb.ID
	}{
		{
			name: "forward to the next entry is zero",
			processes: `/PRO 0
> COGER LLAVE   SKIP $next
$next
> _     _       SYSMESS 0
/END`,
			entry: 0,
			want:  0,
		},
		{
			name: "back to the entry before is 254, which is minus two",
			processes: `/PRO 0
$top
> COGER LLAVE   MESSAGE 0
                DONE
> _     _       SKIP $top
/END`,
			entry: 1,
			want:  254,
		},
		{
			// A label in the middle of a block names the entry after the one
			// the block belongs to, so this one lands on the entry holding the
			// SKIP itself: a jump of minus one, which loops.
			name: "a label inside a block names the entry after it",
			processes: `/PRO 0
> COGER LLAVE   MESSAGE 0
                $here
                DONE
> _     _       SKIP $here
/END`,
			entry: 1,
			want:  255,
		},
		{
			name: "a label used before it is written reads the same",
			processes: `/PRO 0
> COGER LLAVE   SKIP $end
                DONE
> _     _       SYSMESS 0
$end
> _     LLAVE   DONE
/END`,
			entry: 0,
			want:  1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := withProcesses(t, tc.processes)
			require.NoError(t, err)
			require.Equal(t, tc.want, skipIn(t, db, 0, tc.entry))
		})
	}
}

// A label is resolved the same whether the SKIP found it or waited for it, and
// this is the proof: the same two entries, written both ways round.
func TestTheTwoPassesAgree(t *testing.T) {
	forward, err := withProcesses(t, `/PRO 0
> COGER LLAVE   SKIP $there
$there
> _     _       SYSMESS 0
/END`)
	require.NoError(t, err)

	backward, err := withProcesses(t, `/PRO 0
$here
> COGER LLAVE   MESSAGE 0
> _     _       SKIP $here
/END`)
	require.NoError(t, err)

	require.Equal(t, ddb.ID(0), skipIn(t, forward, 0, 0))
	require.Equal(t, ddb.ID(254), skipIn(t, backward, 0, 1))
}

func TestWhatALabelCannotDo(t *testing.T) {
	for _, tc := range []struct{ name, processes, says string }{
		{
			name: "be used and never defined",
			processes: `/PRO 0
> COGER LLAVE   SKIP $nowhere
> _     _       SYSMESS 0
/END`,
			says: "defined nowhere",
		},
		{
			name: "be defined twice",
			processes: `/PRO 0
$twice
> COGER LLAVE   MESSAGE 0
$twice
> _     _       SYSMESS 0
/END`,
			says: "already defined",
		},
		{
			name: "be jumped to from another process",
			processes: `/PRO 0
$mine
> COGER LLAVE   MESSAGE 0
/PRO 1
> _     _       SKIP $mine
/END`,
			says: "cannot leave its process",
		},
		{
			name: "be jumped to from another process, before it exists",
			processes: `/PRO 0
> COGER LLAVE   SKIP $theirs
/PRO 1
$theirs
> _     _       SYSMESS 0
/END`,
			says: "cannot leave its process",
		},
		{
			name: "stand in for a number anywhere else",
			processes: `/PRO 0
> COGER LLAVE   MESSAGE $nowhere
> _     _       SYSMESS 0
/END`,
			says: "SKIP is the only condact that takes one",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := withProcesses(t, tc.processes)
			require.ErrorContains(t, err, tc.says)
		})
	}
}

// The distance travels as one signed byte, so it reaches 127 entries forward
// and 128 back. Past either end the wrap would jump somewhere the author never
// wrote, which is worse than a refusal.
func TestAJumpNoByteCouldHoldIsRefused(t *testing.T) {
	filler := strings.Repeat("> _     _       DONE\n", 129)

	t.Run("too far forward", func(t *testing.T) {
		// The SKIP is entry 0, the filler takes 1 to 129, and the label lands
		// on 130: a distance of 130, where 128 is the most that fits.
		_, err := withProcesses(t, "/PRO 0\n> COGER LLAVE   SKIP $far\n"+filler+
			"$far\n> _     LLAVE   DONE\n/END")
		require.ErrorContains(t, err, "entries ahead")
	})

	t.Run("too far back", func(t *testing.T) {
		_, err := withProcesses(t, "/PRO 0\n$top\n"+filler+
			"> COGER LLAVE   SKIP $top\n/END")
		require.ErrorContains(t, err, "entries back")
	})

	// And the far end that does fit, so that the limit is a limit and not an
	// off-by-one: 128 entries ahead is a distance of 127, the last a byte holds.
	t.Run("as far forward as a byte reaches", func(t *testing.T) {
		db, err := withProcesses(t, "/PRO 0\n> COGER LLAVE   SKIP $far\n"+
			strings.Repeat("> _     _       DONE\n", 127)+
			"$far\n> _     LLAVE   DONE\n/END")
		require.NoError(t, err)
		require.Equal(t, ddb.ID(127), skipIn(t, db, 0, 0))
	})
}

// A label with nothing after it but the end of the process is legal, and it
// names the entry that would have come next had there been one.
func TestALabelMayCloseAProcess(t *testing.T) {
	db, err := withProcesses(t, `/PRO 0
> COGER LLAVE   SKIP $after
> _     _       SYSMESS 0
$after
/END`)
	require.NoError(t, err)

	// Two entries, so the label is entry 2 and the jump from entry 0 is 1.
	require.Equal(t, ddb.ID(1), skipIn(t, db, 0, 0))
}
