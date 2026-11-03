package decompiler

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"

	"github.com/stretchr/testify/require"
)

// The tables move because version 3 keeps its game loop in four of them and
// version 1 kept it in the interpreter. The first three of the old ones have a
// meaning and go where version 3 keeps that meaning; the rest only have to get
// out of the way.
func TestTheTablesMoveToWhereVersionThreeKeepsThem(t *testing.T) {
	require.Equal(t, 5, movedTable(0), "the response table")
	require.Equal(t, 3, movedTable(1), "after the location description")
	require.Equal(t, 4, movedTable(2), "once a turn")

	for old := 3; old < 200; old++ {
		require.Equal(t, old+4, movedTable(old), "a subprocess only steps aside")
	}

	// And nothing lands on a table the loop wants, which is the whole point.
	for old := range 200 {
		require.NotContains(t, v3Reserved, movedTable(old), "table %d", old)
	}
}

// A PROCESS names a table by number, so the numbers inside the condacts have to
// move with the tables. Getting this wrong is quiet: the number is still a
// valid table, just not the right one.
func TestEveryProcessCallFollowsItsTable(t *testing.T) {
	c := &v3Conversion{tables: 21, sysmess: map[int]int{}}

	got := c.convertCondacts([]ddbfile.Condact{
		{Opcode: opPROCESS, Params: []byte{0}},
		{Opcode: opPROCESS, Params: []byte{2}},
		{Opcode: opPROCESS, Params: []byte{13}},
	})

	require.Equal(t, []ddbfile.Condact{
		{Opcode: opPROCESS, Params: []byte{5}},
		{Opcode: opPROCESS, Params: []byte{4}},
		{Opcode: opPROCESS, Params: []byte{17}},
	}, got)

	// An indirect call names a flag and what it holds is not ours to touch.
	got = c.convertCondacts([]ddbfile.Condact{{Opcode: opPROCESS, Indirect: true, Params: []byte{7}}})
	require.Equal(t, byte(7), got[0].Params[0])
}

// The bare DESC of version 1 described the player's location and ended the turn
// there. Version 3's only describes, and the entry falls into the next one.
//
// The version 3 way of saying the old thing is not DESC with a DONE after it,
// which prints the description a step too early: it is RESTART, because RESTART
// restarts table 0 and table 0 is what describes.
func TestTheBareDescBecomesARestart(t *testing.T) {
	c := &v3Conversion{tables: 21, sysmess: map[int]int{}}

	got := c.convertCondacts([]ddbfile.Condact{{Opcode: opDESC}})

	require.Equal(t, []ddbfile.Condact{{Opcode: opCLS}, {Opcode: opRESTART}}, got)

	// A DESC that names a location is version 3's own and is left alone.
	got = c.convertCondacts([]ddbfile.Condact{{Opcode: opDESC, Params: []byte{12}}})
	require.Equal(t, []ddbfile.Condact{{Opcode: opDESC, Params: []byte{12}}}, got)
}

// QUIT ended the game in DAAD 1. In version 3 it only asks, and END is what
// ends it — so an adventure of 1989 has no END anywhere and confirming "quit"
// does nothing at all.
func TestQuitGetsTheEndItNeverNeeded(t *testing.T) {
	c := &v3Conversion{tables: 21, sysmess: map[int]int{}}

	require.Equal(t,
		[]ddbfile.Condact{{Opcode: opQUIT}, {Opcode: opEND}},
		c.convertCondacts([]ddbfile.Condact{{Opcode: opQUIT}}))

	// One that already does the version 3 thing is left as it is, so that
	// converting something twice is not a different answer.
	already := []ddbfile.Condact{{Opcode: opQUIT}, {Opcode: opEND}}
	require.Equal(t, already, c.convertCondacts(already))

	// And the shape both adventures of 1989 really have, which is QUIT with
	// the endgame report after it. The report has to print before END, because
	// END ends the game and nothing after it runs.
	withReport := &v3Conversion{tables: 22, report: 21, sysmess: map[int]int{}}

	require.Equal(t, []ddbfile.Condact{
		{Opcode: opQUIT},
		{Opcode: opPROCESS, Params: []byte{21}},
		{Opcode: opEND},
	}, withReport.convertCondacts([]ddbfile.Condact{
		{Opcode: opQUIT},
		{Opcode: opDPRINT, Params: []byte{21}},
	}))
}

