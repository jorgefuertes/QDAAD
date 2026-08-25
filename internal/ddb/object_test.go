package ddb

import (
	"math"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/qderror"
	"github.com/stretchr/testify/require"
)

// newObject builds a valid object: every one needs its own label, as Add
// rejects a repeated one —including the zero value.
func newObject(labelID ID16, name, adjective ID) Object {
	return Object{LabelID: labelID, Name: name, Adjective: adjective}
}

func TestNewObjectStore(t *testing.T) {
	ost := NewObjectStore()
	require.NotNil(t, ost)
	require.NotNil(t, ost, "the slice is allocated, not nil")
	require.Empty(t, ost)
}

func TestObjectsAdd(t *testing.T) {
	t.Run("assigns sequential IDs starting at zero", func(t *testing.T) {
		ost := NewObjectStore()

		for i := range 3 {
			require.NoError(t, ost.Add(newObject(ID16(i+1), ID(i+1), NO_WORD_ID)))
			require.Equal(t, ID(i), ost[i].ID, "object number %d", i)
		}

		require.Len(t, ost, 3)
	})

	t.Run("an object cannot come with an ID", func(t *testing.T) {
		ost := NewObjectStore()

		obj := newObject(1, 1, NO_WORD_ID)
		obj.ID = 7

		require.ErrorIs(t, ost.Add(obj), qderror.ErrCannotCreateWithID)
		require.Empty(t, ost, "a rejected object is not stored")
	})

	t.Run("duplicated label", func(t *testing.T) {
		ost := NewObjectStore()
		require.NoError(t, ost.Add(newObject(1, 1, NO_WORD_ID)))

		err := ost.Add(newObject(1, 2, NO_WORD_ID))
		require.ErrorIs(t, err, qderror.ErrObjectDuplicatedLabel)
		require.Len(t, ost, 1, "the duplicate is not stored")
	})

	t.Run("duplicated name and adjective", func(t *testing.T) {
		ost := NewObjectStore()
		require.NoError(t, ost.Add(newObject(1, 1, 2)))

		err := ost.Add(newObject(2, 1, 2))
		require.ErrorIs(t, err, qderror.ErrObjectDuplicatedNameAdjective)
		require.Len(t, ost, 1, "the duplicate is not stored")
	})

	t.Run("the adjective tells two objects with the same name apart", func(t *testing.T) {
		ost := NewObjectStore()
		require.NoError(t, ost.Add(newObject(1, 1, 2)))
		require.NoError(t, ost.Add(newObject(2, 1, 3)))
		require.Len(t, ost, 2)
	})

	t.Run("stores every field as given", func(t *testing.T) {
		ost := NewObjectStore()

		obj := Object{
			LabelID:      42,
			Name:         7,
			Adjective:    8,
			Weight:       63,
			Container:    true,
			Wearable:     true,
			InitLocation: LOC_CARRIED,
		}
		obj.Flags[3] = true

		require.NoError(t, ost.Add(obj))

		stored := ost[0]
		require.Equal(t, ID(0), stored.ID, "only the ID is set by the store")

		obj.ID = stored.ID
		require.Equal(t, obj, stored)
	})

	t.Run("id allocation error is propagated", func(t *testing.T) {
		ost := NewObjectStore()
		for i := range math.MaxUint8 + 1 {
			ost = append(ost, Object{ID: ID(i), LabelID: ID16(i + 1)})
		}

		err := ost.Add(newObject(1000, 1, NO_WORD_ID))
		require.ErrorIs(t, err, qderror.ErrObjectStoreIsFull)
		require.Len(t, ost, math.MaxUint8+1, "nothing new is stored")
	})

	t.Run("fills an intermediate gap", func(t *testing.T) {
		ost := NewObjectStore()
		ost = append(ost,
			Object{ID: 0, LabelID: 1, Name: 1},
			Object{ID: 1, LabelID: 2, Name: 2},
			Object{ID: 3, LabelID: 3, Name: 3},
		)

		require.NoError(t, ost.Add(newObject(4, 4, NO_WORD_ID)))
		require.Equal(t, ID(2), ost[3].ID, "the free ID is reused")
	})
}

