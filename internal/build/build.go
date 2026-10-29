// Package build is what the compiler hands to the packager.
//
// The two steps used to be one command, and the seam between them was a
// function call. Pulling them apart needs the seam written down: qdaad
// compiles and writes a directory, qpack reads directories and writes the tape
// or the disk. A directory rather than a file, because a compiled part is more
// than its database — and because an adventure of two parts is two of these.
//
// What goes in it is everything the packager cannot work out for itself:
//
//   - The database. The packager could read the machine and the language out
//     of its byte 1, and does check them, but it has to be told the rest.
//   - The subtarget, which is nowhere in the database. A 48K Spectrum, a +3, a
//     Next and an esxDOS build all write ZX in that byte, and they want
//     different interpreters and different containers.
//   - The character set, when the adventure has one of its own. It travels in
//     a comment in the source, which the packager never sees — and an
//     adventure built with somebody else's font is one that plays with the
//     little icons beside its objects turned into letters.
//   - Which part this is, because a two-part release names its files by part
//     and nothing else says whether this is the first or the second.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
)

// The names inside a build directory. They are plain on purpose: somebody
// looking at one should be able to tell what it holds without a tool.
const (
	manifestFile = "build.toml"
	databaseFile = "database.ddb"
	charsetFile  = "charset.chr"
	// imagesDir holds the pictures, one file per picture, named by the number
	// a PICTURE names it with. What is in them is whatever the target machine
	// takes — a Spectrum screen for a Spectrum — because converting between
	// machines is the compiler's business and not the packager's.
	imagesDir = "images"
)

// Build is one compiled part, ready to be packaged.
type Build struct {
	Machine   arch.Machine
	Subtarget string
	Language  language.Language
	Version   uint8
	// Part is which part of the adventure this is, counting from one. An
	// adventure of one part is part one.
	Part int
	// Tokens is how many entries the compression table has. It is the linker's
	// own count and is nowhere in the database — the table's end is not in any
	// header pointer — so a reader has to deduce it or be told. See
	// internal/pack/qrun.go, which is what being told is for.
	Tokens int

	Database []byte
	// Charset is the adventure's own font, or empty to use the built-in one.
	Charset []byte
	// Images are the pictures, by the number that shows them. A build with
	// none is the usual case: most of the corpus illustrates with nothing, or
	// with vectors that version 3 cannot draw.
	Images map[int][]byte
}

// Write lays the build out in a directory, making it if it is not there.
func (b Build) Write(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, manifestFile), []byte(b.manifest()), 0o644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, databaseFile), b.Database, 0o644); err != nil {
		return err
	}

	if len(b.Charset) > 0 {
		if err := os.WriteFile(filepath.Join(dir, charsetFile), b.Charset, 0o644); err != nil {
			return err
		}
	}

	return b.writeImages(dir)
}

// writeImages puts the pictures in a directory of their own, one file each,
// named by number so that the directory reads like the index it becomes.
func (b Build) writeImages(dir string) error {
	if len(b.Images) == 0 {
		return nil
	}

	into := filepath.Join(dir, imagesDir)
	if err := os.MkdirAll(into, 0o755); err != nil {
		return err
	}

	for number, body := range b.Images {
		name := fmt.Sprintf("%03d.scr", number)
		if err := os.WriteFile(filepath.Join(into, name), body, 0o644); err != nil {
			return err
		}
	}

	return nil
}

// readImages takes them back out. A directory that is not there is a build
// with no pictures, which is not a problem.
func readImages(dir string) (map[int][]byte, error) {
	names, err := filepath.Glob(filepath.Join(dir, imagesDir, "*.scr"))
	if err != nil || len(names) == 0 {
		return nil, err
	}

	out := make(map[int][]byte, len(names))

	for _, name := range names {
		number, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(name), ".scr"))
		if err != nil {
			return nil, fmt.Errorf("%s: a picture is named by its number: %w", name, err)
		}

		if out[number], err = os.ReadFile(name); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// manifest is the file written beside the database, saying what it was built
