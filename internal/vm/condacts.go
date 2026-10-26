package vm

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
)

// The condacts, which are the instruction set.
//
// Each one is a method taking its operands already resolved. Resolving them is
// exec's job and not each condact's, because indirection is a property of the
// opcode byte rather than of what it does: bit 7 says the first operand names a
// flag and the flag's content is the real operand. Doing it once here is what
// keeps a hundred and thirty condacts from each getting it slightly wrong.
//
// A condact that is a condition fails by calling m.fail, which stops the entry
// without ending the process. Anything that "does something" says so with
// m.did, which is what ISDONE asks about afterwards.

// fail gives up on the current entry. The process carries on with the next one.
func (m *Machine) fail() { m.continuing = false }

// did records that this entry acted, which is what ISDONE reports.
func (m *Machine) did() { m.done = true }

// need is a condition: it fails the entry unless the test holds.
func (m *Machine) need(ok bool) {
	if !ok {
		m.fail()
	}
}

// exec runs one condact.
func (m *Machine) exec(c ddbfile.Condact) error {
	def, found := m.db.Condact(c.Opcode)
	if !found {
		return fmt.Errorf("opcode %d is not a condact", c.Opcode)
	}

	// Indirection reaches the first operand only, and turns it into the
	// content of the flag it names.
	args := make([]byte, len(c.Params))
	copy(args, c.Params)

	if c.Indirect && len(args) > 0 {
		args[0] = m.flags[args[0]]
	}

	// And an INDIR in front reaches the second, which is the one bit 7 of the
	// opcode cannot say anything about. See indir.
	if m.indir.pending {
		m.indir.pending = false

		if len(args) > 1 {
			args[1] = m.indir.value
		}
	}

	run, known := condacts[c.Opcode]
	if !known {
		// Saying so is better than doing nothing quietly: an adventure that
		// leans on a condact we have not written would otherwise look like it
		// merely behaves oddly.
		return fmt.Errorf("%s is not implemented yet", def.Name)
	}

	run(m, args)

	return nil
}

// arg is an operand that may not be there, for the condacts whose arity
// changed between the versions.
func arg(args []byte, i int) byte {
	if i < len(args) {
		return args[i]
	}

	return 0
}

