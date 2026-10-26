package vm

import "strings"

// The parser, which turns a line the player typed into the handful of flags an
// entry matches against.
//
// DAAD's parser is not a grammar. It scans the line for words it knows and
// fills in a logical sentence: a verb, a noun, an adjective, an adverb, a
// preposition and a second noun. Everything it does not know it ignores. That
// is why "coger la llave oxidada con cuidado" and "coger llave" answer the same
// entry, and why word order barely matters.
//
// Two rules do the work of a grammar and are worth naming, because an entry
// written for a 1989 adventure leans on both:
//
//   - Only the first five letters of a word count. The vocabulary stores five
//     and the parser compares five, so "llaves" and "llave" are the same word.
//   - A noun whose value is below fourteen can stand in for a verb. That is
//     how typing "norte" on its own means "go north": the directions are nouns,
//     and a sentence with no verb takes its first low noun as one.
const wordLength = 5

// sentence reads a line into the sentence flags. It reports whether anything
// in it was understood.
func (m *Machine) sentence(line string) bool {
	// The sentence starts empty, so that a word the player leaves out does not
	// keep the value it had last turn. Leaving them would make "coger" after
	// "coger llave" mean "coger llave" again.
	for _, f := range []int{
		FlagVerb, FlagNoun1, FlagAdject1, FlagAdverb,
		FlagPrep, FlagNoun2, FlagAdject2,
	} {
		m.setFlag(f, NoWord)
	}

	var found bool

	for _, word := range strings.Fields(strings.ToUpper(line)) {
		value, kind, known := m.lookup(word)
		if !known {
			continue
		}

		found = true

		m.place(value, kind)
	}

	// A sentence with no verb but a low noun is a movement: "norte" means go
	// north. Fourteen is where the directions stop, and the manual says the
	// same thing the other way round when it explains which refusal an
	// unanswered sentence gets.
	const firstRealVerb = 14

	if m.flags[FlagVerb] == NoWord && m.flags[FlagNoun1] != NoWord &&
		m.flags[FlagNoun1] < firstRealVerb {
		m.setFlag(FlagVerb, m.flags[FlagNoun1])
	}

	return found
}

// place files a word in the first slot of its kind that is still empty, which
// is how a second noun becomes the second noun.
func (m *Machine) place(value, kind byte) {
	switch kind {
	case ddbKindVerb:
		if m.flags[FlagVerb] == NoWord {
			m.setFlag(FlagVerb, value)
		}
	case ddbKindNoun:
		if m.flags[FlagNoun1] == NoWord {
			m.setFlag(FlagNoun1, value)

			return
		}

		if m.flags[FlagNoun2] == NoWord {
			m.setFlag(FlagNoun2, value)
		}
	case ddbKindAdjective:
		if m.flags[FlagAdject1] == NoWord {
			m.setFlag(FlagAdject1, value)

			return
		}

		if m.flags[FlagAdject2] == NoWord {
			m.setFlag(FlagAdject2, value)
		}
	case ddbKindAdverb:
		if m.flags[FlagAdverb] == NoWord {
			m.setFlag(FlagAdverb, value)
		}
	case ddbKindPreposition:
		if m.flags[FlagPrep] == NoWord {
			m.setFlag(FlagPrep, value)
		}
	}
}

// The word kinds, as the database numbers them.
const (
	ddbKindVerb        = 0
	ddbKindAdverb      = 1
	ddbKindNoun        = 2
	ddbKindAdjective   = 3
	ddbKindPreposition = 4
)

// lookup finds a word in the vocabulary by its first five letters.
func (m *Machine) lookup(word string) (value, kind byte, found bool) {
	if len(word) > wordLength {
		word = word[:wordLength]
	}

	for _, entry := range m.vocab {
		// The vocabulary pads every word to five characters, so what is
		// stored for "este" is "ESTE " and comparing the two as they come
		// never matches anything.
		if strings.EqualFold(strings.TrimSpace(entry.Word), word) {
			return entry.Value, entry.Kind, true
		}
	}

	return 0, 0, false
}
