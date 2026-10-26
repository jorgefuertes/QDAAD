package vm_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/jorgefuertes/QDAAD/internal/vm"
	"github.com/stretchr/testify/require"
)

// Playing the adventure, which is the only test of an interpreter that is
// worth much.
//
// The unit a condact lives in can be checked on its own and several of them
// here are, but the mistakes that actually happened were not of that kind.
// Every one of them was a condact doing something defensible in isolation and
// wrong in company: a process running its first entry without matching it, a
// PARSE whose answer meant the opposite of how it reads, a MOVE taking the
// flag it was given for a direction when it names a location. None of those
// shows up until something plays.
//
// So this plays. The console is the same three methods a person gets, reading
// from a list and keeping what the adventure said.

// script is a player who already knows what to type.
type script struct {
	lines []string
	said  strings.Builder
}

func (s *script) Print(text string) { s.said.WriteString(text) }

func (s *script) Input() (string, bool) {
	if len(s.lines) == 0 {
		return "", false
	}

	line := s.lines[0]
	s.lines = s.lines[1:]

	return line, true
}

// AnyKey returns at once and takes nothing from the list.
//
// A script has no keyboard, so there is no key to wait for; and taking a line
// would mean counting the adventure's pauses to know how many blank lines to
// put in front of the first real command, which is a thing a test should never
// have to know.
func (s *script) AnyKey() {}

// transcript plays an adventure through a list of commands and gives back
// everything it said.
func transcript(t *testing.T, source string, commands ...string) string {
	t.Helper()

	db, err := legacy.Analyze(source, legacy.Options{
		Machine: arch.PC, Subtarget: "VGA",
		Language: language.LanguageSpanish, Version: 3,
	})
	require.NoError(t, err)

	image, layout, err := back.LinkWithLayout(db, back.Options{
		Machine: arch.PC, Subtarget: "VGA",
		Language: language.LanguageSpanish, Version: 3,
	})
	require.NoError(t, err)
	require.Positive(t, layout.Tokens)

	r, ok := ddbfile.At(image, 0)
	require.True(t, ok)

	player := &script{lines: commands}

	machine, err := vm.New(r, player, nil)
	require.NoError(t, err)

	// Running out of input ends the game, so this returns rather than hanging.
	require.NoError(t, machine.Run())

	return player.said.String()
}

// The template is the smallest adventure there is and it is in the checkout,
// so this runs wherever the repository does.
func TestTheTemplatePlays(t *testing.T) {
	source := templateSource(t)

	said := transcript(t, source, "mirar", "i")

	require.Contains(t, said, "mazmorra",
		"the location the template starts in, printed by the game loop")
	require.Contains(t, said, ">", "and a prompt, so the loop reached the player")
}

// An adventure that does not understand says so, once, and goes on asking.
// Getting this wrong is not loud: PARSE reads as though it succeeds on a good
// sentence, and taking it that way answers every command with the response
// table's first entry.
func TestWhatItSaysToNonsense(t *testing.T) {
	said := transcript(t, templateSource(t), "xyzzy plugh", "i")

	require.Contains(t, said, "No he entendido nada",
		"the template's own words for a sentence it cannot parse")
}

// And the whole of what was verified by hand, kept.
//
// Each line of this was a bug at some point in the afternoon it was written,
// which is the only reason to spell out something so ordinary: walking into a
// building, walking out of it again, and being told why a direction is not a
// direction.
func TestWalkingAroundLaAventuraOriginal(t *testing.T) {
	source := aoSource(t)

	said := transcript(t, source,
		"este",    // into the cabin
		"salidas", // which way out
		"salir",   // and out again, which used to look like the game ending
		"salidas",
		"oeste", // nothing that way: the forest goes north, south, east and in
		"i",
	)

	require.Contains(t, said, "habitación bastante guarra",
		"east from the forest is the cabin")
	require.Contains(t, said, "Oeste, Bajar y Salir",
		"the cabin's exits, which is the response table building a sentence")
	require.Contains(t, said, "Hacia el Sur fluye un rio",
		"and out again: leaving the cabin is a move, not the end of the game")
	require.Contains(t, said, "No puedes",
		"a direction that is not one is refused rather than taken")
	require.Contains(t, said, "No tienes nada", "the inventory, which starts empty")

	// Nothing in that run should have reached a condact we have not written.
	require.NotContains(t, said, "not implemented")
}

// templateSource is the blank adventure DRC ships, converted to the UTF-8 the
// front end reads.
func templateSource(t *testing.T) string {
	t.Helper()

	return utf8Copy(t, "../../work/DRC/BLANK_ES.DSF")
}

// aoSource is La Aventura Original, decompiled and converted for version 3.
// It is a directory of files rather than one, so what is copied is all of it.
func aoSource(t *testing.T) string {
	t.Helper()

	const dir = "../../work/port/ao-zx48"

	if _, err := os.Stat(dir + "/game.sce"); err != nil {
		t.Skipf("%s is not in the checkout: %v", dir, err)
	}

	return dir + "/game.sce"
}

// utf8Copy writes a source the kit ships in ISO-8859-1 somewhere readable.
func utf8Copy(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("%s is not in the checkout: %v", path, err)
	}

	runes := make([]rune, 0, len(raw))
	for _, b := range raw {
		runes = append(runes, rune(b))
	}

	out := t.TempDir() + "/source.dsf"
	require.NoError(t, os.WriteFile(out, []byte(string(runes)), 0o600))

	return out
}
