package decompiler

import (
	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"os"
	"testing"

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

	got := c.convertCondacts([]ddbfile.Condact{{Opcode: opQUIT}, {Opcode: 27, Params: []byte{21}}})

	require.Equal(t, []ddbfile.Condact{
		{Opcode: opQUIT},
		{Opcode: opEND},
		{Opcode: 27, Params: []byte{21}},
	}, got)

	// One that already does the version 3 thing is left as it is, so that
	// converting something twice is not a different answer.
	already := []ddbfile.Condact{{Opcode: opQUIT}, {Opcode: opEND}}
	require.Equal(t, already, c.convertCondacts(already))
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

			require.Len(t, after, len(before)+len(v3Reserved),
				"the loop needs four tables of its own")

			for _, n := range v3Reserved {
				require.Empty(t, after[n], "table %d is the loop's and is written separately", n)
			}

			var quits, bare, restarts int

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

							require.True(t, followedBy(e.Condacts, i, opEND),
								"table %d: a QUIT with no END ends nothing", number)
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
