package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/stretchr/testify/require"
)

// What --test does from outside, which is the only thing a build server reads:
// a script that holds is quiet and worth nothing to anybody, and one that does
// not comes back non-zero.
//
// The language itself is tested in internal/qtest. This is the wiring.

func TestAScriptThatHoldsSaysSoAndOneThatDoesNotFails(t *testing.T) {
	game := compiled(t)

	for _, c := range []struct {
		name   string
		script string
		holds  bool
		says   string
	}{
		{
			name: "it holds",
			script: `
KEY SPACE
KEY SPACE
KEY SPACE

TEST "the forest"
STEP "the first room"
  EXPECT "Estás en el bosque"
  ASSERT AT = 3
END
`,
			holds: true,
			says:  "OK",
		},
		{
			name: "it does not",
			script: `
KEY SPACE
KEY SPACE
KEY SPACE

TEST "the forest"
STEP "the first room"
  EXPECT "Estás en la biblioteca"
END
`,
			holds: false,
			says:  "FAIL",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := written(t, "t.qtest", c.script)

			said, err := qrun(t, "-i", game, "-T", path)

			if c.holds {
				require.NoError(t, err, "the script should have held:\n%s", said)
			} else {
				require.Error(t, err, "a step that does not hold has to come back non-zero")
			}

			require.Contains(t, said, c.says)
		})
	}
}

// A script may name its own database, which is what lets one be run on its own
// without remembering where the build went.
func TestAScriptCanNameItsOwnDatabase(t *testing.T) {
	game := compiled(t)

	path := written(t, "t.qtest", `
GAME "`+game+`"

KEY SPACE
KEY SPACE
KEY SPACE

STEP "the first room"
  EXPECT "Estás en el bosque"
END
`)

	said, err := qrun(t, "-T", path)
	require.NoError(t, err, said)
	require.Contains(t, said, "OK")
}

// And one that names nothing, with nothing given, is told so rather than
// playing nothing quietly.
func TestAScriptWithNoDatabaseIsToldSo(t *testing.T) {
	path := written(t, "t.qtest", "STEP \"nothing\"\n  EXPECT hello\nEND\n")

	_, err := qrun(t, "-T", path)
	require.ErrorContains(t, err, "no database to play")
}

func TestATranscriptKeepsWhatTheAdventureSaid(t *testing.T) {
	game := compiled(t)
	dir := t.TempDir()

	path := written(t, "t.qtest", `
KEY SPACE
KEY SPACE
KEY SPACE

STEP "the first room"
  EXPECT "Estás en el bosque"
END
`)

	words := filepath.Join(dir, "said.txt")

	said, err := qrun(t, "-i", game, "-T", path, "--transcript", words)
	require.NoError(t, err, said)

	kept, err := os.ReadFile(words)
	require.NoError(t, err)

	require.Contains(t, string(kept), "AVENTURA ORIGINAL",
		"the transcript is everything it said, including what no step asked about")

	require.NotContains(t, said, "AVENTURA ORIGINAL",
		"and without --show the report is about the steps")
}

// qrun runs the command the way a person does, and gives back what they would
// have read.
func qrun(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var out strings.Builder

	cmd := command()
	cmd.Writer = &out

	err := cmd.Run(context.Background(), append([]string{"qrun"}, args...))

	return out.String(), err
}

func written(t *testing.T, name, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

// compiled builds La Aventura Original and leaves the database where the
// command can read it, which is what a person would have done with qdaad
// first.
func compiled(t *testing.T) string {
	t.Helper()

	source := filepath.Join("..", "..", "examples", "AO", "ZXSpectrum", "+3", "part1", "game.sce")

	opts := legacy.Options{
		Machine: arch.PC, Subtarget: "VGA",
		Language: language.LanguageSpanish, Version: 3,
	}

	db, err := legacy.Analyze(source, opts)
	require.NoError(t, err)

	image, _, err := back.LinkWithLayout(db, back.Options{
		Machine: opts.Machine, Subtarget: opts.Subtarget,
		Language: opts.Language, Version: opts.Version,
	})
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "AO.ddb")
	require.NoError(t, os.WriteFile(path, image, 0o600))

	return path
}