// PICTURE drew in DAAD 1. In version 3 it only gets the image ready and
// DISPLAY 0 draws it, so a database of 1989 asks for pictures nobody paints.
func TestEveryPictureGetsTheDisplayThatDrawsIt(t *testing.T) {
	c := &v3Conversion{tables: 21, sysmess: map[int]int{}}

	got := c.convertCondacts([]ddbfile.Condact{{Opcode: opPICTURE, Params: []byte{44}}})

	require.Equal(t, []ddbfile.Condact{
		{Opcode: opPICTURE, Params: []byte{44}},
		{Opcode: opDISPLAY, Params: []byte{0}},
	}, got)

	// One that already draws is left as it is, so that converting something
	// twice is not a different answer.
	already := []ddbfile.Condact{
		{Opcode: opPICTURE, Params: []byte{44}},
		{Opcode: opDISPLAY, Params: []byte{0}},
	}
	require.Equal(t, already, c.convertCondacts(already))
}

// La Aventura Original and El Jabato both print the pair 21/22 at the end of
// the game and neither ever writes it, so what reached the screen was a zero.
// What it meant is the endgame report, which the version 1 interpreter printed
// by itself: the turn count, in a sentence.
func TestADeadDPRINTGetsTheTurnCounter(t *testing.T) {
	dprint21 := []ddbfile.Condact{{Opcode: opDPRINT, Params: []byte{21}}}

	t.Run("a pair nobody fills becomes the report", func(t *testing.T) {
		c := &v3Conversion{tables: 22, report: 21, sysmess: map[int]int{}}

		require.Equal(t,
			[]ddbfile.Condact{{Opcode: opPROCESS, Params: []byte{21}}},
			c.convertCondacts(dprint21))
	})

	// The report itself: the count, and the noun with its plural or without.
	t.Run("the report counts in Spanish", func(t *testing.T) {
		report := endgameReport()
		require.Len(t, report, 3)

		require.Equal(t, []ddbfile.Condact{
			{Opcode: opSYSMESS, Params: []byte{sysTurnsBefore}},
			{Opcode: opDPRINT, Params: []byte{turnCounter}},
		}, report[0].Condacts)

		// One turn is both halves of the pair saying so, and it skips the
		// entry that would have added the "es".
		require.Equal(t, []ddbfile.Condact{
			{Opcode: opEQ, Params: []byte{turnCounter + 1, 0}},
			{Opcode: opEQ, Params: []byte{turnCounter, 1}},
			{Opcode: opSYSMESS, Params: []byte{sysTurnsNoun}},
			{Opcode: opSYSMESS, Params: []byte{sysTurnsAfter}},
			{Opcode: opSKIP, Params: []byte{1}},
		}, report[1].Condacts)

		require.Equal(t, []ddbfile.Condact{
			{Opcode: opSYSMESS, Params: []byte{sysTurnsNoun}},
			{Opcode: opSYSMESS, Params: []byte{sysTurnsPlural}},
			{Opcode: opSYSMESS, Params: []byte{sysTurnsAfter}},
		}, report[2].Condacts)
	})

	// The guard, which is the whole reason this is not a blind substitution.
	// Version 1 leaves those flags to the author, so an adventure keeping a
	// number of its own in them keeps its DPRINT.
	for _, named := range []ddbfile.Condact{
		{Opcode: 49, Params: []byte{21, 1}},              // PLUS 21 1, the low half
		{Opcode: 49, Params: []byte{22, 1}},              // PLUS 22 1, the high half
		{Opcode: 48, Params: []byte{21}},                 // CLEAR 21
		{Opcode: 12, Params: []byte{22}},                 // NOTZERO 22, which only reads it
		{Opcode: 53, Indirect: true, Params: []byte{21}}, // PRINT @21
	} {
		t.Run("left alone when something names the pair", func(t *testing.T) {
			c := &v3Conversion{tables: 22, report: 21, sysmess: map[int]int{}}
			c.convertProcesses([][]ddbfile.Entry{{
				{Condacts: []ddbfile.Condact{named}},
				{Condacts: dprint21},
			}})

			require.Equal(t, dprint21, c.convertCondacts(dprint21),
				"%v names the pair, so the DPRINT means something", named)
		})
	}

	// And one that already names the turn counter is not touched, so converting
	// twice is not a different answer.
	t.Run("the turn counter itself is left as it is", func(t *testing.T) {
		c := &v3Conversion{tables: 22, report: 21, sysmess: map[int]int{}}
		already := []ddbfile.Condact{{Opcode: opDPRINT, Params: []byte{turnCounter}}}

		require.Equal(t, already, c.convertCondacts(already))
	})
}