// condacts maps an opcode to what it does. The numbers are the format's; see
// docs/DAAD_v2_v3_Research/04-condactos.md for the whole table.
var condacts = map[byte]func(*Machine, []byte){
	// ---- conditions about where the player is ----
	0:   func(m *Machine, a []byte) { m.need(m.flags[FlagPlayer] == a[0]) },    // AT
	1:   func(m *Machine, a []byte) { m.need(m.flags[FlagPlayer] != a[0]) },    // NOTAT
	2:   func(m *Machine, a []byte) { m.need(m.flags[FlagPlayer] > a[0]) },     // ATGT
	3:   func(m *Machine, a []byte) { m.need(m.flags[FlagPlayer] < a[0]) },     // ATLT
	4:   func(m *Machine, a []byte) { m.need(m.present(int(a[0]))) },           // PRESENT
	5:   func(m *Machine, a []byte) { m.need(!m.present(int(a[0]))) },          // ABSENT
	6:   func(m *Machine, a []byte) { m.need(m.at(int(a[0]), LocWorn)) },       // WORN
	7:   func(m *Machine, a []byte) { m.need(!m.at(int(a[0]), LocWorn)) },      // NOTWORN
	8:   func(m *Machine, a []byte) { m.need(m.at(int(a[0]), LocCarried)) },    // CARRIED
	9:   func(m *Machine, a []byte) { m.need(!m.at(int(a[0]), LocCarried)) },   // NOTCARR
	10:  func(m *Machine, a []byte) { m.need(rand.Intn(100) < int(a[0])) },     //nolint:gosec // CHANCE
	11:  func(m *Machine, a []byte) { m.need(m.flags[a[0]] == 0) },             // ZERO
	12:  func(m *Machine, a []byte) { m.need(m.flags[a[0]] != 0) },             // NOTZERO
	13:  func(m *Machine, a []byte) { m.need(m.flags[a[0]] == a[1]) },          // EQ
	14:  func(m *Machine, a []byte) { m.need(m.flags[a[0]] > a[1]) },           // GT
	15:  func(m *Machine, a []byte) { m.need(m.flags[a[0]] < a[1]) },           // LT
	16:  func(m *Machine, a []byte) { m.need(m.flags[FlagAdject1] == a[0]) },   // ADJECT1
	17:  func(m *Machine, a []byte) { m.need(m.flags[FlagAdverb] == a[0]) },    // ADVERB
	55:  func(m *Machine, a []byte) { m.need(m.at(int(a[0]), a[1])) },          // ISAT
	68:  func(m *Machine, a []byte) { m.need(m.flags[FlagPrep] == a[0]) },      // PREP
	69:  func(m *Machine, a []byte) { m.need(m.flags[FlagNoun2] == a[0]) },     // NOUN2
	70:  func(m *Machine, a []byte) { m.need(m.flags[FlagAdject2] == a[0]) },   // ADJECT2
	76:  func(m *Machine, a []byte) { m.need(m.flags[a[0]] == m.flags[a[1]]) }, // SAME
	79:  func(m *Machine, a []byte) { m.need(m.flags[a[0]] != a[1]) },          // NOTEQ
	80:  func(m *Machine, a []byte) { m.need(m.flags[a[0]] != m.flags[a[1]]) }, // NOTSAME
	88:  func(m *Machine, a []byte) { m.need(!m.at(int(a[0]), a[1])) },         // ISNOTAT
	112: func(m *Machine, a []byte) { m.need(m.flags[a[0]] > m.flags[a[1]]) },  // BIGGER
	113: func(m *Machine, a []byte) { m.need(m.flags[a[0]] < m.flags[a[1]]) },  // SMALLER
	114: func(m *Machine, _ []byte) { m.need(m.done) },                         // ISDONE
	115: func(m *Machine, _ []byte) { m.need(!m.done) },                        // ISNDONE

	// ---- flags ----
	47:  func(m *Machine, a []byte) { m.setFlag(int(a[0]), 255); m.did() },                         // SET
	48:  func(m *Machine, a []byte) { m.setFlag(int(a[0]), 0); m.did() },                           // CLEAR
	49:  func(m *Machine, a []byte) { m.setFlag(int(a[0]), m.flags[a[0]]+a[1]); m.did() },          // PLUS
	50:  func(m *Machine, a []byte) { m.setFlag(int(a[0]), m.flags[a[0]]-a[1]); m.did() },          // MINUS
	51:  func(m *Machine, a []byte) { m.setFlag(int(a[0]), a[1]); m.did() },                        // LET
	71:  func(m *Machine, a []byte) { m.setFlag(int(a[1]), m.flags[a[1]]+m.flags[a[0]]); m.did() }, // ADD
	72:  func(m *Machine, a []byte) { m.setFlag(int(a[1]), m.flags[a[1]]-m.flags[a[0]]); m.did() }, // SUB
	95:  func(m *Machine, a []byte) { m.setFlag(int(a[0]), byte(rand.Intn(100)+1)); m.did() },      //nolint:gosec // RANDOM
	125: func(m *Machine, a []byte) { m.setFlag(int(a[1]), m.flags[a[0]]); m.did() },               // COPYFF
	126: func(m *Machine, a []byte) { m.setFlag(int(a[1]), 255-m.flags[a[0]]); m.did() },           // COPYBF

	// ---- where things are ----
	37: func(m *Machine, a []byte) { m.setFlag(FlagPlayer, a[0]); m.did() },         // GOTO
	39: func(m *Machine, a []byte) { m.moveObject(int(a[0]), LocNotCreated) },       // REMOVE... see below
	43: func(m *Machine, a []byte) { m.moveObject(int(a[0]), LocNotCreated) },       // DESTROY
	44: func(m *Machine, a []byte) { m.moveObject(int(a[0]), m.flags[FlagPlayer]) }, // CREATE
	46: func(m *Machine, a []byte) { m.moveObject(int(a[0]), a[1]) },                // PLACE
	45: func(m *Machine, a []byte) { // SWAP
		if int(a[0]) < len(m.objects) && int(a[1]) < len(m.objects) {
			one, two := m.objects[a[0]].location, m.objects[a[1]].location
			m.move(int(a[0]), two)
			m.move(int(a[1]), one)
		}

		m.did()
	},
	106: func(m *Machine, a []byte) { m.need(m.movePlayer(int(a[0]))) },              // MOVE
	102: func(m *Machine, a []byte) { m.moveObject(int(m.flags[FlagCurObj]), a[0]) }, // PUTO

	// ---- text ----
	52: func(m *Machine, _ []byte) { m.console.Print("\n") },                      // NEWLINE
	57: func(m *Machine, _ []byte) { m.console.Print(" ") },                       // SPACE
	53: func(m *Machine, a []byte) { m.console.Print(fmt.Sprint(m.flags[a[0]])) }, // PRINT
	27: func(m *Machine, a []byte) { // DPRINT, a two-byte number from a flag pair
		m.console.Print(fmt.Sprint(int(m.flags[a[0]]) + 256*int(m.flags[a[0]+1])))
	},
	38: func(m *Machine, a []byte) { m.say(m.texts.messages, int(a[0]), true); m.did() },  // MESSAGE
	77: func(m *Machine, a []byte) { m.say(m.texts.messages, int(a[0]), false); m.did() }, // MES
	54: func(m *Machine, a []byte) { m.say(m.texts.system, int(a[0]), false); m.did() },   // SYSMESS
	19: func(m *Machine, a []byte) { // DESC
		m.say(m.texts.locations, int(arg(a, 0)), true)
		m.did()
	},

	// ---- the turn and the game ----
	22:  func(m *Machine, _ []byte) { m.did(); m.pop() },                                  // DONE
	23:  func(m *Machine, _ []byte) { m.say(m.texts.system, 15, true); m.did(); m.pop() }, // OK
	103: func(m *Machine, _ []byte) { m.done = false; m.pop() },                           // NOTDONE
	108: func(m *Machine, _ []byte) { m.redo() },                                          // REDO
	117: func(m *Machine, _ []byte) { m.restart = true },                                  // RESTART
	116: func(m *Machine, a []byte) { m.skip(int(int8(a[0]))) },                           // SKIP
	75:  func(m *Machine, a []byte) { m.push(int(a[0])) },                                 // PROCESS
	20:  func(m *Machine, _ []byte) { m.need(m.confirm(12)) },                             // QUIT
	21:  func(m *Machine, _ []byte) { m.end() },                                           // END
	24:  func(m *Machine, _ []byte) { m.console.AnyKey() },                                // ANYKEY
	// PARSE carries on with the entry when the sentence made no sense, which
	// is the opposite way round from how it reads. The loop every adventure
	// writes is "PARSE 0 / PROCESS 2 / REDO", and process 2 is what says "I
	// don't understand" — so the condact has to succeed precisely when there
	// is nothing to answer, and fail when there is.
	73:  func(m *Machine, _ []byte) { m.need(m.parse()) },         // PARSE
	92:  func(m *Machine, _ []byte) {},                            // NEWTEXT: nothing to flush here
	127: func(m *Machine, _ []byte) { m.resetObjects(); m.did() }, // RESET

	58:  func(m *Machine, a []byte) { m.need(m.hasBit(a[0])) },  // HASAT
	59:  func(m *Machine, a []byte) { m.need(!m.hasBit(a[0])) }, // HASNAT
	124: func(m *Machine, a []byte) { m.setBit(a[0], a[1]) },    // SETAT

	85: func(m *Machine, a []byte) { m.doall(a[0]) }, // DOALL

	30: func(m *Machine, _ []byte) { m.dropAll() },                                 // DROPALL
	32: func(m *Machine, _ []byte) { m.whatO(); m.drop(int(m.flags[FlagCurObj])) }, // AUTOD
	40: func(m *Machine, a []byte) { m.get(int(a[0])) },                            // GET
	41: func(m *Machine, a []byte) { m.drop(int(a[0])) },                           // DROP
	31: func(m *Machine, _ []byte) { m.whatO(); m.get(int(m.flags[FlagCurObj])) },  // AUTOG
	60: func(m *Machine, _ []byte) { m.listObjects(m.flags[FlagPlayer], true) },    // LISTOBJ
	74: func(m *Machine, a []byte) { m.listObjects(a[0], false) },                  // LISTAT
	93: func(m *Machine, a []byte) { // ABILITY: how much can be carried, and how heavy
		m.setFlag(FlagMaxCarried, a[0])
		m.setFlag(FlagStrength, a[1])
	},
	100: func(m *Machine, _ []byte) { m.whatO() }, // WHATO
	35:  nothing,                                  // PAUSE: a console has no frames to wait for

	// SETCO: make an object the one the messages mean by "_", without moving
	// it or saying anything. do_SETCO is one line — referencedObject — and
	// notably does not mark anything done.
	56: func(m *Machine, a []byte) { m.note(int(a[0])) },

	// EXIT: leave for good on zero, start the game over on anything else.
	// See do_EXIT, daad_condacts.c:2367.
	110: func(m *Machine, a []byte) { m.exit(a[0]) },

	// INDIR: the flag whose value belongs in the next condact's second
	// operand. See the indir field on Machine.
	122: func(m *Machine, a []byte) {
		m.indir.pending = true
		m.indir.value = m.flags[a[0]]
	},

	// ---- things a console has no use for, said plainly rather than omitted ----
	18:  nothing, // SFX
	25:  nothing, // SAVE
	26:  nothing, // LOAD
	28:  nothing, // DISPLAY
	29:  nothing, // CLS: a scrolling console has no screen to clear
	64:  nothing, // BEEP
	65:  nothing, // PAPER
	66:  nothing, // INK
	67:  nothing, // BORDER
	81:  nothing, // MODE
	83:  nothing, // TIME
	84:  nothing, // PICTURE
	86:  nothing, // MOUSE
	87:  nothing, // GFX
	78:  nothing, // WINDOW
	82:  nothing, // WINAT
	107: nothing, // WINSIZE
	109: nothing, // CENTRE
	118: nothing, // TAB
	97:  nothing, // SAVEAT
	98:  nothing, // BACKAT
	99:  nothing, // PRINTAT
	62:  nothing, // RAMSAVE
	63:  nothing, // RAMLOAD
}

