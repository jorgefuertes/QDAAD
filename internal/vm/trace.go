package vm

import (
	"fmt"
	"io"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
)

// Trace writes down everything the machine does, which is the plainest thing a
// Watcher can be and the one a person reads when an adventure misbehaves.
//
// It is deliberately a stream of lines rather than a session: a debugger that
// has to be driven interactively is no use to a test, and one that writes a
// file is useful to a person, to a test and to a program alike.
//
// And it is bounded, which is not a nicety. A trace is one line per condact,
// and what a person usually reaches for a trace to find is a loop that will not
// end — so the cost of tracing is highest exactly when it is most needed. Left
// alone, one stuck SKIP wrote four and a half million lines in eight seconds
// and filled a disk.
type Trace struct {
	to      io.Writer
	condact func(byte) string

	// limit is how many bytes the file may hold before it starts again, and
	// zero means it may hold anything.
	limit   int64
	written int64
	wraps   int
}

// DefaultTraceLimit is a few megabytes: enough to hold a long game, far too
// little to hurt anyone, and more than a person is going to read anyway.
const DefaultTraceLimit = 8 << 20

// rewindable is a writer that can start over, which a file can and a pipe
// cannot. Tracing to something that cannot rewind simply does not wrap.
type rewindable interface {
	io.Writer
	Seek(offset int64, whence int) (int64, error)
	Truncate(size int64) error
}

// NewTrace writes to w, starting the file again whenever it has written limit
// bytes. A limit of zero means no limit.
//
// names turns an opcode into its name and may be nil, in which case the number
// is written instead.
//
// What wrapping loses is the beginning, and that is a real cost rather than a
// detail: it keeps what happened most recently instead of what happened first,
// and for the usual reason to be tracing — something is looping and the
// interesting part is how it got in — the first few thousand lines are the ones
// worth having. So the file says how many times it has gone round, and anyone
// who needs the start raises the limit and runs it again.
func NewTrace(w io.Writer, names func(byte) string, limit int64) *Trace {
	return &Trace{to: w, condact: names, limit: limit}
}

func (t *Trace) Condact(process, entry int, c ddbfile.Condact) {
	name := fmt.Sprintf("op%d", c.Opcode)
	if t.condact != nil {
		name = t.condact(c.Opcode)
	}

	at := ""
	if c.Indirect {
		at = "@"
	}

	t.line("%d/%d  %s %s%v\n", process, entry, name, at, c.Params)
}

func (t *Trace) Flag(number int, was, now byte) {
	t.line("        flag %d: %d -> %d\n", number, was, now)
}

func (t *Trace) Process(number int, running bool) {
	what := "enter"
	if !running {
		what = "leave"
	}

	t.line("%s process %d\n", what, number)
}

func (t *Trace) Moved(object int, from, to byte) {
	t.line("        object %d: %d -> %d\n", object, from, to)
}

// line writes one, and starts the file again when it has grown enough.
func (t *Trace) line(format string, args ...any) {
	if t.limit > 0 && t.written >= t.limit {
		t.restart()
	}

	n, _ := fmt.Fprintf(t.to, format, args...)
	t.written += int64(n)
}

// restart takes the file back to the beginning.
//
// Truncating as well as rewinding is what keeps it readable. Writing over the
// old bytes without truncating would leave the tail of the previous pass behind
// the new one, with the seam in the middle of a line, and a file meant to
// explain something would need explaining itself.
func (t *Trace) restart() {
	// A writer that cannot rewind — a pipe, or the terminal — keeps growing,
	// and there is nothing to be done about that from here. The count still
	// goes back to zero so that it is not asked again on every line.
	file, ok := t.to.(rewindable)
	if !ok {
		t.written = 0

		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.written = 0

		return
	}

	if err := file.Truncate(0); err != nil {
		t.written = 0

		return
	}

	t.wraps++

	n, _ := fmt.Fprintf(t.to,
		"--- restarted %d time(s): everything before this was thrown away to keep "+
			"the file under %d bytes ---\n", t.wraps, t.limit)
	t.written = int64(n)
}
