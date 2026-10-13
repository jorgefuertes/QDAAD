package legacy

import (
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/stretchr/testify/require"
)

// parse runs the whole front end over a snippet, so that a test reads like the
// source an author would write.
func parse(t *testing.T, source string) (*ddb.DDB, error) {
	t.Helper()

	var lines []Line
	for i, text := range strings.Split(source, "\n") {
		lines = append(lines, Line{File: "t.sce", Num: i + 1, Text: text})
	}

	tokens, err := Scan(lines)
	require.NoError(t, err)

	opts := testOptions()

	res, err := Preprocess(tokens, opts)
	require.NoError(t, err)

	return Parse(res, opts)
}

// The smallest source the grammar allows: every section present, none of them
// with anything in it that is not required.
const bare = `
/CTL
_
/VOC
N     1  noun
COGER 20 verb
LLAVE 30 noun
VIEJA 2  adjective
/STX
/0 "It is dark."
/MTX
/0 "You cannot."
/OTX
/0 "a key"
/LTX
/0 "A cave."
/1 "A hall."
/CON
/0
N 1
/1
/OBJ
/0  CARRIED 3 Y N  _ _ _ _ _ _ _ _ _ _ _ _ _ _ Y _  LLAVE VIEJA
/PRO 0
> COGER LLAVE   MESSAGE 0
                DONE
> _     _       SYSMESS 0
/END
`

func TestParsesEverySectionButProcesses(t *testing.T) {
	db, err := parse(t, bare)
	require.NoError(t, err)

	require.Equal(t, byte('_'), db.NullWord)

	t.Run("the vocabulary keeps value and kind", func(t *testing.T) {
		w, found := db.Words().Get(20, ddb.Verb)
		require.True(t, found)
		require.Equal(t, []string{"COGER"}, w.Synonyms)
	})

	t.Run("the texts go to their own tables", func(t *testing.T) {
		got, ok := db.Messages().GetMessage(ddb.SystemMessage, 0)
		require.True(t, ok)
		require.Equal(t, "It is dark.", got)

		got, ok = db.Messages().GetMessage(ddb.UserMessage, 0)
		require.True(t, ok)
		require.Equal(t, "You cannot.", got)
	})

	t.Run("a location keeps its own description", func(t *testing.T) {
		l, err := db.Locations().Get(1)
		require.NoError(t, err)
		require.Equal(t, "A hall.", l.Description)
	})

	// The exit is written with a word and travels as that word's value.
	t.Run("a connection resolves its direction", func(t *testing.T) {
		c, found := db.Conns().Get(0, 1)
		require.True(t, found)
		require.Equal(t, ddb.ID(1), c.To)
	})

	t.Run("an object gathers its text and its twenty-two fields", func(t *testing.T) {
		o, err := db.Objects().Get(0)
		require.NoError(t, err)

		require.Equal(t, "a key", o.Description, "the text comes from /OTX")
		require.Equal(t, ddb.LOC_CARRIED, o.InitLocation)
		require.Equal(t, uint8(3), o.Weight)
		require.True(t, o.Container)
		require.False(t, o.Wearable)
		require.Equal(t, ddb.ID(30), o.Name)
		require.Equal(t, ddb.ID(2), o.Adjective)

		// Written highest bit first, so the Y in column fifteen is bit 1.
		require.True(t, o.Flags[14])
		require.False(t, o.Flags[0])
	})
}

func TestParserRefusesWhatTheFormatCannotHold(t *testing.T) {
	for _, tc := range []struct{ name, swap, with, says string }{
		{"a gap in the text numbering", `/0 "A cave."`, `/1 "A cave."`, "consecutively from zero"},
		{"an unknown word type", "COGER 20 verb", "COGER 20 thing", "is not a word type"},
		{"a direction that is no word", "N 1", "SOUTH 1", "not a verb or a noun"},
		{"a noun that is an adjective", "LLAVE VIEJA", "VIEJA VIEJA", "is not a noun"},
		{"a weight over the limit", "CARRIED 3 Y", "CARRIED 99 Y", "over the 63"},
		{"something that is not Y, N or _", "3 Y N", "3 MAYBE N", "Y, N or _ expected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parse(t, strings.Replace(bare, tc.swap, tc.with, 1))
			require.ErrorContains(t, err, tc.says)
		})
	}
}

// A location with connections but no description would leave the binary's table
// short, and the table is indexed by position.
func TestParserWantsEveryLocationDescribed(t *testing.T) {
	_, err := parse(t, strings.Replace(bare, "/1\n/OBJ", "/1\n/2\n/OBJ", 1))
	require.ErrorContains(t, err, "no description in /LTX")
}

