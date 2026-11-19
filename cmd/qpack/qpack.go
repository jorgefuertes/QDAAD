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
	"strconv"
	"strings"
	"time"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/build"
	"github.com/jorgefuertes/QDAAD/internal/console"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/pack"
	"github.com/jorgefuertes/QDAAD/internal/util"
	"github.com/jorgefuertes/QDAAD/internal/version"
	"github.com/urfave/cli/v3"
)

// What can be written. A directory is the one every machine can have, because
// it is the recipe rather than a container for it; the rest are containers a
// particular machine boots from.
const (
	formatDir  = "dir"
	formatTAP  = "tap"
	formatTZX  = "tzx"
	formatCDT  = "cdt"
	formatDSK  = "dsk"
	formatQRun = "qrun"
)

func main() {
	cmd := &cli.Command{
		Version:   version.Version,
		Name:      "qpack",
		Usage:     "package compiled parts into a tape, a disk or a directory",
		ArgsUsage: "<build directory> [second build directory]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   formatDir,
				Usage: "What to write: " + formatDir + " for everything the machine " +
					"needs as files, " + formatTAP + ", " + formatTZX + " or " +
					formatCDT + " for a tape, " + formatDSK + " for a disk, or " + formatQRun +
					" for a container our own interpreter reads without guessing " +
					"at anything.",
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
			&cli.IntFlag{
				Name:  "lines",
				Value: 0,
				Usage: "How many pixel rows of each picture to carry, for the " +
					"machines that carry them whole. Zero, which is the default, " +
					"carries each picture down to its own last row with a pixel " +
					"in it. It is worth setting on an Amstrad and nowhere else: " +
					"nothing compresses a CPC picture, so a tall one is sixteen " +
					"kilobytes of a disk with a hundred and seventy-eight, and " +
					"the 96 lines DAAD's own IMGLINES defaults to are 96 because " +
					"that is the most that lets two bands share a record.",
			},
			&cli.IntFlag{
				Name:  "generic",
				Value: 0,
				Usage: "The picture to put up in place of the ones that do not fit: " +
					"the one the adventure already shows for a location with no " +
					"drawing of its own, which in La Aventura Original is 27. It is " +
					"worth naming because PICTURE is a condition in version 3 — a " +
					"picture that is not there fails it and abandons the rest of " +
					"the entry, which is how a drawing left off for want of room " +
					"takes a paragraph and the text window with it. Only an " +
					"Amstrad's bank reads this.",
			},
			&cli.IntFlag{
				Name:  "ram",
				Value: 64,
				Usage: "How much memory the machine has, in kilobytes. Only the " +
					"Amstrad reads it, and only to decide where the pictures go: " +
					"64 keeps them in what is left above the game, which is about " +
					"a dozen of them, and 128 puts them in a 6128's four spare " +
					"banks, where a whole adventure fits and no drive or tape is " +
					"touched once it is running. A 128 release will not run on a " +
					"464 or a 664.",
			},
			&cli.StringFlag{
				Name:    "loading",
				Aliases: []string{"l"},
				Usage: "A screen to show while the rest loads, as the 6912 bytes of " +
					"a Spectrum's video memory. The build directory may carry one " +
					"already — the source names it with \"; qdaad: loading=\" — and " +
					"this wins over that one. On an Amstrad the colours come with " +
					"it: a .pal of the same name beside it, which is what qundaad " +
					"writes.",
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		console.Sayln(console.LevelError, console.ErrStyle.Render(err.Error()))
		os.Exit(1)
	}
}

func run(_ context.Context, cmd *cli.Command) error {
	console.Banner("QPACK", console.TitleStyle)

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

	// The flag wins over what the build carries, the same way the compiler's
	// --charset wins over what the source says: the picture belongs to the
	// adventure nearly always, and "nearly" is why there is a flag.
	loadingPalette := []byte(nil)

	if path := cmd.String("loading"); path != "" {
		loading, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		parts[0].Loading = loading

		// The Amstrad's screens carry no colours of their own, so they travel
		// as a pair: qundaad writes loading.scr and loading.pal beside each
		// other and this picks the second one up without being told twice.
		beside := strings.TrimSuffix(path, filepath.Ext(path)) + ".pal"
		if palette, err := os.ReadFile(beside); err == nil {
			loadingPalette = palette
		}
	}

	format := cmd.String("format")

	console.Sayln(console.LevelInfo, "Packaging ", console.PathStyle.Render(dirs[0]),
		" for a ", console.BoldStyle.Render(parts[0].Machine.Model(parts[0].Subtarget)),
		console.MutedStyle.Render(" as a "+format))

	// Timed here rather than inside: what a person waits for is the packaging,
	// and nearly all of it is one picture after another going through ZX0.
	started := time.Now()

	release, err := assemble(format, parts[0], name,
		cmd.Int("lines"), cmd.Int("generic"), cmd.Int("ram"), loadingPalette)
	if err != nil {
		return err
	}

	out, err := write(format, cmd.String("output"), dirs[0], release)
	if err != nil {
		return err
	}

	report(format, parts[0], name, out, release, time.Since(started))

	return nil
}

// report is the table printed after a release is written, in the shape qundaad
// prints its own: what was built, for what, and how big it came out.
func report(
	format string, b build.Build, name, out string, release *pack.Release, took time.Duration,
) {
	console.Sayln(console.LevelOk, "Written to ", console.PathStyle.Render(out))
	console.Sayln(console.LevelInfo, "Packaging report:")

	t := console.NewTable().Horizontal(true)
	t.Row("Machine", b.Machine.Model(b.Subtarget))
	t.Row("Format", format)
	t.Row("Name", name)
	t.Row("Database", util.HumanBytes(int64(len(b.Database))))

	if len(b.Images) > 0 {
		t.Row("Pictures", strconv.Itoa(len(b.Images)))
	}

	if len(b.XMessages) > 0 {
		t.Row("Texts outside it", util.HumanBytes(int64(len(b.XMessages))))
	}

	t.Row("Files", strconv.Itoa(len(release.Files())))
	t.Row("Written", util.HumanBytes(int64(release.Total())))
	t.Row("Took", util.HumanDuration(took))
	t.Print()

	// Which file is which, for a release of several. One file has its own line
	// above and a table of one row would say it twice.
	if len(release.Files()) > 1 {
		files := console.NewTable().Header("File", "Size", "What it is")
		for _, f := range release.Files() {
			files.Row(f.Name, util.HumanBytes(int64(len(f.Body))), f.Note)
		}

		files.Print()
	}
}

// assemble gathers what the asked-for format needs, as a release.
//
// Everything comes back as a release even when it is one file, so that there
// is one thing to write and --format dir works for every format without each
// one knowing about it.
func assemble(
	format string, b build.Build, name string, lines, generic, memory int, loadingPalette []byte,
) (*pack.Release, error) {
	opts := pack.Options{
		Language: b.Language, Name: name,
		Charset: b.Charset, Images: b.Images, Palettes: b.Palettes, Loading: b.Loading,
		XMessages: b.XMessages, Lines: lines, Generic: generic, Memory: memory,
		LoadingPalette: loadingPalette,
		Note:           notes(),
		Progress:       progress(),
	}

	switch format {
	case formatQRun:
		body, err := pack.QRun(b.Database, int(b.Machine.BaseAddress(b.Subtarget)),
			b.Machine.IsBigEndian(), back.Layout{Tokens: b.Tokens})
		if err != nil {
			return nil, err
		}

		release := pack.NewRelease(pack.File{
			Name: name + ".qrun", Body: body,
			Note: "database in our own container, for qrun",
		})

		// And the texts that are not in it, under the name qrun looks for:
		// beside the database, same name, .xmb. Without it an adventure that
		// keeps texts outside would come out playable and short of a third of
		// what it says.
		if len(b.XMessages) > 0 {
			release.Add(name+".xmb",
				fmt.Sprintf("%d bytes of texts read from outside the database",
					len(b.XMessages)),
				b.XMessages)
		}

		return release, nil

	// A directory is the recipe rather than a container for it, so every machine
	// that is packaged at all can have one.
	case formatDir:
		switch b.Machine {
		case arch.ZX:
			return pack.ZXRelease(b.Database, b.Subtarget, opts)
		case arch.CPC:
			return pack.CPCRelease(b.Database, opts)
		default:
			return nil, notPackaged(b.Machine)
		}

	// A TAP is a Spectrum tape and nothing else. It used to share a branch with
	// the directory, which meant a CPC build asked for one got the Amstrad's own
	// single binary in a file called .tap: not a tape, and not a thing that says
	// so until a 464 is asked to load it.
	case formatTAP:
		if b.Machine != arch.ZX {
			return nil, fmt.Errorf(
				"a TAP is a Spectrum tape, and this build is for a %s"+
					"\n\tan Amstrad tape is --format %s", b.Machine, formatCDT)
		}

		return pack.ZXRelease(b.Database, b.Subtarget, opts)

	// The Amstrad's tape is not the Spectrum's wrapped: the machine has a tape
	// format of its own, records and checksums and all, so it is built rather
	// than converted.
	case formatCDT:
		if b.Machine != arch.CPC {
			return nil, fmt.Errorf("a CDT is an Amstrad tape, and this build is for a %s",
				b.Machine)
		}

		body, err := pack.CPCTape(b.Database, opts)
		if err != nil {
			return nil, err
		}

		return pack.NewRelease(pack.File{
			Name: name + ".cdt", Body: body,
			Note: `tape for a 464: RUN"` + name + `"`,
		}), nil

	// A TZX is the tape that was going to be written anyway, wrapped. Building
	// it any other way would be a second assembly of the same blocks and a
	// second place for them to go wrong.
	case formatTZX:
		if b.Machine != arch.ZX {
			return nil, fmt.Errorf("a TZX is a Spectrum tape, and this build is for a %s",
				b.Machine)
		}

		tape, err := pack.ZXRelease(b.Database, b.Subtarget, opts)
		if err != nil {
			return nil, err
		}

		return tape.AsTZX()

	case formatDSK:
		var (
			body []byte
			note string
			err  error
		)

		switch b.Machine {
		case arch.ZX:
			body, err = pack.ZXDisk(b.Database, b.Subtarget, opts)
			note = "177K +3 disk, loaded from the +3's own menu"
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
		return nil, fmt.Errorf("%q is not a format: it is %s, %s, %s, %s, %s or %s",
			format, formatDir, formatTAP, formatTZX, formatCDT, formatDSK, formatQRun)
	}
}

// notes draws what the packager says about the release while it builds it.
//
// What the notice decides is the colour of the caret and nothing else: a note
// that something was left out is a warning and reads as one at a glance, which
// is the whole reason it carries a level.
func notes() pack.Notes {
	return func(level pack.Notice, said string) {
		if level == pack.NoticeLeftOut {
			console.Sayln(console.LevelWarn, said)

			return
		}

		console.Sayln(console.LevelOk, said)
	}
}

// progressWidth is how wide the line is drawn. Every line is drawn to the same
// width so that the next one covers the whole of the last, with no tail of a
// longer phrase left behind it.
const progressWidth = 52

// progress draws what the packager is working on, each line over the last, and
// rubs the line out when the work it was about is done.
//
// It goes to the error stream and only when there is a person on the other end
// of it. A log wants the result and not the sixty-seven lines it took to get
// there, and a progress line in a file is sixty-seven lines with carriage
// returns in them.
//
// The stream is taken now rather than read at every line, because console.Make
// points os.Stderr at the null device while it runs something: a line drawn
// through the name would land there instead of on the terminal.
func progress() pack.Progress {
	out := os.Stderr
	if !isTerminal(out) {
		return nil
	}

	caret := console.MutedStyle.Render(console.RightCaret)

	// "7 of 28" rather than "7/28", because the two numbers of a picture are
	// not the same number: the pictures are numbered as the adventure asks for
	// them and that numbering has holes, so picture 011 is the seventh of the
	// twenty-eight a part carries. A slash between them reads like a fraction
	// of something and invites being read as the picture's own number.
	return func(done, total int, what string) {
		count := fmt.Sprintf("%d of %d", done, total)

		// Padded by what the line shows and not by what it holds: the styling
		// is escape codes, which take plenty of string and no room on screen,
		// and the caret is three bytes of one column.
		shown := 1 + 1 + len(what) + 3 + len(count)

		// Nothing useful can be done about a progress line that cannot be
		// written, and it is the one line of the output nobody needs: what
		// matters is said again at the end, where a failure to write it is
		// worth reporting.
		_, _ = fmt.Fprintf(out, "\r%s %s   %s%*s", caret, what,
			console.MutedStyle.Render(count), max(progressWidth-shown, 0), "")

		if done >= total {
			_, _ = fmt.Fprintf(out, "\r%*s\r", progressWidth, "")
		}
	}
}

// isTerminal says whether there is a person on the other end of a file. A
// character device is a terminal; a redirected file or a pipe is not.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func notPackaged(machine arch.Machine) error {
	return fmt.Errorf("only the Spectrum and the CPC are packaged so far, not the %s", machine)
}

// write puts the release where it was asked for, as a directory or as the one
// file it holds, and says where that was: the name is worked out here and the
// report wants to show it.
func write(format, out, from string, release *pack.Release) (string, error) {
	if out == "" {
		out = strings.TrimSuffix(from, string(filepath.Separator)) + "-" + format
	}

	if format == formatDir {
		if err := release.WriteDir(out); err != nil {
			return "", err
		}

		return out, nil
	}

	file, err := release.Single()
	if err != nil {
		return "", fmt.Errorf("%s: %w\n\tpackage it as --format %s instead", format, err, formatDir)
	}

	if filepath.Ext(out) == "" {
		out += "." + format
	}

	if err := os.WriteFile(out, file.Body, 0o644); err != nil {
		return "", err
	}

	return out, nil
}
