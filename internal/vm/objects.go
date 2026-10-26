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

// DOALL, which turns one command into one per object.
//
// "coger todo" is not a condact that picks everything up: it is a loop. DOALL
// remembers where it stands, finds an object in the place its operand names,
// puts that object's noun into the sentence, and lets the rest of the entry
// run as though the player had named it. Running off the end of the table
// comes back here for the next one instead of returning.
//
// Two details are the reference's rather than anything one would invent. The
// object named by the second noun is skipped, which is how "coge todo excepto
// la lámpara" works without a word for "except". And the counter starts at
// the null word, so that adding one to it gives object zero.
func (m *Machine) doall(where byte) {
	f := m.top()

	f.doallEntry = f.entry
	f.doallCondact = f.condact
	f.doallWhere = where

	m.setFlag(FlagDoallObj, NoWord)

	if m.doallSearch() {
		return
	}

	// Nothing to do it to, which is not the same as having finished: the
	// entry never ran even once, so nothing was done and the sentence is
	// cleared rather than answered.
	f.doallCondact = -1
	m.newText()
	m.done = false
	m.continuing = false
	m.pop()
}

// doallSearch finds the next object of the loop and puts it in the sentence.
// It reports whether there was one.
func (m *Machine) doallSearch() bool {
	f := m.top()

	where := f.doallWhere
	if where == LocHere {
		where = m.flags[FlagPlayer]
	}

	// Adding one to the null word is how the first time round starts at
	// object zero, so it has to wrap the way a byte does.
	const wrap = 0x100

	for n := (int(m.flags[FlagDoallObj]) + 1) % wrap; n < len(m.objects); n++ {
		o := m.objects[n]

		if o.location != where {
			continue
		}

		// The one the player asked to leave out.
		if o.noun == m.flags[FlagNoun2] && o.adjective == m.flags[FlagAdject2] {
			continue
		}

		m.setFlag(FlagDoallObj, byte(n))
		m.setFlag(FlagNoun1, o.noun)
		m.setFlag(FlagAdject1, o.adjective)

		f.entry = f.doallEntry
		f.condact = f.doallCondact
		f.running = true

		return true
	}

	return false
}

// newText is NEWTEXT: forget the rest of the sentence, so that what the player
// typed after this point is not answered twice.
func (m *Machine) newText() {
	for _, flag := range []int{
		FlagVerb, FlagNoun1, FlagAdject1, FlagAdverb,
		FlagPrep, FlagNoun2, FlagAdject2,
	} {
		m.setFlag(flag, NoWord)
	}
}

// The object a message is about, which is what "_" and "@" stand for.
//
// Both name the object flag 51 holds, and the only difference between them is
// the capital: "@" starts a sentence and "_" does not. What makes it more than
// a substitution is that the article changes with it — an object is written
// down indefinite, "Una linterna", and read out definite, "la linterna" — so
// "Has cogido _." comes out as "Has cogido la linterna.".
//
// Spanish changes the article and English drops it, which is why this needs to
// know which language the database was built for. See
// daad_msg.c:107-136, printObjectMsgModif.
func (m *Machine) objectName(capital bool) string {
	n := int(m.flags[FlagCurObj])
	if n < 0 || n >= len(m.texts.objects) {
		return ""
	}

	// A name is read into a sentence, so whatever ended the line it was
	// written on is not part of it.
	name := strings.TrimRight(expand(m.texts.objects[n]), " .\n")

	switch article, rest := splitArticle(name, m.spanish); {
	case article == "":
		return rest
	case capital:
		return strings.ToUpper(article[:1]) + article[1:] + rest
	default:
		return article + rest
	}
}

// splitArticle takes the indefinite article off a name and says what definite
// one belongs in its place. An empty article means there was none to change,
// or that the language does not use one here.
func splitArticle(name string, spanish bool) (article, rest string) {
	lower := strings.ToLower(name)

	// Both keep the two letters after the article, which is why "una" gives
	// back an "l" and not a "la": the "a" of "una" is the one that stays.
	// That is the reference's own trick and it is cheaper than it looks —
	// "Una linterna" becomes "la linterna" by writing one character.
	const kept = 2

	if spanish {
		switch {
		case strings.HasPrefix(lower, "una "):
			return "l", name[kept:]
		case strings.HasPrefix(lower, "un "):
			return "el", name[kept:]
		}

		return "", name
	}

	// English has nothing to put back: "a lantern" is read as "lantern".
	switch {
	case strings.HasPrefix(lower, "an "):
		return "", name[len("an "):]
	case strings.HasPrefix(lower, "a "):
		return "", name[len("a "):]
	}

	return "", name
}
