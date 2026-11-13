package legacy

import (
	"testing"
	"time"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/stretchr/testify/require"
)

// The symbols a machine brings with it are what a source branches on, so every
// one of these numbers moves text on a real screen or picks a different block
// of code. They are checked against drf.pas:60-137 and not against each other.
func TestTargetSymbols(t *testing.T) {
	for _, tc := range []struct {
		name      string
		machine   arch.Machine
		subtarget string
		cols      int
		rows      int
		width     string
	}{
		{name: "a 128K Spectrum", machine: arch.ZX, subtarget: "128K", cols: 42, rows: 24, width: "BIT8"},
		{name: "a Next", machine: arch.ZX, subtarget: "NEXT", cols: 42, rows: 24, width: "BIT8"},
		{name: "an Amstrad PCW", machine: arch.PCW, cols: 90, rows: 32, width: "BIT8"},
		{name: "a CPC", machine: arch.CPC, cols: 40, rows: 25, width: "BIT8"},
		{name: "a Commodore 64", machine: arch.C64, cols: 40, rows: 25, width: "BIT8"},
		{name: "a Plus/4", machine: arch.CP4, cols: 40, rows: 25, width: "BIT8"},
		{name: "an MSX", machine: arch.MSX, cols: 42, rows: 24, width: "BIT8"},
		{name: "an MSX2 in mode 6", machine: arch.MSX2, subtarget: "6_6", cols: 85, rows: 26, width: "BIT8"},
		{name: "an MSX2 in mode 8", machine: arch.MSX2, subtarget: "8_8", cols: 32, rows: 26, width: "BIT8"},
		{name: "a generic CP/M", machine: arch.CPM, cols: 80, rows: 25, width: "BIT8"},
		{name: "a 16K ZX81", machine: arch.ZX81, subtarget: arch.SubZX81_16K, cols: 32, rows: 24, width: "BIT8"},
		{name: "an Atari ST", machine: arch.ST, cols: 53, rows: 25, width: "BIT16"},
		{name: "an Amiga", machine: arch.AMIGA, cols: 53, rows: 25, width: "BIT16"},
		{name: "a PC in text mode", machine: arch.PC, subtarget: "TEXT", cols: 80, rows: 25, width: "BIT16"},
		{name: "a PC in VGA256", machine: arch.PC, subtarget: "VGA256", cols: 53, rows: 25, width: "BIT16"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := predefined(Options{
				Machine: tc.machine, Subtarget: tc.subtarget, Version: 3,
			})

			require.Equal(t, tc.cols, s["COLS"])
			require.Equal(t, tc.rows, s["ROWS"])

			require.Contains(t, s, tc.width)
			require.Equal(t, 1, s[tc.machine.Name()])

			// One width and never both: a source picks a branch on these.
			other := map[string]string{"BIT8": "BIT16", "BIT16": "BIT8"}[tc.width]
			require.NotContains(t, s, other)

			if tc.subtarget != "" {
				require.Equal(t, 1, s[tc.subtarget], "the bare subtarget name")
				require.Equal(t, 1, s["MODE_"+tc.subtarget], "the MODE_ spelling")
			}
		})
	}
}

// HTML is the machine that is neither, and a source that only handles the two
// it knows is meant to fail to compile for it rather than pick a branch by
// accident. See drf.pas:242.
func TestHTMLHasNeitherWidth(t *testing.T) {
	s := predefined(Options{Machine: arch.HTML, Version: 3})

	require.NotContains(t, s, "BIT8")
	require.NotContains(t, s, "BIT16")
	require.Equal(t, 53, s["COLS"])
}

func TestVersionSymbol(t *testing.T) {
	v3 := predefined(Options{Machine: arch.ZX, Version: 3})
	require.Equal(t, 1, v3["V3"])
	require.NotContains(t, v3, "V2")

	v2 := predefined(Options{Machine: arch.ZX, Version: 2})
	require.Equal(t, 1, v2["V2"])
	require.NotContains(t, v2, "V3")
}

func TestDateSymbols(t *testing.T) {
	s := predefined(Options{
		Machine: arch.ZX, Version: 3,
		Date: time.Date(2026, time.September, 26, 13, 0, 0, 0, time.UTC),
	})

	// The year is split because neither half fits in a byte whole.
	require.Equal(t, 20, s["YEARHIGH"])
	require.Equal(t, 26, s["YEARLOW"])
	require.Equal(t, 9, s["MONTH"])
	require.Equal(t, 26, s["DAY"])
}

// The command line symbols are worth their position rather than 1, which looks
// like a bug in the compiler of reference and is the behaviour a source may be
// relying on. See drf.pas:288-296.
func TestExtraSymbolsAreNumberedByPosition(t *testing.T) {
	s := predefined(Options{
		Machine: arch.ZX, Version: 3,
		Extra: []string{"DEMO", "NOSOUND", "TESTING"},
	})

	require.Equal(t, 1, s["DEMO"])
	require.Equal(t, 2, s["NOSOUND"])
	require.Equal(t, 3, s["TESTING"])
}

func TestConstantSymbols(t *testing.T) {
	s := predefined(Options{Machine: arch.ZX, Version: 3})

	// The two spellings of the same place, which drf accepts both of.
	require.Equal(t, s["NOT_CREATED"], s["NON_CREATED"])

	require.Equal(t, 1, s["PLAYSFX"])
	require.Equal(t, 10, s["PLAYFLIL"])
	require.Equal(t, 0, s["RESETMS"])
	require.Equal(t, 7, s["DELTAYMS"])

	// RESETMS is worth zero, so being there and being zero are different
	// questions and a lookup that only returns the value cannot tell them apart.
	require.Contains(t, s, "RESETMS")
}

func TestSubtargetValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opts    Options
		wantErr string
	}{
		{
			name: "a machine with no variants takes none",
			opts: Options{Machine: arch.CPC, Subtarget: "TURBO", Version: 3},
			// The CPC has none at all, which is a different complaint.
			wantErr: "has no subtargets",
		},
		{
			name:    "a variant the machine does not have",
			opts:    Options{Machine: arch.ZX, Subtarget: "TIMEX", Version: 3},
			wantErr: "is not a subtarget of ZX",
		},
		{
			name: "no variant at all is allowed",
			opts: Options{Machine: arch.ZX, Version: 3},
		},
		{
			name: "the case is not the source's problem",
			opts: Options{Machine: arch.ZX, Subtarget: "plus3", Version: 3},
		},
		{
			name:    "a version that does not exist",
			opts:    Options{Machine: arch.ZX, Version: 4},
			wantErr: "version 4 does not exist",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()

			if tc.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// A subtarget written in lower case still reaches the source upper case, which
// is how it is spelled where it is used.
func TestSubtargetSymbolIsUpperCase(t *testing.T) {
	s := predefined(Options{Machine: arch.ZX, Subtarget: "plus3", Version: 3})

	require.Equal(t, 1, s["PLUS3"])
	require.Equal(t, 1, s["MODE_PLUS3"])
	require.NotContains(t, s, "plus3")
}
