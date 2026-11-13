package build

import (
	"path/filepath"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/stretchr/testify/require"
)

// The seam between the two tools, written down and read back.
//
// Everything in here is something the packager cannot work out for itself. The
// subtarget is the plainest: a 48K Spectrum, a +3, a Next and an esxDOS build
// all write ZX in the database's byte 1, and they want different interpreters
// and different containers. The token count is the subtlest — the compression
// table's end is in no header pointer, so it is the linker's own count or
// nothing.
func TestABuildSurvivesTheTripThroughADirectory(t *testing.T) {
	want := Build{
		Machine:   arch.ZX,
		Subtarget: "plus3",
		Language:  language.LanguageSpanish,
		Version:   3,
		Part:      2,
		Tokens:    97,
		Database:  []byte("the adventure"),
		Charset:   []byte("its own font"),
	}

	dir := filepath.Join(t.TempDir(), "nuevo", "AO-part2")
	require.NoError(t, want.Write(dir))

	got, err := Read(dir)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// A build with no font of its own is the usual case, and reads back as one.
func TestABuildWithNoFontOfItsOwn(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, Build{
		Machine: arch.CPC, Language: language.LanguageEnglish,
		Version: 3, Part: 1, Database: []byte("x"),
	}.Write(dir))

	got, err := Read(dir)
	require.NoError(t, err)
	require.Empty(t, got.Charset)
	require.Equal(t, arch.CPC, got.Machine)
}

// A build directory is written over rather than made afresh, so what the last
// build left in it has to go. What a packager reads is the directory, not the
// source, and a picture that outlives its source file is a picture that goes on
// being packaged.
//
// This is not hypothetical. Drawing 4 was taken out of La Aventura Original's
// second part — a piece of another drawing that had landed at the number of a
// location people visit — and it went on turning up on the +3 disk, because the
// file from the build before was still sitting there.
func TestABuildDirectoryKeepsNothingFromTheBuildBefore(t *testing.T) {
	dir := t.TempDir()

	before := Build{
		Machine: arch.ZX, Subtarget: "plus3", Language: language.LanguageSpanish,
		Version: 3, Part: 1, Database: []byte("x"),
		Charset: []byte("a font"),
		Loading: []byte("a screen"),
		Images:  map[int][]byte{3: []byte("the cabin"), 4: []byte("a piece")},
	}
	require.NoError(t, before.Write(dir))

	after := before
	after.Images = map[int][]byte{3: []byte("the cabin")}
	after.Charset = nil
	after.Loading = nil
	require.NoError(t, after.Write(dir))

	got, err := Read(dir)
	require.NoError(t, err)

	require.Equal(t, after.Images, got.Images, "picture 4 outlived its source")
	require.Empty(t, got.Charset, "the font outlived its source")
	require.Empty(t, got.Loading, "the loading screen outlived its source")

	require.NoFileExists(t, filepath.Join(dir, imagesDir, "004.scr"))
	require.NoFileExists(t, filepath.Join(dir, charsetFile))
	require.NoFileExists(t, filepath.Join(dir, loadingFile))
}

// Something that is not a build directory says so, naming what it looked for.
func TestWhatIsNotABuildDirectory(t *testing.T) {
	_, err := Read(t.TempDir())
	require.ErrorContains(t, err, "not a build directory")
}

// And a manifest that says something we do not understand is an error rather
// than something to skip, for the same reason the source settings are.
func TestAManifestWeDoNotUnderstand(t *testing.T) {
	for _, text := range []string{
		`maquina = "zx"`,
		`machine = "una tostadora"`,
		"machine",
		`language = "klingon"`,
	} {
		_, err := parseManifest(text)
		require.Error(t, err, "manifest %q was accepted", text)
	}

	// And one that never says what it is for is not usable at all.
	_, err := parseManifest(`part = 1`)
	require.ErrorContains(t, err, "no machine")
}

// Two parts of one adventure have to agree about the machine they are for.
//
// Two that disagree would make a disk rather than an error: one that boots,
// loads the first part, and does something unaccountable when the player asks
// for the second.
func TestTwoPartsHaveToBeTheSameAdventure(t *testing.T) {
	zx := Build{Machine: arch.ZX, Subtarget: "plus3", Language: language.LanguageSpanish}

	require.NoError(t, SameAdventure([]Build{zx, zx}))
	require.NoError(t, SameAdventure([]Build{zx}), "one part agrees with itself")

	for _, other := range []Build{
		{Machine: arch.CPC, Subtarget: "plus3", Language: language.LanguageSpanish},
		{Machine: arch.ZX, Subtarget: "48k", Language: language.LanguageSpanish},
		{Machine: arch.ZX, Subtarget: "plus3", Language: language.LanguageEnglish},
	} {
		require.Error(t, SameAdventure([]Build{zx, other}),
			"a part for a %s %q speaking %s was accepted beside a %s %q speaking %s",
			other.Machine, other.Subtarget, other.Language, zx.Machine, zx.Subtarget, zx.Language)
	}
}
