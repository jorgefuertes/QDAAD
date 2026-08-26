package ddb

import (
	"math"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb/dberrors"
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
		{MAX_DIRECTION_WORD, true},
		{MAX_DIRECTION_WORD + 1, false},
		{MAX_CONVERTIBLE_NAME, false},
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
		{MAX_DIRECTION_WORD, true},
		{MAX_DIRECTION_WORD + 1, true},
		{MAX_CONVERTIBLE_NAME, true},
		{MAX_CONVERTIBLE_NAME + 1, false},
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
	require.Equal(t, NO_WORD_ID, w.ID)
	require.Zero(t, w.LabelID)
	require.Equal(t, NoWord, w.Kind)
	require.Equal(t, []string{"_"}, w.Synonyms)
}

func TestWordsNew(t *testing.T) {
	t.Run("invalid kind", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.New(1, NoWord, None)
		require.ErrorIs(t, err, dberrors.ErrInvalidWordKind)
		require.Zero(t, id)
		require.Len(t, ws, baseWordsCount, "a rejected word is not stored")
	})

	t.Run("duplicated label", func(t *testing.T) {
		ws := NewWordStore()
		_, err := ws.New(7, Verb, None, "COGER")
		require.NoError(t, err)

		id, err := ws.New(7, Noun, None, "LLAVE")
		require.ErrorIs(t, err, dberrors.ErrWordWithDuplicatedLabel)
		require.Zero(t, id)
		require.Len(t, ws, baseWordsCount+1, "the duplicate is not stored")
	})

	t.Run("id allocation error is propagated", func(t *testing.T) {
		ws := NewWordStore()
		fill(&ws, 0, MAX_DIRECTION_WORD)

		id, err := ws.New(1000, Verb, Direction, "NORTE")
		require.ErrorIs(t, err, dberrors.ErrWordConnectionIDsExhausted)
		require.Zero(t, id)
		require.Len(t, ws, baseWordsCount+MAX_DIRECTION_WORD.Int()+1, "nothing new is stored")
	})

	t.Run("stores the word", func(t *testing.T) {
		ws := NewWordStore()

		id, err := ws.New(42, Verb, None, "COGER", "TOMAR")
		require.NoError(t, err)
		require.Equal(t, MAX_PROPER_NOUN+1, id, "first general ID")
		require.Len(t, ws, baseWordsCount+1)

		w := ws[baseWordsCount]
		require.Equal(t, id, w.ID)
		require.Equal(t, ID16(42), w.LabelID)
		require.Equal(t, Verb, w.Kind)
		require.Equal(t, []string{"COGER", "TOMAR"}, w.Synonyms)
	})

	t.Run("stores a word without synonyms", func(t *testing.T) {
		ws := NewWordStore()

		id, err := ws.New(1, Noun, None)
		require.NoError(t, err)
		require.Equal(t, MAX_PROPER_NOUN+1, id)
		require.Empty(t, ws[baseWordsCount].Synonyms)
	})

	t.Run("stores the synonyms in vocabulary form", func(t *testing.T) {
		ws := NewWordStore()

		_, err := ws.New(1, Noun, None, " araña ", "bicho")
		require.NoError(t, err)
		require.Equal(t, []string{"ARANA", "BICHO"}, ws[baseWordsCount].Synonyms)
	})

	t.Run("each option draws from its own range", func(t *testing.T) {
		ws := NewWordStore()

		cases := []struct {
			option WordOption
			want   ID
			name   string
		}{
			{Direction, 0, "NORTE"},
			{Convertible, MAX_DIRECTION_WORD + 1, "LLAVE"},
			{ProperNoun, MAX_CONVERTIBLE_NAME + 1, "PACO"},
			{None, MAX_PROPER_NOUN + 1, "COGER"},
			{NotPronominal, LAST_PRONOMINAL_VERB + 1, "HABLA"},
		}

		for i, c := range cases {
			id, err := ws.New(ID16(i+1), Noun, c.option, c.name)
			require.NoError(t, err, "adding %q", c.name)
			require.Equal(t, c.want, id, "first ID of the %q range", c.name)
		}
	})
}

