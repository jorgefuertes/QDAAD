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

// The pirate, which is the bug this was found as.
//
// WHATO points the five flags that describe "the object this is about" at
// whatever the sentence named. When the sentence named nothing it used to set
// the number to 255 and leave the other four holding the last object's own,
// which is not what the reference does: referencedObject builds the block out
// of a null object of six zeros rather than leaving early — daad_objects.c:47,
// nullObjFake at daad_init.c:14.
//
// The table that reads those flags is the Terrible Pirate of La Aventura
// Original's second part. Every turn away from six named rooms it calls
// PROCESS 20, which asks WHATO for the object in the sentence and then gives
// up unless it is carried or here:
//
//	> _ _ NOTEQ 54 254
//	      NOTSAME 54 38
//	      DONE
//
// Type a bare direction and WHATO finds nothing. With flag 54 still holding
// the 254 of whatever was last picked up, that guard passes, a one-in-five
// CHANCE fires, and the pirate steals object 255 and announces it with message
// 97 — "¡Te quito _ y me voy a mi entrelazada guarida!" — whose _ is the name
// of an object that does not exist. What a player saw was the hole:
//
//	¡Te quito  y me voy a mi entrelazada guarida!
func TestNoObjectIsSaidInEveryFlagAndNotJustTheNumber(t *testing.T) {
	const room = 19

	m := &Machine{watch: NoWatcher{}, objects: []object{
		{noun: 10, location: LocCarried, weight: 5, container: true, wearable: true},
	}}
	m.setFlag(FlagPlayer, room)
	m.setFlag(FlagAdject1, NoWord)

	// Something the sentence does name, so there is a block to leave behind.
	m.setFlag(FlagNoun1, 10)
	m.whatO()

	require.Equal(t, byte(0), m.Flag(FlagCurObj))
	require.Equal(t, byte(LocCarried), m.Flag(FlagCurObjLoc))
	require.Equal(t, byte(5), m.Flag(FlagCurObjWei))

	// And then one it does not.
	m.setFlag(FlagNoun1, 99)
	m.whatO()

	require.Equal(t, byte(NoWord), m.Flag(FlagCurObj), "the number says there is no object")

	require.Zero(t, m.Flag(FlagCurObjLoc),
		"and so must the location, or the pirate's guard reads the last object's 254")
	require.Zero(t, m.Flag(FlagCurObjWei))
	require.Zero(t, m.Flag(FlagCurObjCon))
	require.Zero(t, m.Flag(FlagCurObjWR))
}

// The graphics half of a character set, which an adventure of 1989 used for
// two things that have to come out of a terminal differently.
//
// A title is the letters drawn larger and says what it spells, so it comes out
// as itself: "#gAVENTURA ORIGINAL I#t" is readable text and the access code
// the first part prints is the whole proof that it was won. A drawing is not
// text at all — "Una llave#g#f,-#t" is the key and then two cells of a picture
// of one — and letting those through as themselves puts a comma and a hyphen
// in the middle of a sentence where the author put a drawing.
//
// What tells them apart is the case, measured and not supposed: of the 84
// distinct graphics runs in the five Aventuras AD adventures, across every
// edition, 56 are words in capitals and the other 28 are one or two characters
// of lower case, digits or punctuation. The display face is a capitals-only
// font, so the slots a capital never needs were free for the sprites.
func TestTheGraphicsHalfIsEitherALetterOrADrawing(t *testing.T) {
	for _, c := range []struct {
		name, text, want string
	}{
		{
			name: "a drawing beside an object's name",
			text: "Una llave#g#f,-#t",
			want: "Una llave [UDG:44][UDG:45]",
		},
		{
			name: "a title, which is the letters and says so",
			text: "#gAVENTURA ORIGINAL I#t",
			want: "AVENTURA ORIGINAL I",
		},
		{
			name: "the access code, for the same reason",
			text: "   CLAVE DE ACCESO: #gTIMACUS#t",
			want: "   CLAVE DE ACCESO: TIMACUS",
		},
		{
			name: "lower case in the graphics half is a sprite, not a word",
			text: "Una varita mágica#g#fef#t",
			want: "Una varita mágica [UDG:101][UDG:102]",
		},
		{
			name: "and the other escapes are what they were",
			text: "dos#nlíneas",
			want: "dos\nlíneas",
		},
		{
			name: "an escape this does not know is left alone",
			text: "#z",
			want: "#z",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, expand(c.text))
		})
	}
}