// Some system messages are not the adventure's: a version 3 interpreter prints
// them itself, as separators. An adventure of 1989 that put its own text there
// has it moved, and every SYSMESS that named the old slot has to follow.
func TestTheReservedMessagesMoveAndTheirCallersFollow(t *testing.T) {
	c := &v3Conversion{tables: 21, sysmess: map[int]int{3: 34}}

	texts := make([]string, 40)
	texts[3] = "ir al "

	got := c.convertSysmess(texts)

	require.Equal(t, "#n", got[3], "what the interpreter expects to print")
	require.Equal(t, "ir al ", got[34])

	require.Equal(t,
		[]ddbfile.Condact{{Opcode: opSYSMESS, Params: []byte{34}}},
		c.convertCondacts([]ddbfile.Condact{{Opcode: opSYSMESS, Params: []byte{3}}}))

	// One that names a slot nobody moved is left alone.
	require.Equal(t,
		[]ddbfile.Condact{{Opcode: opSYSMESS, Params: []byte{7}}},
		c.convertCondacts([]ddbfile.Condact{{Opcode: opSYSMESS, Params: []byte{7}}}))
}

// A space inside graphics mode is drawn with the glyph from the top half of the
// font, and the font changed between the versions: where the one of 1989 had
// nothing, AD8x6.CHR has an Á. So the titles of La Aventura Original came out
// as "AVENTURAÁORIGINALÁI".
func TestASpaceNeverStaysInGraphicsMode(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{
			name: "a title with spaces",
			in:   "#gAVENTURA ORIGINAL I#t",
			want: "#gAVENTURA#t #gORIGINAL#t #gI#t",
		},
		{
			name: "text around it is untouched",
			in:   `Pone: "#gCUIDADO CON TUS OBJETOS#t".`,
			want: `Pone: "#gCUIDADO#t #gCON#t #gTUS#t #gOBJETOS#t".`,
		},
		{
			name: "an accented capital is a run of its own and has no spaces",
			in:   "Est#g{#ts en la cueva",
			want: "Est#g{#ts en la cueva",
		},
		{"nothing to do", "un texto corriente", "un texto corriente"},
		{"two runs", "#gUNO DOS#t y #gTRES#t", "#gUNO#t #gDOS#t y #gTRES#t"},
		// An unterminated run is left alone rather than guessed at.
		{"a run that never closes", "#gSIN CERRAR", "#gSIN CERRAR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, convertText(tc.in))
		})
	}
}

