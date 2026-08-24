package ddb

import (
	"math"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/qderror"
	"github.com/stretchr/testify/require"
)

// fill occupies the word IDs in [from, to] with placeholder entries so the
// allocators can be driven into their exhaustion branches. The store is a
// slice, so it takes a pointer: appending to a copy would not reach the caller.
func fill(ws *Words, from, to ID) {
	// The counter must be an int: with "to" at 255 an ID counter would overflow
	// back to 0 on the last increment and the loop would never end.
	for i := int(from); i <= to.Int(); i++ {
		*ws = append(*ws, Word{ID: ID(i)})
	}
}

var kinds = []struct {
	kind WordKind
	name string
}{
	{Verb, "Verb"},
	{Adverb, "Adverb"},
	{Noun, "Noun"},
	{Adjective, "Adjective"},
	{Preposition, "Preposition"},
	{Conjunction, "Conjunction"},
	{Pronoun, "Pronoun"},
}

func TestWordKindString(t *testing.T) {
	for _, c := range kinds {
		require.Equal(t, c.name, c.kind.String(), "name of kind %d", c.kind)
	}

	require.Equal(t, "_", NoWord.String(), "NoWord is the empty word")
	require.Empty(t, WordKind(200).String(), "an unassigned kind has no name")
}

func TestWordKindIsValid(t *testing.T) {
	for _, c := range kinds {
		require.True(t, c.kind.IsValid(), "%s should be valid", c.name)
	}

	require.False(t, NoWord.IsValid(), "NoWord should not be valid")
	require.False(t, WordKind(200).IsValid(), "an unassigned kind should not be valid")
}

func TestWordAddSynonym(t *testing.T) {
	w := Word{ID: 1, LabelID: 1, Kind: Verb}
	require.Nil(t, w.Synonyms, "a fresh word has no synonyms")

	w.AddSynonym("COGER")
	require.Equal(t, []string{"COGER"}, w.Synonyms)

	w.AddSynonym("TOMAR")
	require.Equal(t, []string{"COGER", "TOMAR"}, w.Synonyms, "synonyms keep insertion order")

	// Duplicates are not filtered out.
	w.AddSynonym("COGER")
	require.Equal(t, []string{"COGER", "TOMAR", "COGER"}, w.Synonyms)
}

func TestWordIsDirection(t *testing.T) {
	cases := []struct {
		id   ID
		want bool
	}{
		{0, true},
		{7, true},
		{MaxDirectionWord, true},
		{MaxDirectionWord + 1, false},
		{MaxConvertibleToVerb, false},
		{math.MaxUint8, false},
	}

	for _, c := range cases {
		w := Word{ID: c.id}
		require.Equal(t, c.want, w.IsDirection(), "ID %d", c.id)
	}
}

func TestWordIsConvertible(t *testing.T) {
	cases := []struct {
		id   ID
		want bool
	}{
		// The direction range is included: any word below the convertible
		// limit can be converted into a noun.
		{0, true},
		{MaxDirectionWord, true},
		{MaxDirectionWord + 1, true},
		{MaxConvertibleToVerb, true},
		{MaxConvertibleToVerb + 1, false},
		{math.MaxUint8, false},
	}

	for _, c := range cases {
		w := Word{ID: c.id}
		require.Equal(t, c.want, w.IsConvertible(), "ID %d", c.id)
	}
}

// A fresh store is not empty: it holds the empty word, so the counts below are
// always one above the number of words added by the test.
const baseWordsCount = 1

func TestNewWordStore(t *testing.T) {
	ws := NewWordStore()
	require.NotNil(t, ws)
	require.Len(t, ws, baseWordsCount, "the store is born with the empty word")

	w := &ws[0]
	require.Equal(t, NoWordID, w.ID)
	require.Zero(t, w.LabelID)
	require.Equal(t, NoWord, w.Kind)
	require.Equal(t, []string{"_"}, w.Synonyms)
}

