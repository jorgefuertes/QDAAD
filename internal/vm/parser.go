package vm

import (
	"strings"
	"unicode/utf8"
)

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
//
// Anything in quotes is taken out first and kept for PARSE 1 rather than
// parsed here. That is how an adventure lets the player say something to
// somebody: "DECIR "SIGUEME"" is the verb DECIR and a phrase that belongs to
// whatever the verb hands it to, and the two are parsed by different condacts
// into the same flags at different moments.
func (m *Machine) sentence(line string) bool {
	line, m.literal = splitQuoted(line)

	return m.words(line)
}

// words fills the sentence flags from a line that has had its quotes taken
// out, and reports whether anything in it was understood.
func (m *Machine) words(line string) bool {
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

	// A sentence with no verb but a low noun becomes that noun: "norte" means
	// go north, and "i" means take inventory.
	//
	// Twenty is the manual's own number, in the section on what a sentence is
	// made of: "la mínima frase es un verbo, o un nombre convertible a verbo
	// (los nombres con valor menor de 20 que si no hay verbo en la frase se
	// convertirán en verbo, por ejemplo, NORTE)" — daad-ready/DOC/doc_es.html.
	//
	// It used to be fourteen here, which is where the directions stop, and
	// that was a guess dressed as a reading. It cost the second part of La
	// Aventura Original its inventory: I and INVEN are nouns of value 14, its
	// response table answers them with "> I _ PROCESS 10", and with the lower
	// number the verb stayed unset and the entry never matched. Nothing was
	// printed at all, which is a hard symptom to trace back to a constant.
	//
	// The two numbers do not contradict each other, which is what made the
	// guess look right: a noun of twenty or more does not convert, so a bare
	// "llave" still leaves no verb, and La Aventura Original's own wildcards
	// go on telling that case apart with EQ 33 255.
	const firstRealVerb = 20

	if m.flags[FlagVerb] == NoWord && m.flags[FlagNoun1] != NoWord &&
		m.flags[FlagNoun1] < firstRealVerb {
		m.setFlag(FlagVerb, m.flags[FlagNoun1])
	}

	m.remember()

	return found
}

// remember keeps this sentence's noun as the one a pronoun will stand for.
//
// Fifty is the line the format draws between a proper noun and one "it" can
// replace: below it the word is a direction or a word of the machine's own and
// is not worth remembering, at or above it it is a thing. See
// docs/DAAD_v2_v3_Research/06-parser.md section 4.3, and the C at
// daad_parser_sentences.c:233.
//
// It outlives the sentence on purpose. Every other sentence flag is cleared at
// the top of words, and these two are not, because the whole point of them is
// to still be there next turn when the player says "cógelo" — or when a table
// asks what the player was last talking about. La Aventura Original's pirate
// is the one that asks: "COPYFF 46 34" then WHATO, which is its second chance
// at a treasure when the sentence it was given named something small.
func (m *Machine) remember() {
	const firstPronounable = 50

	if noun := m.flags[FlagNoun1]; noun != NoWord && noun >= firstPronounable {
		m.setFlag(FlagCPNoun, noun)
		m.setFlag(FlagCPAdject, m.flags[FlagAdject1])
	}
}

// splitQuoted takes the quoted phrase out of a line, giving back the rest and
// the phrase without its quotes.
//
// Only the first pair counts, and an opening quote with no closing one runs to
// the end of the line — which is the forgiving reading, and the one a player
// who forgot the second quote deserves.
func splitQuoted(line string) (rest, quoted string) {
	const quote = '"'

	open := strings.IndexRune(line, quote)
	if open < 0 {
		return line, ""
	}

	tail := line[open+1:]

	close := strings.IndexRune(tail, quote)
	if close < 0 {
		return line[:open], tail
	}

	return line[:open] + tail[close+1:], tail[:close]
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
//
// Five letters and not five bytes. The machine this came from counted
// characters because a character was a byte, and our sources are UTF-8 — so
// cutting the line's words by byte takes "BAÑAR" down to four letters, which
// no longer matches the "BAÑAR" the vocabulary holds. It also leaves half an
// eñe at the end of the string, which is no word in any encoding.
func (m *Machine) lookup(word string) (value, kind byte, found bool) {
	if utf8.RuneCountInString(word) > wordLength {
		word = string([]rune(word)[:wordLength])
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
