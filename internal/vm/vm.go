// Package vm runs a DAAD database.
//
// It is an interpreter, not an emulator: it runs the adventure rather than a
// machine that runs the adventure. What it follows is msx2daad's engine, which
// is DAAD written in C by Natalia Pujol — work/msx2daad/src/daad_condacts.c and
// its neighbours — with NextDAAD's assembler as the tie-breaker where the C is
// unclear. The JavaScript port is not used as a source: it has bugs of its own,
// so it can confirm a reading and never decide one.
//
// One thing is done differently from every interpreter of the period, and on
// purpose. They walk the database in memory, so a process is a pair of pointers
// and a jump is pointer arithmetic. Here the database arrives already decoded —
// ddbfile.Reader hands back entries holding condacts holding operands — so a
// position is a pair of indices instead. It is the same machine and it cannot
// read off the end of a block.
//
// Nothing here knows what a terminal is. Output goes to a Console and input
// comes from one, so the same machine serves a person typing, a test asserting,
// and a program driving it. What it does as it goes is reported to a Watcher,
// which is what a debugger listens to.
package vm

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
)

// The flags DAAD gives a meaning to. The numbers are the format's and are not
// ours to choose; the names are the ones the 1991 manual uses.
//
// Only the ones this package needs by name are here. The rest are flags like
// any other and the adventure decides what they mean.
const (
	FlagDark       = 0  // non-zero when the game is dark; see also object 0
	FlagCarried    = 1  // how many objects the player is carrying, not counting worn
	FlagDarkF      = 28 // what the loop sets after working out whether it is dark here
	FlagGraphics   = 29 // what the machine can do, which here is nothing
	FlagScore      = 30
	FlagTurns      = 31 // two bytes, little-endian, with 32
	FlagVerb       = 33 // the verb of the sentence being answered
	FlagNoun1      = 34
	FlagAdject1    = 35
	FlagAdverb     = 36
	FlagMaxCarried = 37 // how many objects can be carried at once
	FlagPlayer     = 38 // where the player is
	FlagInStream   = 41
	FlagPrompt     = 42
	FlagPrep       = 43
	FlagNoun2      = 44
	FlagAdject2    = 45
	FlagCPNoun     = 46 // the noun "it" currently stands for
	FlagCPAdject   = 47
	FlagTime       = 48
	FlagTimeFlags  = 49
	FlagDoallObj   = 50 // the object a DOALL loop is on
	FlagCurObj     = 51 // the last object GET, DROP, WEAR and friends touched
	FlagStrength   = 52 // the most weight the player can carry
	FlagObjFlags   = 53
	FlagCurObjLoc  = 54
	FlagCurObjWei  = 55
	FlagCurObjCon  = 56
	FlagCurObjWR   = 57
	FlagCurObjAtt  = 58
	FlagKey1       = 60
	FlagKey2       = 61
	FlagScreenMode = 62
	FlagCurWindow  = 63
)

// The markers that stand where a location would, which are the same four the
// format uses everywhere. They are repeated rather than imported so that this
// package reads without one.
const (
	LocNotCreated = 252
	LocWorn       = 253
	LocCarried    = 254
	LocHere       = 255
)

// NoWord is the word value that matches anything, in an entry and in a
// sentence alike.
const NoWord = 0xFF

// Console is everything the machine needs of the world outside it.
//
// It is small on purpose. A console interpreter, a test and a program driving
// the adventure from somewhere else differ in what they do with a line of
// text, not in what the machine asks of them.
type Console interface {
	// Print writes text where the player will see it. The machine has already
	// turned tokens and escapes into characters; what arrives is what should
	// appear, with "\n" for a new line.
	Print(text string)
	// Input asks the player for a line and gives back what was typed. The
	// second result is false when there is no more input, which ends the game
	// as surely as QUIT does.
	Input() (string, bool)
	// AnyKey waits for a keypress. A driver with nobody at the keyboard
	// returns straight away.
	AnyKey()
}

// Watcher is told what the machine does, for a debugger, a test or a
// transcript to listen to. Every method may be left empty.
//
// It is reported to rather than asked, and it cannot change what happens: a
// debugger that can alter the run by observing it is not showing the run.
type Watcher interface {
	// Condact is called before each one runs, with the process and entry it
	// belongs to.
	Condact(process, entry int, c ddbfile.Condact)
	// Flag is called when one changes, with what it held before.
	Flag(number int, was, now byte)
	// Process is called when one is pushed, and again with running false when
	// it is popped.
	Process(number int, running bool)
	// Moved is called when an object changes place.
	Moved(object int, from, to byte)
}

