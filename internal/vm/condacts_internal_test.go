package vm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Counting a flag, which stops at each end of the byte rather than going round.
//
// It is in the package because what is being fixed is arithmetic and the
// cheapest honest test of arithmetic is the arithmetic. What it is really
// guarding is a flag used as a switch: SET leaves 255, and a process that adds
// one to that flag every turn used to turn the switch off on the next turn.
// See plus, and do_PLUS at daad_condacts.c:1445.
func TestFlagArithmeticStopsAtTheEnds(t *testing.T) {
	t.Run("adding", func(t *testing.T) {
		for _, c := range []struct {
			name              string
			flag, value, want byte
		}{
			{"the ordinary case", 7, 3, 10},
			{"landing exactly on the top", 250, 5, 255},
			{"one past the top", 255, 1, 255},
			{"well past the top", 250, 20, 255},
			{"the largest there is, twice over", 255, 255, 255},
			{"adding nothing", 255, 0, 255},
		} {
			t.Run(c.name, func(t *testing.T) {
				require.Equal(t, c.want, plus(c.flag, c.value))
			})
		}
	})

	t.Run("subtracting", func(t *testing.T) {
		for _, c := range []struct {
			name              string
			flag, value, want byte
		}{
			{"the ordinary case", 10, 3, 7},
			{"landing exactly on zero", 5, 5, 0},
			{"one below zero", 0, 1, 0},
			{"well below zero", 5, 20, 0},
			{"subtracting nothing", 0, 0, 0},
		} {
			t.Run(c.name, func(t *testing.T) {
				require.Equal(t, c.want, minus(c.flag, c.value))
			})
		}
	})
}

// Asking for an object "here", which is a question about the player and not
// only about the object.
//
// 255 stands where a location would, and the object lying on that floor does
// not hold 255: it holds the room's number. Comparing the two as they come
// answers no to every ISAT 255 there is. See at, and do_ISAT at
// daad_condacts.c:376.
func TestHereMeansWhereThePlayerIs(t *testing.T) {
	const room = 19

	for _, c := range []struct {
		name  string
		where byte
		want  bool
	}{
		{"lying in the room the player is in", room, true},
		{"lying in another room", room + 1, false},
		{"carried", LocCarried, false},
		{"not created", LocNotCreated, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := &Machine{objects: []object{{location: c.where}}}
			m.flags[FlagPlayer] = room

			require.Equal(t, c.want, m.at(0, LocHere))
		})
	}

	// And the other places are still themselves: the new reading is an extra
	// way to be somewhere, not a looser comparison.
	t.Run("carried is still carried and nothing else", func(t *testing.T) {
		m := &Machine{objects: []object{{location: LocCarried}}}
		m.flags[FlagPlayer] = room

		require.True(t, m.at(0, LocCarried))
		require.False(t, m.at(0, room))
	})
}

// The snake, which is the bug this was found as.
//
// Flag 104 of La Aventura Original's second part is the snake in the elf
// kings' chamber. Letting the bird out of the cage SETs it, and the process
// that runs every turn adds one to it. Wrapping put it back to zero, the snake
// returned the turn after it fled, and the poison killed the player seven
// turns later with half the cave still unreached.
func TestASwitchedOnFlagSurvivesTheTurnCounter(t *testing.T) {
	const on = 255 // what SET leaves

	flag := byte(on)
	for turn := range 20 {
		flag = plus(flag, 1)
		require.EqualValues(t, on, flag, "the switch went off on turn %d", turn+1)
	}
}

// An adventure that has not asked for a timeout never gets one, however
// patient the console is willing to be.
//
// Flag 48 is the duration and zero means there is no clock, which is what
// every adventure starts with and what most of them stay at. A console that
// can wait must not be a console that waits when nobody asked.
func TestNoTimeWantedIsNoTimeoutGiven(t *testing.T) {
	m := &Machine{watch: NoWatcher{}, console: neverAnswers{}}

	m.setFlag(FlagTime, 0)

	_, _, expired := m.waitForLine()
	require.False(t, expired, "flag 48 is zero, so there is no clock to run out")

	// And with a duration, the same console does time out — so the test above
	// is about the flag and not about the console refusing to play along.
	m.setFlag(FlagTime, 30)

	_, _, expired = m.waitForLine()
	require.True(t, expired)
}

// neverAnswers is a player who says nothing, ever.
type neverAnswers struct{}

func (neverAnswers) Print(string) {}
func (neverAnswers) AnyKey()      {}

func (neverAnswers) Input() (string, bool) { return "", true }

func (neverAnswers) InputWithin(time.Duration) (string, bool, bool) { return "", true, true }
