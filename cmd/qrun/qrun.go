// Command qrun plays a DAAD database on a console.
//
// It is an interpreter rather than an emulator: there is no Spectrum behind
// it, only the adventure. What it runs is a database any of the compilers
// writes, and what it needs of the world is a line in and a line out — so the
// same program serves a person at a terminal, a test asserting on what the
// adventure said, and another program driving it.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/vm"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "qrun",
		Usage: "play a DAAD database on a console",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "input",
				Aliases:  []string{"i"},
				Usage:    "Database to play.",
				Required: true,
			},
			&cli.IntFlag{
				Name:  "trace-limit",
				Value: vm.DefaultTraceLimit,
				Usage: "How many bytes a trace file may hold before it starts again from " +
					"the top. A trace is one line per condact and the usual reason to want " +
					"one is a loop that will not end, so left alone it fills a disk. Zero " +
					"removes the limit.",
			},
			&cli.StringFlag{
				Name:    "trace",
				Aliases: []string{"t"},
				Usage: "Write a trace of every condact, flag change and process call " +
					"to this file, or to - for standard error. A bare name with no " +
					"directory in it is written inside --trace-dir.",
			},
			&cli.StringFlag{
				Name:  "trace-dir",
				Value: defaultTraceDir,
				Usage: "Where a --trace given as a bare name goes. It defaults to somewhere " +
					"with room, because a trace is the one thing this writes that grows and " +
					"a full disk is a worse failure than a missing trace.",
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(_ context.Context, cmd *cli.Command) error {
	data, err := os.ReadFile(cmd.String("input"))
	if err != nil {
		return err
	}

	db, err := open(data)
	if err != nil {
		return fmt.Errorf("%s: %w", cmd.String("input"), err)
	}

	watch, closeWatch, err := tracer(cmd.String("trace"), cmd.String("trace-dir"),
		int64(cmd.Int("trace-limit")))
	if err != nil {
		return err
	}

	defer closeWatch()

	machine, err := vm.New(db, &console{
		out:   os.Stdout,
		in:    bufio.NewScanner(os.Stdin),
		waits: isTerminal(os.Stdin),
	}, watch)
	if err != nil {
		return err
	}

	return machine.Run()
}

// open reads the database.
//
// Our own container first, because it is the one that states where everything
// is instead of leaving it to be worked out. Anything else is tried the old
// way, which is what makes an original tape from 1989 playable too — it just
// cannot promise as much.
func open(data []byte) (*ddbfile.Reader, error) {
	db, err := ddbfile.OpenQRun(data)
	if err == nil {
		return db, nil
	}

	db, ok := ddbfile.At(data, 0)
	if !ok {
		return nil, fmt.Errorf("neither ours (%w) nor one we can work out", err)
	}

	return db, nil
}

// defaultTraceDir is where a trace goes when only a name is given.
//
// It is a second disk rather than a directory under the project on purpose.
// A trace is bounded now, but it is still the largest thing this writes, and
// the boot volume is the one whose filling up stops everything — the editor,
// the compiler, and the very commands you would reach for to clear it.
const defaultTraceDir = "/Volumes/QueruData/qdaad-trace"

// tracer builds the watcher the --trace flag asks for.
func tracer(to, dir string, limit int64) (vm.Watcher, func(), error) {
	switch to {
	case "":
		return vm.NoWatcher{}, func() {}, nil
	case "-":
		// Standard error cannot be rewound, so the limit does not apply: what
		// is on a terminal has already been read by the time it scrolls away.
		return vm.NewTrace(os.Stderr, nil, 0), func() {}, nil
	}

	path, err := tracePath(to, dir)
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}

	return vm.NewTrace(f, nil, limit), func() { _ = f.Close() }, nil
}

// tracePath works out where the trace is written, and refuses rather than
// falling back.
//
// A name with a directory in it is taken as given: whoever typed a path meant
// it. A bare name goes into the trace directory, and if that directory cannot
// be had — the usual reason being that the disk is not plugged in — this says
// so. Quietly writing to the working directory instead would put the trace on
// exactly the volume it was moved off, and on the day that matters that volume
// is already full.
func tracePath(name, dir string) (string, error) {
	if filepath.Base(name) != name {
		return name, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("no trace directory at %s, so %q has nowhere to go: %w\n"+
			"\tgive --trace a path with a directory in it to write somewhere else",
			dir, name, err)
	}

	return filepath.Join(dir, name), nil
}

// isTerminal says whether there is a person on the other end of a file, which
// is what decides whether a pause is worth anything. A character device is a
// terminal; a redirected file or a pipe is not.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// console is a person at a terminal: text out, a line in.
type console struct {
	out io.Writer
	in  *bufio.Scanner

	// waits says whether a pause should stop for anything. A person wants it
	// to; a file being fed in has nothing to press.
	waits bool
}

func (c *console) Print(text string) { _, _ = fmt.Fprint(c.out, text) }

func (c *console) Input() (string, bool) {
	if !c.in.Scan() {
		return "", false
	}

	return c.in.Text(), true
}

// AnyKey waits for a line rather than a key, because a console has no way to
// read one key.
//
// It only waits for a person. Reading a line from a file would spend one of
// the commands in it on a pause, and the cost of that is out of all proportion
// to the convenience: an adventure pauses wherever its author felt like it, so
// a walkthrough fed in from a file is silently shifted by one from the first
// pause onwards. Every command after it is answered in the wrong room, and
// nothing says so — a whole afternoon went on a tortilla that was never taken
// because the move to the cabin had been eaten by the title screen.
func (c *console) AnyKey() {
	if c.waits {
		c.in.Scan()
	}
}