// NoWatcher ignores everything, which is what a run nobody is watching wants.
type NoWatcher struct{}

func (NoWatcher) Condact(int, int, ddbfile.Condact) {}
func (NoWatcher) Flag(int, byte, byte)              {}
func (NoWatcher) Process(int, bool)                 {}
func (NoWatcher) Moved(int, byte, byte)             {}

// object is one object's mutable state. Its name and weight come from the
// database and do not change; where it is does.
type object struct {
	location  byte
	weight    byte
	container bool
	wearable  bool
	noun      byte
	adjective byte
}

// frame is one process on the stack: which one, how far through it, and what a
// DOALL running inside it has to come back to.
//
// The C keeps pointers into the database for all of this. Indices say the same
// thing and cannot point between two condacts.
type frame struct {
	process int
	entry   int // which entry of that process
	condact int // which condact of that entry

	// running says whether the entry this frame points at has been matched
	// against the sentence and is being executed. A frame that is not running
	// has to find its next matching entry before anything happens, which is
	// what makes a process start at the first entry that answers rather than
	// at the first entry.
	//
	// It is the C's checkEntry, and leaving it out is not a small mistake: a
	// process would run its entry zero whatever the player typed. In La
	// Aventura Original that is "> AD _ MESSAGE 139", so every command
	// answered with the list of people who also worked on the game.
	running bool

	// doall is where a DOALL in this process sends execution back to, and -1
	// in the condact when there is no loop running. doallWhere is the place
	// it is going through, which the operand named and nothing else records.
	doallEntry   int
	doallCondact int
	doallWhere   byte
}

// Machine runs one database.
type Machine struct {
	db *ddbfile.Reader

	processes [][]ddbfile.Entry
	texts     texts
	objects   []object
	conns     [][]ddbfile.Movement
	vocab     []ddbfile.VocabularyEntry

	flags [256]byte
	stack []frame

	console Console
	watch   Watcher

	// done is the "something happened" that ISDONE asks about, and lastDone
	// what it held when the process it belongs to started.
	done     bool
	lastDone bool
	// continuing is false once a condition in the current entry has failed,
	// which is how an entry gives up without ending the process.
	continuing bool
	// finished is set by QUIT and by running out of input, and is what stops
	// the machine rather than any process returning.
	finished bool
	// restart asks the loop to begin again from process zero.
	restart bool
	// inputs is how many lines the player has given, which is the only measure
	// of whether a turn got anywhere. See Run.
	inputs int

	// indir carries what an INDIR condact left for the one after it.
	//
	// Bit 7 of an opcode makes the first operand indirect and there is nowhere
	// in the encoding to say the same of the second, so version 3 does it with
	// a condact: INDIR names a flag, and the interpreter writes that flag's
	// value over the second operand of whatever comes next. In a database that
	// is literally a poke into the process block, *(pPROC + 2) — see
	// do_INDIR, daad_condacts.c:2495. Here the condacts arrive decoded, so
	// what the poke would have changed is held until the next one runs.
	indir struct {
		pending bool
		value   byte
	}
	// spanish is what the database was built for, which decides how an
	// object's article is read out. See objectName.
	spanish bool
}

// texts is the four tables, read once.
type texts struct {
	system    []string
	messages  []string
	objects   []string
	locations []string
}

// New reads a database and gets it ready to run.
func New(r *ddbfile.Reader, console Console, watch Watcher) (*Machine, error) {
	if watch == nil {
		watch = NoWatcher{}
	}

	m := &Machine{db: r, console: console, watch: watch}

	var err error

	if m.processes, err = r.Processes(); err != nil {
		return nil, fmt.Errorf("reading the processes: %w", err)
	}

	if m.conns, err = r.Connections(); err != nil {
		return nil, fmt.Errorf("reading the connections: %w", err)
	}

	if m.texts.system, err = r.SystemMessages(); err != nil {
		return nil, fmt.Errorf("reading the system messages: %w", err)
	}

	if m.texts.messages, err = r.Messages(); err != nil {
		return nil, fmt.Errorf("reading the messages: %w", err)
	}

	if m.texts.objects, err = r.ObjectTexts(); err != nil {
		return nil, fmt.Errorf("reading the object texts: %w", err)
	}

	if m.texts.locations, err = r.LocationTexts(); err != nil {
		return nil, fmt.Errorf("reading the location texts: %w", err)
	}

	objects, err := r.Objects()
	if err != nil {
		return nil, fmt.Errorf("reading the objects: %w", err)
	}

	m.objects = make([]object, len(objects))
	for i, o := range objects {
		m.objects[i] = object{
			location: o.InitialLocation, weight: o.Weight,
			container: o.Container, wearable: o.Wearable,
			noun: o.Noun, adjective: o.Adjective,
		}
	}

	m.vocab = r.Vocabulary()

	const spanish = 1

	m.spanish = r.Language() == spanish

	return m, nil
}

