package vm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/vm"
	"github.com/stretchr/testify/require"
)

// A trace has to stay small, and this is the only thing worth asserting about
// it: that a run which would write gigabytes writes a file somebody can open.
//
// The number is not academic. One stuck SKIP wrote four and a half million
// lines in eight seconds and filled the disk of the machine this was written
// on, which is how the limit came to exist.
func TestATraceStaysSmall(t *testing.T) {
	const limit = 4096

	path := filepath.Join(t.TempDir(), "trace.log")

	f, err := os.Create(path)
	require.NoError(t, err)

	trace := vm.NewTrace(f, nil, limit)

	// Far more than the limit holds, so it has to go round several times.
	for range 20000 {
		trace.Process(7, true)
	}

	require.NoError(t, f.Close())

	written, err := os.ReadFile(path)
	require.NoError(t, err)

	require.LessOrEqual(t, len(written), limit*2,
		"a trace that cannot be opened is no use to anyone")

	// And it says what it threw away, so that nobody reads the first line and
	// believes it is the beginning of the run.
	require.Contains(t, string(written), "restarted",
		"a wrapped trace has to admit it wrapped")

	// What is left has to be whole lines. Rewinding without truncating would
	// leave the tail of the previous pass behind the new one, and the seam in
	// the middle of a line.
	for _, line := range strings.Split(strings.TrimRight(string(written), "\n"), "\n") {
		require.True(t,
			strings.HasPrefix(line, "enter process") || strings.HasPrefix(line, "---"),
			"a line left half-written: %q", line)
	}
}

// Nothing to rewind is not an error. Tracing to a pipe or a terminal writes
// everything, because what is on a terminal has been read by the time it
// scrolls away.
func TestATraceToSomethingUnrewindableJustKeepsGoing(t *testing.T) {
	var sink strings.Builder

	trace := vm.NewTrace(&sink, nil, 64)

	for range 100 {
		trace.Process(1, true)
	}

	require.Greater(t, sink.Len(), 64*2, "nothing was dropped and nothing broke")
}

// And the names, because a trace of numbers is a worse trace.
func TestATraceNamesTheCondactsWhenItCan(t *testing.T) {
	var sink strings.Builder

	trace := vm.NewTrace(&sink, func(op byte) string {
		require.Equal(t, byte(19), op)

		return "DESC"
	}, 0)

	trace.Condact(0, 1, ddbfile.Condact{Opcode: 19, Indirect: true, Params: []byte{38}})

	require.Contains(t, sink.String(), "DESC @[38]")
}
