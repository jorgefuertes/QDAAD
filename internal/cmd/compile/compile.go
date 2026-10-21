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
	"github.com/jorgefuertes/QDAAD/internal/pack"
	"github.com/urfave/cli/v3"
)

// What the compiler can be asked to write. The database is the compiler's own
// output and the one every target has; a tape is that database with an
// interpreter and a loader around it, which only some machines know how to
// boot from.
const (
	formatDDB = "ddb"
	formatTAP = "tap"
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
			&cli.StringFlag{
				Name:     "format",
				Category: "files",
				Aliases:  []string{"f"},
				Value:    formatDDB,
				Usage: "What to write: " + formatDDB + " for the database alone, or " +
					formatTAP + " for a tape that boots it.",
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

	format := cmd.String("format")

	out := cmd.String("output")
	if out == "" {
		out = strings.TrimSuffix(source, path.Ext(source))
	}

	if path.Ext(out) == "" {
		out += "." + format
	}

	// The name the ROM shows while loading is the one the file was given,
	// which is what the reference does too and the only name there is to use.
	body, err := wrap(format, image, opts, name(out))
	if err != nil {
		return err
	}

	if err := os.WriteFile(out, body, 0o644); err != nil {
		return err
	}

	if format == formatTAP {
		fmt.Printf("%s: %d bytes of tape, %d of database laid at 0x%04X\n",
			out, len(body), len(image), opts.Machine.BaseAddress(opts.Subtarget))

		return nil
	}

	fmt.Printf("%s: %d bytes, laid at 0x%04X\n", out, len(image),
		opts.Machine.BaseAddress(opts.Subtarget))

	return nil
}

// wrap puts the database into whatever container was asked for.
//
// Only the 48K Spectrum is packed so far. The other machines refuse rather than
// quietly writing the database under a .tap extension, which would load on
// nothing and say so nowhere.
func wrap(format string, image []byte, opts legacy.Options, name string) ([]byte, error) {
	switch format {
	case formatDDB:
		return image, nil
	case formatTAP:
		if opts.Machine != arch.ZX {
			return nil, fmt.Errorf("a tape is a Spectrum thing and this is a %s build", opts.Machine)
		}

		if opts.Subtarget != "" && !strings.EqualFold(opts.Subtarget, "48k") {
			return nil, fmt.Errorf("only the 48K Spectrum is packed into a tape so far, not the %s",
				opts.Subtarget)
		}

		return pack.ZX48(image, pack.Options{Language: opts.Language, Name: name})
	default:
		return nil, fmt.Errorf("%q is not a format: it is %s or %s", format, formatDDB, formatTAP)
	}
}

// name is the output file's own name, without its directory or extension.
func name(out string) string {
	return strings.TrimSuffix(path.Base(out), path.Ext(out))
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
