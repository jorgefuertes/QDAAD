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
type Trace struct {
	to      io.Writer
	condact func(byte) string
}

// NewTrace writes to w. names turns an opcode into its name, and may be nil,
// in which case the number is written instead.
func NewTrace(w io.Writer, names func(byte) string) *Trace {
	return &Trace{to: w, condact: names}
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

	_, _ = fmt.Fprintf(t.to, "%d/%d  %s %s%v\n", process, entry, name, at, c.Params)
}

func (t *Trace) Flag(number int, was, now byte) {
	_, _ = fmt.Fprintf(t.to, "        flag %d: %d -> %d\n", number, was, now)
}

func (t *Trace) Process(number int, running bool) {
	what := "enter"
	if !running {
		what = "leave"
	}

	_, _ = fmt.Fprintf(t.to, "%s process %d\n", what, number)
}

func (t *Trace) Moved(object int, from, to byte) {
	_, _ = fmt.Fprintf(t.to, "        object %d: %d -> %d\n", object, from, to)
}