// The location markers are symbols, not names some section knows about. That is
// what makes them worth a value anywhere a value is expected, and it is how the
// compiler of reference does it: drf.pas seeds them before reading a line.
func TestLocationMarkersAreOrdinarySymbols(t *testing.T) {
	t.Run("they resolve where a location is expected", func(t *testing.T) {
		db, err := parse(t, bare)
		require.NoError(t, err)

		o, err := db.Objects().Get(0)
		require.NoError(t, err)
		require.Equal(t, ddb.LOC_CARRIED, o.InitLocation)
	})

	// The proof that they are symbols and not a special case: arithmetic over
	// them works, which a name known only to /OBJ could never do.
	t.Run("they work inside an expression", func(t *testing.T) {
		_, symbols, err := preprocess(t, `#define BELOW "CARRIED - 2"`)
		require.NoError(t, err)
		require.Equal(t, int(ddb.LOC_NOT_CREATED), symbols["BELOW"])
	})

	t.Run("a conditional sees them", func(t *testing.T) {
		got, _, err := preprocess(t, "#ifdef \"CARRIED\"\nYES\n#endif\n")
		require.NoError(t, err)
		require.Equal(t, []string{"YES"}, texts(got))
	})
}

// The underscore means the null word almost everywhere, but in this one slot it
// means the object is not created.
func TestUnderscoreMeansNotCreatedForAnObject(t *testing.T) {
	db, err := parse(t, strings.Replace(bare, "/0  CARRIED", "/0  _", 1))
	require.NoError(t, err)

	o, err := db.Objects().Get(0)
	require.NoError(t, err)
	require.Equal(t, ddb.LOC_NOT_CREATED, o.InitLocation)
}

// HERE resolves as a symbol but is refused here: an object cannot start where
// the player happens to be. The compiler of reference checks the same.
func TestAnObjectCannotStartAtHere(t *testing.T) {
	_, err := parse(t, strings.Replace(bare, "/0  CARRIED", "/0  HERE", 1))
	require.ErrorContains(t, err, "cannot start at HERE")
}

// The processes are the program, and until they were read a database was a
// world with nothing happening in it.
func TestParsesProcesses(t *testing.T) {
	db, err := parse(t, bare)
	require.NoError(t, err)
	require.Equal(t, 1, db.Processes().Len())

	proc, found := db.Processes().Get(0)
	require.True(t, found)
	require.Len(t, proc.Entries, 2)

	t.Run("an entry names its words and keeps its block", func(t *testing.T) {
		e := proc.Entries[0]
		require.Equal(t, ddb.ID(20), e.Verb, "COGER")
		require.Equal(t, ddb.ID(30), e.Noun, "LLAVE")
		require.Len(t, e.Condacts, 2)
		require.Equal(t, "DONE", nameOf(t, e.Condacts[1].Opcode))
	})

	t.Run("the wildcard matches anything", func(t *testing.T) {
		e := proc.Entries[1]
		require.Equal(t, ddb.NO_WORD_ID, e.Verb)
		require.Equal(t, ddb.NO_WORD_ID, e.Noun)
	})
}

// Several entries in a row share one block, which is how a source says that two
// sentences do the same thing without writing it twice.
func TestEntriesInARowShareTheirBlock(t *testing.T) {
	db, err := parse(t, strings.Replace(bare,
		"> COGER LLAVE   MESSAGE 0",
		"> COGER LLAVE\n> _     LLAVE   MESSAGE 0", 1))
	require.NoError(t, err)

	proc, found := db.Processes().Get(0)
	require.True(t, found)
	require.Len(t, proc.Entries, 3)

	require.Equal(t, proc.Entries[0].Condacts, proc.Entries[1].Condacts,
		"both heads run the same block")
	require.Len(t, proc.Entries[0].Condacts, 2)
}

// The @ says the operand names a flag whose contents are used instead.
func TestIndirection(t *testing.T) {
	db, err := parse(t, strings.Replace(bare, "SYSMESS 0", "SYSMESS @38", 1))
	require.NoError(t, err)

	proc, _ := db.Processes().Get(0)
	param := proc.Entries[1].Condacts[0].Params[0]

	require.True(t, param.Indirect)
	require.Equal(t, ddb.ID(38), param.Value)
}

// A condact spelled wrong is reported where it is. The compiler of reference
// takes anything it cannot place as the end of the block, and the complaint
// surfaces somewhere else entirely.
func TestAMisspeltCondactIsCaughtWhereItIs(t *testing.T) {
	_, err := parse(t, strings.Replace(bare, "SYSMESS 0", "SYSMES 0", 1))
	require.ErrorContains(t, err, `"SYSMES" is not a condact`)
}

func nameOf(t *testing.T, op ddb.Opcode) string {
	t.Helper()

	def, found := ddb.LookupCondact(op)
	require.True(t, found)

	return def.Name
}
