package main

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/build"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/stretchr/testify/require"
)

// A container belongs to a machine, and asking for the wrong one is refused
// with the right one named.
//
// The TAP is why this is here. It used to share a branch with the directory, so
// a CPC build asked for one got the Amstrad's single binary in a file called
// .tap — not a tape, and nothing said so until a 464 was asked to load it.
func TestAContainerIsRefusedForTheWrongMachine(t *testing.T) {
	for _, tc := range []struct {
		name, format string
		machine      arch.Machine
		says         string
	}{
		{"a TAP for a CPC", formatTAP, arch.CPC, "a TAP is a Spectrum tape"},
		{"a TZX for a CPC", formatTZX, arch.CPC, "a TZX is a Spectrum tape"},
		{"a CDT for a Spectrum", formatCDT, arch.ZX, "a CDT is an Amstrad tape"},
		{"a disk for a PC", formatDSK, arch.PC, "the Spectrum, the CPC and the PCW"},
		{"a tape for a PC", formatTAP, arch.PC, "a TAP is a Spectrum tape"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := assemble(tc.format, build.Build{Machine: tc.machine}, "AO", 0, 0, 64, nil, false)
			require.ErrorContains(t, err, tc.says)
		})
	}
}

// And a format nobody has heard of names the ones there are.
func TestAnUnknownFormatSaysWhatThereIs(t *testing.T) {
	_, err := assemble("floppy", build.Build{Machine: arch.CPC}, "AO", 0, 0, 64, nil, false)
	require.ErrorContains(t, err, "is not a format")
}
