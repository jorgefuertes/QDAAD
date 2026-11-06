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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/qtest"
	"github.com/jorgefuertes/QDAAD/internal/vm"
	"github.com/urfave/cli/v3"
)

func main() {
	if err := command().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// command is what the flags and the action are, built apart from main so that
// a test can run the whole of it the way a person does.
func command() *cli.Command {
	return &cli.Command{
		Name:        "qrun",
		Usage:       "play a DAAD database on a console",
		Description: testLanguage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "input",
				Aliases: []string{"i"},
				Usage: "Database to play. Required to play, and optional with --test " +
					"when the script names its own with GAME.",
			},
			&cli.StringSliceFlag{
				Name:    "test",
				Aliases: []string{"T"},
				Usage: "Play a .qtest script instead of a person: wait for text, type, " +
					"check flags. Give it more than once to run several. Nothing is " +
					"printed but the steps unless --show is given, and a step that " +
					"does not hold stops the run and exits non-zero.",
			},
			&cli.BoolFlag{
				Name: "show",
				Usage: "With --test, print what the adventure says as well as what the " +
					"steps did. It is off by default because a test is about the steps.",
			},
			&cli.StringFlag{
				Name: "transcript",
				Usage: "Write everything the adventure said to this file. It is the first " +
					"thing to read when a step fails for a reason the report cannot show.",
			},
			&cli.IntFlag{
				Name: "seed",
				Usage: "Fix the dice, so that an adventure with a CHANCE or a RANDOM in it " +
					"plays the same game twice. It wins over a script's own SEED.",
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
}

func run(_ context.Context, cmd *cli.Command) error {
	watch, closeWatch, err := tracer(cmd.String("trace"), cmd.String("trace-dir"),
		int64(cmd.Int("trace-limit")))
	if err != nil {
		return err
	}

	defer closeWatch()

	if scripts := cmd.StringSlice("test"); len(scripts) > 0 {
		return test(cmd, scripts, watch)
	}

	if cmd.String("input") == "" {
		return errors.New("nothing to play: give a database with --input")
	}

	db, err := load(cmd.String("input"))
	if err != nil {
		return err
	}

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

// test plays scripts instead of a person.
//
// A script that cannot be read comes back as an error, because that is a
// mistake in the test rather than a failure of the adventure, and the
// difference is worth keeping: one is fixed in the .qtest and the other in the
// game.
func test(cmd *cli.Command, scripts []string, watch vm.Watcher) error {
	// A trace is only handed over if one was asked for. Given a watcher that
	// writes nowhere, a script's TRACE ON would look like it did something.
	if cmd.String("trace") == "" {
		watch = nil
	}

	transcript, closeTranscript, err := transcriptTo(cmd.String("transcript"))
	if err != nil {
		return err
	}

	defer closeTranscript()

	failed := 0

	for _, path := range scripts {
		script, err := qtest.Parse(path)
		if err != nil {
			return err
		}

		where := cmd.String("input")
		if where == "" {
			where = script.Game
		}

		if where == "" {
			return fmt.Errorf("%s: %w", path, qtest.ErrNoDatabase)
		}

		db, err := load(where)
		if err != nil {
			return err
		}

		options := qtest.Options{
			Report:     qtest.NewReport(writer(cmd)),
			Show:       cmd.Bool("show"),
			Transcript: transcript,
			Trace:      watch,

			// What a BREAKPOINT hands the adventure to is the console a person
			// plays on, not an imitation of one: the clock and the pauses then
			// behave for them exactly as they do with no script in the way.
			Breakpoint: &console{
				out:   os.Stdout,
				in:    bufio.NewScanner(os.Stdin),
				waits: isTerminal(os.Stdin),
			},
		}

		if cmd.IsSet("seed") {
			options.Seed, options.HasSeed = int64(cmd.Int("seed")), true
		}

		if err := qtest.Play(db, script, options); err != nil {
			failed++
		}
	}

	if failed == 0 {
		return nil
	}

	if len(scripts) == 1 {
		return fmt.Errorf("%s did not pass", scripts[0])
	}

	return fmt.Errorf("%d of %d scripts did not pass", failed, len(scripts))
}

// writer is where a report goes. It is the command's own, so that a test can
// read what a person would have seen.
func writer(cmd *cli.Command) io.Writer {
	if out := cmd.Root().Writer; out != nil {
		return out
	}

	return os.Stdout
}

// load reads a database and says which file it was when it cannot.
func load(path string) (*ddbfile.Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	db, err := open(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return db, nil
}

// transcriptTo opens where everything the adventure said is to be kept.
func transcriptTo(path string) (io.Writer, func(), error) {
	if path == "" {
		return nil, func() {}, nil
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}

	return f, func() { _ = f.Close() }, nil
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

	// lines is where the reading goroutine puts them, and the only place the
	// scanner is read from. See reading.
	lines chan string
}

func (c *console) Print(text string) { _, _ = fmt.Fprint(c.out, text) }

// reading is the one goroutine that touches the scanner, started the first
// time somebody wants a line.
//
// A read of a terminal cannot be called off once it has begun, so a timeout
// cannot be a deadline on the read: it has to be a race between the read and a
// clock, and that needs the read to be somewhere else. What the goroutine
// hands over is lines, and a closed channel is the end of the input.
//
// A line typed after the clock ran out is not thrown away — it is still in the
// channel and the next turn takes it. That is the kinder reading and it is the
// one DAAD has a bit for, "recuperar automáticamente el buffer tras el
// timeout".
func (c *console) reading() <-chan string {
	if c.lines == nil {
		c.lines = make(chan string)

		go func() {
			defer close(c.lines)

			for c.in.Scan() {
				c.lines <- c.in.Text()
			}
		}()
	}

	return c.lines
}

func (c *console) Input() (string, bool) {
	line, more := <-c.reading()

	return line, more
}

// InputWithin is Input with the adventure's TIME on it: whoever gets there
// first, the player or the clock.
func (c *console) InputWithin(limit time.Duration) (line string, more, expired bool) {
	clock := time.NewTimer(limit)
	defer clock.Stop()

	select {
	case line, more := <-c.reading():
		return line, more, false
	case <-clock.C:
		return "", true, true
	}
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
		<-c.reading()
	}
}