// nothing is a condact a console has no use for. It is written out rather than
// left out of the table so that the difference between "does nothing here" and
// "not written yet" stays visible: the second is an error.
func nothing(_ *Machine, _ []byte) {}

// at is whether an object is in a given place.
func (m *Machine) at(object int, where byte) bool {
	return object >= 0 && object < len(m.objects) && m.objects[object].location == where
}

// present is whether an object is somewhere the player can reach: carried,
// worn, or lying in the room.
func (m *Machine) present(object int) bool {
	if object < 0 || object >= len(m.objects) {
		return false
	}

	switch at := m.objects[object].location; at {
	case LocCarried, LocWorn:
		return true
	default:
		return at == m.flags[FlagPlayer]
	}
}

// moveObject is PLACE and its relatives, which all end by noting which object
// was last touched.
func (m *Machine) moveObject(object int, to byte) {
	m.move(object, to)
	m.setFlag(FlagCurObj, byte(object))
	m.did()
}

// movePlayer is MOVE, and its two halves come from different places.
//
// The operand names the flag holding the location to move FROM, and the new
// location is written back into that same flag. The direction is not in the
// operand at all: it is whatever fVerb holds.
//
// Getting that round the wrong way is not a small error and does not look like
// one. Matching the connection against the operand's flag means matching it
// against the location, so "salir" from location 10 found a connection whose
// word happened to be 10 and went to location 0 — and location 0 is where the
// main loop starts a new game, so leaving the cabin looked like the adventure
// ending.
//
// That the flag is the location and not the direction is also what makes AO's
// own exit-lister work: it does "COPYFF 38 202" and then "MOVE 202", walking a
// copy of where the player is while fVerb counts through the directions.
func (m *Machine) movePlayer(flag int) bool {
	from := int(m.flags[flag])
	if from < 0 || from >= len(m.conns) {
		return false
	}

	for _, c := range m.conns[from] {
		if c.Word == m.flags[FlagVerb] {
			m.setFlag(flag, c.To)
			m.did()

			return true
		}
	}

	return false
}