// Run starts the adventure and returns when it ends.
//
// What that means is the whole of the main loop, which in DAAD is four lines:
// clear the flags, empty the process stack, push process zero and run it. Every
// game loop a player would recognise is written in the database rather than
// here — see docs/DAAD_v2_v3_Research/07-daad-v3.md section 11.
func (m *Machine) Run() error {
	m.reset()

	// Turns that have gone round without the player saying anything. A RESTART
	// is how the loop asks for another go, and the one thing it cannot do is
	// ask forever: a turn that restarts without having read a line has changed
	// nothing that could make the next turn end differently.
	//
	// Left unguarded that is not a hang but a flood, because every one of those
	// turns prints the location first. One of them wrote 59 GB to a file that
	// had already been deleted, so nothing on disk showed it and only the
	// process holding it open did.
	spins := 0

	const spinLimit = 1000

	for !m.finished {
		before := m.inputs

		m.restart = false
		m.stack = m.stack[:0]
		m.push(0)

		if err := m.runStack(); err != nil {
			return err
		}

		// Only a RESTART asks for another turn of this loop. Anything else
		// that empties the stack has ended the game.
		if !m.restart {
			return nil
		}

		if m.inputs > before {
			spins = 0

			continue
		}

		if spins++; spins > spinLimit {
			return fmt.Errorf("the adventure restarted %d times without reading a line: "+
				"it is looping, and every turn of it prints", spins)
		}
	}

	return nil
}

// reset puts the machine where initFlags and initializePROC leave it.
func (m *Machine) reset() {
	m.flags = [256]byte{}
	m.stack = m.stack[:0]
	m.done, m.lastDone, m.finished = false, false, false

	// A half-finished indirection belongs to the condact that was going to
	// consume it, and that condact is not going to run now. inputs is left
	// alone on purpose: it counts the player, not the game, and the loop's
	// guard against restarting forever needs it to carry across.
	m.indir.pending = false

	// What the machine can do. A console has no graphics and no mouse, and
	// saying so is better than leaving an adventure to find out.
	m.flags[FlagGraphics] = 0
}

func (m *Machine) push(process int) {
	m.watch.Process(process, true)

	// Entry -1 and not 0: the first thing the process does is step forward to
	// an entry that matches, and it has to be able to land on entry zero.
	m.stack = append(m.stack, frame{
		process: process, entry: -1, condact: 0,
		doallEntry: -1, doallCondact: -1,
	})
}

func (m *Machine) pop() {
	if len(m.stack) == 0 {
		return
	}

	m.watch.Process(m.top().process, false)
	m.stack = m.stack[:len(m.stack)-1]
}

func (m *Machine) top() *frame { return &m.stack[len(m.stack)-1] }

// holds says whether f is still the frame on top.
//
// It cannot be a comparison against top(), because the thing it is asked about
// is a condact having changed the stack, and one of the ways it can change it
// is to empty it: the outermost process is allowed to return. La Aventura
// Original's second part does exactly that on its first turn, and asking for
// the frame afterwards read off the end of the slice and crashed.
func (m *Machine) holds(f *frame) bool {
	return len(m.stack) > 0 && m.top() == f
}

// setFlag is the only way a flag changes, so that a watcher sees every one.
func (m *Machine) setFlag(number int, value byte) {
	was := m.flags[number&0xFF]
	m.flags[number&0xFF] = value

	if was != value {
		m.watch.Flag(number&0xFF, was, value)
	}
}

// move is the only way an object changes place, for the same reason.
func (m *Machine) move(object int, to byte) {
	if object < 0 || object >= len(m.objects) {
		return
	}

	from := m.objects[object].location
	m.objects[object].location = to

	if from != to {
		m.watch.Moved(object, from, to)
	}
}
