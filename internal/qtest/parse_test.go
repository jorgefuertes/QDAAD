package qtest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Reading a script.
//
// What is tested here is mostly the saying-no: a script is a thing somebody
// types at midnight with the game open in the other window, and every way of
// getting it slightly wrong should come back with the line it is on rather
// than with an adventure that plays oddly.

func TestEveryOrderReads(t *testing.T) {
	script, err := ParseText("t.qtest", `
; a comment, and a blank line above it
TEST    "everything"
SEED    1989

STEP    "the lot"
  EXPECT  "Los rayos del sol", 2
  EXPECT  NOT "not implemented"
  INPUT   coger tortilla
  KEY     RETURN
  KEY     SPACE
  KEY     A
  WAIT    5
  DEBUG   FLAG 1
  DEBUG   FLAGS
  DEBUG   FLAGS SET
  DEBUG   AT
  DEBUG   PICTURE
  DEBUG   OBJECTS
  DEBUG   SAID
  ASSERT  FLAG 10 = 5
  ASSERT  FLAG 10 != 0
  ASSERT  AT = 3
  ASSERT  PICTURE = 1
  ASSERT  SCORE > 0
  ASSERT  TURNS < 300
  ASSERT  CARRIED 12
  ASSERT  WORN 3
  ASSERT  HERE 7
  ASSERT  PRESENT 7
  ASSERT  OBJECT 12 AT 254
  ASSERT  ENDED
  TRACE   ON
  TRACE   OFF
END
`)
	require.NoError(t, err, "every order in the language should read")

	require.Equal(t, "everything", script.Name)
	require.True(t, script.HasSeed, "SEED was given")
	require.Equal(t, int64(1989), script.Seed)
	require.True(t, script.tracing, "a script with TRACE in it manages the trace itself")

	// Two for the STEP and its END, and one for each order between them.
	const orders = 30

	require.Equal(t, orders, script.Steps(), "one step per order, and no more")
}

// The seconds of an EXPECT are optional, and what they default to is not zero.
//
// Wanting to wait is the common case: an adventure with a TIME running says
// things on its own and the point of the language is to be able to test that.
// A default of none would make every line that waits spell out a number, and
// the number would be the same number every time.
func TestAnExpectWithoutSecondsStillWaits(t *testing.T) {
	script, err := ParseText("t.qtest", `
EXPECT  "the dwarf walks in"
EXPECT  "and again", 3
EXPECT  the dwarf walks in
EXPECT  bare with a number, 7
`)
	require.NoError(t, err)

	require.Equal(t, defaultSeconds, script.steps[0].secs, "a quoted text with no seconds")
	require.Equal(t, 3, script.steps[1].secs)
	require.Equal(t, defaultSeconds, script.steps[2].secs, "an unquoted text with no seconds")

	require.Equal(t, 7, script.steps[3].secs,
		"an unquoted text may end in a comma and a number, and then it is the budget")
	require.Equal(t, "bare with a number", script.steps[3].text)
}

func TestWhatAScriptSaysAboutItself(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.qtest")

	require.NoError(t, os.WriteFile(path, []byte("GAME \"AO.qrun\"\n"), 0o600))

	script, err := Parse(path)
	require.NoError(t, err)

	require.Equal(t, filepath.Join(dir, "AO.qrun"), script.Game,
		"a file named in a script is beside the script, which is where its author was looking")
}

// A prologue is the point of INCLUDE: getting past the title screen and
// through the access code is the same twenty lines for every test of the
// second part.
func TestAPrologueComesIn(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "prologue.qtest"), []byte(`
STEP "the access code"
  INPUT TIMACUS
END
`), 0o600))

	path := filepath.Join(dir, "part2.qtest")
	require.NoError(t, os.WriteFile(path, []byte(`
TEST "the second part"
INCLUDE "prologue.qtest"
STEP "and now the game"
  EXPECT "Una llave"
END
`), 0o600))

	script, err := Parse(path)
	require.NoError(t, err)

	require.Equal(t, "the second part", script.Name)

	const orders = 6

	require.Equal(t, orders, script.Steps(), "the prologue's orders should be in the middle of it")
	require.Equal(t, "the access code", script.steps[0].text, "and first, where the INCLUDE was")
}