// The three tables the loop calls are not dead code, and the source has to
// stop saying they might be.
//
// What the comment is written from is a count of PROCESS condacts, and in DAAD
// 1 these three were the interpreter's: no condact named them, so the count is
// zero and the source said "either dead code or entered some other way". True
// of the database, false of the conversion, and it was saying it of the
// response table — which is most of the adventure.
func TestTheTablesTheLoopCallsAreNotCalledDeadCode(t *testing.T) {
	const deadCode = "either dead code"

	for _, db := range databases {
		t.Run(db.name, func(t *testing.T) {
			r := openDatabase(t, db.path)

			// Without converting, the count is right and the wording with it:
			// nothing in the database does reach tables 1 and 2.
			plain, err := writeProcesses(r)
			require.NoError(t, err)
			require.Contains(t, plain, deadCode,
				"an unconverted source has three tables nothing names")

			_, err = convertToV3(r)
			require.NoError(t, err)

			converted, err := writeProcesses(r)
			require.NoError(t, err)

			// Two fewer: the old tables 1 and 2. The old table 0 said MAIN
			// LOOP rather than dead code, and the sub-processes that really
			// are unreachable keep saying so, which is the point of the
			// wording being there at all.
			require.Equal(t,
				strings.Count(plain, deadCode)-2,
				strings.Count(converted, deadCode))

			for old, role := range map[int]string{
				0: "The response table",
				1: "Runs after the location description",
				2: "Runs once a turn",
			} {
				require.Contains(t, converted,
					fmt.Sprintf("; TABLE %d - %s. The game loop calls it, in %s.",
						movedTable(old), role, v3LoopFile),
					"table %d went to %d", old, movedTable(old))
			}
		})
	}
}

// The loops in examples carry things this decompiler cannot write yet, and
// they have to go on carrying them.
//
// The loop it generates is still the one for an adventure with no pictures:
// it takes the surface away from the graphics window, because version 3 draws
// no vectors and vectors are all a Spectrum adventure of 1989 has. The ported
// copies draw, and what makes them draw is hand-written.
//
// This is here because that hand-written part has now been overwritten twice
// by somebody regenerating the sources — both times by replacing the file with
// the decompiler's own output, which is exactly the thing that looks right.
// The symptom is a long way from the cause: the text comes out on top of the
// pictures, and nobody thinks of a loop.
//
// When the decompiler learns to write these itself, this test is what says so:
// it will start failing on the generated loop matching the kept one, and the
// answer then is to delete it.
func TestTheExampleLoopsKeepWhatTheDecompilerCannotWriteYet(t *testing.T) {
	kept := []string{
		"PICTURE @38",  // the location's own picture, by indirection
		"DISPLAY 0",    // which PICTURE only prepares
		"CLEAR   40",   // "this location has a drawing", for the generic one
		"SET     40",   //
		"WINAT   14 0", // and the text below the picture rather than on it
		"WINSIZE 24 42",
		// And the timeout asked for the way HASAT names a bit, which is not by
		// flag number: 87 is flag 49 bit 7, and the 49 that used to be here is
		// flag 53 bit 1 and never true when the clock runs out.
		"HASAT   87",
		// The room is not drawn in the dark. Without the ZERO 28 it was drawn
		// and then blanked by the adventure's own table 3, which is a flash of
		// the room you are not supposed to be able to see.
		"ZERO    28\n                PICTURE @38",
		"SYSMESS 0", // and the darkness is said out loud
	}

	loops, err := filepath.Glob(filepath.Join("..", "..", "..",
		"examples", "AO", "ZXSpectrum", "*", "part*", v3LoopFile))
	require.NoError(t, err)
	require.Len(t, loops, 4, "two machines, two parts")

	for _, path := range loops {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			body, err := os.ReadFile(path)
			require.NoError(t, err)

			for _, want := range kept {
				require.Contains(t, string(body), want, path)
			}

			require.NotContains(t, string(body), "WINSIZE 0 0",
				"this is the loop for an adventure with no pictures")

			// And the clearing comes before anything is printed, which is an
			// order and not a presence, so it needs a check of its own.
			//
			// The CLS went in to stop the last turn's picture sitting under the
			// new location's text, and it went in below the darkness message.
			// So being in the dark printed "No se vé nada." and then wiped it:
			// no picture, which is right, and no text either, which left the
			// input cursor alone on a black screen.
			cleared := strings.Index(string(body), "CLS")
			dark := strings.Index(string(body), "SYSMESS 0")

			require.Positive(t, cleared)
			require.Positive(t, dark)
			require.Less(t, cleared, dark,
				"the screen is cleared after the darkness message, which erases it")
		})
	}
}

