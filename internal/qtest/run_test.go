package qtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/stretchr/testify/require"
)

// Playing a script, which is the only test of a test language worth much.
//
// The adventure is La Aventura Original, compiled from the source in examples
// the way internal/vm's own play tests do it. A prebuilt database would be
// quicker and it is exactly the mistake those tests record: a copy left in
// work/ goes on passing against a week-old build long after the thing it is
// testing has been fixed.

// past is the three keypresses AO asks for before the game begins: the "?",
// the credits and the page about what the game is.
//
// Every script that is about something else still has to answer them. A pause
// is the adventure's own behaviour and not a thing a test may decide to skip,
// so they are here to be written once rather than to be hidden — the orders go
// in front of the first STEP, which is where anything that is not part of a
// step belongs.
const past = "KEY SPACE\nKEY SPACE\nKEY SPACE\n"

func TestAScriptThatPlays(t *testing.T) {
	said, err := play(t, ao(t), `
TEST "AO: the forest"
SEED 1989

STEP "the title screen and the forest"
  EXPECT "?"
  KEY SPACE
  EXPECT "AVENTURA ORIGINAL"
  KEY SPACE
  EXPECT "LA RESPUESTA DEPENDE DE TU VALOR"
  KEY SPACE
  EXPECT "Estás en el bosque"
  EXPECT NOT "not implemented"
  ASSERT AT = 3
  ASSERT PICTURE = 3
END

STEP "east is the cabin"
  INPUT este
  EXPECT "habitación bastante guarra"
  ASSERT AT = 10
END
`)

	require.NoError(t, err, "every step should have held:\n%s", said)

	require.Equal(t, 2, strings.Count(said, "OK"), "one OK per step and nothing else")
	require.NotContains(t, said, "FAIL")
	require.Contains(t, said, "the title screen and the forest")
}

// A pause waits, because the adventure asked it to.
//
// Letting it through would be the harness deciding that this pause does not
// count, and a pause is the adventure's own behaviour: how many there are and
// where they fall is part of what a test of it is for. So a script with
// nothing to press fails here, and says so.
func TestAPauseWaitsForItsKey(t *testing.T) {
	said, err := play(t, ao(t), `
STEP "a script with nothing to press"
  EXPECT "VAS EN BUSCA DE LA GRAN CAVERNA"
END
`)

	require.Error(t, err, "the title screen pauses, and nothing in the script answers it")
	require.Contains(t, said, "waiting for a keypress",
		"the reason is the useful half of the failure")
}

// And a key given is a key taken, which is how a title screen is stepped
// through.
//
// Three of them, because that is what AO asks for: the "?", the credits and
// the page about what the game is. A test that needed two would be a test that
// had not noticed one.
func TestAKeyIsTakenByAPause(t *testing.T) {
	said, err := play(t, ao(t), `
STEP "three keys at the title"
  EXPECT "?"
  KEY SPACE
  EXPECT "AVENTURA ORIGINAL"
  KEY SPACE
  EXPECT "LA RESPUESTA DEPENDE DE TU VALOR"
  KEY SPACE
  EXPECT "Estás en el bosque"
END
`)

	require.NoError(t, err, "a key offered at a pause should be taken, not typed into a command:\n%s", said)
}

// What a failure says, which is what the whole thing is for.
func TestAFailingStepSaysWhereItIsWritten(t *testing.T) {
	said, err := play(t, ao(t), past+`
TEST "AO"
STEP "one that holds"
  EXPECT "Estás en el bosque"
END

STEP "one that does not"
  INPUT este
  EXPECT "la biblioteca de Alejandría"
END
`)

	require.Error(t, err, "the second step cannot have held")

	require.Contains(t, said, "OK", "the first step held and should say so")
	require.Contains(t, said, "FAIL")
	require.Contains(t, said, "one that does not", "a failure is reported against its step's name")
	require.Contains(t, said, "t.qtest:12:", "and against the line it is written on")

	require.Contains(t, said, "habitación bastante guarra",
		"what the adventure did say is nearly always the answer, so it goes in the report")

	require.Contains(t, said, "1 step passed")
	require.Contains(t, said, "1 step failed")
}