func TestAScriptThatIncludesItselfIsSaidSo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "loop.qtest")

	require.NoError(t, os.WriteFile(path, []byte("INCLUDE \"loop.qtest\"\n"), 0o600))

	_, err := Parse(path)
	require.ErrorContains(t, err, "includes itself",
		"a cycle has to be refused rather than followed until the stack gives up")
}

// Every way of getting it wrong, and what each one should say.
//
// The line number is the part that matters. A script is read before the
// adventure is started, so a mistake in one costs nothing to find — as long as
// it says where it is.
func TestWhatIsWrongAndWhereItIs(t *testing.T) {
	for _, c := range []struct {
		name, script, says string
		line               int
	}{
		{
			name:   "a step that is never closed",
			script: "STEP \"one\"\n  EXPECT hello\n",
			says:   "never closed",
			line:   1,
		},
		{
			name:   "an END with nothing open",
			script: "EXPECT hello\nEND\n",
			says:   "END with no STEP",
			line:   2,
		},
		{
			name:   "steps do not nest",
			script: "STEP \"one\"\nSTEP \"two\"\nEND\nEND\n",
			says:   "do not nest",
			line:   2,
		},
		{
			name:   "a step with no name",
			script: "STEP\nEND\n",
			says:   "STEP needs a name",
			line:   1,
		},
		{
			name:   "an order that does not exist",
			script: "EXPECT hello\nPRESS RETURN\n",
			says:   "not an order this language has",
			line:   2,
		},
		{
			name:   "a key that is not one",
			script: "KEY CONTROL\n",
			says:   "is not a key",
			line:   1,
		},
		{
			name:   "an EXPECT with nothing to wait for",
			script: "EXPECT\n",
			says:   "needs something to wait for",
			line:   1,
		},
		{
			name:   "an EXPECT NOT with seconds",
			script: "EXPECT NOT \"no puedo\", 5\n",
			says:   "takes no seconds",
			line:   1,
		},
		{
			name:   "a quote that never closes",
			script: "EXPECT \"los rayos\n",
			says:   "never closes",
			line:   1,
		},
		{
			name:   "a flag that is not a byte",
			script: "ASSERT FLAG 300 = 1\n",
			says:   "outside 0 to 255",
			line:   1,
		},
		{
			name:   "an ASSERT it does not know",
			script: "ASSERT DAYLIGHT = 1\n",
			says:   "ASSERT does not know",
			line:   1,
		},
		{
			name:   "a comparison that is not one",
			script: "ASSERT AT ~ 3\n",
			says:   "is not one of",
			line:   1,
		},
		{
			name:   "a TRACE that is neither",
			script: "TRACE SOMETIMES\n",
			says:   "TRACE is ON or OFF",
			line:   1,
		},
		{
			name:   "a name given twice",
			script: "TEST \"one\"\nTEST \"two\"\n",
			says:   "already given a name",
			line:   2,
		},
		{
			name:   "a WAIT with no number",
			script: "WAIT soon\n",
			says:   "is not a number",
			line:   1,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseText("t.qtest", c.script)

			require.Error(t, err, "this should not have read")
			require.ErrorContains(t, err, c.says)
			require.ErrorContains(t, err, fmt.Sprintf("t.qtest:%d:", c.line),
				"the line it is on is the whole value of the message")
		})
	}
}

// Comments and quotes, which is the one place the two meet.
func TestASemicolonInsideQuotesIsText(t *testing.T) {
	script, err := ParseText("t.qtest", `
EXPECT "a cave; and a door"   ; this bit is the comment
`)
	require.NoError(t, err)

	require.Equal(t, "a cave; and a door", script.steps[0].text)
}

// An operator may be written up against its numbers, because at midnight it
// will be.
func TestOperatorsNeedNoSpace(t *testing.T) {
	script, err := ParseText("t.qtest", "ASSERT FLAG 10=5\nASSERT AT!=3\nASSERT SCORE>0\n")
	require.NoError(t, err)

	require.Equal(t, cmpEq, script.steps[0].cmp)
	require.Equal(t, 10, script.steps[0].num)
	require.Equal(t, 5, script.steps[0].val)

	require.Equal(t, cmpNot, script.steps[1].cmp)
	require.Equal(t, 3, script.steps[1].val)

	require.Equal(t, cmpGt, script.steps[2].cmp)
}
