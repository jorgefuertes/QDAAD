package legacy_test

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/stretchr/testify/require"
)

// Where script/decomp-all.sh leaves what it read out of the original releases.
const decompiled = "../../../work/decomp"

// Every edition we can decompile has to read back in.
//
// It is a round trip and that is what makes it worth running: the sources are
// not written by hand to suit the parser, they come out of our own decompiler
// reading a real 1980s binary, and there are sixty-six of them across five
// adventures and a dozen machines. Anything the grammar gets wrong about the
// format shows up in one of them.
//
// What it does not check is that the database is right — only that the whole
// source is read and understood. Comparing the contents against the compiler of
// reference is the next step, and it needs the JSON.
func TestReadsEveryDecompiledSource(t *testing.T) {
	mains := findMainSources(t)
	if len(mains) == 0 {
		t.Skip("work/decomp is empty: run script/decomp-all.sh first")
	}

	for _, main := range mains {
		t.Run(editionOf(main), func(t *testing.T) {
			_, err := legacy.Analyze(main, legacy.Options{
				Machine:   arch.ZX,
				Subtarget: "128K",
				Language:  language.LanguageSpanish,
				Version:   3,
			})
			require.NoError(t, err)
		})
	}
}

// findMainSources collects the game.sce of every edition, which is the file
// that pulls the rest in with #include.
func findMainSources(t *testing.T) []string {
	t.Helper()

	var found []string

	err := filepath.WalkDir(decompiled, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// The tree is not in the checkout, which is not a failure: the
			// sources are built by a script and are not committed.
			return fs.SkipAll
		}

		if !d.IsDir() && d.Name() == "game.sce" {
			found = append(found, path)
		}

		return nil
	})
	if err != nil {
		t.Skipf("work/decomp cannot be walked: %v", err)
	}

	return found
}

// editionOf names a run after the adventure and the machine rather than the
// path, so that a failure says which release broke.
func editionOf(main string) string {
	rel, err := filepath.Rel(decompiled, filepath.Dir(main))
	if err != nil {
		return main
	}

	return strings.ReplaceAll(rel, string(filepath.Separator), "/")
}
