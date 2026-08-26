package ddb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLabelStore(t *testing.T) {
	ls := NewLabelStore()
	require.NotNil(t, ls)
	require.NotNil(t, ls, "the slice is allocated, not nil")
	require.Len(t, ls, 1, "the store starts with the undefined label")
}

func TestLabelStoreAdd(t *testing.T) {
	t.Run("assigns sequential IDs starting at one", func(t *testing.T) {
		ls := NewLabelStore()

		for i, name := range []string{"_PUERTA", "_LLAVE", "_COGER"} {
			id, err := ls.Add(name)
			require.NoError(t, err)
			require.Equal(t, ID16(i+1), id, "adding %q", name)
		}

		require.Len(t, ls, 3)
	})

	t.Run("zero is never assigned", func(t *testing.T) {
		// Get and GetByName report absence with a zero ID, so no label may own it.
		ls := NewLabelStore()

		id, err := ls.Add("_PUERTA")
		require.NoError(t, err)
		require.NotZero(t, id)
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
		require.Len(t, ls, 2, "nothing new is stored")
	})

	t.Run("names are compared literally", func(t *testing.T) {
		ls := NewLabelStore()

		upper, err := ls.Add("_PUERTA")
		require.NoError(t, err)

		lower, err := ls.Add("_puerta")
		require.NoError(t, err)

		require.NotEqual(t, upper, lower, "case tells two labels apart")
		require.Len(t, ls, 2)
	})

	t.Run("an empty name is a label like any other", func(t *testing.T) {
		ls := NewLabelStore()

		id, err := ls.Add("")
		require.NoError(t, err)
		require.Equal(t, ID16(1), id)

		again, err := ls.Add("")
		require.NoError(t, err)
		require.Equal(t, id, again)
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
