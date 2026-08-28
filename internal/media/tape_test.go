package media_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	spectrumTape = "../../work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original - Part 1.tzx"
	amstradTape  = "../../work/aventuras/La_Aventura_Original/AmstradCPC/La aventura original (Cara A).cdt"
)

// The Amstrad world calls the format CDT and the Spectrum world TZX, but the
// signature is the same and so is everything after it.
func TestOpenTapes(t *testing.T) {
	for _, path := range []string{spectrumTape, amstradTape} {
		require.Equal(t, "TZX tape, version 1.10", openDSK(t, path).Format())
	}
}

func TestTapeHasNoFiles(t *testing.T) {
	files, err := openDSK(t, spectrumTape).Files()
	require.Error(t, err)
	require.ErrorContains(t, err, "no filesystem")
	require.Nil(t, files)
}

// TestTapePayload checks the two things the walk can get wrong: stepping over a
// block by the wrong number of bytes, which turns everything after it into
// nonsense, and keeping blocks that hold pulses rather than bytes.
//
// The two tapes exercise both kinds of data block. The Spectrum one is all
// standard speed blocks, written by the ROM, so each has a flag and a checksum
// around it that have to come off; the Amstrad one uses the turbo blocks of its
// own loader, which are left as they are.
func TestTapePayload(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		size    int
		at      int
		machine byte
	}{
		{name: "spectrum", path: spectrumTape, size: 47703, at: 0x3ED7, machine: 1},
		{name: "amstrad", path: amstradTape, size: 57706, at: 0x6561, machine: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := openDSK(t, tc.path).Payload()
			require.Len(t, payload, tc.size)

			ddb := payload[tc.at:]

			require.Equal(t, byte(1), ddb[0], "DAAD version")
			require.Equal(t, tc.machine, ddb[1]>>4, "machine")
			require.Equal(t, byte(95), ddb[2], "the null word character")

			require.Equal(t, byte(9), ddb[3], "objects")
			require.Equal(t, byte(49), ddb[4], "locations")
			require.Equal(t, byte(140), ddb[5], "user messages")
			require.Equal(t, byte(63), ddb[6], "system messages")
			require.Equal(t, byte(17), ddb[7], "processes")

			word := make([]byte, 5)
			for i := range word {
				word[i] = ddb[32+i] ^ 0xFF
			}

			require.Equal(t, "\x15RBOL", string(word), "the first word is ÁRBOL, with its accent")
		})
	}
}
