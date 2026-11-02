package vm_test

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

// transcript is everything the adventure said, which is what a test reads.
func (s *script) transcript() string { return s.said.String() }

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
// everything it said. It builds for a PC, which is the plainest layout there
// is: laid at zero, little-endian, sections aligned to even addresses.
func transcript(t *testing.T, source string, commands ...string) string {
	t.Helper()

	return transcriptFor(t, source, arch.PC, "VGA", commands...)
}

// transcriptFor is the same for a named machine, because the layout is not a
// detail the interpreter is supposed to notice and the only way to know it
// does not is to play the same adventure on two of them.
//
// A Spectrum is the one that matters: it is laid at 0x8400 instead of zero and
// packs its sections tight instead of aligning them, and it is the build that
// goes on a tape. Playing only the PC build would leave the published one
// resting on the roundtrip tests, which read every section but answer nothing
// about whether the adventure can be won.
func transcriptFor(t *testing.T, source string, target arch.Machine, subtarget string,
	commands ...string,
) string {
	t.Helper()

	opts := legacy.Options{
		Machine: target, Subtarget: subtarget,
		Language: language.LanguageSpanish, Version: 3,
	}

	db, err := legacy.Analyze(source, opts)
	require.NoError(t, err)

	image, layout, err := back.LinkWithLayout(db, back.Options{
		Machine: opts.Machine, Subtarget: opts.Subtarget,
		Language: opts.Language, Version: opts.Version,
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

// playWith is the same, for a player that is not a plain list of commands.
func playWith(t *testing.T, source string, player interface {
	vm.Console
	transcript() string
},
) string {
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

	r, ok := ddbfile.At(image, 0)
	require.True(t, ok)

	machine, err := vm.New(r, player, nil)
	require.NoError(t, err)
	require.NoError(t, machine.Run())

	return player.transcript()
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

// And the whole of the first part, played to the end.
//
// This is the test that is worth more than all the others together, because
// finishing an adventure exercises the machine the way nothing smaller does:
// 84 commands, every kind of condact, a conversation, a puzzle chain with four
// stages and an ending that will not come unless all of it worked. The access
// code it prints is AO's own answer, so there is no judging what counts as
// success — either the adventure says TIMACUS or it does not.
//
// Three bugs hid behind this one transcript and all three were the same shape,
// a machine helping itself to the player's input:
//
//   - AnyKey read a line even with nobody at the keyboard, so every pause ate
//     a command and everything after it happened in the wrong room.
//   - PARSE ignored its operand and always asked for a line. The phrase in
//     quotes is what a conversation parses — that is PARSE 1 — so DECIR
//     "SIGUEME" threw the phrase away and ate the player's next command
//     instead, answering one turn late and attached to the wrong order.
//   - And the carrying limit was the template's guessed 4, which no source
//     should be guessing: La Aventura Original says ABILITY 5 255 in its own
//     initialisation. The cave then asks for EQ 1 5 — exactly five objects in
//     hand — which is unreachable if anything has overridden the adventure's
//     own answer.
//
// It is played on both layouts. The Spectrum build is the one that goes on a
// tape, and until this ran twice the published one rested on tests that read
// every section of it and answered nothing about whether it could be won.
func TestFinishingTheFirstPartOfLaAventuraOriginal(t *testing.T) {
	for _, target := range []struct {
		name      string
		machine   arch.Machine
		subtarget string
	}{
		{"a 48K Spectrum, laid at 0x8400 and packed tight", arch.ZX, "48K"},
		{"a PC, laid at zero and word-aligned", arch.PC, "VGA"},
	} {
		t.Run(target.name, func(t *testing.T) {
			finishTheFirstPart(t, target.machine, target.subtarget)
		})
	}
}

func finishTheFirstPart(t *testing.T, machine arch.Machine, subtarget string) {
	t.Helper()

	source := aoSource(t)

	said := transcriptFor(t, source, machine, subtarget,
		// The bottle, down the valley and back.
		"S", "S", "S", "S", "COGER BOTELLA", "N", "N", "N",
		// The boots, east and south.
		"E", "E", "S", "S", "S", "S", "COGER BOTAS",
		"N", "N", "N", "N", "O", "O", "N", "N",
		// The battery, and the lamp Don Enano gives for the boots.
		"E", "E", "S", "COGER PILA", "N", "N",
		"DAR BOTAS", "COGER LINTERNA", "PONER PILAS", "ENCENDER LINTERNA",
		// The key, out of the tree.
		"S", "O", "O", "O", "S", "S", "S", "E", "E", "SE", "S",
		"SUBIR ARBOL", "EXAMINAR RAMAS", "SACUDIR RAMAS", "BAJAR ARBOL",
		"COGER LLAVE",
		// The coin, in the pool.
		"N", "S", "O", "N", "E", "BAJAR", "EXAMINAR CHARCO", "COGER MONEDA",
		"SUBIR", "O", "S", "LLENAR BOTELLA",
		// The elf, talked to the way the game's own instructions say: a verb
		// and a phrase in quotes, which PARSE 1 is what reads.
		"E", "E", "S", "EXAMINAR FIGURA",
		`DECIR "QUE TE PASA?"`,
		`DECIR "SIGUEME"`,
		// The grating, with the elf following.
		"N", "O", "O", "S", "S", "ABRIR CANDADO", "COGER CANDADO", "EMPUJAR REJA",
		// The tortilla it leaves in the picnic area, which is the fifth object.
		"N", "N", "E", "COGER TORTILLA", "O", "S", "S", "I",
		"ENTRAR CAVERNA",
	)

	require.Contains(t, said, "TIMACUS",
		"the access code to the second part, which only prints when the first is won")

	// The five the game's own note says to arrive with, and the five the cave
	// counts.
	for _, thing := range []string{
		"linterna encendida", "moneda", "tortilla", "botella con agua", "llave",
	} {
		require.Contains(t, said, thing, "missing from the inventory at the end")
	}

	require.NotContains(t, said, "not implemented",
		"the whole of the first part and not one condact we have yet to write")
}

// Ending the game, which has a right way round and had the wrong one.
//
// "FIN" asks whether you are sure, and then END asks whether you want another
// game. Saying no to the second is the end of the session: on a machine of the
// period it dropped you back to BASIC, and here the process exits. Saying yes
// is a fresh game, which is the other thing those machines did.
//
// It was inverted, and the reason is worth keeping: the two questions are held
// against different words. QUIT compares the reply with system message 30, the
// adventure's own letter for yes; END compares it with message 31, its letter
// for no. Treating the second as the negation of the first makes confirming
// that you have had enough start a new game instead of giving the terminal
// back. See do_END, daad_condacts.c:2340.
func TestSayingYouHaveHadEnoughEndsIt(t *testing.T) {
	source := aoSource(t)

	t.Run("no to another game stops", func(t *testing.T) {
		// The "mirar" is the trap: if it is answered, the game did not stop.
		said := transcript(t, source, "fin", "s", "n", "mirar")

		require.Contains(t, said, "¿Hace otra partidita?", "it never got that far")
		require.Equal(t, 1, strings.Count(said, "Estás en el bosque"),
			"the forest was described twice, so a new game started after the player declined")
	})

	t.Run("yes to another game starts one", func(t *testing.T) {
		said := transcript(t, source, "fin", "s", "s", "mirar")

		require.Greater(t, strings.Count(said, "Estás en el bosque"), 1,
			"the game did not begin again")
	})
}

// The second part, which is the one the first part's ending is a ticket to.
//
// It opens by asking for an access code, and that one question is why it sat
// untested for so long: it is the only thing in the corpus that uses INPUT,
// and INPUT does not read anything. It names the window the player's next line
// comes from and sets three option bits — the name promises a prompt and what
// it does is set two flags — so an interpreter that has not written it stops
// dead on the title screen of every part two there is.
//
// TIMACUS is what the first part prints when it is won, which makes this the
// two halves of the adventure meeting: the code is not ours and not the
// test's, it is what TestFinishingTheFirstPartOfLaAventuraOriginal watches for.
func TestTheSecondPartOfLaAventuraOriginalTakesItsCode(t *testing.T) {
	source := ao2Source(t)

	said := transcript(t, source, "TIMACUS", "mirar", "salidas")

	require.Contains(t, said, "Clave de acceso",
		"it never asked, so INPUT is not reaching the question")
	require.Contains(t, said, "la puerta se cierra",
		"the code was not accepted and the adventure did not begin")
	require.Contains(t, said, "Los rayos del sol",
		"the first location was never described")

	require.NotContains(t, said, "not implemented")
}

// impatient is a player who sometimes says nothing at all.
//
// A line of "..." is the clock running out rather than an order: the console
// says the wait expired and the machine goes on without a sentence. It is the
// smallest thing that makes a timeout testable, and it is what a test language
// for qrun would end up needing underneath.
type impatient struct {
	script

	waited int
}

const letTheClockRun = "..."

func (p *impatient) InputWithin(time.Duration) (line string, more, expired bool) {
	if len(p.lines) > 0 && p.lines[0] == letTheClockRun {
		p.lines = p.lines[1:]
		p.waited++

		return "", true, true
	}

	line, more = p.Input()

	return line, more, false
}

// When the clock runs out the adventure says so, in its own words.
//
// This is the whole of what TIME is for: the player says nothing, the tables
// run on without an order, and the loop's own table asks whether it was the
// clock — HASAT 87, flag 49 bit 7 — and answers with system message 35 rather
// than the one for a sentence it could not parse.
//
// It was asking with HASAT 49, which by the same arithmetic is flag 53 bit 1
// and never set when a clock runs out, so La Aventura Original answered "Lo
// siento, no entiendo" where it had "El tiempo pasa... ¡Y no vuelve!" written.
// Nothing in a transcript showed it, because nothing in a transcript waits.
func TestTheClockRunningOutSaysSo(t *testing.T) {
	source := aoSource(t)

	player := &impatient{script: script{lines: []string{letTheClockRun, letTheClockRun, "fin", "s"}}}

	said := playWith(t, source, player)

	require.Equal(t, 2, player.waited, "the console was never asked to wait")
	require.Contains(t, said, "El tiempo pasa",
		"the adventure has the words and the loop did not ask for them")
	require.NotContains(t, said, "no te entiendo",
		"a clock running out is not a sentence it failed to parse")
}

// And once inside, the first thing a player does is look at what the first
// part left them holding.
//
// Two things had to be right for this to work and neither was. The adventure
// answers INVENTARIO with "> I _ PROCESS 10", and I is a noun of value 14, so
// the word only reaches that entry if a verbless sentence promotes its noun to
// the verb — which the manual says happens below twenty and we were doing
// below fourteen. Nothing at all was printed.
//
// Then, with the entry reached, it tests NOTZERO 1, and flag 1 is how many
// objects are carried. RESET put the objects back where the database starts
// them and left the count alone, so the answer was "no llevas nada" with the
// lamp, a key, the food and the water in hand.
func TestTheSecondPartKnowsWhatTheFirstOneHandedOver(t *testing.T) {
	source := ao2Source(t)

	said := transcript(t, source, "TIMACUS", "i", "dejar moneda", "i")

	require.Contains(t, said, "Una llave",
		"the first inventory of the game is the one that was empty")
	require.NotContains(t, said, "No tienes nada")

	// And what is put down leaves it, which is the other half of the count
	// being real rather than a number somebody wrote once.
	_, after, found := strings.Cut(said, "Has dejado la moneda")
	require.True(t, found, "the coin was carried and could be dropped")
	require.NotContains(t, after, "Una moneda")
}

// Every machine we compile for, played.
//
// This is the promise qrun is for: an adventure built for any target can be
// played and debugged here, without a tape, a disk or an emulator. The layouts
// disagree about everything that matters to a reader — where the database is
// laid, which way round its words go, whether sections are padded to even
// addresses — so the only way to know none of that reaches the interpreter is
// to build the same adventure for all of them and play each one.
//
// It goes through MachineNames rather than a list written here, so that adding
// a target to the compiler adds it to this test too and a new one cannot
// arrive unplayed.
func TestEveryMachineWeCompileForPlays(t *testing.T) {
	source := templateSource(t)

	for _, name := range arch.MachineNames() {
		machine, err := arch.Parse(strings.ToLower(name))
		require.NoError(t, err)

		t.Run(name, func(t *testing.T) {
			said := transcriptFor(t, source, machine, "", "mirar", "i")

			require.Contains(t, said, "mazmorra",
				"the location the template starts in, so the loop ran and the text was read")
			require.NotContains(t, said, "not implemented",
				"a condact we have yet to write")
		})
	}
}

// ao2Source is the second part, decompiled and converted the same way as the
// first.
func ao2Source(t *testing.T) string {
	t.Helper()

	return examplePart(t, "part2")
}

// templateSource is the blank adventure DRC ships, converted to the UTF-8 the
// front end reads.
func templateSource(t *testing.T) string {
	t.Helper()

	return utf8Copy(t, "../../work/DRC/BLANK_ES.DSF")
}

// aoSource is La Aventura Original, decompiled and converted for version 3 by
// the decompiler itself — "qundaad decompile --v3" on the original tape and
// nothing else. The whole conversion lives in the model now, so there is no
// pipeline of text substitutions between the tape and this.
func aoSource(t *testing.T) string {
	t.Helper()

	return examplePart(t, "part1")
}

// examplePart is one part of the ported adventure as the repository carries
// it, in examples.
//
// It used to be work/port, which is not in the repository, so these tests
// skipped on a fresh checkout and — worse — went on passing against whatever
// an old build had left there. One of them did exactly that: the second part's
// inventory was fixed and the test still failed, because the copy it was
// reading predated the fix by a week.
func examplePart(t *testing.T, part string) string {
	t.Helper()

	return filepath.Join("..", "..", "examples", "AO", "ZXSpectrum", "+3", part, "game.sce")
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