func TestWordsNew(t *testing.T) {
	t.Run("invalid kind", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.New(1, NoWord, false, false)
		require.ErrorIs(t, err, qderror.ErrInvalidWordKind)
		require.Zero(t, id)
		require.Len(t, ws, baseWordsCount, "a rejected word is not stored")
	})

	t.Run("direction and convertible at once", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.New(1, Noun, true, true)
		require.ErrorIs(t, err, qderror.ErrWordDirectionAndConvertible)
		require.Zero(t, id)
		require.Len(t, ws, baseWordsCount)
	})

	t.Run("duplicated label", func(t *testing.T) {
		ws := NewWordStore()
		_, err := ws.New(7, Verb, false, false, "COGER")
		require.NoError(t, err)

		id, err := ws.New(7, Noun, false, false, "LLAVE")
		require.ErrorIs(t, err, qderror.ErrWordWithDuplicatedLabel)
		require.Zero(t, id)
		require.Len(t, ws, baseWordsCount+1, "the duplicate is not stored")
	})

	t.Run("id allocation error is propagated", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, 0, MaxDirectionWord)

		id, err := ws.New(1000, Verb, true, false, "NORTE")
		require.ErrorIs(t, err, qderror.ErrWordConnectionIDsExhausted)
		require.Zero(t, id)
		require.Len(t, ws, baseWordsCount+MaxDirectionWord.Int()+1, "nothing new is stored")
	})

	t.Run("stores the word", func(t *testing.T) {
		ws := NewWordStore()

		id, err := ws.New(42, Verb, false, false, "COGER", "TOMAR")
		require.NoError(t, err)
		require.Equal(t, ID(MaxConvertibleToVerb+1), id, "first general ID")
		require.Len(t, ws, baseWordsCount+1)

		w := ws[baseWordsCount]
		require.Equal(t, id, w.ID)
		require.Equal(t, ID16(42), w.LabelID)
		require.Equal(t, Verb, w.Kind)
		require.Equal(t, []string{"COGER", "TOMAR"}, w.Synonyms)
	})

	t.Run("stores a word without synonyms", func(t *testing.T) {
		ws := NewWordStore()

		id, err := ws.New(1, Noun, false, false)
		require.NoError(t, err)
		require.Equal(t, ID(MaxConvertibleToVerb+1), id)
		require.Empty(t, ws[baseWordsCount].Synonyms)
	})

	t.Run("stores the synonyms in vocabulary form", func(t *testing.T) {
		ws := NewWordStore()

		_, err := ws.New(1, Noun, false, false, " araña ", "bicho")
		require.NoError(t, err)
		require.Equal(t, []string{"ARANA", "BICHO"}, ws[baseWordsCount].Synonyms)
	})

	t.Run("assigns each range its own IDs", func(t *testing.T) {
		ws := NewWordStore()

		dirID, err := ws.New(1, Noun, true, false, "NORTE")
		require.NoError(t, err)
		require.Equal(t, ID(0), dirID)

		convID, err := ws.New(2, Noun, false, true, "LLAVE")
		require.NoError(t, err)
		require.Equal(t, ID(MaxDirectionWord+1), convID)

		genID, err := ws.New(3, Verb, false, false, "COGER")
		require.NoError(t, err)
		require.Equal(t, ID(MaxConvertibleToVerb+1), genID)
	})
}

func TestSynonymPersistence(t *testing.T) {
	ws := NewWordStore()

	id, err := ws.New(1, Verb, false, false, "COGER")
	require.NoError(t, err)

	// A second word so the lookups have to pick the right one.
	_, err = ws.New(2, Noun, false, false, "LLAVE")
	require.NoError(t, err)

	// AddSynonym does not normalize, so the fixtures are given in vocabulary
	// form —uppercase and no longer than VocabularyLength— to stay searchable.
	want := []string{"COGER", "TOMAR", "ASIR", "ROBAR"}

	t.Run("through Get", func(t *testing.T) {
		w, ok := ws.Get(id)
		require.True(t, ok)
		w.AddSynonym("TOMAR")
	})

	t.Run("through GetByLabelID", func(t *testing.T) {
		w, ok := ws.GetByLabelID(1)
		require.True(t, ok)
		require.Equal(t, []string{"COGER", "TOMAR"}, w.Synonyms, "the previous change is there")
		w.AddSynonym("ASIR")
	})

	t.Run("through GetBySynonym", func(t *testing.T) {
		w, err := ws.GetBySynonym("ASIR")
		require.NoError(t, err, "a synonym added afterwards is searchable")
		w.AddSynonym("ROBAR")
	})

	t.Run("all of them are kept in the store", func(t *testing.T) {
		require.Equal(t, want, ws[baseWordsCount].Synonyms)

		w, ok := ws.Get(id)
		require.True(t, ok)
		require.Equal(t, want, w.Synonyms)

		w, err := ws.GetByTypeAndSynonym(Verb, "ROBAR")
		require.NoError(t, err)
		require.Equal(t, id, w.ID)
		require.Equal(t, want, w.Synonyms)
	})

	t.Run("the other word is untouched", func(t *testing.T) {
		w, ok := ws.GetByLabelID(2)
		require.True(t, ok)
		require.Equal(t, []string{"LLAVE"}, w.Synonyms)
	})

	t.Run("changes survive new words", func(t *testing.T) {
		// Appending may reallocate the backing array: the store, not any
		// pointer taken earlier, is the source of truth.
		for i := ID16(3); i < 20; i++ {
			_, err := ws.New(i, Noun, false, false, "PALABRA")
			require.NoError(t, err)
		}

		w, ok := ws.Get(id)
		require.True(t, ok)
		require.Equal(t, want, w.Synonyms)
	})
}

