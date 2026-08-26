package ddb

import (
	"slices"
	"unicode"

	qderror "github.com/jorgefuertes/QDAAD/internal/ddb/dberrors"
)

type WordKind uint8

const (
	// MAX_WORD_LEN is the significant length of a vocabulary word: DAAD only
	// stores and compares the first five characters of each word.
	MAX_WORD_LEN            = 5
	MAX_DIRECTION_WORD   ID = 13
	MAX_CONVERTIBLE_NAME ID = 39
	MAX_PROPER_NOUN      ID = 49
	LAST_PRONOMINAL_VERB ID = 239
	MAX_WORD_ID          ID = 254
	NO_WORD_ID           ID = 255
)

const (
	Verb        WordKind = 0
	Adverb      WordKind = 1
	Noun        WordKind = 2
	Adjective   WordKind = 3
	Preposition WordKind = 4
	Conjunction WordKind = 5
	Pronoun     WordKind = 6
	NoWord      WordKind = 255
)

func (wk WordKind) String() string {
	switch wk {
	case Verb:
		return "Verb"
	case Adverb:
		return "Adverb"
	case Noun:
		return "Noun"
	case Adjective:
		return "Adjective"
	case Preposition:
		return "Preposition"
	case Conjunction:
		return "Conjunction"
	case Pronoun:
		return "Pronoun"
	case NoWord:
		return "_"
	default:
		return ""
	}
}

func (wk WordKind) IsValid() bool {
	switch wk {
	case Verb, Adverb, Noun, Adjective, Preposition, Conjunction, Pronoun:
		return true
	default:
		return false
	}
}

type Word struct {
	ID       ID
	LabelID  ID16
	Kind     WordKind
	Synonyms []string
}

func (w *Word) AddSynonym(synonym string) {
	w.Synonyms = append(w.Synonyms, NormalizeWord(synonym))
}

func (w *Word) IsDirection() bool {
	return w.ID <= MAX_DIRECTION_WORD
}

func (w *Word) IsConvertible() bool {
	return w.ID <= MAX_CONVERTIBLE_NAME
}

type Words []Word

func NewWordStore() Words {
	return Words{
		{ID: NO_WORD_ID, LabelID: 0, Kind: NoWord, Synonyms: []string{"_"}},
	}
}

type WordOption int

const (
	None WordOption = iota
	Direction
	Convertible
	NotPronominal
	ProperNoun
)

func (ws *Words) New(labelID ID16, kind WordKind, option WordOption, synonyms ...string) (ID, error) {
	if !kind.IsValid() {
		return 0, qderror.ErrInvalidWordKind
	}

	if _, exists := ws.GetByLabelID(labelID); exists {
		return 0, qderror.ErrWordWithDuplicatedLabel
	}

	id, err := ws.getNextID(option)
	if err != nil {
		return 0, err
	}

	var normalized []string
	for _, s := range synonyms {
		normalized = append(normalized, NormalizeWord(s))
	}

	*ws = append(*ws, Word{
		ID:       id,
		LabelID:  labelID,
		Kind:     kind,
		Synonyms: normalized,
	})

	return id, nil
}

// The getters return a pointer into the store, not to a copy, so changes made
// through it —AddSynonym, for instance— are kept.
func (ws Words) Get(id ID) (*Word, bool) {
	for i := range ws {
		if ws[i].ID == id {
			return &ws[i], true
		}
	}

	return nil, false
}

func (ws Words) GetByLabelID(labelID ID16) (*Word, bool) {
	for i := range ws {
		if ws[i].LabelID == labelID {
			return &ws[i], true
		}
	}

	return nil, false
}

func (ws Words) GetBySynonym(synonym string) (*Word, error) {
	synonym = NormalizeWord(synonym)

	for i := range ws {
		if slices.Contains(ws[i].Synonyms, synonym) {
			return &ws[i], nil
		}
	}

	return nil, qderror.ErrWordNotFound
}

