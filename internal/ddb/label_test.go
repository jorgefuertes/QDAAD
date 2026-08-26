package ddb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A fresh store is not empty: it holds the undefined label at ID 0, so the
// counts below are always one above the number of labels added by the test.
const baseLabelsCount = 1

// firstLabelID is the ID the store hands out first: ID 0 belongs to the
// undefined label, so real labels start at 1 and run dense from there.
const firstLabelID ID16 = 1

func TestNewLabelStore(t *testing.T) {
	ls := NewLabelStore()
	require.NotNil(t, ls, "the slice is allocated, not nil")
	require.Len(t, ls, baseLabelsCount, "the store starts with the undefined label")

	require.Equal(t, LabelUndefined, ls[0].ID)
	require.Equal(t, "undefined", ls[0].Name)
}

func TestLabelStoreAdd(t *testing.T) {
	t.Run("assigns ascending IDs", func(t *testing.T) {
		ls := NewLabelStore()

		for i, name := range []string{"_PUERTA", "_LLAVE", "_COGER"} {
			id, err := ls.Add(name)
			require.NoError(t, err)
			require.Equal(t, firstLabelID+ID16(i), id, "adding %q", name)
		}

		require.Len(t, ls, baseLabelsCount+3)
	})

	t.Run("the undefined ID is never assigned", func(t *testing.T) {
		// Get and GetByName report absence with a zero ID, so no real label may
		// own it: it belongs to the undefined entry the store is born with.
		ls := NewLabelStore()

		id, err := ls.Add("_PUERTA")
		require.NoError(t, err)
		require.NotEqual(t, LabelUndefined, id)
	})

	t.Run("a repeated name returns the same ID", func(t *testing.T) {
		ls := NewLabelStore()

		first, err := ls.Add("_PUERTA")
		require.NoError(t, err)

		_, err = ls.Add("_LLAVE")
		require.NoError(t, err)

		again, err := ls.Add("_PUERTA")
		require.NoError(t, err)
		require.Equal(t, first, again, "the label is not duplicated")
		require.Len(t, ls, baseLabelsCount+2, "nothing new is stored")
	})

	t.Run("names are compared literally", func(t *testing.T) {
		ls := NewLabelStore()

		upper, err := ls.Add("_PUERTA")
		require.NoError(t, err)

		lower, err := ls.Add("_puerta")
		require.NoError(t, err)

		require.NotEqual(t, upper, lower, "case tells two labels apart")
		require.Len(t, ls, baseLabelsCount+2)
	})

	t.Run("an empty name is a label like any other", func(t *testing.T) {
		ls := NewLabelStore()

		id, err := ls.Add("")
		require.NoError(t, err)
		require.Equal(t, firstLabelID, id)

		again, err := ls.Add("")
		require.NoError(t, err)
		require.Equal(t, id, again)
	})

	t.Run("numbering is dense: no ID is skipped", func(t *testing.T) {
		// The undefined label owns 0, and the next one must be 1. Starting the
		// search one too high leaves a hole that nothing else would catch.
		ls := NewLabelStore()

		for i := range 10 {
			id, err := ls.Add("_ETIQUETA" + string(rune('A'+i)))
			require.NoError(t, err)
			require.Equal(t, ID16(i)+firstLabelID, id, "label number %d", i)
		}

		for i, l := range ls {
			require.Equal(t, ID16(i), l.ID, "position %d holds a different ID", i)
		}
	})

	t.Run("the undefined label is reachable by name", func(t *testing.T) {
		ls := NewLabelStore()

		id, exists := ls.GetByName("undefined")
		require.True(t, exists)
		require.Equal(t, LabelUndefined, id)

		name, exists := ls.Get(LabelUndefined)
		require.True(t, exists)
		require.Equal(t, "undefined", name)
	})

	t.Run("the next ID follows the highest one, not the count", func(t *testing.T) {
		ls := NewLabelStore()
		ls = append(ls, Label{ID: 40, Name: "_PUERTA"}, Label{ID: 7, Name: "_LLAVE"})

		id, err := ls.Add("_COGER")
		require.NoError(t, err)
		require.Equal(t, ID16(41), id, "IDs stay unique even with gaps or out of order")
	})
}

func TestLabelStoreGet(t *testing.T) {
	ls := NewLabelStore()
	id, err := ls.Add("_PUERTA")
	require.NoError(t, err)

	// A second label so the lookup has to pick the right one.
	_, err = ls.Add("_LLAVE")
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		name, ok := ls.Get(id)
		require.True(t, ok)
		require.Equal(t, "_PUERTA", name)
	})

	t.Run("not found", func(t *testing.T) {
		name, ok := ls.Get(1000)
		require.False(t, ok)
		require.Empty(t, name)
	})

	t.Run("not found on an empty store", func(t *testing.T) {
		name, ok := NewLabelStore().Get(1)
		require.False(t, ok)
		require.Empty(t, name)
	})
}

func TestLabelStoreGetByName(t *testing.T) {
	ls := NewLabelStore()
	id, err := ls.Add("_PUERTA")
	require.NoError(t, err)

	_, err = ls.Add("_LLAVE")
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, ok := ls.GetByName("_PUERTA")
		require.True(t, ok)
		require.Equal(t, id, got)
	})

	t.Run("not found", func(t *testing.T) {
		got, ok := ls.GetByName("_VENTANA")
		require.False(t, ok)
		require.Zero(t, got)
	})

	t.Run("not found on an empty store", func(t *testing.T) {
		got, ok := NewLabelStore().GetByName("_PUERTA")
		require.False(t, ok)
		require.Zero(t, got)
	})
}

func TestLabelRoundTrip(t *testing.T) {
	ls := NewLabelStore()
	names := []string{"_NORTE", "_SUR", "_ESTE", "_OESTE"}

	ids := make([]ID16, 0, len(names))
	for _, name := range names {
		id, err := ls.Add(name)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	for i, name := range names {
		id, ok := ls.GetByName(name)
		require.True(t, ok, "looking up %q", name)
		require.Equal(t, ids[i], id)

		got, ok := ls.Get(id)
		require.True(t, ok, "looking up ID %d", id)
		require.Equal(t, name, got, "name and ID map back to each other")
	}
}
