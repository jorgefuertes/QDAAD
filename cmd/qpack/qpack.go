// Command qpack turns compiled parts into something a machine will load.
//
// It is the second half of what used to be one command. qdaad compiles a
// source into a build directory; this reads those directories and writes the
// tape, the disk, or the loose files. The two were split because they are two
// jobs: compiling is about one source, and packaging is about a release, which
// may be of two parts and is always about a machine rather than a language.
//
// The name is packing and not linking on purpose. Linking already means
// something here — it is what the back end does to produce a database, and
// what script/compare-drb.sh checks — so a tool called qlinker would make the
// word mean two things in one project.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/build"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/pack"
	"github.com/urfave/cli/v3"
)

// What can be written. A directory is the one every machine can have, because
// it is the recipe rather than a container for it; the rest are containers a
// particular machine boots from.
const (
	formatDir  = "dir"
	formatTAP  = "tap"
	formatDSK  = "dsk"
	formatQRun = "qrun"
)

func main() {
	cmd := &cli.Command{
		Name:      "qpack",
		Usage:     "package compiled parts into a tape, a disk or a directory",
		ArgsUsage: "<build directory> [second build directory]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   formatDir,
				Usage: "What to write: " + formatDir + " for everything the machine " +
					"needs as files, " + formatTAP + " for a tape, " + formatDSK +
					" for a disk, or " + formatQRun + " for a container our own " +
					"interpreter reads without guessing at anything.",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Where to write it. Without this the build directory's own name is used.",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "The name the machine shows while loading, and the name of the files on a disk.",
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
	dirs := cmd.Args().Slice()
	if len(dirs) == 0 {
		return errors.New("nothing to package: name a build directory that qdaad wrote")
	}

	parts := make([]build.Build, 0, len(dirs))

	for _, dir := range dirs {
		b, err := build.Read(dir)
		if err != nil {
			return err
		}

		parts = append(parts, b)
	}

	// Two parts that disagree about the machine would make a disk rather than
	// an error: one that boots, loads the first part, and does something
	// unaccountable when the player asks for the second.
	if err := build.SameAdventure(parts); err != nil {
		return fmt.Errorf("these are not two parts of one adventure: %w", err)
	}

	if len(parts) > 1 {
		return fmt.Errorf(
			"%d parts given, and only one can be packaged so far: a two-part release "+
				"needs a loader that asks which part, which is not written yet", len(parts))
	}

	name := cmd.String("name")
	if name == "" {
		name = strings.ToUpper(filepath.Base(strings.TrimSuffix(dirs[0], string(filepath.Separator))))
	}

	release, err := assemble(cmd.String("format"), parts[0], name)
	if err != nil {
		return err
	}

	return write(cmd.String("format"), cmd.String("output"), dirs[0], release)
}

// assemble gathers what the asked-for format needs, as a release.
//
// Everything comes back as a release even when it is one file, so that there
// is one thing to write and --format dir works for every format without each
// one knowing about it.
func assemble(format string, b build.Build, name string) (*pack.Release, error) {
	opts := pack.Options{Language: b.Language, Name: name, Charset: b.Charset}

	switch format {
	case formatQRun:
		body, err := pack.QRun(b.Database, int(b.Machine.BaseAddress(b.Subtarget)),
			b.Machine.IsBigEndian(), back.Layout{Tokens: b.Tokens})
		if err != nil {
			return nil, err
		}

		return pack.NewRelease(pack.File{
			Name: name + ".qrun", Body: body,
			Note: "database in our own container, for qrun",
		}), nil

	case formatDir, formatTAP:
		switch b.Machine {
		case arch.ZX:
			return pack.ZXRelease(b.Database, b.Subtarget, opts)
		case arch.CPC:
			return pack.CPCRelease(b.Database, opts)
		default:
			return nil, notPackaged(b.Machine)
		}

	case formatDSK:
		var (
			body []byte
			note string
			err  error
		)

		switch b.Machine {
		case arch.ZX:
			body, err = pack.ZXDisk(b.Database, b.Subtarget, opts)
			note = "180K +3 disk, bootable from the +3's own menu"
		case arch.CPC:
			body, err = pack.CPCDisk(b.Database, opts)
			note = `180K CPC disk in DATA format: RUN"` + name + `"`
		default:
			return nil, notPackaged(b.Machine)
		}

		if err != nil {
			return nil, err
		}

		return pack.NewRelease(pack.File{Name: name + ".dsk", Body: body, Note: note}), nil

	default:
		return nil, fmt.Errorf("%q is not a format: it is %s, %s, %s or %s",
			format, formatDir, formatTAP, formatDSK, formatQRun)
	}
}

func notPackaged(machine arch.Machine) error {
	return fmt.Errorf("only the Spectrum and the CPC are packaged so far, not the %s", machine)
}

// write puts the release where it was asked for, as a directory or as the one
// file it holds.
func write(format, out, from string, release *pack.Release) error {
	if out == "" {
		out = strings.TrimSuffix(from, string(filepath.Separator)) + "-" + format
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
		return fmt.Errorf("%s: %w\n\tpackage it as --format %s instead", format, err, formatDir)
	}

	if filepath.Ext(out) == "" {
		out += "." + format
	}

	if err := os.WriteFile(out, file.Body, 0o644); err != nil {
		return err
	}

	fmt.Printf("%s: %d bytes — %s\n", out, len(file.Body), file.Note)

	return nil
}
