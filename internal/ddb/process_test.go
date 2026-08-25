package ddb

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/qderror"
	"github.com/stretchr/testify/require"
)

// Opcodes used as fixtures, picked for their arity.
const (
	opCLS     Opcode = 29  // 0 parameters
	opDESC    Opcode = 19  // 1 parameter
	opLET     Opcode = 51  // 2 parameters
	opPROCESS Opcode = 75  // 1 parameter
	opSETAT   Opcode = 124 // 2 parameters
)

func TestNewProcessStore(t *testing.T) {
	ps := NewProcessStore()
	require.NotNil(t, ps, "the slice is allocated, not nil")
	require.Empty(t, ps)
}

func TestProcessesAdd(t *testing.T) {
	t.Run("numbering is sequential and dense from zero", func(t *testing.T) {
		ps := NewProcessStore()

		for i := range 4 {
			id, err := ps.Add(ID16(i + 1))
			require.NoError(t, err)
			require.Equal(t, ID(i), id, "process number %d", i)
		}

		require.Len(t, ps, 4)

		for i := range ps {
			require.Equal(t, ID(i), ps[i].ID)
			require.Equal(t, ID16(i+1), ps[i].LabelID)
			require.Empty(t, ps[i].Entries, "a fresh process has no entries")
		}
	})

	t.Run("exhausted", func(t *testing.T) {
		ps := NewProcessStore()

		for i := 0; i <= MAX_PROCESS_ID.Int(); i++ {
			_, err := ps.Add(ID16(i))
			require.NoError(t, err, "process number %d", i)
		}

		require.Len(t, ps, MAX_PROCESS_ID.Int()+1, "255 processes, numbered 0 to 254")

		id, err := ps.Add(1000)
		require.ErrorIs(t, err, qderror.ErrProcessStoreIsFull)
		require.Zero(t, id)
		require.Len(t, ps, MAX_PROCESS_ID.Int()+1, "nothing new is stored")
	})
}

func TestProcessesAddEntry(t *testing.T) {
	t.Run("without a process", func(t *testing.T) {
		ps := NewProcessStore()
		require.ErrorIs(t, ps.AddEntry(1, 2), qderror.ErrProcessNoProcess)
	})

	t.Run("entries land on the last process", func(t *testing.T) {
		ps := NewProcessStore()

		_, err := ps.Add(1)
		require.NoError(t, err)
		require.NoError(t, ps.AddEntry(10, 20))

		_, err = ps.Add(2)
		require.NoError(t, err)
		require.NoError(t, ps.AddEntry(30, 40))
		require.NoError(t, ps.AddEntry(50, 60))

		require.Len(t, ps[0].Entries, 1, "the first process is closed and untouched")
		require.Equal(t, Entry{Verb: 10, Noun: 20}, ps[0].Entries[0])

		require.Len(t, ps[1].Entries, 2)
		require.Equal(t, ID(30), ps[1].Entries[0].Verb)
		require.Equal(t, ID(50), ps[1].Entries[1].Verb)
	})

	t.Run("wildcards", func(t *testing.T) {
		// NO_WORD_ID is the "_" of the source: it matches anything.
		ps := NewProcessStore()

		_, err := ps.Add(1)
		require.NoError(t, err)
		require.NoError(t, ps.AddEntry(NO_WORD_ID, NO_WORD_ID))

		require.Equal(t, NO_WORD_ID, ps[0].Entries[0].Verb)
		require.Equal(t, NO_WORD_ID, ps[0].Entries[0].Noun)
	})
}

