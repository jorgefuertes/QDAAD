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
	// formatQRun is the database in a container that states what the readers
	// of the old formats have to guess: where it was laid and which way round
	// its words go. See internal/pack/qrun.go.
	formatQRun = "qrun"
	// formatDSK is a disk image the machine boots from, which on a Spectrum
	// means a 180K +3 disk. See internal/media/plus3disk.go.
	formatDSK = "dsk"
	// formatDir is everything the machine needs, as files in a directory.
	//
	// It is the one format every machine can have, because it is the recipe
	// rather than a container for it — and a machine whose release really is
	// one file gets a directory with that file in it. See internal/pack/release.go.
	formatDir = "dir"
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
				Value:    formatDDB,
				Usage: "What to write: " + formatDDB + " for the database alone, " +
					formatTAP + " for a tape that boots it, " + formatDSK +
					" for a disk that does, " + formatDir +
					" for a directory holding everything the machine needs, or " +
					formatQRun + " for one our own interpreter reads without guessing " +
					"at anything.",
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

	// A directory is named, not suffixed. Everything else gets the format for
	// an extension when none was given.
	if path.Ext(out) == "" && format != formatDir {
		out += "." + format
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

	// The name the ROM shows while loading is the one the file was given,
	// which is what the reference does too and the only name there is to use.
	release, err := assemble(format, image, opts, name(out), charset, layout)
	if err != nil {
		return err
	}

	if format == formatDir {
		if err := release.WriteDir(out); err != nil {
			return err
		}

		files := "files"
		if len(release.Files()) == 1 {
			files = "file"
		}

		fmt.Printf("%s: %d %s, %d bytes in all\n%s", out,
			len(release.Files()), files, release.Total(), release.Listing())

		return nil
	}

	file, err := release.Single()
	if err != nil {
		return fmt.Errorf("%s: %w\n\tbuild it as --format %s instead", format, err, formatDir)
	}

	if err := os.WriteFile(out, file.Body, 0o644); err != nil {
		return err
	}

	// A container says its own size as well as the database's, because the two
	// are different numbers and the one that matters depends on what you are
	// about to do with it: a tape has to fit in a machine, a disk on a disk.
	if format == formatTAP || format == formatDSK {
		fmt.Printf("%s: %d bytes of %s, %d of database laid at 0x%04X\n",
			out, len(file.Body), format, len(image),
			opts.Machine.BaseAddress(opts.Subtarget))

		return nil
	}

	fmt.Printf("%s: %d bytes, laid at 0x%04X\n", out, len(image),
		opts.Machine.BaseAddress(opts.Subtarget))

	return nil
}

// assemble gathers what the asked-for format needs, as a release.
//
// Everything comes back as a release even when it is one file, so that the
// caller has one thing to write and --format dir works for every format
// without each one knowing about it. A database on its own is a release of one;
// so is a tape, which only looks like a container because the gluing happened
// inside it.
//
// Only the Spectrum is packaged so far. The other machines refuse rather than
// quietly writing the database under a .tap extension, which would load on
// nothing and say so nowhere.
func assemble(format string, image []byte, opts legacy.Options, name string, charset []byte,
	layout back.Layout,
) (*pack.Release, error) {
	switch format {
	case formatDDB:
		return pack.NewRelease(pack.File{
			Name: name + ".ddb", Body: image,
			Note: fmt.Sprintf("database, laid at 0x%04X",
				opts.Machine.BaseAddress(opts.Subtarget)),
		}), nil

	case formatQRun:
		body, err := pack.QRun(image, int(opts.Machine.BaseAddress(opts.Subtarget)),
			opts.Machine.IsBigEndian(), layout)
		if err != nil {
			return nil, err
		}

		return pack.NewRelease(pack.File{
			Name: name + ".qrun", Body: body,
			Note: "database in our own container, for qrun",
		}), nil

	case formatDir, formatTAP:
		packOpts := pack.Options{Language: opts.Language, Name: name, Charset: charset}

		switch opts.Machine {
		case arch.ZX:
			return pack.ZXRelease(image, opts.Subtarget, packOpts)
		case arch.CPC:
			return pack.CPCRelease(image, packOpts)
		default:
			return nil, fmt.Errorf("only the Spectrum and the CPC are packaged so far, not the %s",
				opts.Machine)
		}

	case formatDSK:
		packOpts := pack.Options{Language: opts.Language, Name: name, Charset: charset}

		var (
			body []byte
			note string
			err  error
		)

		switch opts.Machine {
		case arch.ZX:
			body, err = pack.ZXDisk(image, opts.Subtarget, packOpts)
			note = "180K +3 disk, bootable from the +3's own menu"
		case arch.CPC:
			body, err = pack.CPCDisk(image, packOpts)
			note = "180K CPC disk in DATA format: RUN\"" + name + "\""
		default:
			return nil, fmt.Errorf("only the Spectrum and the CPC are packaged so far, not the %s",
				opts.Machine)
		}

		if err != nil {
			return nil, err
		}

		return pack.NewRelease(pack.File{Name: name + ".dsk", Body: body, Note: note}), nil

	default:
		return nil, fmt.Errorf("%q is not a format: it is %s, %s, %s, %s or %s",
			format, formatDDB, formatTAP, formatDSK, formatDir, formatQRun)
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