func TestWordsGet(t *testing.T) {
	ws := NewWordStore()
	id, err := ws.New(9, Verb, false, false, "COGER")
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		w, ok := ws.Get(id)
		require.True(t, ok)
		require.NotNil(t, w)
		require.Equal(t, id, w.ID)
		require.Equal(t, ID16(9), w.LabelID)
		require.Equal(t, Verb, w.Kind)
	})

	t.Run("not found", func(t *testing.T) {
		w, ok := ws.Get(id + 1)
		require.False(t, ok)
		require.Nil(t, w)
	})
}

func TestWordsGetByLabelID(t *testing.T) {
	ws := NewWordStore()
	id, err := ws.New(9, Verb, false, false, "COGER")
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		w, ok := ws.GetByLabelID(9)
		require.True(t, ok)
		require.NotNil(t, w)
		require.Equal(t, id, w.ID)
		require.Equal(t, ID16(9), w.LabelID)
	})

	t.Run("not found", func(t *testing.T) {
		w, ok := ws.GetByLabelID(10)
		require.False(t, ok)
		require.Nil(t, w)
	})
}

func TestWordsGetBySynonym(t *testing.T) {
	ws := NewWordStore()
	_, err := ws.New(1, Verb, false, false, "COGER", "TOMAR")
	require.NoError(t, err)
	_, err = ws.New(2, Noun, false, false, "LLAVE")
	require.NoError(t, err)

	t.Run("found by any of its synonyms", func(t *testing.T) {
		for _, synonym := range []string{"COGER", "TOMAR"} {
			w, err := ws.GetBySynonym(synonym)
			require.NoError(t, err, "looking up %q", synonym)
			require.Equal(t, ID16(1), w.LabelID)
			require.Equal(t, Verb, w.Kind)
		}
	})

	t.Run("not found", func(t *testing.T) {
		w, err := ws.GetBySynonym("DEJAR")
		require.ErrorIs(t, err, qderror.ErrWordNotFound)
		require.Nil(t, w)
	})

	t.Run("the query is normalized", func(t *testing.T) {
		for _, synonym := range []string{"coger", " Coger ", "cóger"} {
			w, err := ws.GetBySynonym(synonym)
			require.NoError(t, err, "looking up %q", synonym)
			require.Equal(t, ID16(1), w.LabelID)
		}
	})
}

func TestWordsGetByTypeAndSynonym(t *testing.T) {
	ws := NewWordStore()

	// The same synonym under two different kinds: only the kind tells them apart.
	_, err := ws.New(1, Verb, false, false, "PUERTA")
	require.NoError(t, err)
	_, err = ws.New(2, Noun, false, false, "PUERTA")
	require.NoError(t, err)

	t.Run("discriminates by kind", func(t *testing.T) {
		w, err := ws.GetByTypeAndSynonym(Verb, "PUERTA")
		require.NoError(t, err)
		require.Equal(t, ID16(1), w.LabelID)

		w, err = ws.GetByTypeAndSynonym(Noun, "PUERTA")
		require.NoError(t, err)
		require.Equal(t, ID16(2), w.LabelID)
	})

	t.Run("not found: right synonym, wrong kind", func(t *testing.T) {
		w, err := ws.GetByTypeAndSynonym(Adjective, "PUERTA")
		require.ErrorIs(t, err, qderror.ErrWordNotFound)
		require.Nil(t, w)
	})

	t.Run("not found: unknown synonym", func(t *testing.T) {
		w, err := ws.GetByTypeAndSynonym(Verb, "VENTANA")
		require.ErrorIs(t, err, qderror.ErrWordNotFound)
		require.Nil(t, w)
	})
}