// The clock, which is the reason the seconds in this language are the
// adventure's and not the wall's.
//
// This is TestTheClockRunningOutSaysSo written in the language: the player says
// nothing, the tables run on without an order, and the adventure answers with
// the words it has for that rather than with "I do not understand". Nothing is
// slept through — AO gives 47 seconds a turn and this test takes none of them.
func TestTheClockMakesTheAdventureSpeak(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "the time passes, and does not come back"
  EXPECT "Estás en el bosque"
  WAIT 50
  EXPECT "El tiempo pasa", 100
  EXPECT NOT "no te entiendo"
END
`)

	require.NoError(t, err, "the clock should have run and the adventure should have said so:\n%s", said)
}

// An EXPECT that cannot be met says which kind of cannot it is.
//
// With no TIME running there is nothing to wait for: the adventure is holding
// out for an order and will say nothing until it gets one. Failing at once and
// saying that is worth more than a timeout, which would spend real seconds to
// reach the same answer.
func TestWaitingForWhatWillNeverComeFailsAtOnce(t *testing.T) {
	said, err := play(t, template(t), `
KEY SPACE                       ; the template's own title line pauses too

STEP "nothing is going to happen"
  EXPECT "Has entrado a la mazmorra"
  EXPECT "a dwarf walks in", 30
END
`)

	require.Error(t, err)
	require.Contains(t, said, "no clock running on it",
		"the reason is the useful half of the failure")
}

// EXPECT eats what it matched, so the order things are said in is part of the
// test.
func TestExpectIsAboutOrder(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "the forest comes after the title, not before"
  EXPECT "Estás en el bosque"
  EXPECT "AVENTURA ORIGINAL"
END
`)

	require.Error(t, err, "the title was said first, so by here it has been eaten")
	require.Contains(t, said, "AVENTURA ORIGINAL")
}

// One order is one command: INPUT types and presses return, which is what a
// walkthrough is made of and the only reason it is not twice as long.
//
// The EXPECT is what proves the return: without one the line would still be
// sitting in the keyboard buffer, the adventure would be waiting for it, and
// the cabin would never be described. The turn count is the other half — it
// was sent once rather than not at all.
func TestAnInputIsOneCommand(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "one order, one command"
  EXPECT "Estás en el bosque"
  INPUT  este
  EXPECT "habitación bastante guarra"
  ASSERT AT = 10
  ASSERT TURNS = 1
END
`)

	require.NoError(t, err, "one INPUT should have been one turn:\n%s", said)
}

// And a RETURN that is not needed is not refused either: it hands over an empty
// line and the adventure answers it with the words it keeps for a sentence it
// could not read.
//
// It is written down because the arithmetic is the opposite of what anybody
// would assume. The turn counter does not move: flag 31 is the adventure's own
// bookkeeping, done in its own tables, and a line it did not understand is not
// a turn to it. So a script left with a stray return is not a script a TURNS
// assertion will catch — it is one extra "no te entiendo" in the transcript,
// and the transcript is where to look.
func TestAStrayReturnIsAnEmptyCommand(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "an order and a half"
  EXPECT "Estás en el bosque"
  INPUT  este
  KEY    RETURN
  EXPECT "habitación bastante guarra"
  EXPECT "no te entiendo"
  ASSERT TURNS = 1
END
`)

	require.NoError(t, err, "the empty line should have been answered, and counted as nothing:\n%s", said)
}

// A line can still be built a keypress at a time, which is what answering a
// yes-or-no question is: one letter, then the return that hands it over.
//
// Both halves are tested by it holding. If KEY sent the line itself, the
// adventure would be answering an empty one and would never get to the second
// question.
func TestALineBuiltOutOfKeypresses(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "one letter, then the return"
  EXPECT "Estás en el bosque"
  INPUT  fin
  EXPECT "seguro"
  KEY    S
  KEY    RETURN
  EXPECT "otra partidita"
END
`)

	require.NoError(t, err, "KEY types without sending, and KEY RETURN sends:\n%s", said)
}

// Where things are, which no transcript answers: the adventure prints the
// room's description and never its number.
func TestWhatIsCarriedAndWhereItIs(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "down the valley for the bottle"
  EXPECT "Estás en el bosque"
  INPUT S
  INPUT S
  INPUT S
  INPUT S
  INPUT COGER BOTELLA
  EXPECT "Has cogido la botella"
  ASSERT AT = 28
  ASSERT CARRIED 4
  ASSERT PRESENT 4
  ASSERT FLAG 1 = 1
  ASSERT TURNS > 0
END
`)

	require.NoError(t, err, "the bottle is object 4 and it was picked up:\n%s", said)
}

// And what is not where it is said to be comes back with where it really is,
// because "it is at 254, carried" is the answer and "it is not here" is only
// the question again.
func TestWhereSomethingReallyIsGetsSaid(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "the bottle is not in the forest"
  EXPECT "Estás en el bosque"
  ASSERT CARRIED 4
END
`)

	require.Error(t, err)
	require.Contains(t, said, "object 4 is not carried: it is at 28")
}

// The end of the game, and the two orders that are about it.
func TestTheEndOfTheGame(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "FIN, and no to another game"
  EXPECT "Estás en el bosque"
  INPUT fin
  EXPECT "seguro"
  INPUT s
  EXPECT "otra partidita"
  INPUT n
  ASSERT ENDED
END
`)

	require.NoError(t, err, "the game should have ended and said so:\n%s", said)
}

