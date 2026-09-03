package ddb

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb/dberrors"
	"github.com/stretchr/testify/require"
)

func TestLocationsAdd(t *testing.T) {
	ls := NewLocationStore()

	first, err := ls.Add(10, "A dark cave.")
	require.NoError(t, err)
	require.Equal(t, ID(0), first, "the first location takes zero")

	second, err := ls.Add(11, "A lit cave.")
	require.NoError(t, err)
	require.Equal(t, ID(1), second, "and they go on from there")

	got, err := ls.Get(second)
	require.NoError(t, err)
	require.Equal(t, "A lit cave.", got.Description)
	require.Equal(t, ID16(11), got.LabelID)
}

// Add hands out the lowest free number rather than the next one after the last,
// so a store built with explicit numbers and then added to does not collide.
func TestLocationsAddFillsAHole(t *testing.T) {
	ls := NewLocationStore()

	require.NoError(t, ls.NewLegacy(0, "zero"))
	require.NoError(t, ls.NewLegacy(2, "two"))

	id, err := ls.Add(0, "the hole")
	require.NoError(t, err)
	require.Equal(t, ID(1), id)
}

func TestLocationsNewLegacy(t *testing.T) {
	t.Run("takes the number the source gives", func(t *testing.T) {
		ls := NewLocationStore()

		require.NoError(t, ls.NewLegacy(7, "A clearing."))

		got, err := ls.Get(7)
		require.NoError(t, err)
		require.Equal(t, "A clearing.", got.Description)
		require.Equal(t, LabelUndefined, got.LabelID,
			"a legacy source names nothing, so there is no label")
	})

	// 252 to 255 are the markers —not created, worn, carried, here— so a real
	// location cannot reach them.
	t.Run("refuses a number the markers own", func(t *testing.T) {
		ls := NewLocationStore()

		for _, id := range []ID{LOC_NOT_CREATED, LOC_WORN, LOC_CARRIED, LOC_HERE} {
			require.ErrorIs(t, ls.NewLegacy(id, "nowhere"), dberrors.ErrLocationIDOutOfRange,
				"location %d", id)
		}

		require.NoError(t, ls.NewLegacy(MAX_LOCATION_ID, "the last one that fits"))
	})

	t.Run("refuses a number already used", func(t *testing.T) {
		ls := NewLocationStore()

		require.NoError(t, ls.NewLegacy(3, "first"))
		require.ErrorIs(t, ls.NewLegacy(3, "second"), dberrors.ErrLocationAlreadyExists)

		got, err := ls.Get(3)
		require.NoError(t, err)
		require.Equal(t, "first", got.Description, "the first one stays")
	})
}

func TestLocationsGet(t *testing.T) {
	ls := NewLocationStore()
	require.NoError(t, ls.NewLegacy(4, "somewhere"))

	t.Run("by number", func(t *testing.T) {
		got, err := ls.Get(4)
		require.NoError(t, err)
		require.Equal(t, ID(4), got.ID)
	})

	t.Run("by label", func(t *testing.T) {
		id, err := ls.Add(9, "named")
		require.NoError(t, err)

		got, err := ls.GetByLabelID(9)
		require.NoError(t, err)
		require.Equal(t, id, got.ID)
	})

	t.Run("what is not there", func(t *testing.T) {
		_, err := ls.Get(200)
		require.ErrorIs(t, err, dberrors.ErrLocationNotFound)

		_, err = ls.GetByLabelID(200)
		require.ErrorIs(t, err, dberrors.ErrLocationNotFound)
	})
}

// The getter hands back a pointer into the store, so a description can be
// rewritten through it. That is deliberate and worth pinning down.
func TestLocationsGetReachesTheStore(t *testing.T) {
	ls := NewLocationStore()
	require.NoError(t, ls.NewLegacy(0, "before"))

	got, err := ls.Get(0)
	require.NoError(t, err)
	got.Description = "after"

	again, err := ls.Get(0)
	require.NoError(t, err)
	require.Equal(t, "after", again.Description)
}

func TestLocationsFillUp(t *testing.T) {
	ls := NewLocationStore()

	for id := ID(0); id <= MAX_LOCATION_ID; id++ {
		require.NoError(t, ls.NewLegacy(id, "somewhere"))
	}

	_, err := ls.Add(0, "one too many")
	require.ErrorIs(t, err, dberrors.ErrLocationStoreIsFull)
}
