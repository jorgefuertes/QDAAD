package legacy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// What a source says about how to build it.
//
// The font is the case this exists for. La Aventura Original draws the little
// icons beside its objects with glyphs of its own, and built with anybody
// else's they come out as "ßÊì" — an adventure that compiles, boots, plays,
// and is wrong, with nothing said. The font is part of the game rather than an
// opinion about the build, so the source is where it belongs.
func TestASourceSaysWhatToBuildItWith(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "game.sce")

	require.NoError(t, os.WriteFile(source, []byte(`
; -----------------------------------------------
; Decompiled by QUnDAAD from a tape
; qdaad: charset=gfx/chr/udg.chr
; -----------------------------------------------

/CTL
_
; qdaad: charset=no-se-lee-aqui.chr
`), 0o600))

	got, err := readHeaderSettings(source)
	require.NoError(t, err)

	// Resolved against the source and not against whatever directory the
	// compiler happened to be run from, so a decompiled adventure can be moved
	// around whole.
	require.Equal(t, filepath.Join(dir, "gfx/chr/udg.chr"), got.Charset)
}

// Only the header is read. A setting a thousand lines into the tables would be
// a setting nobody finds, and looking for one would mean reading the whole
// source twice.
func TestSettingsPastTheHeaderAreNotSettings(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "game.sce")

	require.NoError(t, os.WriteFile(source, []byte(
		"/CTL\n_\n\n; qdaad: charset=tarde.chr\n"), 0o600))

	got, err := readHeaderSettings(source)
	require.NoError(t, err)
	require.Empty(t, got.Charset, "a setting below the header was obeyed")
}

// A source with none of these is every source written before they existed, and
// is not a problem.
func TestASourceWithNoSettingsIsFine(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "game.sce")

	require.NoError(t, os.WriteFile(source, []byte("; una aventura\n\n/CTL\n_\n"), 0o600))

	got, err := readHeaderSettings(source)
	require.NoError(t, err)
	require.Empty(t, got.Charset)
}

// A setting we do not understand is an error rather than something to skip.
//
// Silently ignoring it is the worst of the options: the source says what it
// wants, the build does something else, and the only sign is in the result —
// which is the exact failure this whole mechanism exists to stop happening.
func TestASettingWeDoNotKnowIsRefused(t *testing.T) {
	for _, line := range []string{
		"; qdaad: fuente=udg.chr",
		"; qdaad: charset",
		"; qdaad: charset=",
	} {
		dir := t.TempDir()
		source := filepath.Join(dir, "game.sce")

		require.NoError(t, os.WriteFile(source, []byte(line+"\n/CTL\n_\n"), 0o600))

		_, err := readHeaderSettings(source)
		require.Error(t, err, "line %q was accepted", line)
		require.Contains(t, err.Error(), "game.sce:1", "the error does not say where")
	}
}

// An absolute path is taken as given: somebody who wrote one meant it.
func TestAnAbsolutePathIsLeftAlone(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "game.sce")

	require.NoError(t, os.WriteFile(source,
		[]byte("; qdaad: charset=/opt/fuentes/udg.chr\n/CTL\n_\n"), 0o600))

	got, err := readHeaderSettings(source)
	require.NoError(t, err)
	require.Equal(t, "/opt/fuentes/udg.chr", got.Charset)
}