// And the loading screen is the screen, not the twelve bytes in front of it.
//
// The AO$ block of the original tape is 6924 bytes and loads at 30000, which is
// not video memory: the first twelve are a copier, and line 20 of the BASIC
// calls it.
//
//	21 3C 75  ld hl,30012   ; itself, just past these twelve
//	11 00 40  ld de,0x4000  ; video memory
//	01 00 1B  ld bc,6912
//	ED B0     ldir
//	C9        ret
//
// Done that way so the picture appears all at once instead of painting as it
// comes off the tape. Our loaders put it straight at 16384 and let it paint,
// which is the better of the two for a disk and no worse for a tape — but it
// means the file has to be the 6912 bytes the copier moves and not the first
// 6912 of the block.
//
// Taking them from the front cost twelve bytes of the first pixel row and put
// the copier in their place. Each row of a Spectrum screen is 32 bytes, so a
// shift of twelve rotates every one of them by twenty characters: the picture
// came out with its halves swapped and a bar of machine code across the top.
func TestTheExampleLoadingScreenIsNotTheCopierInFrontOfIt(t *testing.T) {
	copier := []byte{0x21, 0x3C, 0x75, 0x11, 0x00, 0x40, 0x01, 0x00, 0x1B, 0xED, 0xB0, 0xC9}

	screens, err := filepath.Glob(filepath.Join("..", "..", "..",
		"examples", "AO", "ZXSpectrum", "*", "part*", "gfx", "loading.scr"))
	require.NoError(t, err)
	require.Len(t, screens, 4, "two machines, two parts")

	for _, path := range screens {
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(path))), func(t *testing.T) {
			body, err := os.ReadFile(path)
			require.NoError(t, err)

			require.Len(t, body, 6912, "a Spectrum screen is 6144 of pixels and 768 of attributes")
			require.NotEqual(t, copier, body[:len(copier)],
				"this is the block as it comes off the tape, copier and all")
		})
	}
}

// And over the databases themselves, which is where a mistake in any of the
// above actually shows. These are the two adventures of 1989, the only ones
// there is anything to convert in.
func TestConvertingTheVersionOneAdventures(t *testing.T) {
	for _, db := range databases {
		t.Run(db.name, func(t *testing.T) {
			r := openDatabase(t, db.path)

			before, err := r.Processes()
			require.NoError(t, err)

			plan, err := convertToV3(r)
			require.NoError(t, err)

			after, err := r.Processes()
			require.NoError(t, err)

			// Four tables for the loop, and a fifth for the endgame report,
			// which both adventures need because both print the dead pair.
			require.Positive(t, plan.report, "both adventures end with a report")
			require.Len(t, after, len(before)+len(v3Reserved)+1,
				"the loop needs four tables of its own, and the report one more")
			require.Equal(t, endgameReport(), after[plan.report])

			for _, n := range v3Reserved {
				require.Empty(t, after[n], "table %d is the loop's and is written separately", n)
			}

			var quits, bare, restarts, dprints, pictures int

			for number, entries := range after {
				for _, e := range entries {
					for i, c := range e.Condacts {
						switch {
						case c.Opcode == opDESC && len(c.Params) == 0:
							bare++
						case c.Opcode == opRESTART:
							restarts++
						case c.Opcode == opQUIT:
							quits++

							// The END comes right after, or one past the
							// report these adventures print before it.
							require.True(t,
								followedBy(e.Condacts, i, opEND) ||
									(followedBy(e.Condacts, i, opPROCESS) &&
										followedBy(e.Condacts, i+1, opEND)),
								"table %d: a QUIT with no END ends nothing", number)
						case c.Opcode == opPICTURE:
							pictures++

							require.True(t, followedBy(e.Condacts, i, opDISPLAY),
								"table %d: a PICTURE with no DISPLAY draws nothing", number)
						case c.Opcode == opDPRINT && !c.Indirect:
							dprints++

							require.Equal(t, []byte{turnCounter}, c.Params,
								"table %d: a DPRINT of anything these adventures never fill "+
									"prints a zero", number)
						}

						// Every PROCESS has to name a table that exists, and
						// never one the loop took.
						if c.Opcode == opPROCESS && !c.Indirect && len(c.Params) == 1 {
							to := int(c.Params[0])
							require.Less(t, to, len(after))
							require.NotContains(t, v3Reserved, to,
								"table %d calls into the loop's own tables", number)
						}
					}
				}
			}

			require.Zero(t, bare, "a bare DESC means the same as nothing under version 3")
			require.Positive(t, restarts, "both adventures describe the player's location")
			require.Positive(t, quits, "both adventures let the player give up")
			require.Positive(t, dprints, "both adventures print a number at the end")
			require.Positive(t, pictures, "both adventures draw pictures of their own")

			// The system messages the interpreter reserves are now what it
			// expects to print, and whatever the adventure had there is
			// somewhere it can still be reached.
			texts, err := r.SystemMessages()
			require.NoError(t, err)

			for from, to := range plan.sysmess {
				require.Equal(t, "#n", texts[from], "message %d", from)
				require.NotEmpty(t, texts[to], "message %d went to %d", from, to)
				require.NotContains(t, reservedSysmess, to, "and not into another reserved slot")
			}
		})
	}
}