func (ws Words) GetByTypeAndSynonym(kind WordKind, synonym string) (*Word, error) {
	synonym = NormalizeWord(synonym)

	for i := range ws {
		if ws[i].Kind == kind && slices.Contains(ws[i].Synonyms, synonym) {
			return &ws[i], nil
		}
	}

	return nil, qderror.ErrWordNotFound
}

func (ws Words) getNextID(option WordOption) (ID, error) {
	switch option {
	case Direction:
		return ws.getNextDirectionID()
	case Convertible:
		return ws.getNextConvertibleID()
	case ProperNoun:
		return ws.getNextProperNounID()
	case NotPronominal:
		return ws.getNextNonPronominalVerbID()
	default:
		return ws.getNextGeneralID()
	}
}

func (ws Words) getNextDirectionID() (ID, error) {
	for i := ID(0); i <= MAX_DIRECTION_WORD; i++ {
		if _, exists := ws.Get(i); !exists {
			return i, nil
		}
	}

	return 0, qderror.ErrWordConnectionIDsExhausted
}

func (ws Words) getNextConvertibleID() (ID, error) {
	for i := ID(MAX_DIRECTION_WORD + 1); i <= MAX_CONVERTIBLE_NAME; i++ {
		if _, exists := ws.Get(i); !exists {
			return i, nil
		}
	}

	return 0, qderror.ErrWordConvertibleIDsExhausted
}

func (ws Words) getNextProperNounID() (ID, error) {
	for id := MAX_CONVERTIBLE_NAME + 1; id <= MAX_PROPER_NOUN; id++ {
		if _, exists := ws.Get(id); !exists {
			return id, nil
		}
	}

	return 0, qderror.ErrWordProperNounIDsExhausted
}

func (ws Words) getNextNonPronominalVerbID() (ID, error) {
	for id := LAST_PRONOMINAL_VERB + 1; id <= MAX_WORD_ID; id++ {
		if _, exists := ws.Get(id); !exists {
			return id, nil
		}
	}

	return 0, qderror.ErrWordNonPronominalIDsExhausted
}

func (ws Words) getNextGeneralID() (ID, error) {
	for id := MAX_PROPER_NOUN + 1; id <= LAST_PRONOMINAL_VERB; id++ {
		if _, exists := ws.Get(id); !exists {
			return id, nil
		}
	}

	return 0, qderror.ErrWordGeneralIDsExhausted
}

// precomposed joins a letter and a combining mark into the single character
// DAAD has a code for. They are the eight pairs of the low charset, and no
// other accented letter fits in a vocabulary word.
var precomposed = map[[2]rune]rune{
	{'A', '́'}: 'Á',
	{'E', '́'}: 'É',
	{'I', '́'}: 'Í',
	{'O', '́'}: 'Ó',
	{'U', '́'}: 'Ú',
	{'N', '̃'}: 'Ñ',
	{'U', '̈'}: 'Ü',
	{'C', '̧'}: 'Ç',
}

// NormalizeWord returns the vocabulary form of a word: blanks removed,
// uppercase and truncated to MAX_WORD_LEN characters.
//
// Accents and tildes are kept: DAAD stores them, it does not strip them. What
// it does do is fold them back to their lowercase code —there is no uppercase
// "Á" in the low charset— but that is a property of the DDB character set, so
// it belongs to the emitter and not here.
func NormalizeWord(s string) string {
	word := make([]rune, 0, MAX_WORD_LEN)

	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}

		// A combining mark joins the letter already written instead of taking a
		// slot of its own, so decomposed input —"N" plus a combining tilde—
		// ends up like its precomposed form.
		if unicode.Is(unicode.Mn, r) {
			if len(word) > 0 {
				if joined, found := precomposed[[2]rune{word[len(word)-1], r}]; found {
					word[len(word)-1] = joined
				}
			}

			continue
		}

		if len(word) == MAX_WORD_LEN {
			break
		}

		word = append(word, unicode.ToUpper(r))
	}

	return string(word)
}