// The dwarf's axe, which is the bug this was found as.
//
// Location 255 is not a location: it is the word for wherever the player is
// standing. An object sent there has to land in the room the player is in,
// because that is the only place the rest of the machine will look for it —
// listObjects names what holds the room's own number and get picks up what
// holds the room's own number, and 255 is neither.
//
// It was being stored as it came, which is what both references do: msx2daad
// at _internal_placeObject, daad_condacts.c:879, and NextDAAD by calling it
// runtime error 2 outright. The interpreter of 1989 cannot have done either.
// The second part of La Aventura Original has the dwarf Maluva throw his axe
// and leave it behind with "PLACE 14 255" so that the player can pick it up
// and throw it back at him, and the other half of that is written out in full:
//
//	> BAÑAR HACHA  CARRIED 14 / ZERO 119 / EQ 118 2 / CHANCE 50
//	               MESSAGE 90      "¡Has fallado! El hacha cae al suelo."
//	               PLACE 14 255
//	> BAÑAR HACHA  CARRIED 14 / EQ 118 2
//	               MESSAGE 91      "¡Le has cortado la cabeza!"
//	               DESTROY 14
//
// With the axe unreachable, CARRIED 14 is never true and all of that — the
// fifty-fifty, the latch in flag 119, the count of dead dwarves in 120 — was
// dead code the day the game shipped. Three of the five adventures use the
// idiom, thirteen times between them.
func TestAnObjectSentHereLandsWhereThePlayerIs(t *testing.T) {
	const (
		room = 47
		axe  = 0
	)

	for _, c := range []struct {
		name string
		put  func(m *Machine)
	}{
		{
			name: "PLACE, which is how the dwarf drops it",
			put:  func(m *Machine) { m.moveObject(axe, LocHere) },
		},
		{
			name: "and PUTO, the other one with a location for an operand",
			put: func(m *Machine) {
				m.setFlag(FlagCurObj, axe)
				m.moveObject(int(m.Flag(FlagCurObj)), LocHere)
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := &Machine{
				watch:   NoWatcher{},
				console: neverAnswers{},
				objects: []object{{noun: 10, location: LocNotCreated}},
				texts:   texts{system: systemMessages()},
			}
			m.setFlag(FlagPlayer, room)
			m.setFlag(FlagStrength, 255)
			m.setFlag(FlagMaxCarried, 5)

			c.put(m)

			where, _ := m.ObjectAt(axe)
			require.Equal(t, byte(room), where,
				"what is left on the floor holds the room's own number, not 255")

			// Which is what makes the rest of it work: asking still says yes…
			require.True(t, m.at(axe, LocHere), "ISAT 14 255 has to go on being true")
			require.True(t, m.present(axe))

			// …and the axe can be picked up, which is the whole point.
			m.get(axe)

			where, _ = m.ObjectAt(axe)
			require.Equal(t, byte(LocCarried), where, "and then the player can throw it back")
		})
	}
}

// systemMessages is enough of the table for the refusals get prints.
func systemMessages() []string {
	const most = 40

	out := make([]string, most)
	for i := range out {
		out[i] = "system message"
	}

	return out
}

// neverAnswers is a player who says nothing, ever.
type neverAnswers struct{}

func (neverAnswers) Print(string) {}
func (neverAnswers) AnyKey()      {}

func (neverAnswers) Input() (string, bool) { return "", true }

func (neverAnswers) InputWithin(time.Duration) (string, bool, bool) { return "", true, true }
