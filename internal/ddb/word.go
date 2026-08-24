package ddb

import (
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/jorgefuertes/QDAAD/internal/qderror"
)

type WordKind uint8

const (
	MaxDirectionWord     ID = 13
	MaxConvertibleToVerb ID = 39
	// MaxWordLen is the significant length of a vocabulary word: DAAD only
	// stores and compares the first five characters of each word.
	MaxWordLen    = 5
	MaxWord    ID = 254
	NoWordID   ID = 255
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
	return w.ID <= MaxDirectionWord
}

func (w *Word) IsConvertible() bool {
	return w.ID <= MaxConvertibleToVerb
}

type Words []Word

func NewWordStore() Words {
	return Words{
		{ID: NoWordID, LabelID: 0, Kind: NoWord, Synonyms: []string{"_"}},
	}
}

func (ws *Words) New(labelID ID16, kind WordKind, isDirection, isConvertible bool, synonyms ...string) (ID, error) {
	if !kind.IsValid() {
		return 0, qderror.ErrInvalidWordKind
	}

	if isDirection && isConvertible {
		return 0, qderror.ErrWordDirectionAndConvertible
	}

	if _, exists := ws.GetByLabelID(labelID); exists {
		return 0, qderror.ErrWordWithDuplicatedLabel
	}

	id, err := ws.getNextID(isDirection, isConvertible)
	if err != nil {
		return 0, err
	}

	for i, synonym := range synonyms {
		synonyms[i] = NormalizeWord(synonym)
	}

	*ws = append(*ws, Word{
		ID:       id,
		LabelID:  labelID,
		Kind:     kind,
		Synonyms: synonyms,
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

func (ws Words) getNextID(isDirection, isConvertible bool) (ID, error) {
	if isDirection {
		return ws.getNextDirectionID()
	}

	if isConvertible {
		id, err := ws.getNextConvertibleID()
		if err == nil {
			return id, nil
		}

		id, err = ws.getNextDirectionID()
		if err == nil {
			return id, nil
		}

		return 0, qderror.ErrWordConvertibleIDsExhausted
	}

	return ws.getNextGeneralID()
}

func (ws Words) getNextDirectionID() (ID, error) {
	for i := ID(0); i <= MaxDirectionWord; i++ {
		if _, exists := ws.Get(i); !exists {
			return i, nil
		}
	}

	return 0, qderror.ErrWordConnectionIDsExhausted
}

func (ws Words) getNextConvertibleID() (ID, error) {
	for i := ID(MaxDirectionWord + 1); i <= MaxConvertibleToVerb; i++ {
		if _, exists := ws.Get(i); !exists {
			return i, nil
		}
	}

	return 0, qderror.ErrWordConvertibleIDsExhausted
}

func (ws Words) getNextGeneralID() (ID, error) {
	// The counters must be int: ID is an uint8, so "i <= math.MaxUint8" would
	// always hold and the loop would never end.
	for i := MaxConvertibleToVerb + 1; i < math.MaxUint8; i++ {
		if _, exists := ws.Get(ID(i)); !exists {
			return ID(i), nil
		}
	}

	// No general ID left: fall back to the reserved ranges.
	for id := ID(0); id <= MaxConvertibleToVerb; id++ {
		if _, exists := ws.Get(id); !exists {
			return id, nil
		}
	}

	return 0, qderror.ErrWordStoreIsFull
}

// NormalizeWord returns the vocabulary form of a word: blanks removed,
// uppercase, without accents or tildes —"ñ" becomes "n"— and truncated to
// VocabularyLength characters.
func NormalizeWord(s string) string {
	// accentReplacements maps the uppercase accented characters of the Spanish
	// alphabet to their plain equivalent: the DAAD vocabulary is ASCII only.
	accentReplacements := map[rune]rune{
		'Á': 'A', 'À': 'A', 'Ä': 'A', 'Â': 'A',
		'É': 'E', 'È': 'E', 'Ë': 'E', 'Ê': 'E',
		'Í': 'I', 'Ì': 'I', 'Ï': 'I', 'Î': 'I',
		'Ó': 'O', 'Ò': 'O', 'Ö': 'O', 'Ô': 'O',
		'Ú': 'U', 'Ù': 'U', 'Ü': 'U', 'Û': 'U',
		'Ñ': 'N', 'Ç': 'C',
	}

	var (
		sb     strings.Builder
		length int
	)

	sb.Grow(MaxWordLen)

	for _, r := range s {
		// Combining marks are dropped so that decomposed input —"n" plus a
		// combining tilde— normalizes just like its precomposed form.
		if unicode.IsSpace(r) || unicode.Is(unicode.Mn, r) {
			continue
		}

		r = unicode.ToUpper(r)
		if replacement, found := accentReplacements[r]; found {
			r = replacement
		}

		sb.WriteRune(r)

		length++
		if length == MaxWordLen {
			break
		}
	}

	return sb.String()
}
