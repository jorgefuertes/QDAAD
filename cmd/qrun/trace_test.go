package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Where a trace ends up, which is worth a test because getting it wrong is
// silent and expensive.
//
// The whole point of the trace directory is that it is not the boot volume. A
// resolver that fell back to the working directory when the disk was missing
// would undo that on exactly the day it mattered — the day the boot volume is
// full — and it would do it without saying anything.
func TestWhereATraceGoes(t *testing.T) {
	dir := t.TempDir()

	t.Run("a bare name goes in the trace directory", func(t *testing.T) {
		got, err := tracePath("partida.log", dir)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "partida.log"), got)
	})

	t.Run("a path with a directory is taken as given", func(t *testing.T) {
		// Not under dir: whoever typed a path meant that path, and the trace
		// directory has nothing to say about it.
		got, err := tracePath("sub/partida.log", dir)
		require.NoError(t, err)
		require.Equal(t, "sub/partida.log", got)
	})

	t.Run("the directory is made if it is not there", func(t *testing.T) {
		made := filepath.Join(dir, "nuevo", "mas")

		got, err := tracePath("partida.log", made)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(made, "partida.log"), got)
		require.DirExists(t, made, "the directory was not made")
	})

	t.Run("a directory that cannot be had is refused, not replaced", func(t *testing.T) {
		// A file where the directory should be is the cheapest stand-in for a
		// disk that is not plugged in: MkdirAll fails either way.
		blocked := filepath.Join(dir, "soy-un-fichero")
		require.NoError(t, os.WriteFile(blocked, nil, 0o600))

		_, err := tracePath("partida.log", blocked)
		require.Error(t, err, "it fell back somewhere instead of refusing")
		require.ErrorContains(t, err, blocked, "the error does not say where it tried")
		require.ErrorContains(t, err, "--trace", "the error does not say what to do instead")
	})
}
