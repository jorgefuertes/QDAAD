package decompiler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Several words can share a value and a type — they are synonyms — and only one
// of them can be written where the database holds the value. Which one is a
// convention, because the binary keeps the number and not the spelling the
// author typed.
//
// The table comes out of the database in the order the compiler's tree left it,
// which is the DAAD character set's order, and that set keeps the accented
// letters below "A". So the accented spelling of a word sorts first and taking
// the first match wrote "LáMPA" where every other tool and every source writes
// "LAMPA". It is the same word; it just looked like a different one in a diff.
// Issue #7.
func TestWordForPrefersTheSpellingWithoutAccents(t *testing.T) {
	const noun = kindNoun

	for _, tc := range []struct {
		name  string
		words []VocabularyEntry
		want  string
	}{
		{
			name: "the plain twin wins however far down it is",
			words: []VocabularyEntry{
				{Word: "LáMPA", Value: 50, Kind: noun},
				{Word: "LINTE", Value: 50, Kind: noun},
				{Word: "LAMPA", Value: 50, Kind: noun},
			},
			want: "LAMPA",
		},
		{
			name: "a word with no twin keeps its accents",
			words: []VocabularyEntry{
				{Word: "CABAÑ", Value: 56, Kind: noun},
				{Word: "CASA", Value: 56, Kind: noun},
				{Word: "CAVER", Value: 56, Kind: noun},
			},
			want: "CABAÑ",
		},
		{
			// The ñ is a letter and not an accent. These entries are there for
			// a player with no ñ key, and the spider is spelled with one.
			name: "the n is never taken for the enye",
			words: []VocabularyEntry{
				{Word: "ARAÑA", Value: 70, Kind: noun},
				{Word: "ARANA", Value: 70, Kind: noun},
			},
			want: "ARAÑA",
		},
		{
			name: "a plain word first is left where it is",
			words: []VocabularyEntry{
				{Word: "ARBOL", Value: 58, Kind: noun},
				{Word: "áRBOL", Value: 58, Kind: noun},
			},
			want: "ARBOL",
		},
		{
			name: "one word on its own",
			words: []VocabularyEntry{
				{Word: "PIRáM", Value: 70, Kind: noun},
			},
			want: "PIRáM",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Reader{vocabulary: tc.words}

			got, found := r.WordFor(tc.words[0].Value, noun)
			require.True(t, found)
			require.Equal(t, tc.want, got)
		})
	}
}

// The kinds are tried in the order the caller gives them, and a value that has
// a word of the first kind never falls through to the second.
func TestWordForTakesTheKindsInOrder(t *testing.T) {
	r := &Reader{vocabulary: []VocabularyEntry{
		{Word: "NORTE", Value: 1, Kind: kindNoun},
		{Word: "N", Value: 1, Kind: kindVerb},
	}}

	got, found := r.WordFor(1, kindVerb, kindNoun)
	require.True(t, found)
	require.Equal(t, "N", got, "the verb is asked for first")

	got, found = r.WordFor(1, kindNoun, kindVerb)
	require.True(t, found)
	require.Equal(t, "NORTE", got)

	_, found = r.WordFor(99, kindNoun)
	require.False(t, found)
}
