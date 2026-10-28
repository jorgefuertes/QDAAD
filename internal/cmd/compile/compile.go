package compile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/build"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/urfave/cli/v3"
)

// What the compiler can be asked to write. The database is the compiler's own
// output and the one every target has; a tape is that database with an
// interpreter and a loader around it, which only some machines know how to
// boot from.
const (
	formatDDB = "ddb"
	// formatBuild is the directory qpack reads: the database, what it was
	// built for, and the font if the adventure has one of its own. It is what
	// the compiler writes now that packaging is a tool of its own.
	formatBuild = "build"
	// formatQRun is the database in a container that states what the readers
	// of the old formats have to guess: where it was laid and which way round
	// its words go. See internal/pack/qrun.go.
	// formatDSK is a disk image the machine boots from, which on a Spectrum
	// means a 180K +3 disk. See internal/media/plus3disk.go.
	// formatDir is everything the machine needs, as files in a directory.
	//
	// It is the one format every machine can have, because it is the recipe
	// rather than a container for it — and a machine whose release really is
	// one file gets a directory with that file in it. See internal/pack/release.go.
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
			&cli.UintFlag{
				Name:     "part",
				Category: "compiler flags",
				Value:    1,
				Usage: "Which part of the adventure this is. A two-part release names " +
					"its files by part, and nothing in a database says which one it is.",
			},
			&cli.StringFlag{
				Name:     "charset",
				Category: "files",
				Usage: "Character set to load with the game, 2048 bytes or that plus a " +
					"128-byte header. Without it the one built in is used, which is the " +
					"one every DAAD READY build loads.",
			},
			&cli.StringFlag{
				Name:     "format",
				Category: "files",
				Aliases:  []string{"f"},
				Value:    formatBuild,
				Usage: "What to write: " + formatBuild + " for a directory qpack can " +
					"package, or " + formatDDB + " for the database on its own. " +
					"Tapes, disks and the rest are qpack's job now.",
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

	db, read, err := legacy.Compile(source, opts)
	if err != nil {
		return err
	}

	image, layout, err := back.LinkWithLayout(db, back.Options{
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

	// The flag wins over what the source says about itself. The source's own
	// answer is the right one nearly always — the font is part of the
	// adventure, not an opinion about the build — but "nearly" is why there is
	// a flag, and an override that can be overridden is no override.
	charsetPath := cmd.String("charset")
	if charsetPath == "" {
		charsetPath = read.Settings.Charset
	}

	var charset []byte

	if charsetPath != "" {
		if charset, err = os.ReadFile(charsetPath); err != nil {
			return err
		}
	}

	switch format {
	case formatDDB:
		if path.Ext(out) == "" {
			out += "." + formatDDB
		}

		if err := os.WriteFile(out, image, 0o644); err != nil {
			return err
		}

		fmt.Printf("%s: %d bytes, laid at 0x%04X\n", out, len(image),
			opts.Machine.BaseAddress(opts.Subtarget))

		return nil

	case formatBuild:
		b := build.Build{
			Machine:   opts.Machine,
			Subtarget: opts.Subtarget,
			Language:  opts.Language,
			Version:   opts.Version,
			Part:      int(cmd.Uint("part")),
			Tokens:    layout.Tokens,
			Database:  image,
			Charset:   charset,
		}

		if err := b.Write(out); err != nil {
			return err
		}

		fmt.Printf("%s: part %d for a %s, %d bytes of database laid at 0x%04X\n",
			out, b.Part, opts.Machine, len(image),
			opts.Machine.BaseAddress(opts.Subtarget))

		return nil

	default:
		return fmt.Errorf("%q is not a format: it is %s or %s, and the rest is qpack's",
			format, formatBuild, formatDDB)
	}
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
