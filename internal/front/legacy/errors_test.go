package legacy

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A failure says where it is once, and at the front.
//
// The front end finds a problem deep in a parameter and then adds context on
// the way back up — the parameter, the condact, the process — and each of those
// layers knows a position. Each of them writing its own put four copies of the
// same file:line in one sentence and buried the precise one in the middle of
// it. See errors.go.
func TestAFailureSaysWhereItIsOnceAndInFront(t *testing.T) {
	deep := placedf("t.sce:241:25", "%q is not defined", "Un")

	// The layers above, in the order the parser adds them.
	err := placedf("t.sce:242:17", "parameter %d: %w", 1, deep)
	err = placedf("t.sce:242:17", "%s: %w", "XMES", err)
	err = placedf("t.sce:242:17", "process %d: %w", 0, err)

	require.Equal(t,
		`t.sce:241:25: process 0: XMES: parameter 1: "Un" is not defined`,
		err.Error())

	require.Equal(t, 1, strings.Count(err.Error(), "t.sce:"),
		"the position is in there more than once")
	require.Equal(t, 0, strings.Count(err.Error(), "242:17"),
		"a layer that only adds context should not bring its own position")
}

// And it is still an error chain, so errors.Is and errors.As see through it.
func TestAPlacedErrorStillWraps(t *testing.T) {
	sentinel := errors.New("the root of it")

	err := placedf("t.sce:1:1", "in a parameter: %w", sentinel)
	err = placedf("t.sce:2:2", "in a process: %w", err)

	require.ErrorIs(t, err, sentinel)
}

// An error with nothing to wrap reports where it was raised.
func TestAFailureWithNothingUnderItReportsItsOwnPosition(t *testing.T) {
	err := placedf("t.sce:7:3", "%s expected", "/VOC")
	require.Equal(t, "t.sce:7:3: /VOC expected", err.Error())
}

// The whole front end, to show it is not only true of placedf.
//
// A label nobody defines is raised where it is used and reported from there,
// however many sections the parser has climbed back out of since.
func TestTheFrontEndReportsOnePositionForANestedFailure(t *testing.T) {
	_, err := parse(t, strings.Replace(bare,
		"> COGER LLAVE   MESSAGE 0",
		"> COGER LLAVE   MESSAGE NOSUCHTHING", 1))

	require.Error(t, err)
	require.Equal(t, 1, strings.Count(err.Error(), "t.sce:"),
		"one failure, one position: %s", err.Error())
	require.True(t, strings.HasPrefix(err.Error(), "t.sce:"),
		"the position should come first, not from inside the sentence: %s", err.Error())
}