// for.
//
// It is TOML, in the sense that TOML would read it. Nothing reads it but us
// and it holds six keys, so it is written and read here rather than by a
// library — the same choice as the header comments a source carries, and for
// the same reason.
func (b Build) manifest() string {
	var sb strings.Builder

	sb.WriteString("# Written by qdaad. This says what the database beside it was\n")
	sb.WriteString("# built for, which is what qpack cannot work out on its own.\n\n")

	fmt.Fprintf(&sb, "machine   = %q\n", strings.ToLower(b.Machine.String()))
	fmt.Fprintf(&sb, "subtarget = %q\n", b.Subtarget)
	fmt.Fprintf(&sb, "language  = %q\n", b.Language.String())
	fmt.Fprintf(&sb, "version   = %d\n", b.Version)
	fmt.Fprintf(&sb, "part      = %d\n", b.Part)
	fmt.Fprintf(&sb, "tokens    = %d\n", b.Tokens)

	return sb.String()
}

// Read takes a build back out of a directory.
func Read(dir string) (Build, error) {
	manifest, err := os.ReadFile(filepath.Join(dir, manifestFile))
	if err != nil {
		return Build{}, fmt.Errorf("%s is not a build directory: %w", dir, err)
	}

	out, err := parseManifest(string(manifest))
	if err != nil {
		return Build{}, fmt.Errorf("%s: %w", filepath.Join(dir, manifestFile), err)
	}

	if out.Database, err = os.ReadFile(filepath.Join(dir, databaseFile)); err != nil {
		return Build{}, err
	}

	// A build with no font of its own is the usual case, so a missing one is
	// not a problem. Anything else going wrong with it is.
	charset, err := os.ReadFile(filepath.Join(dir, charsetFile))
	if err == nil {
		out.Charset = charset
	} else if !os.IsNotExist(err) {
		return Build{}, err
	}

	if out.Images, err = readImages(dir); err != nil {
		return Build{}, err
	}

	return out, nil
}

func parseManifest(text string) (Build, error) {
	var out Build

	for n, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Build{}, fmt.Errorf("line %d: %q is not key = value", n+1, line)
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)

		if err := out.set(key, value); err != nil {
			return Build{}, fmt.Errorf("line %d: %w", n+1, err)
		}
	}

	if out.Machine == 0 {
		return Build{}, fmt.Errorf("no machine: a build has to say what it is for")
	}

	return out, nil
}

func (b *Build) set(key, value string) error {
	switch key {
	case "machine":
		machine, err := arch.Parse(value)
		if err != nil {
			return err
		}

		b.Machine = machine

	case "subtarget":
		b.Subtarget = value

	case "language":
		lang := language.LanguageUndefined.Parse(value)
		if !lang.IsValid() {
			return fmt.Errorf("%q is not a language", value)
		}

		b.Language = lang

	case "version", "part", "tokens":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}

		switch key {
		case "version":
			b.Version = uint8(n)
		case "part":
			b.Part = n
		default:
			b.Tokens = n
		}

	default:
		return fmt.Errorf("%q is not something a build says about itself", key)
	}

	return nil
}

// SameAdventure reports whether these builds belong together, and says what is
// wrong when they do not.
//
// Packaging two parts means putting two databases on one disk for one
// interpreter, so they have to agree about the machine, the variant of it and
// the language. Two parts that disagree is the kind of mistake that produces a
// disk rather than an error — one that boots, loads part one, and does
// something unaccountable when the player asks for part two.
func SameAdventure(parts []Build) error {
	if len(parts) < 2 {
		return nil
	}

	first := parts[0]

	for _, p := range parts[1:] {
		switch {
		case p.Machine != first.Machine:
			return fmt.Errorf("one part is for a %s and another for a %s", first.Machine, p.Machine)
		case !strings.EqualFold(p.Subtarget, first.Subtarget):
			return fmt.Errorf("one part is for a %s %q and another for a %q",
				first.Machine, first.Subtarget, p.Subtarget)
		case p.Language != first.Language:
			return fmt.Errorf("one part speaks %s and another %s", first.Language, p.Language)
		}
	}

	return nil
}
