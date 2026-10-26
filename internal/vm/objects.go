package vm

import "strings"

// What an adventure does with objects, which is most of what it does.
//
// These follow daad_condacts.c and daad_objects.c. The system messages they
// print are the ones the 1991 manual names in its description of each condact,
// and the numbers are not ours to choose: an adventure writes its own words
// into those slots and expects them back.
const (
	msgAlreadyHave  = 25 // "I already have the _."
	msgNoneHere     = 26 // "There isn't one of those here."
	msgTooMany      = 27 // "I can't carry any more things."
	msgCanSee       = 1  // "I can also see:"
	msgNowHave      = 36 // "I now have the _."
	msgDropped      = 39 // "I have dropped the _."
	msgTooHeavy     = 43 // "The _ weighs too much for me."
	msgNothingThere = 53 // "Nothing."
)

// carrying counts what the player holds, not counting what is worn. It is what
// flag 1 is supposed to hold, and recounting beats trusting the flag.
func (m *Machine) carrying() int {
	n := 0

	for _, o := range m.objects {
		if o.location == LocCarried {
			n++
		}
	}

	return n
}

// weightCarried is everything held and worn together, which is what the
// strength limit is measured against.
func (m *Machine) weightCarried() int {
	total := 0

	for _, o := range m.objects {
		if o.location == LocCarried || o.location == LocWorn {
			total += int(o.weight)
		}
	}

	return total
}

// note records an object as the one most recently referred to, filling in the
// flags that describe it. Every condact that touches an object ends this way.
func (m *Machine) note(object int) {
	m.setFlag(FlagCurObj, byte(object))

	if object < 0 || object >= len(m.objects) {
		return
	}

	o := m.objects[object]
	m.setFlag(FlagCurObjLoc, o.location)
	m.setFlag(FlagCurObjWei, o.weight)
	m.setFlag(FlagCurObjCon, boolFlag(o.container))
	m.setFlag(FlagCurObjWR, boolFlag(o.wearable))
}

func boolFlag(b bool) byte {
	if b {
		return 128
	}

	return 0
}

// get is GET and the body of AUTOG: pick an object up, with the four refusals
// the manual lists, in the order it lists them.
func (m *Machine) get(object int) {
	if object < 0 || object >= len(m.objects) {
		m.say(m.texts.system, msgNoneHere, true)
		m.fail()

		return
	}

	m.note(object)

	switch at := m.objects[object].location; at {
	case LocCarried, LocWorn:
		m.say(m.texts.system, msgAlreadyHave, true)
		m.fail()

		return
	default:
		if at != m.flags[FlagPlayer] {
			m.say(m.texts.system, msgNoneHere, true)
			m.fail()

			return
		}
	}

	if m.weightCarried()+int(m.objects[object].weight) > int(m.flags[FlagStrength]) {
		m.say(m.texts.system, msgTooHeavy, true)
		m.fail()

		return
	}

	if m.carrying() >= int(m.flags[FlagMaxCarried]) {
		m.say(m.texts.system, msgTooMany, true)
		m.fail()

		return
	}

	m.move(object, LocCarried)
	m.setFlag(FlagCarried, byte(m.carrying()))
	m.say(m.texts.system, msgNowHave, true)
	m.did()
}

// drop puts an object down where the player is.
func (m *Machine) drop(object int) {
	if object < 0 || object >= len(m.objects) {
		m.fail()

		return
	}

	m.note(object)
	m.move(object, m.flags[FlagPlayer])
	m.setFlag(FlagCarried, byte(m.carrying()))
	m.say(m.texts.system, msgDropped, true)
	m.did()
}

// dropAll is DROPALL: everything held or worn goes to the floor, with nothing
// said about any of it.
func (m *Machine) dropAll() {
	for i, o := range m.objects {
		if o.location == LocCarried || o.location == LocWorn {
			m.move(i, m.flags[FlagPlayer])
		}
	}

	m.setFlag(FlagCarried, 0)
	m.did()
}

// whatO is WHATO: work out which object the sentence is talking about, and
// leave it in flag 51.
//
// The noun alone is not enough when two objects share it — a bottle and a
// bottle of water — so an adjective narrows it. What decides between the rest
// is nearness: something carried or worn answers before something merely lying
// in the room, which is why "drop bottle" never reaches for the one on the
// floor.
func (m *Machine) whatO() {
	noun, adjective := m.flags[FlagNoun1], m.flags[FlagAdject1]

	best, bestRank := -1, 0

	for i, o := range m.objects {
		if o.noun != noun {
			continue
		}

		if adjective != NoWord && o.adjective != adjective && o.adjective != NoWord {
			continue
		}

		rank := 0

		switch o.location {
		case LocCarried, LocWorn:
			rank = 3
		case m.flags[FlagPlayer]:
			rank = 2
		default:
			rank = 1
		}

		if rank > bestRank {
			best, bestRank = i, rank
		}
	}

	if best < 0 {
		m.setFlag(FlagCurObj, NoWord)

		return
	}

	m.note(best)
}

// listObjects is LISTOBJ and LISTAT: name everything in a place.
//
// What it prints is the objects' own descriptions run together with commas,
// which is what the interpreters do and why an object's text reads as a
// fragment rather than a sentence.
func (m *Machine) listObjects(where byte, header bool) {
	var found []string

	for i, o := range m.objects {
		if o.location != where {
			continue
		}

		if i < len(m.texts.objects) {
			found = append(found, strings.TrimSpace(expand(m.texts.objects[i])))
		}
	}

	if len(found) == 0 {
		if header {
			m.say(m.texts.system, msgNothingThere, true)
		}

		return
	}

	if header {
		m.say(m.texts.system, msgCanSee, false)
	}

	m.console.Print(strings.Join(found, ", ") + ".\n")
	m.did()
}
