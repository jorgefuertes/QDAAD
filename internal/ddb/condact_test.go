package ddb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The one hole left in the opcode space: 137, the XUNDONE that v3 deprecated.
//
// 120 used to be the other. It is the translated XMES, and it was empty for as
// long as nothing wrote one; now that the back end does, it is a condact like
// any other except that no author can name it. See Emitted.
var reservedOpcodes = []Opcode{137}

func TestCondactTableIsCoherent(t *testing.T) {
	t.Run("every opcode is either named or reserved", func(t *testing.T) {
		for op := range condacts {
			def := condacts[op]
			if def.Name != "" {
				continue
			}

			require.Contains(t, reservedOpcodes, Opcode(op),
				"opcode %d has no name and is not one of the reserved holes", op)
		}
	})

	t.Run("arity never exceeds the three parameter slots", func(t *testing.T) {
		for op := range condacts {
			require.LessOrEqual(t, condacts[op].NumParams(), MAX_CONDACT_PARAMS,
				"opcode %d (%s)", op, condacts[op].Name)
		}
	})

	t.Run("the reserved holes take no parameters", func(t *testing.T) {
		for _, op := range reservedOpcodes {
			require.Zero(t, condacts[op].NumParams(), "opcode %d", op)
		}
	})

	t.Run("only the real opcodes reach the binary", func(t *testing.T) {
		for op := range condacts {
			if condacts[op].Name == "" {
				continue // a reserved slot is on neither side
			}

			require.Equal(t, Opcode(op) >= NUM_CONDACTS, condacts[op].Fake,
				"opcode %d (%s) is on the wrong side of the fake boundary", op, condacts[op].Name)
		}
	})

	// Unique among the ones a source can name, which is not the same as unique.
	// A pseudo-condact and the opcode it is translated into share a name on
	// purpose — XMES is 128 written and 120 emitted — and what must not happen
	// is a source naming the one it cannot write.
	t.Run("names a source can write are unique", func(t *testing.T) {
		seen := make(map[string]int, len(condacts))

		for op := range condacts {
			name := condacts[op].Name
			if name == "" || condacts[op].Emitted {
				continue
			}

			previous, duplicated := seen[name]
			require.False(t, duplicated, "%s is both opcode %d and %d", name, previous, op)
			seen[name] = op
		}
	})

	t.Run("an emitted condact shares its name with the one it translates", func(t *testing.T) {
		for op := range condacts {
			if !condacts[op].Emitted {
				continue
			}

			written, _, known := LookupCondactByName(condacts[op].Name)

			require.True(t, known, "opcode %d is emitted as %s and no source writes that",
				op, condacts[op].Name)
			require.True(t, condacts[written].Fake,
				"%s resolves to opcode %d, which is not a pseudo-condact",
				condacts[op].Name, written)
		}
	})
}

func TestCondactTerminals(t *testing.T) {
	// A terminal condact ends the block by itself, so the emitter can save the
	// 0xFF terminator (drb.php:1050).
	want := []Opcode{22, 23, 103, 108, 116, 117}

	var got []Opcode

	for op := range condacts {
		if condacts[op].Terminal {
			got = append(got, Opcode(op))
		}
	}

	require.Equal(t, want, got, "DONE, OK, NOTDONE, REDO, SKIP and RESTART, and no others")
}

func TestCondactConditions(t *testing.T) {
	// The 32 conditions, from 04-condactos.md.
	want := []Opcode{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17,
		55, 58, 59, 68, 69, 70, 76, 79, 80, 88, 112, 113, 114, 115,
	}

	var got []Opcode

	for op := range condacts {
		if condacts[op].Condition {
			got = append(got, Opcode(op))
		}
	}

	require.Equal(t, want, got)
	require.Len(t, got, 32)
}

func TestLookupCondact(t *testing.T) {
	t.Run("resolves by opcode", func(t *testing.T) {
		cases := []struct {
			op        Opcode
			name      string
			numParams int
		}{
			{51, "LET", 2},
			{29, "CLS", 0},
			{122, "INDIR", 1},
			{124, "SETAT", 2},
			{143, "GETKEY", 0},
		}

		for _, c := range cases {
			def, found := LookupCondact(c.op)
			require.True(t, found, "opcode %d should resolve", c.op)
			require.Equal(t, c.name, def.Name)
			require.Equal(t, c.numParams, def.NumParams())
		}
	})

	t.Run("the reserved opcode does not resolve", func(t *testing.T) {
		for _, op := range reservedOpcodes {
			def, found := LookupCondact(op)
			require.False(t, found, "opcode %d should not resolve", op)
			require.Empty(t, def.Name)
		}
	})

	// The translated XMES resolves by opcode, because a reader of a database
	// meets it and has to know its arity, and does not resolve by name,
	// because a source writing XMES means the one that takes a text.
	t.Run("the translated XMES resolves one way only", func(t *testing.T) {
		def, found := LookupCondact(OpXMesFinal)
		require.True(t, found)
		require.Equal(t, "XMES", def.Name)
		require.Equal(t, 2, def.NumParams(), "an offset in two bytes")
		require.True(t, def.Emitted)

		op, written, known := LookupCondactByName("XMES")
		require.True(t, known)
		require.Equal(t, OpXMes, op, "a source that writes XMES means the one with the text")
		require.True(t, written.Fake)
	})

	t.Run("an opcode beyond the table does not resolve", func(t *testing.T) {
		_, found := LookupCondact(Opcode(len(condacts)))
		require.False(t, found)

		_, found = LookupCondact(255)
		require.False(t, found)
	})
}