func TestGetNextID(t *testing.T) {
	t.Run("direction", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.getNextID(true, false)
		require.NoError(t, err)
		require.Equal(t, ID(0), id)
	})

	t.Run("convertible", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.getNextID(false, true)
		require.NoError(t, err)
		require.Equal(t, ID(MaxDirectionWord+1), id)
	})

	t.Run("convertible falls back to the direction range", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, MaxDirectionWord+1, MaxConvertibleToVerb)
		fill(&ws, 0, MaxDirectionWord-1)

		id, err := ws.getNextID(false, true)
		require.NoError(t, err)
		require.Equal(t, ID(MaxDirectionWord), id, "the last free direction ID")
	})

	t.Run("convertible exhausted", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, 0, MaxConvertibleToVerb)

		id, err := ws.getNextID(false, true)
		require.ErrorIs(t, err, qderror.ErrWordConvertibleIDsExhausted)
		require.Zero(t, id)
	})

	t.Run("general", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.getNextID(false, false)
		require.NoError(t, err)
		require.Equal(t, ID(MaxConvertibleToVerb+1), id)
	})
}

func TestGetNextDirectionID(t *testing.T) {
	t.Run("first free on an empty store", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.getNextDirectionID()
		require.NoError(t, err)
		require.Equal(t, ID(0), id)
	})

	t.Run("fills an intermediate gap", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, 0, 4)
		fill(&ws, 6, MaxDirectionWord)

		id, err := ws.getNextDirectionID()
		require.NoError(t, err)
		require.Equal(t, ID(5), id)
	})

	t.Run("exhausted", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, 0, MaxDirectionWord)

		id, err := ws.getNextDirectionID()
		require.ErrorIs(t, err, qderror.ErrWordConnectionIDsExhausted)
		require.Zero(t, id)
	})
}

func TestGetNextConvertibleID(t *testing.T) {
	t.Run("first free on an empty store", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.getNextConvertibleID()
		require.NoError(t, err)
		require.Equal(t, ID(MaxDirectionWord+1), id)
	})

	t.Run("ignores the direction range", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, 0, MaxDirectionWord)

		id, err := ws.getNextConvertibleID()
		require.NoError(t, err)
		require.Equal(t, ID(MaxDirectionWord+1), id)
	})

	t.Run("fills an intermediate gap", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, MaxDirectionWord+1, 19)
		fill(&ws, 21, MaxConvertibleToVerb)

		id, err := ws.getNextConvertibleID()
		require.NoError(t, err)
		require.Equal(t, ID(20), id)
	})

	t.Run("exhausted", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, MaxDirectionWord+1, MaxConvertibleToVerb)

		id, err := ws.getNextConvertibleID()
		require.ErrorIs(t, err, qderror.ErrWordConvertibleIDsExhausted)
		require.Zero(t, id)
	})
}

func TestGetNextGeneralID(t *testing.T) {
	t.Run("first free on an empty store", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.getNextGeneralID()
		require.NoError(t, err)
		require.Equal(t, ID(MaxConvertibleToVerb+1), id)
	})

	t.Run("fills an intermediate gap", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, MaxConvertibleToVerb+1, 99)
		fill(&ws, 101, math.MaxUint8)

		id, err := ws.getNextGeneralID()
		require.NoError(t, err)
		require.Equal(t, ID(100), id)
	})

	t.Run("wraps around into the reserved ranges", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, MaxConvertibleToVerb+1, math.MaxUint8)
		fill(&ws, 0, 6)
		fill(&ws, 8, MaxConvertibleToVerb)

		id, err := ws.getNextGeneralID()
		require.NoError(t, err)
		require.Equal(t, ID(7), id, "the only free ID is in the direction range")
	})

	t.Run("exhausted", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, 0, math.MaxUint8)

		id, err := ws.getNextGeneralID()
		require.ErrorIs(t, err, qderror.ErrWordStoreIsFull)
		require.Zero(t, id)
	})
}

func TestNormalizeWord(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"coger", "COGER"},
		{"COGER", "COGER"},
		{"  coger  ", "COGER"},
		{"salir al norte", "SALIR"},
		{"araña", "ARANA"},
		{"ÑU", "NU"},
		{"ñu", "NU"},
		{"cañón", "CANON"},
		{"pingüino", "PINGU"},
		{"murciélago", "MURCI"},
		{"IRÁ", "IRA"},
		// Decomposed input: "n" plus a combining tilde.
		{"an\u0303o", "ANO"},
		{"abcdefgh", "ABCDE"},
		// Blanks are dropped before the cut, so this yields five letters.
		{"a b c d e f", "ABCDE"},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			require.Equal(t, c.want, NormalizeWord(c.in), "normalizing %q", c.in)
			require.LessOrEqual(t, len([]rune(NormalizeWord(c.in))), MaxWordLen)
		})
	}
}
