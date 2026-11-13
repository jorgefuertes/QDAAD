package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/media"
)

// A release is what a machine needs in order to play, which is almost never
// one thing.
//
// Every machine DAAD targets wants the same handful: the database, an
// interpreter that can read it, a character set, a graphics block, and on the
// disk machines a loader to start the lot. The nine xREADME.TXT files that come
// with DAAD all say the same sentence — "add game DB, graphics, interpreter and
// charset on disk" — and what differs between them is the container, not the
// recipe.
//
// So this is the recipe, and the containers are written from it. A directory is
// the plainest of them and the one every machine can have: the files as they
// are, named as the machine expects them. A tape or a disk image is the same
// set put somewhere a real machine can boot from.
//
// Where a release really is one file — a 48K Spectrum tape holds the
// interpreter, the database and the graphics glued into one stream — it is a
// release of one, and writing it to a directory puts that file in it. That is
// not a special case to carry around: a caller asks for a directory and gets
// one, whatever the machine does inside.
type Release struct {
	files []File
}

// File is one named blob of a release. The name is the name the machine's own
// loader or interpreter expects, which is why it is not derived from anything:
// DAAD's loaders look for DISK, PART1.DDB and PART1.SDG, and no rule generates
// those from a target.
type File struct {
	Name string
	Body []byte
	// Note says what this file is for, in one short phrase, so that a person
	// looking at an unpacked release can tell the loader from the font.
	Note string
}

// NewRelease gathers files in the order given, which is the order a listing
// shows them in and, on the tape machines, the order they are loaded.
func NewRelease(files ...File) *Release {
	return &Release{files: files}
}

// Add puts one more file at the end.
func (r *Release) Add(name, note string, body []byte) {
	r.files = append(r.files, File{Name: name, Body: body, Note: note})
}

// AsTZX turns a one-tape release into the same tape as a TZX.
//
// It is a conversion and not a second way of building, so the blocks are the
// ones already assembled and the two files load the same. A release of more
// than one file is not a tape and says so.
func (r *Release) AsTZX() (*Release, error) {
	if len(r.files) != 1 {
		return nil, fmt.Errorf(
			"a TZX is one tape and this release has %d files: it is not a tape build",
			len(r.files))
	}

	body, err := media.TZX(r.files[0].Body)
	if err != nil {
		return nil, err
	}

	return NewRelease(File{
		Name: strings.TrimSuffix(r.files[0].Name, ".tap") + ".tzx",
		Body: body,
		Note: r.files[0].Note + ", as a TZX",
	}), nil
}

// Files is what the release holds, in order.
func (r *Release) Files() []File { return r.files }

// Single is the release's only file, for a format that is one.
//
// It refuses rather than picking, because a release of several written as one
// file would be a release with things missing and nothing said about it.
func (r *Release) Single() (File, error) {
	if len(r.files) != 1 {
		return File{}, fmt.Errorf("this release is %d files and cannot be written as one: %s",
			len(r.files), strings.Join(r.names(), ", "))
	}

	return r.files[0], nil
}

func (r *Release) names() []string {
	out := make([]string, 0, len(r.files))
	for _, f := range r.files {
		out = append(out, f.Name)
	}

	return out
}

// WriteDir writes the release into a directory, making it if it is not there.
//
// An existing directory is written into rather than emptied, and that is the
// cautious choice on purpose: the name given is somebody's own path, and a
// packager that cleared it would be a packager that can delete work. What it
// will not do is write outside it — a file named with a path separator or a
// parent is refused, because the names come from a machine's conventions and
// none of those conventions has a directory in it.
func (r *Release) WriteDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for _, f := range r.files {
		if f.Name != filepath.Base(f.Name) || f.Name == "." || f.Name == ".." {
			return fmt.Errorf("%q is not a file name", f.Name)
		}

		if err := os.WriteFile(filepath.Join(dir, f.Name), f.Body, 0o644); err != nil {
			return err
		}
	}

	return nil
}

// Listing describes the release for a person, one line per file.
//
// It is printed after a build rather than written into the directory. A README
// in there would end up on the disk image beside the game, and a 48K Spectrum
// has better uses for 180 bytes.
func (r *Release) Listing() string {
	var sb strings.Builder

	width := 0
	for _, f := range r.files {
		if len(f.Name) > width {
			width = len(f.Name)
		}
	}

	for _, f := range r.files {
		fmt.Fprintf(&sb, "  %-*s  %7d  %s\n", width, f.Name, len(f.Body), f.Note)
	}

	return sb.String()
}

// Total is how many bytes the release comes to, which on the machines with a
// disk is the number that decides whether it fits on one.
func (r *Release) Total() int {
	total := 0
	for _, f := range r.files {
		total += len(f.Body)
	}

	return total
}

// sortedNames is for tests and messages that want the names regardless of load
// order.
func (r *Release) sortedNames() []string {
	out := r.names()
	sort.Strings(out)

	return out
}