// And typing into a game that has ended is a failure rather than a hang.
func TestTypingAfterTheEndIsRefused(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "one order too many"
  EXPECT "Estás en el bosque"
  INPUT fin
  EXPECT "seguro"
  INPUT s
  EXPECT "otra partidita"
  INPUT n
  INPUT mirar
END
`)

	require.Error(t, err)
	require.Contains(t, said, "the adventure has ended and will say nothing more")
}

// What DEBUG prints, in the shape something else can read.
func TestWhatDebugSays(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "ask the machine about itself"
  EXPECT "Estás en el bosque"
  DEBUG AT
  DEBUG PICTURE
  DEBUG FLAG 38
  DEBUG FLAGS SET
END
`)

	require.NoError(t, err)

	lines := strings.Split(said, "\n")

	// Each on a line of its own with nothing around it, which is what makes it
	// readable by something other than a person.
	require.Contains(t, lines, "AT=3", "the location")
	require.Contains(t, lines, "PICTURE=3")
	require.Contains(t, lines, "FLAG_38=3")
	require.Contains(t, lines, "FLAG_48=47", "AO gives the player 47 seconds a turn")

	require.NotContains(t, lines, "FLAG_2=0", "SET is only the ones holding something")
}

// The dice can be asked for the same game twice, which is what makes a script
// over an adventure with a CHANCE in it worth running.
func TestTheSameSeedPlaysTheSameGame(t *testing.T) {
	db := ao(t)

	const script = past + `
STEP "walk about a bit"
  EXPECT "Estás en el bosque"
  INPUT S
  INPUT S
  INPUT N
END
`

	first := transcriptOf(t, db, script, 1989)
	again := transcriptOf(t, db, script, 1989)
	other := transcriptOf(t, db, script, 1)

	require.Equal(t, first, again, "the same seed should play the same game")
	require.NotEmpty(t, first)

	// Nothing says a different seed has to differ — it only does when the dice
	// are actually rolled — so this is not asserted, only that it still plays.
	require.NotEmpty(t, other)
}

// The scripts in examples, played as part of the suite.
//
// They are the dogfood and they are also the only thing keeping the language
// honest: part1.qtest is the same walkthrough as
// internal/vm/play_test.go's TestFinishingTheFirstPartOfLaAventuraOriginal,
// written in the language instead of in Go. If the two ever disagree, one of
// them is lying, and running both is the only way to find out which.
func TestTheExampleScriptsPlay(t *testing.T) {
	for _, c := range []struct{ part, script string }{
		{"part1", "part1.qtest"},
		{"part2", "part2.qtest"},
	} {
		t.Run(c.part, func(t *testing.T) {
			db := database(t, filepath.Join("..", "..", "examples", "AO", "ZXSpectrum", "+3",
				c.part, "game.sce"))

			script, err := Parse(filepath.Join("..", "..", "examples", "AO", "test", c.script))
			require.NoError(t, err)

			var report strings.Builder

			require.NoError(t, Play(db, script, Options{Report: NewReport(&report)}),
				"the example script did not play:\n%s", report.String())
		})
	}
}

// play compiles an adventure, reads a script and plays one against the other,
// giving back the report and the verdict.
func play(t *testing.T, db *ddbfile.Reader, text string) (string, error) {
	t.Helper()

	script, err := ParseText("t.qtest", text)
	require.NoError(t, err)

	var report strings.Builder

	err = Play(db, script, Options{Report: NewReport(&report)})

	return report.String(), err
}

// transcriptOf plays a script with a fixed seed and gives back everything the
// adventure said.
func transcriptOf(t *testing.T, db *ddbfile.Reader, text string, seed int64) string {
	t.Helper()

	script, err := ParseText("t.qtest", text)
	require.NoError(t, err)

	var said strings.Builder

	require.NoError(t, Play(db, script, Options{Transcript: &said, Seed: seed, HasSeed: true}))

	return said.String()
}

// ao is La Aventura Original as the repository carries it, built for a PC,
// which is the plainest layout there is.
func ao(t *testing.T) *ddbfile.Reader {
	t.Helper()

	return database(t, filepath.Join("..", "..", "examples", "AO", "ZXSpectrum", "+3", "part1", "game.sce"))
}

// template is the blank adventure DRC ships, which is the one to hand with no
// TIME in it.
func template(t *testing.T) *ddbfile.Reader {
	t.Helper()

	path := filepath.Join("..", "..", "work", "DRC", "BLANK_ES.DSF")

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("%s is not in the checkout: %v", path, err)
	}

	// The kit ships it in ISO-8859-1 and the front end reads UTF-8.
	runes := make([]rune, 0, len(raw))
	for _, b := range raw {
		runes = append(runes, rune(b))
	}

	out := filepath.Join(t.TempDir(), "source.dsf")
	require.NoError(t, os.WriteFile(out, []byte(string(runes)), 0o600))

	return database(t, out)
}

// database compiles a source and hands back what the machine reads.
func database(t *testing.T, source string) *ddbfile.Reader {
	t.Helper()

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

	reader, ok := ddbfile.At(image, 0)
	require.True(t, ok)

	return reader
}
