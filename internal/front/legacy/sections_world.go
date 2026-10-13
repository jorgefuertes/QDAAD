package legacy

import (
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// connections reads /CON: for every location, where its exits lead.
//
// Two kinds of record alternate. A numbered entry opens a location, and after
// it come zero or more pairs of a vocabulary word and a destination. Every
// location has to appear even with no exits at all, which is the compiler's
// rule and also the binary's: the table is indexed by location number.
func (p *parser) connections() error {
	if err := p.expectSection(SecCON); err != nil {
		return err
	}

	want := ddb.ID(0)

	for !p.atSection(SecOBJ) {
		entry := p.next()
		if entry.Kind != ListEntry || entry.Text != "" {
			return p.errorf("a numbered location or /OBJ expected, but %s is there",
				entry.Describe())
		}

		if entry.Num != want.Int() {
			return p.errorf("locations must run consecutively from zero: %d expected, %d found",
				want, entry.Num)
		}

		if _, err := p.db.Locations().Get(want); err != nil {
			return p.errorf("location %d has connections but no description in /LTX", want)
		}

		if err := p.exits(want); err != nil {
			return err
		}

		want++
	}

	// A location with no entry of its own would leave the table short, and the
	// binary indexes it by position.
	if left := p.db.Locations().Len(); want.Int() < left {
		return p.errorf("/CON stops at location %d, and there are %d", want, left)
	}

	return nil
}

// exits reads the pairs of one location, up to the next location or /OBJ.
func (p *parser) exits(from ddb.ID) error {
	for {
		t := p.peek()
		if t.Kind == ListEntry || p.atSection(SecOBJ) || t.Kind == EOF {
			return nil
		}

		word := p.next()
		if word.Kind != Ident && word.Kind != Number {
			return p.errorf("a direction or a new location expected, but %s is there",
				word.Describe())
		}

		// A direction is named by a word, and the compiler asks only that it be
		// a verb or a noun: it is the value that travels to the binary.
		value, err := p.direction(word)
		if err != nil {
			return err
		}

		to, err := p.byteValue(p.next(), "the destination of "+word.Describe())
		if err != nil {
			return err
		}

		if _, taken := p.db.Conns().Get(from, value); taken {
			return p.errorf("location %d already leads somewhere through %s",
				from, word.Describe())
		}

		p.db.Conns().Add(from, value, to)
	}
}

// direction resolves the word that names an exit.
func (p *parser) direction(t Token) (ddb.ID, error) {
	name := t.Text
	if t.Kind == Number {
		name = t.Describe()
	}

	for _, kind := range []ddb.WordKind{ddb.Verb, ddb.Noun} {
		if w, err := p.db.Words().GetByTypeAndSynonym(kind, name); err == nil {
			return w.ID, nil
		}
	}

	return 0, p.errorf("%q is not a verb or a noun in the vocabulary", name)
}

// objects reads /OBJ, whose record is the widest in the language: the number,
// where it starts, what it weighs, two flags, sixteen more, and two words.
//
// The objects already exist, created by /OTX with their description, so this
// completes them rather than adding them.
func (p *parser) objects() error {
	if err := p.expectSection(SecOBJ); err != nil {
		return err
	}

	want := ddb.ID(0)

	for !p.atSection(SecPRO) && !p.atSection(SecEND) && p.peek().Kind != EOF {
		entry := p.next()
		if entry.Kind != ListEntry || entry.Text != "" {
			return p.errorf("a numbered object or /PRO expected, but %s is there",
				entry.Describe())
		}

		if entry.Num != want.Int() {
			return p.errorf("objects must run consecutively from zero: %d expected, %d found",
				want, entry.Num)
		}

		obj, err := p.db.Objects().Get(want)
		if err != nil {
			return p.errorf("object %d is defined but has no text in /OTX", want)
		}

		if err := p.objectBody(obj); err != nil {
			return err
		}

		want++
	}

	if left := p.db.Objects().Len(); want.Int() < left {
		return p.errorf("/OBJ stops at object %d, and there are %d", want, left)
	}

	p.defineInventory()

	return nil
}

// defineInventory counts what the player starts the game holding.
//
// Both are worked out from where each object begins, exactly as the compiler of
// reference does it in UObjects.pas:45-46. A source reads them to check a limit
// — you cannot carry more than so many things — without having to keep a count
// by hand that goes stale the moment an object is added.
func (p *parser) defineInventory() {
	var carried, worn int

	for i := range p.db.Objects().Len() {
		obj, err := p.db.Objects().Get(ddb.ID(i))
		if err != nil {
			continue
		}

		switch obj.InitLocation {
		case ddb.LOC_CARRIED:
			carried++
		case ddb.LOC_WORN:
			worn++
		}
	}

	p.define("NUM_CARRIED", carried)
	p.define("NUM_WORN", worn)
}

// objectBody fills in the twenty-two fields that follow the object's number.
func (p *parser) objectBody(obj *ddb.Object) error {
	where, err := p.objectPlace()
	if err != nil {
		return err
	}

	obj.InitLocation = where

	weight, err := p.byteValue(p.next(), "the weight")
	if err != nil {
		return err
	}

	if weight > ddb.MAX_WEIGHT {
		return p.errorf("a weight of %d is over the %d a database allows", weight, ddb.MAX_WEIGHT)
	}

	obj.Weight = uint8(weight)

	if obj.Container, err = p.yesNo("the container flag"); err != nil {
		return err
	}

	if obj.Wearable, err = p.yesNo("the wearable flag"); err != nil {
		return err
	}

	// Written highest bit first, so the leftmost column is bit 15.
	for i := range obj.Flags {
		on, err := p.yesNo("attribute bit")
		if err != nil {
			return err
		}

		obj.Flags[i] = on
	}

	if obj.Name, err = p.objectWord(ddb.Noun, "a noun"); err != nil {
		return err
	}

	obj.Adjective, err = p.objectWord(ddb.Adjective, "an adjective")

	return err
}

// objectPlace reads where an object starts.
//
// It is a location number, or one of the markers — which arrive as ordinary
// symbols, so nothing special happens here — or the underscore. The underscore
// is the one thing worth spelling out: everywhere else it stands for the null
// word, 255, but in this one slot it means the object is not created, 252.
//
// HERE is refused although the symbol exists and resolves: an object cannot
// start where the player happens to be. The compiler of reference makes the
// same check, at USintactic.pas:461.
func (p *parser) objectPlace() (ddb.ID, error) {
	if t := p.peek(); t.Kind == Wildcard {
		p.next()

		return ddb.LOC_NOT_CREATED, nil
	}

	t := p.next()

	where, err := p.byteValue(t, "the initial location")
	if err != nil {
		return 0, err
	}

	switch where {
	case ddb.LOC_NOT_CREATED, ddb.LOC_WORN, ddb.LOC_CARRIED:
		return where, nil
	case ddb.LOC_HERE:
		return 0, p.errorf("an object cannot start at HERE")
	}

	if _, err := p.db.Locations().Get(where); err != nil {
		return 0, p.errorf("%s is no location and no marker", t.Describe())
	}

	return where, nil
}

// yesNo reads one of the flag columns, which is Y, N or the underscore.
func (p *parser) yesNo(what string) (bool, error) {
	t := p.next()

	switch {
	case t.Kind == Wildcard:
		return false, nil
	case t.Kind == Ident && strings.EqualFold(t.Text, "Y"):
		return true, nil
	case t.Kind == Ident && strings.EqualFold(t.Text, "N"):
		return false, nil
	}

	return false, p.errorf("Y, N or _ expected at %s, but %s is there", what, t.Describe())
}

// objectWord resolves the noun or the adjective that names an object, either
// of which may be absent.
func (p *parser) objectWord(kind ddb.WordKind, what string) (ddb.ID, error) {
	t := p.next()
	if t.Kind == Wildcard {
		return ddb.NO_WORD_ID, nil
	}

	name := t.Text
	if t.Kind == Number {
		name = t.Describe()
	}

	w, err := p.db.Words().GetByTypeAndSynonym(kind, name)
	if err != nil {
		return 0, p.errorf("%q is not %s in the vocabulary", name, what)
	}

	return w.ID, nil
}