// Converting a database that needs no converting does nothing, and says so by
// handing back no plan rather than an error.
//
// Version 2 onwards already keeps the loop where version 3 looks for it, so
// --v3 on one of those is a question whose answer is "it already is". It used
// to be refused, and that made the flag unusable over the corpus we have: the
// five adventures are two formats, and a harness walking all of them had to
// know which each was before it could ask. Worth being careful about one thing
// — a nil plan must also mean no loop file is written, or a database that
// carries its own loop would be given a second one.
func TestConvertingWhatIsAlreadyVersionThreeDoesNothing(t *testing.T) {
	r := openDatabase(t, "../../../work/aventuras/Cozumel/PC/PART1.DDB")

	require.False(t, r.OldFormat())

	plan, err := convertToV3(r)
	require.NoError(t, err)
	require.Nil(t, plan, "a conversion was planned for a database that needs none")

	// And the loop file is the thing a nil plan has to keep from happening.
	_, err = writeV3Loop(plan, arch.ZX)
	require.Error(t, err, "it wrote a game loop for a database that already has one")
}

// The two adventures of 1989, which are the only ones there is anything to
// convert in. The reader's own tests have a list like this; the two are
// separate on purpose, so that moving one package's fixtures does not quietly
// empty the other package's tests.
var databases = []struct{ name, path string }{
	{"AO part 1", "../../../work/aventuras/La_Aventura_Original/PC/PART1.DDB"},
	{"AO part 2", "../../../work/aventuras/La_Aventura_Original/PC/PART2.DDB"},
	{"Jabato part 1", "../../../work/aventuras/El_Jabato/PC/CGA/PART1.DDB"},
	{"Jabato part 2", "../../../work/aventuras/El_Jabato/PC/CGA/PART2.DDB"},
}

func openDatabase(t *testing.T, path string) *ddbfile.Reader {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("%s is not in the checkout: %v", path, err)
	}

	r, ok := ddbfile.At(data, 0)
	require.True(t, ok, "%s does not read as a database", path)

	return r
}

