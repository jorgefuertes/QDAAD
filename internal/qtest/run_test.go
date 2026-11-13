package qtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/jorgefuertes/QDAAD/internal/vm"
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
  EXPECT "la biblioteca de Alejandría", 1
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

// The clock, and who it is that runs the turn.
//
// This is TestTheClockRunningOutSaysSo written in the language: the player says
// nothing, the tables run on without an order, and the adventure answers with
// the words it has for that rather than with "I do not understand".
//
// The waiting is real, and that is the point of it. The script asks for two
// seconds of silence; the adventure's own timeout is shorter, so it fires
// while the script is still waiting and the runtime runs its own turn for its
// own reasons. Nobody told it anything untrue about the time.
func TestTheClockMakesTheAdventureSpeak(t *testing.T) {
	said, err := play(t, hurried(t), past+`
STEP "the time passes, and does not come back"
  EXPECT "Estás en el bosque"
  WAIT 2
  EXPECT "El tiempo pasa"
  EXPECT NOT "no te entiendo"
END
`)

	require.NoError(t, err, "the clock should have run and the adventure should have said so:\n%s", said)
}

// And an EXPECT that gives up before the adventure's clock does leaves the game
// exactly where it was.
//
// This is the Schrödinger half: a harness that reported a timeout without the
// time having passed would make the adventure run a turn it would never have
// run, and the only sign of it is a line in the transcript that a player would
// never have seen. One second of silence is one second: the adventure gives
// the player forty-seven, so nothing of its own happens in that time.
func TestGivingUpFirstLeavesTheGameAlone(t *testing.T) {
	said, err := play(t, ao(t), past+`
STEP "a second of silence, and no turn of the game"
  EXPECT "Estás en el bosque"
  EXPECT "a dwarf walks in", 1
END
`)

	require.Error(t, err)

	require.Contains(t, said, "it did not say that in 1 second",
		"the seconds reported are the ones the script asked for")

	require.NotContains(t, said, "El tiempo pasa",
		"the adventure's clock never ran out, so it had nothing to say about time")
}