func TestObjectsGet(t *testing.T) {
	ost := NewObjectStore()
	require.NoError(t, ost.Add(newObject(1, 1, NO_WORD_ID)))
	require.NoError(t, ost.Add(newObject(2, 2, NO_WORD_ID)))

	t.Run("found", func(t *testing.T) {
		obj, err := ost.Get(1)
		require.NoError(t, err)
		require.Equal(t, ID16(2), obj.LabelID, "the right one, not the first")
	})

	t.Run("not found", func(t *testing.T) {
		obj, err := ost.Get(2)
		require.ErrorIs(t, err, qderror.ErrObjectNotFound)
		require.Nil(t, obj)
	})

	t.Run("the getter returns a pointer into the store", func(t *testing.T) {
		obj, err := ost.Get(0)
		require.NoError(t, err)

		obj.Weight = 12

		obj, err = ost.Get(0)
		require.NoError(t, err)
		require.Equal(t, uint8(12), obj.Weight, "the change is kept")
	})
}

func TestObjectsGetByLabelID(t *testing.T) {
	ost := NewObjectStore()
	require.NoError(t, ost.Add(newObject(9, 1, NO_WORD_ID)))
	require.NoError(t, ost.Add(newObject(10, 2, NO_WORD_ID)))

	t.Run("found", func(t *testing.T) {
		obj, err := ost.GetByLabelID(10)
		require.NoError(t, err)
		require.Equal(t, ID(1), obj.ID)
	})

	t.Run("not found", func(t *testing.T) {
		obj, err := ost.GetByLabelID(11)
		require.ErrorIs(t, err, qderror.ErrObjectNotFound)
		require.Nil(t, obj)
	})
}

func TestObjectsGetByName(t *testing.T) {
	ost := NewObjectStore()
	require.NoError(t, ost.Add(newObject(1, 7, 20)))
	require.NoError(t, ost.Add(newObject(2, 7, 21)))
	require.NoError(t, ost.Add(newObject(3, 8, 20)))

	t.Run("returns the first match", func(t *testing.T) {
		obj, err := ost.GetByName(7)
		require.NoError(t, err)
		require.Equal(t, ID(0), obj.ID, "the adjective is not taken into account")
	})

	t.Run("not found", func(t *testing.T) {
		obj, err := ost.GetByName(9)
		require.ErrorIs(t, err, qderror.ErrObjectNotFound)
		require.Nil(t, obj)
	})
}

func TestObjectsGetByNameAdjective(t *testing.T) {
	ost := NewObjectStore()
	require.NoError(t, ost.Add(newObject(1, 7, 20)))
	require.NoError(t, ost.Add(newObject(2, 7, 21)))

	t.Run("discriminates by adjective", func(t *testing.T) {
		obj, err := ost.GetByNameAdjective(7, 20)
		require.NoError(t, err)
		require.Equal(t, ID16(1), obj.LabelID)

		obj, err = ost.GetByNameAdjective(7, 21)
		require.NoError(t, err)
		require.Equal(t, ID16(2), obj.LabelID)
	})

	t.Run("not found: right name, wrong adjective", func(t *testing.T) {
		obj, err := ost.GetByNameAdjective(7, 22)
		require.ErrorIs(t, err, qderror.ErrObjectNotFound)
		require.Nil(t, obj)
	})

	t.Run("not found: unknown name", func(t *testing.T) {
		obj, err := ost.GetByNameAdjective(8, 20)
		require.ErrorIs(t, err, qderror.ErrObjectNotFound)
		require.Nil(t, obj)
	})
}

func TestObjectsGetNextID(t *testing.T) {
	t.Run("first free on an empty store", func(t *testing.T) {
		ost := NewObjectStore()
		id, err := ost.getNextID()
		require.NoError(t, err)
		require.Equal(t, ID(0), id)
	})

	t.Run("fills an intermediate gap", func(t *testing.T) {
		ost := NewObjectStore()
		for i := range 100 {
			ost = append(ost, Object{ID: ID(i), LabelID: ID16(i + 1)})
		}
		ost = append(ost[:50], ost[51:]...)

		id, err := ost.getNextID()
		require.NoError(t, err)
		require.Equal(t, ID(50), id)
	})

	t.Run("exhausted", func(t *testing.T) {
		ost := NewObjectStore()
		for i := range math.MaxUint8 + 1 {
			ost = append(ost, Object{ID: ID(i), LabelID: ID16(i + 1)})
		}

		id, err := ost.getNextID()
		require.ErrorIs(t, err, qderror.ErrObjectStoreIsFull)
		require.Zero(t, id)
	})
}