// resetObjects is RESET: every object back where the database says it starts.
func (m *Machine) resetObjects() {
	objects, err := m.db.Objects()
	if err != nil {
		return
	}

	for i, o := range objects {
		if i < len(m.objects) {
			m.move(i, o.InitialLocation)
		}
	}
}

// say prints one text of a table, with a new line after it when the condact is
// the kind that adds one.
func (m *Machine) say(table []string, number int, newline bool) {
	if number < 0 || number >= len(table) {
		return
	}

	text := table[number]
	if newline {
		text += "\n"
	}

	m.console.Print(m.fill(expand(text)))
}

// fill puts the object a message is about where the message leaves room for
// it. See objectName.
func (m *Machine) fill(text string) string {
	if !strings.ContainsAny(text, "_@") {
		return text
	}

	var sb strings.Builder

	for _, r := range text {
		switch r {
		case '_':
			sb.WriteString(m.objectName(false))
		case '@':
			sb.WriteString(m.objectName(true))
		default:
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// expand turns the escapes a source writes into what a console should show.
//
// The text arrives already decoded by internal/charset, so what is left is the
// handful that mean something to a screen rather than to a character set. Two
// of those mean nothing here at all: #g and #t switch to the graphics half of
// a character set and back, which is how an adventure of 1989 drew its titles
// and its little pictures beside an object's name. A console has one set of
// letters, so the markers come out and what is between them stays.
func expand(text string) string {
	return strings.NewReplacer(
		"#n", "\n",
		"#r", "\n",
		"#f", " ", // the forced space, which is a space anywhere with no font
		"#s", " ",
		"#g", "",
		"#t", "",
	).Replace(text)
}

// end is END: ask whether to play again, and either start over or stop.
func (m *Machine) end() {
	if m.confirm(13) {
		m.finished = true

		return
	}

	m.reset()
	m.restart = true
}

// exit is EXIT, which is two condacts wearing one name.
//
// Zero leaves the interpreter altogether — no asking, no new game, the way a
// loading screen's menu quits rather than the way END does. Anything else
// starts the adventure over: the flags go back to nothing, the objects go back
// where they began, and the loop begins again at process zero.
//
// It asks nothing in either case, which is the whole difference from END. See
// do_EXIT, daad_condacts.c:2367.
func (m *Machine) exit(value byte) {
	if value == 0 {
		m.finished = true

		return
	}

	m.reset()
	m.resetObjects()
	m.restart = true

	// The entry it was written in has nothing more to say: whatever follows
	// belongs to the game that just ended.
	m.continuing = false
}

// confirm prints a system message and reads a yes or no, the way QUIT and END
// do. Yes is a reply starting with the first letter of system message 30.
func (m *Machine) confirm(message int) bool {
	m.say(m.texts.system, message, true)

	answer, ok := m.console.Input()
	if !ok {
		m.finished = true

		return true
	}

	// This counts as the player having spoken, so that saying no to "another
	// game?" is not mistaken for a turn that got nowhere.
	m.inputs++

	yes := "S"
	if len(m.texts.system) > 30 && m.texts.system[30] != "" {
		yes = m.texts.system[30]
	}

	answer = strings.TrimSpace(strings.ToUpper(answer))

	return answer != "" && strings.HasPrefix(answer, strings.ToUpper(yes[:1]))
}

// redo starts the current process again from its first entry.
func (m *Machine) redo() {
	f := m.top()
	f.entry, f.condact, f.running = -1, 0, false
	m.continuing = false
}

// skip jumps a number of entries, forwards or back. The operand is signed,
// which is the one place in DAAD where a byte is.
func (m *Machine) skip(by int) {
	f := m.top()

	// The operand is a delta the compiler works out as "target - here - 1", so
	// the entry it means is here + delta + 1. One less than that is set,
	// because what happens next is the step forward that looks for a matching
	// entry and it will add the one back.
	f.entry += by
	f.condact, f.running = 0, false
	m.continuing = false
}

// doallNext is what a DOALL does with the end of its table: go round again
// with the next object, or give up and let the process return.
func (m *Machine) doallNext() bool {
	if m.doallSearch() {
		return true
	}

	m.top().doallCondact = -1

	return false
}

// parse takes the next sentence from the player. It reports whether the
// sentence could not be understood, which is the way round PARSE is used:
// "PARSE 0 / PROCESS 2" runs process 2 when the input made no sense.
func (m *Machine) parse() bool {
	m.prompt()

	line, ok := m.console.Input()
	if !ok {
		m.finished = true

		return true
	}

	m.inputs++

	return !m.sentence(line)
}

// prompt writes what goes in front of the player's line.
//
// Which system message that is lives in a flag, so an adventure can change it
// as it goes — a sinking ship asking differently from a quiet room. Zero means
// the interpreter picks one of the first four at random, which is a flourish
// nobody used; the usual answer is 33.
func (m *Machine) prompt() {
	const defaultPrompt = 33

	which := int(m.flags[FlagPrompt])
	if which == 0 {
		which = defaultPrompt
	}

	m.say(m.texts.system, which, false)
}

// A bit, the way HASAT, HASNAT and SETAT name one.
//
// The operand is not a flag number and a bit number side by side: it is one
// byte that packs both, counting flags downwards from 59. Nothing says so in
// the manual, but msx2daad spells out five of them as constants — timeout is
// flag 49 bit 7 and is written 87, graphics mode is flag 29 bit 7 and is
// written 247 — and this is the only arithmetic that gives all five.
const bitsPerFlag = 8

func bitOf(value byte) (flag, bit int) {
	return FlagCurObjAtt + 1 - int(value)/bitsPerFlag, int(value) % bitsPerFlag
}

func (m *Machine) hasBit(value byte) bool {
	flag, bit := bitOf(value)
	if flag < 0 || flag > 255 {
		return false
	}

	return m.flags[flag]&(1<<bit) != 0
}

// setBit is SETAT, whose second operand says what to do: clear, set, or turn
// the bit over.
func (m *Machine) setBit(value, how byte) {
	flag, bit := bitOf(value)
	if flag < 0 || flag > 255 {
		return
	}

	const (
		clear = 0
		set   = 1
	)

	was := m.flags[flag]

	switch how {
	case clear:
		m.setFlag(flag, was&^(1<<bit))
	case set:
		m.setFlag(flag, was|(1<<bit))
	default:
		m.setFlag(flag, was^(1<<bit))
	}

	m.did()
}