// And TIMEOUT is how a whole script says how patient it is.
//
// The waiting is real, so the number is a bill. A script being written and run
// over and over wants it small; one about an adventure that takes its time
// wants it large. Saying it once at the top beats saying it on every line.
func TestATimeoutSetsWhatEveryExpectWaits(t *testing.T) {
	said, err := play(t, ao(t), past+`
TIMEOUT 1

STEP "one second, because the script said so"
  EXPECT "Estás en el bosque"
  EXPECT "a dwarf walks in"
END
`)

	require.Error(t, err)
	require.Contains(t, said, "it did not say that in 1 second",
		"the seconds are the script's, and the report says which they were")
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
  EXPECT "AVENTURA ORIGINAL", 1
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
//
// The -extended pair is the same two walkthroughs against the +3_Extended
// tree, which carries the 16-bit editions' long texts. Same moves, and the
// EXPECT lines that quote a text quote the long one.
func TestTheExampleScriptsPlay(t *testing.T) {
	for _, c := range []struct{ tree, part, script string }{
		{"+3", "part1", "part1.qtest"},
		{"+3", "part2", "part2.qtest"},
		{"+3_Extended", "part1", "part1-extended.qtest"},
		{"+3_Extended", "part2", "part2-extended.qtest"},
	} {
		t.Run(c.tree+"/"+c.part, func(t *testing.T) {
			db := database(t, filepath.Join("..", "..", "examples", "AO", "ZXSpectrum", c.tree,
				c.part, "game.sce"))

			script, err := Parse(filepath.Join("..", "..", "examples", "AO", "test", c.script))
			require.NoError(t, err)

			var report strings.Builder

			require.NoError(t, Play(db, script, Options{Report: NewReport(&report)}),
				"the example script did not play:\n%s", report.String())
		})
	}
}

// The xmessages play, which is the one thing a comparison of bytes cannot say.
//
// compare-drb holds this same source's database and .XMB against the reference
// byte for byte, and that is worth a great deal and is not this: a text whose
// offset is one byte out is still a file identical to nobody's, and what says
// whether the right text comes out is reading it back the way the interpreter
// does. So the script plays the adventure and asks for the words.
func TestTheXMessagesScriptPlays(t *testing.T) {
	const dir = "../../script/sources"

	db, xmb := databaseFor(t, filepath.Join(dir, "xmessages.sce"), "VGA256")
	require.NotEmpty(t, xmb, "the source is the one that keeps its texts outside")

	script, err := Parse(filepath.Join(dir, "xmessages.qtest"))
	require.NoError(t, err)

	var report strings.Builder

	require.NoError(t, Play(db, script, Options{Report: NewReport(&report), XMessages: xmb}),
		"the xmessages did not play:\n%s", report.String())
}

// And without the file, an XMES says so rather than printing nothing.
//
// This is the failure worth being loud about. A database names offsets into a
// file nobody handed over, so every xmessage in the adventure is a line that
// does not appear — and an adventure missing a third of its text while
// answering every command looks like an adventure, not like a mistake.
func TestAnXMessageWithNoFileSaysSo(t *testing.T) {
	db, xmb := databaseFor(t, "../../script/sources/xmessages.sce", "VGA256")
	require.NotEmpty(t, xmb)

	script, err := ParseText("no file", `
STEP "it stops instead of printing nothing"
  INPUT leer carta
END
`)
	require.NoError(t, err)

	var report strings.Builder

	// The same adventure, and this time nothing is handed over.
	require.Error(t, Play(db, script, Options{Report: NewReport(&report)}))
	require.Contains(t, report.String(), "XMES")
	require.Contains(t, report.String(), "none was given")
}

// A BREAKPOINT hands the adventure to whoever is at the terminal, and Resume
// is the one line that is not for the adventure.
//
// What the person does is theirs: the script does not advance a step however
// many turns they take, and what they are told goes to their console and not
// into the report. Then the line gives it back and the script carries on from
// the step after the breakpoint, which is the whole point of stopping there.
func TestABreakpointHandsTheAdventureOver(t *testing.T) {
	person := &person{lines: []string{"mirar", "salidas", "continue test"}}

	said, err := playWith(t, ao(t), person, past+`
STEP "the forest"
  EXPECT "Estás en el bosque"
END

BREAKPOINT

STEP "and the script picks up where it left off"
  INPUT este
  EXPECT "habitación bastante guarra"
  ASSERT AT = 10
END
`)

	require.NoError(t, err, "both steps should have held:\n%s", said)

	require.Equal(t, 3, person.asked, "the person was asked for every line they had")

	require.Contains(t, person.heard.String(), "Las salidas posibles",
		"what they asked for was answered to them")
	require.NotContains(t, said, "Las salidas posibles",
		"and not into the report, which is about the steps")

	require.Contains(t, said, "BREAKPOINT")
	require.Contains(t, said, "the script has the adventure again")

	// Resume is read by the harness and never handed on. If the adventure had
	// been given it, it would have answered that it did not understand.
	require.NotContains(t, person.heard.String(), "no te entiendo",
		"CONTINUE TEST is a word of the language and not a command of the game")
}

// Sitting at a breakpoint and saying nothing is not typing an empty line.
//
// The adventure gave them TIME seconds and they let them go by, which is a
// thing it has words for: "El tiempo pasa... ¡Y no vuelve!", its system
// message 35, reached by asking HASAT 87 — flag 49 bit 7, the one the
// interpreter raises when a wait runs out.
//
// The expiry was being thrown away on its way through: the breakpoint's
// console reported it, the harness turned it into an empty line and told the
// machine nobody had timed out. So the adventure answered "Lo siento, no
// entiendo", which is what it says to somebody who pressed return with nothing
// typed — and nobody had pressed anything at all. It is the same lie the clock
// and the pauses were fixed for, pointing the other way: the console decides
// when a wait is over, and what it decides has to arrive whole.
//
// What hid it is in this file rather than in the harness. A person double with
// no InputWithin is a player who can never sit still, so every breakpoint test
// typed at once and the clock was never carried through one.
func TestLettingTheClockRunOutAtABreakpointIsNotAnEmptyLine(t *testing.T) {
	person := &person{silences: 2, lines: []string{"continue test"}}

	said, err := playWith(t, ao(t), person, past+`
STEP "the forest"
  EXPECT "Estás en el bosque"
END

BREAKPOINT

STEP "and the script picks up afterwards"
  INPUT este
  EXPECT "habitación bastante guarra"
END
`)

	require.NoError(t, err, "both steps should have held:\n%s", said)

	heard := person.heard.String()

	require.Equal(t, 2, strings.Count(heard, "El tiempo pasa"),
		"they sat out two of the adventure's own timeouts and it has words for both")

	require.NotContains(t, heard, "no te entiendo",
		"which is what it says to an empty line, and nothing was typed until the last one")
}

// And a script with one left in it still runs where there is nobody to hand it
// to, which is what a suite is.
func TestABreakpointWithNobodyThereSaysSoAndCarriesOn(t *testing.T) {
	said, err := play(t, ao(t), past+`
BREAKPOINT

STEP "the forest, reached anyway"
  EXPECT "Estás en el bosque"
END
`)

	require.NoError(t, err, "the step should have held:\n%s", said)
	require.Contains(t, said, "nobody is at a terminal")
}

// Walking away from a breakpoint ends the run, the same as walking away from
// the adventure does.
//
// Nothing hangs and nothing is invented: the person's input runs out, which is
// the end of the game for them and for the script. The steps after the
// breakpoint never ran, so the summary says so rather than counting them as
// anything.
func TestWalkingAwayFromABreakpointEndsIt(t *testing.T) {
	person := &person{lines: []string{"mirar"}}

	said, err := playWith(t, ao(t), person, past+`
STEP "the forest"
  EXPECT "Estás en el bosque"
END

BREAKPOINT

STEP "never reached"
  EXPECT "a dwarf walks in"
END
`)

	require.NoError(t, err, "nothing failed; the run simply ended:\n%s", said)
	require.Contains(t, said, "1 step passed")
	require.NotContains(t, said, "FAIL")
}

// person is somebody at a terminal during a breakpoint.
type person struct {
	lines []string
	heard strings.Builder
	asked int

	// silences is how many times they let the adventure's clock run out before
	// starting to type. A console is the one thing allowed to decide that a
	// wait is over, and this is a console.
	silences int
}

func (p *person) Print(text string) { p.heard.WriteString(text) }
func (p *person) AnyKey()           {}

func (p *person) Input() (string, bool) {
	p.asked++

	if len(p.lines) == 0 {
		return "", false
	}

	line := p.lines[0]
	p.lines = p.lines[1:]

	return line, true
}

// InputWithin is the half of a person the breakpoint tests used to leave out,
// and leaving it out is why the clock could be dropped there without anything
// noticing: a double with no InputWithin is a player who can never sit still.
func (p *person) InputWithin(time.Duration) (line string, more, expired bool) {
	if p.silences > 0 {
		p.silences--
		p.asked++

		return "", true, true
	}

	line, more = p.Input()

	return line, more, false
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

// playWith is the same with somebody at the terminal for a BREAKPOINT to hand
// the adventure to.
func playWith(t *testing.T, db *ddbfile.Reader, who vm.Console, text string) (string, error) {
	t.Helper()

	script, err := ParseText("t.qtest", text)
	require.NoError(t, err)

	var report strings.Builder

	err = Play(db, script, Options{Report: NewReport(&report), Breakpoint: who})

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

// hurried is La Aventura Original with its clock turned down to one second.
//
// A timeout can only be tested by being timed out, and the waiting is real: the
// adventure's own forty-seven seconds would be forty-seven seconds of the
// suite. So the fixture is a different adventure, not a different machine —
// the same database in every other respect, handed to the same interpreter,
// which gives the player a second instead of most of a minute.
//
// The sources are copied and gfx/ is linked rather than copied, which is what
// the comparison harnesses in script/ had to learn too: the compiler wants the
// charset and the pictures beside the source, and the pictures are large.
func hurried(t *testing.T) *ddbfile.Reader {
	t.Helper()

	from := filepath.Join("..", "..", "examples", "AO", "ZXSpectrum", "+3", "part1")
	dir := t.TempDir()

	sources, err := filepath.Glob(filepath.Join(from, "*.sce"))
	require.NoError(t, err)
	require.NotEmpty(t, sources, "the adventure's sources are not where this expects them")

	for _, path := range sources {
		body, err := os.ReadFile(path)
		require.NoError(t, err)

		if filepath.Base(path) == "processes.sce" {
			hurry := strings.Replace(string(body), "TIME 47 1", "TIME 1 1", 1)
			require.NotEqual(t, string(body), hurry,
				"the adventure no longer gives the player 47 seconds, so this fixture is out of date")

			body = []byte(hurry)
		}

		require.NoError(t, os.WriteFile(filepath.Join(dir, filepath.Base(path)), body, 0o600))
	}

	gfx, err := filepath.Abs(filepath.Join(from, "gfx"))
	require.NoError(t, err)
	require.NoError(t, os.Symlink(gfx, filepath.Join(dir, "gfx")))

	return database(t, filepath.Join(dir, "game.sce"))
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

	db, _ := databaseFor(t, source, "VGA")

	return db
}

// databaseFor is the same with the subtarget asked for, and with the .XMB the
// link produced.
//
// The subtarget is a parameter because of the xmessages: a PC that is not
// VGA256 has nowhere to read them from, so an adventure with any is refused
// for it. See internal/back/xmessages.go.
func databaseFor(t *testing.T, source, subtarget string) (*ddbfile.Reader, []byte) {
	t.Helper()

	opts := legacy.Options{
		Machine: arch.PC, Subtarget: subtarget,
		Language: language.LanguageSpanish, Version: 3,
	}

	db, err := legacy.Analyze(source, opts)
	require.NoError(t, err)

	image, layout, err := back.LinkWithLayout(db, back.Options{
		Machine: opts.Machine, Subtarget: opts.Subtarget,
		Language: opts.Language, Version: opts.Version,
	})
	require.NoError(t, err)

	reader, ok := ddbfile.At(image, 0)
	require.True(t, ok)

	return reader, layout.XMessages
}
