package pack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Writing a release out, which is the step every machine shares.
func TestWritingAReleaseToADirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nuevo", "AO")

	r := NewRelease(
		File{Name: "DISK", Body: []byte("loader"), Note: "the BASIC loader"},
		File{Name: "PART1.DDB", Body: []byte("database"), Note: "the game"},
	)

	require.NoError(t, r.WriteDir(dir), "it would not make the directory it was given")

	for name, want := range map[string]string{"DISK": "loader", "PART1.DDB": "database"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		require.Equal(t, want, string(body))
	}

	require.Equal(t, 14, r.Total())
	require.Equal(t, []string{"DISK", "PART1.DDB"}, r.sortedNames())
}

// A release of one can be written as one file. A release of several cannot, and
// says what it holds instead of picking.
//
// Getting this wrong would be quiet and bad: writing the first file of a +3
// release under a .dsk name gives you an interpreter with no game in it, on a
// disk image that is not one.
func TestOnlyAReleaseOfOneIsOneFile(t *testing.T) {
	one, err := NewRelease(File{Name: "AO.tap", Body: []byte("tape")}).Single()
	require.NoError(t, err)
	require.Equal(t, "AO.tap", one.Name)

	_, err = NewRelease(
		File{Name: "DISK"},
		File{Name: "DAAD.BIN"},
		File{Name: "PART1.DDB"},
	).Single()
	require.ErrorContains(t, err, "3 files")
	require.ErrorContains(t, err, "PART1.DDB", "the error does not say what it holds")
}

// Nothing in a release writes outside the directory it was given.
//
// The names are a machine's own conventions — DISK, PART1.DDB, DAAD.FNT — and
// not one of those conventions has a directory in it, so a name that does is a
// mistake somewhere upstream rather than something to honour. A packager is
// handed a path by whoever ran it and has no business writing above it.
func TestAReleaseCannotEscapeItsDirectory(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"../escapado", "sub/fichero", "/absoluto", "..", "."} {
		err := NewRelease(File{Name: name, Body: []byte("x")}).WriteDir(dir)
		require.ErrorContains(t, err, "is not a file name", "name %q was accepted", name)
	}

	left, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, left, "something was written anyway")
}

// And the listing a person reads after a build names every file and says what
// each is for, because a directory of DISK, DAAD.BIN and PART1.SDG explains
// itself to nobody.
func TestTheListingSaysWhatEachFileIsFor(t *testing.T) {
	r := NewRelease(
		File{Name: "DISK", Body: make([]byte, 180), Note: "the BASIC loader"},
		File{Name: "PART1.SDG", Body: make([]byte, 2076), Note: "graphics stub and font"},
	)

	listing := r.Listing()
	require.Contains(t, listing, "DISK")
	require.Contains(t, listing, "the BASIC loader")
	require.Contains(t, listing, "2076")
}
