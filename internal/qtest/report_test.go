package qtest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// How a run reads.
//
// Most of what the report does is taste and is not tested. Two things are not
// taste: a step is one line with its verdict at the end of it, and anything
// said in the middle of a step breaks that line — because a verdict left
// hanging at the end of a stretch of other output reads as belonging to the
// wrong thing.
//
// None of this needs a terminal. Lipgloss drops the colour when it is writing
// somewhere that is not one and leaves the widths alone, so what a test reads
// is the layout without the escapes.

func TestAStepIsOneLineWithItsVerdictAtTheEnd(t *testing.T) {
	var out strings.Builder

	report := NewReport(&out)
	report.Align(len("the longer one"))

	report.Step("short")
	report.Pass()
	report.Step("the longer one")
	report.Pass()

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 2, "one line per step and nothing else")

	for _, line := range lines {
		require.Contains(t, line, "OK", "the verdict belongs on the step's own line")
	}

	require.Equal(t, strings.Index(lines[0], "OK"), strings.Index(lines[1], "OK"),
		"the names are padded to the widest, so the badges stand in one column")
}

// OK and FAIL are drawn the same width, which is what keeps that column from
// wobbling.
func TestTheTwoVerdictsAreTheSameWidth(t *testing.T) {
	var ok, bad strings.Builder

	passed := NewReport(&ok)
	passed.Step("a step")
	passed.Pass()

	failed := NewReport(&bad)
	failed.Step("a step")
	failed.Fail(Place{File: "t.qtest", Line: 1, Col: 1}, "EXPECT x", "it did not", "")

	first, _, _ := strings.Cut(bad.String(), "\n")

	require.Equal(t, len(strings.TrimRight(ok.String(), "\n")), len(first),
		"a step that passed and a step that failed should be the same length")
}

// Anything said inside a step breaks its line, and the verdict then stands on
// its own at the margin.
func TestWhatIsSaidInsideAStepBreaksItsLine(t *testing.T) {
	var out strings.Builder

	report := NewReport(&out)

	report.Step("ask the machine about itself")
	report.Debug("FLAG_1=23")
	report.Pass()

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 3)

	require.NotContains(t, lines[0], "FLAG_1", "the debug line is its own, so something can read it")
	require.Equal(t, "FLAG_1=23", lines[1], "and it is bare, with nothing in front of it")
	require.Contains(t, lines[2], "OK")
	require.NotContains(t, lines[2], "ask the machine",
		"the name is already up there; what is left is the verdict")
}

// The summary is a line for what passed and, only when there is something to
// say, a line for what did not.
func TestTheSummarySaysWhatThereIsToSay(t *testing.T) {
	t.Run("everything held", func(t *testing.T) {
		var out strings.Builder

		report := NewReport(&out)
		report.Step("a step")
		report.Pass()
		report.Summary()

		require.Contains(t, out.String(), "1 step passed")
		require.NotContains(t, out.String(), "failed", "nothing failed, so there is no line for it")
		require.NoError(t, report.Err())
	})

	t.Run("something did not", func(t *testing.T) {
		var out strings.Builder

		report := NewReport(&out)
		report.Step("a step")
		report.Pass()
		report.Step("another")
		report.Fail(Place{File: "t.qtest", Line: 7, Col: 3}, "EXPECT x", "it did not", "what it said")
		report.Summary()

		said := out.String()

		require.Contains(t, said, "1 step passed")
		require.Contains(t, said, "1 step failed")
		require.Contains(t, said, "t.qtest:7:3", "a failure says where it is written")
		require.Contains(t, said, "what it said", "and what the adventure had said instead")
		require.ErrorContains(t, report.Err(), "1 step failed of 2 steps")
	})
}

// A run with no name has no frame, because an empty box is worse than no box.
func TestANamelessRunHasNoFrame(t *testing.T) {
	var out strings.Builder

	NewReport(&out).Title("")

	require.Empty(t, out.String())
}
