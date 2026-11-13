package legacy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// write puts a file in the test's own directory and returns its path.
func write(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

// What matters is not only that the lines arrive, but that each one still says
// which file the author wrote it in and on what line of that file. Everything
// downstream reports errors with it.
func TestResolveKeepsWhereEachLineCameFrom(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "voc.sce", "/VOC\nN 1 noun\n")
	main := write(t, dir, "game.sce", "/CTL\n_\n#include voc.sce\n/END\n")

	lines, err := Resolve(main)
	require.NoError(t, err)

	require.Equal(t, []Line{
		{File: main, Num: 1, Text: "/CTL"},
		{File: main, Num: 2, Text: "_"},
		{File: filepath.Join(dir, "voc.sce"), Num: 1, Text: "/VOC"},
		{File: filepath.Join(dir, "voc.sce"), Num: 2, Text: "N 1 noun"},
		{File: main, Num: 4, Text: "/END"},
	}, lines)
}

// The name is taken from a fixed column, without quotes, and a comment after it
// is not part of it.
func TestResolveTakesTheNameWithoutItsComment(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "voc.sce", "/VOC\n")
	main := write(t, dir, "game.sce", "#include voc.sce   ; the vocabulary\n")

	lines, err := Resolve(main)
	require.NoError(t, err)
	require.Equal(t, "/VOC", lines[0].Text)
}

func TestResolveRefusesNestedIncludes(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "inner.sce", "/VOC\n")
	write(t, dir, "outer.sce", "#include inner.sce\n")
	main := write(t, dir, "game.sce", "#include outer.sce\n")

	_, err := Resolve(main)
	require.ErrorContains(t, err, "nested includes are not allowed")
}

func TestResolveComplainsAboutAMissingFile(t *testing.T) {
	dir := t.TempDir()
	main := write(t, dir, "game.sce", "#include nowhere.sce\n")

	_, err := Resolve(main)
	require.ErrorContains(t, err, "nowhere.sce")
}

// A source has an encoding and it is UTF-8. Anything else is an error to fix
// before compiling, not something to guess at.
func TestResolveRefusesWhatIsNotUTF8(t *testing.T) {
	dir := t.TempDir()
	// "ñ" as ISO-8859-1 is a byte that cannot start a UTF-8 sequence.
	main := write(t, dir, "game.sce", "; a\xf1o\n")

	_, err := Resolve(main)
	require.ErrorContains(t, err, "not UTF-8")
}

// A word that merely starts like the directive is not the directive.
func TestResolveDoesNotMistakeALookalike(t *testing.T) {
	dir := t.TempDir()
	main := write(t, dir, "game.sce", "#includes something\n")

	lines, err := Resolve(main)
	require.NoError(t, err)
	require.Equal(t, "#includes something", lines[0].Text)
}
