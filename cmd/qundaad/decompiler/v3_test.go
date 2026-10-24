package decompiler

import (
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

	got := c.convertCondacts([]Condact{
		{Opcode: opPROCESS, Params: []byte{0}},
		{Opcode: opPROCESS, Params: []byte{2}},
		{Opcode: opPROCESS, Params: []byte{13}},
	})

	require.Equal(t, []Condact{
		{Opcode: opPROCESS, Params: []byte{5}},
		{Opcode: opPROCESS, Params: []byte{4}},
		{Opcode: opPROCESS, Params: []byte{17}},
	}, got)

	// An indirect call names a flag and what it holds is not ours to touch.
	got = c.convertCondacts([]Condact{{Opcode: opPROCESS, Indirect: true, Params: []byte{7}}})
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

	got := c.convertCondacts([]Condact{{Opcode: opDESC}})

	require.Equal(t, []Condact{{Opcode: opCLS}, {Opcode: opRESTART}}, got)

	// A DESC that names a location is version 3's own and is left alone.
	got = c.convertCondacts([]Condact{{Opcode: opDESC, Params: []byte{12}}})
	require.Equal(t, []Condact{{Opcode: opDESC, Params: []byte{12}}}, got)
}

// QUIT ended the game in DAAD 1. In version 3 it only asks, and END is what
// ends it — so an adventure of 1989 has no END anywhere and confirming "quit"
// does nothing at all.
func TestQuitGetsTheEndItNeverNeeded(t *testing.T) {
	c := &v3Conversion{tables: 21, sysmess: map[int]int{}}

	got := c.convertCondacts([]Condact{{Opcode: opQUIT}, {Opcode: 27, Params: []byte{21}}})

	require.Equal(t, []Condact{
		{Opcode: opQUIT},
		{Opcode: opEND},
		{Opcode: 27, Params: []byte{21}},
	}, got)

	// One that already does the version 3 thing is left as it is, so that
	// converting something twice is not a different answer.
	already := []Condact{{Opcode: opQUIT}, {Opcode: opEND}}
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
		[]Condact{{Opcode: opSYSMESS, Params: []byte{34}}},
		c.convertCondacts([]Condact{{Opcode: opSYSMESS, Params: []byte{3}}}))

	// One that names a slot nobody moved is left alone.
	require.Equal(t,
		[]Condact{{Opcode: opSYSMESS, Params: []byte{7}}},
		c.convertCondacts([]Condact{{Opcode: opSYSMESS, Params: []byte{7}}}))
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

			require.NoError(t, r.ConvertToV3())

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
			texts, err := r.TextTable(ptrSysMessageText, r.NumSysMessages())
			require.NoError(t, err)

			for from, to := range r.v3.sysmess {
				require.Equal(t, "#n", texts[from], "message %d", from)
				require.NotEmpty(t, texts[to], "message %d went to %d", from, to)
				require.NotContains(t, reservedSysmess, to, "and not into another reserved slot")
			}
		})
	}
}

// Converting a database that needs no converting is refused rather than done
// badly: version 2 onwards already keeps the loop where version 3 looks for it.
func TestOnlyAVersionOneDatabaseIsConverted(t *testing.T) {
	r := openDatabase(t, "../../../work/aventuras/Cozumel/PC/PART1.DDB")

	require.False(t, r.OldFormat())
	require.ErrorContains(t, r.ConvertToV3(), "needs no converting")
}

// The illustration condacts come out when the build has no pictures. It is not
// part of the version conversion — version 3 draws pictures — and it is tested
// apart for the same reason.
func TestPicturesComeOutWhenAskedFor(t *testing.T) {
	got := withoutPictures([][]Entry{{{
		Verb: 1, Noun: 2,
		Condacts: []Condact{
			{Opcode: opPICTURE, Params: []byte{4}},
			{Opcode: opDISPLAY, Params: []byte{0}},
			{Opcode: opCLS},
		},
	}}})

	require.Equal(t, []Condact{{Opcode: opCLS}}, got[0][0].Condacts)
	require.Equal(t, byte(1), got[0][0].Verb, "the entry itself is untouched")
}