// What a +3 disk carries is the drawings the adventure can ask for, and no
// others.
//
// A drawing on a +3 is a file, and a file is a name out of the 128 the disk
// has, so a drawing nothing can ask for is a name and a kilobyte spent on
// nothing. There were 13 of them in part one and 27 in part two — more than a
// third of what was being packaged.
//
// They are not stray. They are the library: pieces and backgrounds that other
// drawings CALL. On the machine of 1989 that mattered, because the engine
// followed the CALL at draw time; what we package is each drawing already
// traced with its calls followed, so the pieces are inside the drawings that
// use them and nothing ever opens them by name.
//
// Asking for one means one of two things, and this works out both:
//
//   - an explicit PICTURE n somewhere in the processes;
//   - PICTURE @38, which is the loop drawing the picture of the location the
//     player is at — so any location the player can reach.
//
// Reachable is taken widely on purpose: described, or with an exit, or the far
// end of somebody's exit, or the target of a GOTO. Wider than it needs to be,
// because the cost of keeping one too many is a kilobyte and the cost of
// dropping one too few is a location with no picture.
//
// The one way round this the adventure has is COPYFF 55 38 in part one, which
// puts the player somewhere to drop an object down it. It is transient — AUTOD
// drops at flag 38 and the GOTO 18 two condacts later puts them back, inside
// the one entry — so no turn ever ends there and no picture is ever asked for
// by it.
func TestThePlus3TreesCarryWhatTheAdventureCanAskFor(t *testing.T) {
	for _, part := range []string{"part1", "part2"} {
		t.Run(part, func(t *testing.T) {
			root := filepath.Join("..", "..", "..",
				"examples", "AO", "ZXSpectrum", "+3", part)

			have := drawingsOf(t, root)
			asked, explicit := drawingsAsked(t, root)

			var spare []int

			for _, n := range slices.Sorted(maps.Keys(have)) {
				if !asked[n] {
					spare = append(spare, n)
				}
			}

			// All of them at once, because the answer to one is the answer to
			// the lot and a failure that names the first is a failure you fix
			// forty times.
			require.Empty(t, spare,
				"packaged and nothing can ask for them: library material, already "+
					"traced into the drawings that CALL it")

			for _, n := range slices.Sorted(maps.Keys(explicit)) {
				require.Contains(t, have, n, "PICTURE %d names a drawing that is not there", n)
			}
		})
	}
}

// drawingsOf is what a tree packages.
func drawingsOf(t *testing.T, root string) map[int]bool {
	t.Helper()

	found, err := filepath.Glob(filepath.Join(root, "gfx", "scr", "*.scr"))
	require.NoError(t, err)
	require.NotEmpty(t, found)

	out := map[int]bool{}

	for _, path := range found {
		n, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(path), ".scr"))
		require.NoError(t, err)

		out[n] = true
	}

	return out
}

var (
	exampleDescribed = regexp.MustCompile(`^/(\d+) "(.+)"\s*$`)
	exampleLocation  = regexp.MustCompile(`^/(\d+)\s*$`)
	// No leading space is required: a connection line starts in column one,
	// "O        4". Demanding one matched nothing, the whole connection map
	// went missing, and sixteen drawings of locations the player walks to were
	// counted as unreachable.
	exampleExit    = regexp.MustCompile(`^\s*([A-Za-zÑÁÉÍÓÚÜ][^\s;]*)\s+(\d+)\s*$`)
	examplePicture = regexp.MustCompile(`\bPICTURE\s+(\d+)`)
	exampleGoto    = regexp.MustCompile(`\bGOTO\s+(\d+)`)
)

// drawingsAsked works out which drawing numbers the adventure can ask for, and
// separately the ones it names outright.
func drawingsAsked(t *testing.T, root string) (asked, explicit map[int]bool) {
	t.Helper()

	read := func(name string) string {
		body, err := os.ReadFile(filepath.Join(root, name))
		require.NoError(t, err)

		return string(body)
	}

	asked, explicit = map[int]bool{}, map[int]bool{}

	for _, line := range strings.Split(read("location-text.sce"), "\n") {
		if m := exampleDescribed.FindStringSubmatch(line); m != nil && strings.TrimSpace(m[2]) != "" {
			asked[atoi(t, m[1])] = true
		}
	}

	here := -1

	for _, line := range strings.Split(read("connections.sce"), "\n") {
		if m := exampleLocation.FindStringSubmatch(line); m != nil {
			here = atoi(t, m[1])

			continue
		}

		if m := exampleExit.FindStringSubmatch(line); m != nil && here >= 0 {
			asked[here] = true
			asked[atoi(t, m[2])] = true
		}
	}

	processes := read("processes.sce")

	for _, m := range exampleGoto.FindAllStringSubmatch(processes, -1) {
		asked[atoi(t, m[1])] = true
	}

	for _, m := range examplePicture.FindAllStringSubmatch(processes, -1) {
		explicit[atoi(t, m[1])] = true
		asked[atoi(t, m[1])] = true
	}

	return asked, explicit
}

func atoi(t *testing.T, s string) int {
	t.Helper()

	n, err := strconv.Atoi(s)
	require.NoError(t, err)

	return n
}
