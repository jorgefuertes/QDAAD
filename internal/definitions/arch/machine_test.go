package arch

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The identifiers documented in drb.php:1264-1280.
var all = []struct {
	m    Machine
	id   byte
	name string
}{
	{PC, 0x00, "PC"},
	{ZX, 0x01, "ZX"},
	{C64, 0x02, "C64"},
	{CPC, 0x03, "CPC"},
	{MSX, 0x04, "MSX"},
	{ST, 0x05, "ST"},
	{AMIGA, 0x06, "AMIGA"},
	{PCW, 0x07, "PCW"},
	{ZX81, 0x08, "ZX81"},
	{CPM, 0x0B, "CPM"},
	{NEXT, 0x0C, "NEXTDAAD"},
	{HTML, 0x0D, "HTML"},
	{CP4, 0x0E, "CP4"},
	{MSX2, 0x0F, "MSX2"},
}

func TestID(t *testing.T) {
	require.Len(t, machines, len(all), "table and test must cover the same machines")

	for _, c := range all {
		require.Equal(t, c.id, c.m.ID(), "identifier of %s", c.name)
		require.Equal(t, c.name, c.m.String(), "name of 0x%02X", c.id)
		require.True(t, c.m.Valid(), "%s should be valid", c.name)
	}
}

func TestFromHeaderByte(t *testing.T) {
	for _, c := range all {
		// The low nibble carries the language and must not interfere.
		for _, lang := range []byte{0x00, 0x01} {
			m, err := FromHeaderByte(c.id<<4 | lang)
			require.NoError(t, err)
			require.Equal(t, c.m, m)
		}
	}

	// 0x09 and 0x0A are unassigned in the format.
	for _, unassigned := range []byte{0x90, 0xA0} {
		_, err := FromHeaderByte(unassigned)
		require.Error(t, err, "0x%02X should not resolve to a machine", unassigned)
	}
}

func TestBaseAddress(t *testing.T) {
	cases := []struct {
		m         Machine
		subtarget string
		want      uint16
	}{
		{ZX, "128K", 0x8400},
		{CPC, "", 0x2880},
		{MSX, "", 0x0100},
		{PCW, "", 0x0100},
		{CPM, "", 0x2000},
		{CP4, "", 0x7080},
		{C64, "", 0x3880},
		{PC, "VGA256", 0x0000},
		{ST, "", 0x0000},
		{AMIGA, "", 0x0000},
		{HTML, "", 0x0000},
		{NEXT, "", 0x0000},

		// MSX2 falls through to the default in drb.php: base 0, not 0x0100.
		// Verified against a real DDB: word(0x20) = 0x1B76 = 7030 = the file
		// size, which only happens with a base of 0.
		{MSX2, "8_6", 0x0000},

		// ZX81 is the only machine whose base depends on the subtarget.
		{ZX81, SubZX81_16K, 0x0000},
		{ZX81, SubZX81_SD81B, 0x8400},
		{ZX81, "sd81b", 0x8400}, // case-insensitive
		{ZX81, "", 0x0000},
	}

	for _, c := range cases {
		require.Equal(t, c.want, c.m.BaseAddress(c.subtarget),
			"base address of %s with subtarget %q", c.m, c.subtarget)
	}
}

func TestEndiannessAndAlignment(t *testing.T) {
	bigEndian := map[Machine]bool{ST: true, AMIGA: true}
	aligned := map[Machine]bool{PC: true, ST: true, AMIGA: true, HTML: true}

	for m := range machines {
		require.Equal(t, bigEndian[m], m.Data().bigEndian, "endianness of %s", m)
		require.Equal(t, aligned[m], m.Data().wordAligned, "alignment of %s", m)
	}
}

func TestParse(t *testing.T) {
	for _, c := range all {
		m, err := Parse(c.name)
		require.NoError(t, err)
		require.Equal(t, c.m, m)

		// Parse is case-insensitive.
		m, err = Parse(strings.ToLower(c.name))
		require.NoError(t, err)
		require.Equal(t, c.m, m)
	}

	// "NEXT" is an alias for "NEXTDAAD".
	m, err := Parse("next")
	require.NoError(t, err)
	require.Equal(t, NEXT, m)

	_, err = Parse("SPECTRUM")
	require.Error(t, err)
}