// A variadic parameter is a plain slice: calling New with "synonyms..." hands
// it the caller's own array, with no copy in between. New must normalize into
// a slice of its own so that neither side can reach into the other's.
func TestWordsNewDoesNotAliasSynonyms(t *testing.T) {
	t.Run("the caller's slice is left untouched", func(t *testing.T) {
		ws := NewWordStore()
		mine := []string{"araña", "bicho"}

		_, err := ws.New(1, Noun, None, mine...)
		require.NoError(t, err)

		require.Equal(t, []string{"araña", "bicho"}, mine,
			"New normalized in place instead of into its own slice")
		require.Equal(t, []string{"ARANA", "BICHO"}, ws[baseWordsCount].Synonyms,
			"the store does keep the vocabulary form")
	})

	t.Run("writing to the caller's slice does not reach the store", func(t *testing.T) {
		ws := NewWordStore()
		mine := []string{"ARANA", "BICHO"}

		_, err := ws.New(1, Noun, None, mine...)
		require.NoError(t, err)

		// The caller reuses its slice for something else.
		mine[0] = "PERRO"

		w, err := ws.GetBySynonym("ARANA")
		require.NoError(t, err, "the stored synonym survives the caller's write")
		require.Equal(t, []string{"ARANA", "BICHO"}, w.Synonyms)
	})

	t.Run("AddSynonym does not write into the caller's spare capacity", func(t *testing.T) {
		// The subtlest vector: if the store kept the caller's array, appending
		// a synonym would land in the unused tail the caller still owns.
		ws := NewWordStore()
		mine := make([]string, 0, 8)
		mine = append(mine, "ARANA", "BICHO")

		_, err := ws.New(1, Noun, None, mine...)
		require.NoError(t, err)

		w, ok := ws.Get(MAX_PROPER_NOUN + 1)
		require.True(t, ok)
		w.AddSynonym("MOSCA")

		tail := mine[:cap(mine)]
		require.Empty(t, tail[len(mine)], "the new synonym leaked into the caller's array")
		require.Equal(t, []string{"ARANA", "BICHO", "MOSCA"}, w.Synonyms)
	})

	t.Run("a word without synonyms keeps a nil slice", func(t *testing.T) {
		ws := NewWordStore()

		_, err := ws.New(1, Noun, None)
		require.NoError(t, err)
		require.Nil(t, ws[baseWordsCount].Synonyms)
	})
}

func TestSynonymPersistence(t *testing.T) {
	ws := NewWordStore()

	id, err := ws.New(1, Verb, None, "COGER")
	require.NoError(t, err)

	// A second word so the lookups have to pick the right one.
	_, err = ws.New(2, Noun, None, "LLAVE")
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
			_, err := ws.New(i, Noun, None, "PALABRA")
			require.NoError(t, err)
		}

		w, ok := ws.Get(id)
		require.True(t, ok)
		require.Equal(t, want, w.Synonyms)
	})
}

func TestWordsGet(t *testing.T) {
	ws := NewWordStore()
	id, err := ws.New(9, Verb, None, "COGER")
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
	id, err := ws.New(9, Verb, None, "COGER")
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
	_, err := ws.New(1, Verb, None, "COGER", "TOMAR")
	require.NoError(t, err)
	_, err = ws.New(2, Noun, None, "LLAVE")
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
		require.ErrorIs(t, err, dberrors.ErrWordNotFound)
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
	_, err := ws.New(1, Verb, None, "PUERTA")
	require.NoError(t, err)
	_, err = ws.New(2, Noun, None, "PUERTA")
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
		require.ErrorIs(t, err, dberrors.ErrWordNotFound)
		require.Nil(t, w)
	})

	t.Run("not found: unknown synonym", func(t *testing.T) {
		w, err := ws.GetByTypeAndSynonym(Verb, "VENTANA")
		require.ErrorIs(t, err, dberrors.ErrWordNotFound)
		require.Nil(t, w)
	})
}

