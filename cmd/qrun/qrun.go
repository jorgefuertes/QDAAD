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
			&cli.StringFlag{
				Name:    "trace",
				Aliases: []string{"t"},
				Usage: "Write a trace of every condact, flag change and process call " +
					"to this file, or to - for standard error.",
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

	db, ok := ddbfile.At(data, 0)
	if !ok {
		return fmt.Errorf("%s does not read as a database", cmd.String("input"))
	}

	watch, closeWatch, err := tracer(cmd.String("trace"))
	if err != nil {
		return err
	}

	defer closeWatch()

	machine, err := vm.New(db, &console{
		out: os.Stdout,
		in:  bufio.NewScanner(os.Stdin),
	}, watch)
	if err != nil {
		return err
	}

	return machine.Run()
}

// tracer builds the watcher the --trace flag asks for.
func tracer(to string) (vm.Watcher, func(), error) {
	switch to {
	case "":
		return vm.NoWatcher{}, func() {}, nil
	case "-":
		return vm.NewTrace(os.Stderr, nil), func() {}, nil
	}

	f, err := os.Create(to)
	if err != nil {
		return nil, nil, err
	}

	return vm.NewTrace(f, nil), func() { _ = f.Close() }, nil
}

// console is a person at a terminal: text out, a line in.
type console struct {
	out io.Writer
	in  *bufio.Scanner
}

func (c *console) Print(text string) { _, _ = fmt.Fprint(c.out, text) }

func (c *console) Input() (string, bool) {
	if !c.in.Scan() {
		return "", false
	}

	return c.in.Text(), true
}

// AnyKey waits for a line rather than a key. A console has no way to read one
// key, and a driver feeding it from a file has no keyboard at all; a line is
// what both can give.
func (c *console) AnyKey() { c.in.Scan() }
