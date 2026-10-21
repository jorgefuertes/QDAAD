package compile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:    "compile",
		Usage:   "compile a .DDB database",
		Aliases: []string{"c"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "machine",
				Category: "compiler flags",
				Aliases:  []string{"m"},
				Usage: "Target machine for the compiled database. " +
					"Valid values are: " + arch.MachineHumanList(),
			},
			&cli.StringFlag{
				Name:     "input",
				Category: "files",
				Aliases:  []string{"i"},
				Usage:    "Main source file to compile.",
			},
			&cli.StringFlag{
				Name:     "subtarget",
				Category: "compiler flags",
				Aliases:  []string{"s"},
				Usage: "Variant of the target machine, where it has one: " +
					"48k, 128k, plus3, next, uno or esxdos on a Spectrum, " +
					"text, cga, ega, vga or vga256 on a PC.",
			},
			&cli.StringFlag{
				Name:     "language",
				Category: "compiler flags",
				Aliases:  []string{"l"},
				Value:    "spanish",
				Usage:    "Language the interpreter speaks: english or spanish.",
			},
			&cli.StringSliceFlag{
				Name:     "define",
				Category: "compiler flags",
				Aliases:  []string{"D"},
				Usage:    "Extra symbol to define before reading the source. Repeatable.",
			},
			&cli.UintFlag{
				Name:     "version",
				Category: "compiler flags",
				Aliases:  []string{"v"},
				Value:    3,
				Usage:    "Database format to emit, 2 or 3.",
			},
			&cli.StringFlag{
				Name:     "output",
				Category: "files",
				Aliases:  []string{"o"},
				Usage:    "Output file for the compiled database, without extension.",
			},
		},
		Action: compileAction,
	}
}

func compileAction(_ context.Context, cmd *cli.Command) error {
	source := cmd.String("input")
	if source == "" {
		return errors.New("no source to compile: name one with -i")
	}

	if ext := path.Ext(source); ext != ".sce" && ext != ".dsf" && ext != ".DSF" {
		return fmt.Errorf("%q is not a source this reads", ext)
	}

	opts, err := options(cmd)
	if err != nil {
		return err
	}

	db, err := legacy.Analyze(source, opts)
	if err != nil {
		return err
	}

	image, err := back.Link(db, back.Options{
		Machine:   opts.Machine,
		Subtarget: opts.Subtarget,
		Language:  opts.Language,
		Version:   opts.Version,
	})
	if err != nil {
		return err
	}

	out := cmd.String("output")
	if out == "" {
		out = strings.TrimSuffix(source, path.Ext(source))
	}

	if path.Ext(out) == "" {
		out += ".ddb"
	}

	if err := os.WriteFile(out, image, 0o644); err != nil {
		return err
	}

	fmt.Printf("%s: %d bytes, laid at 0x%04X\n", out, len(image),
		opts.Machine.BaseAddress(opts.Subtarget))

	return nil
}

// options turns the flags into what the front end needs to know.
//
// The machine is not optional and has no default: a source branches on what it
// is being built for, so guessing one would quietly compile the wrong thing.
func options(cmd *cli.Command) (legacy.Options, error) {
	machine, err := arch.Parse(cmd.String("machine"))
	if err != nil {
		return legacy.Options{}, err
	}

	lang := language.LanguageUndefined.Parse(cmd.String("language"))
	if !lang.IsValid() {
		return legacy.Options{}, fmt.Errorf("%q is not a language, it is english or spanish",
			cmd.String("language"))
	}

	opts := legacy.Options{
		Machine:   machine,
		Subtarget: cmd.String("subtarget"),
		Language:  lang,
		Version:   uint8(cmd.Uint("version")),
		Extra:     cmd.StringSlice("define"),
	}

	return opts, opts.Validate()
}