func TestProcessesAddCondact(t *testing.T) {
	t.Run("without a process", func(t *testing.T) {
		ps := NewProcessStore()
		require.ErrorIs(t, ps.AddCondact(opCLS), qderror.ErrProcessNoProcess)
	})

	t.Run("without an entry", func(t *testing.T) {
		ps := NewProcessStore()
		_, err := ps.Add(1)
		require.NoError(t, err)

		require.ErrorIs(t, ps.AddCondact(opCLS), qderror.ErrProcessNoEntry)
	})

	t.Run("arity", func(t *testing.T) {
		cases := []struct {
			name   string
			op     Opcode
			params []Param
			err    error
		}{
			{"no parameters", opCLS, nil, nil},
			{"one parameter", opDESC, []Param{{Value: 7}}, nil},
			{"two parameters", opLET, []Param{{Value: 100}, {Value: 200}}, nil},
			{"too few", opLET, []Param{{Value: 100}}, qderror.ErrCondactParamCount},
			{"too many", opCLS, []Param{{Value: 1}}, qderror.ErrCondactParamCount},
			{"reserved opcode", 120, nil, qderror.ErrInvalidOpcode},
			{"beyond the table", 200, nil, qderror.ErrInvalidOpcode},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				ps := NewProcessStore()
				_, err := ps.Add(1)
				require.NoError(t, err)
				require.NoError(t, ps.AddEntry(1, 2))

				err = ps.AddCondact(c.op, c.params...)
				if c.err != nil {
					require.ErrorIs(t, err, c.err)
					require.Empty(t, ps[0].Entries[0].Condacts, "a rejected condact is not stored")

					return
				}

				require.NoError(t, err)
				require.Len(t, ps[0].Entries[0].Condacts, 1)
				require.Equal(t, c.op, ps[0].Entries[0].Condacts[0].Opcode)
				require.Len(t, ps[0].Entries[0].Condacts[0].Params, len(c.params))
			})
		}
	})

	t.Run("SETAT, which only exists in v3, is a condact like any other", func(t *testing.T) {
		ps := NewProcessStore()
		_, err := ps.Add(1)
		require.NoError(t, err)
		require.NoError(t, ps.AddEntry(1, 2))

		require.NoError(t, ps.AddCondact(opSETAT, Param{Value: 3}, Param{Value: 1}))
	})

	t.Run("condacts land on the last entry of the last process", func(t *testing.T) {
		ps := NewProcessStore()

		_, err := ps.Add(1)
		require.NoError(t, err)
		require.NoError(t, ps.AddEntry(1, 2))
		require.NoError(t, ps.AddCondact(opCLS))

		require.NoError(t, ps.AddEntry(3, 4))
		require.NoError(t, ps.AddCondact(opDESC, Param{Value: 9}))
		require.NoError(t, ps.AddCondact(opPROCESS, Param{Value: 3}))

		require.Len(t, ps[0].Entries[0].Condacts, 1, "the first entry is closed and untouched")
		require.Equal(t, opCLS, ps[0].Entries[0].Condacts[0].Opcode)

		require.Len(t, ps[0].Entries[1].Condacts, 2)
		require.Equal(t, opDESC, ps[0].Entries[1].Condacts[0].Opcode)
		require.Equal(t, opPROCESS, ps[0].Entries[1].Condacts[1].Opcode)
	})

	t.Run("indirection is kept per parameter", func(t *testing.T) {
		// "LET 100 @200": how that reaches the binary —bit 7 of the opcode, or
		// a whole INDIR condact in front— is the emitter's business.
		ps := NewProcessStore()

		_, err := ps.Add(1)
		require.NoError(t, err)
		require.NoError(t, ps.AddEntry(1, 2))
		require.NoError(t, ps.AddCondact(opLET, Param{Value: 100}, Param{Value: 200, Indirect: true}))

		params := ps[0].Entries[0].Condacts[0].Params
		require.Equal(t, Param{Value: 100, Indirect: false}, params[0])
		require.Equal(t, Param{Value: 200, Indirect: true}, params[1])
	})
}

// A variadic parameter is a plain slice: calling AddCondact with "params..."
// hands it the caller's own array, with no copy in between.
func TestProcessesAddCondactDoesNotAliasParams(t *testing.T) {
	ps := NewProcessStore()

	_, err := ps.Add(1)
	require.NoError(t, err)
	require.NoError(t, ps.AddEntry(1, 2))

	mine := []Param{{Value: 100}, {Value: 200}}

	require.NoError(t, ps.AddCondact(opLET, mine...))
	require.NoError(t, ps.AddCondact(opLET, mine...))

	// The caller reuses its slice for something else.
	mine[0] = Param{Value: 7, Indirect: true}

	condacts := ps[0].Entries[0].Condacts
	require.Len(t, condacts, 2)

	for i, condact := range condacts {
		require.Equal(t, Param{Value: 100}, condact.Params[0],
			"condact %d shares its array with the caller", i)
		require.Equal(t, Param{Value: 200}, condact.Params[1], "condact %d", i)
	}
}