// The five ID ranges, each one the exclusive territory of a WordOption.
var ranges = []struct {
	option       WordOption
	name         string
	first        ID
	last         ID
	exhaustedErr error
}{
	{Direction, "direction", 0, MAX_DIRECTION_WORD, dberrors.ErrWordConnectionIDsExhausted},
	{Convertible, "convertible", MAX_DIRECTION_WORD + 1, MAX_CONVERTIBLE_NAME, dberrors.ErrWordConvertibleIDsExhausted},
	{ProperNoun, "proper noun", MAX_CONVERTIBLE_NAME + 1, MAX_PROPER_NOUN, dberrors.ErrWordProperNounIDsExhausted},
	{None, "general", MAX_PROPER_NOUN + 1, LAST_PRONOMINAL_VERB, dberrors.ErrWordGeneralIDsExhausted},
	{NotPronominal, "non pronominal", LAST_PRONOMINAL_VERB + 1, MAX_WORD_ID, dberrors.ErrWordNonPronominalIDsExhausted},
}

func TestGetNextID(t *testing.T) {
	for _, r := range ranges {
		t.Run(r.name, func(t *testing.T) {
			t.Run("first free on an empty store", func(t *testing.T) {
				ws := NewWordStore()
				id, err := ws.getNextID(r.option)
				require.NoError(t, err)
				require.Equal(t, r.first, id)
			})

			t.Run("stays inside its own range", func(t *testing.T) {
				// Every other range is occupied: the allocator must not stray
				// into them, as they no longer fall back to one another.
				ws := NewWordStore()
				if r.first > 0 {
					fill(&ws, 0, r.first-1)
				}

				if r.last < MAX_WORD_ID {
					fill(&ws, r.last+1, MAX_WORD_ID)
				}

				id, err := ws.getNextID(r.option)
				require.NoError(t, err)
				require.Equal(t, r.first, id)
			})

			t.Run("fills an intermediate gap", func(t *testing.T) {
				ws := NewWordStore()
				gap := r.first + 1
				require.LessOrEqual(t, gap, r.last, "the range needs at least two IDs")

				fill(&ws, r.first, gap-1)
				fill(&ws, gap+1, r.last)

				id, err := ws.getNextID(r.option)
				require.NoError(t, err)
				require.Equal(t, gap, id)
			})

			t.Run("exhausted", func(t *testing.T) {
				ws := NewWordStore()
				fill(&ws, r.first, r.last)

				id, err := ws.getNextID(r.option)
				require.ErrorIs(t, err, r.exhaustedErr)
				require.Zero(t, id)
			})
		})
	}

	t.Run("an unknown option falls back to the general range", func(t *testing.T) {
		ws := NewWordStore()
		id, err := ws.getNextID(WordOption(99))
		require.NoError(t, err)
		require.Equal(t, MAX_PROPER_NOUN+1, id)
	})

	t.Run("the ranges cover 0 to MAX_WORD_ID without gaps or overlaps", func(t *testing.T) {
		require.Equal(t, ID(0), ranges[0].first, "the first range starts at zero")
		require.Equal(t, MAX_WORD_ID, ranges[len(ranges)-1].last, "the last one ends at MAX_WORD_ID")

		for i := 1; i < len(ranges); i++ {
			require.Equal(t, ranges[i-1].last+1, ranges[i].first,
				"the %q range must start right after %q", ranges[i].name, ranges[i-1].name)
		}
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
			require.LessOrEqual(t, len([]rune(NormalizeWord(c.in))), MAX_WORD_LEN)
		})
	}
}
