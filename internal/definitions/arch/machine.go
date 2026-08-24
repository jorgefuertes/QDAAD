package arch

import (
	"fmt"
	"strings"
)

// Machine identifies the target machine. Its value is the DDB identifier.
type Machine uint8

const (
	PC    Machine = 0x00
	ZX    Machine = 0x01
	C64   Machine = 0x02
	CPC   Machine = 0x03
	MSX   Machine = 0x04
	ST    Machine = 0x05
	AMIGA Machine = 0x06
	PCW   Machine = 0x07
	ZX81  Machine = 0x08
	// 0x09 and 0x0A are unassigned in the format.
	CPM  Machine = 0x0B
	NEXT Machine = 0x0C
	HTML Machine = 0x0D
	CP4  Machine = 0x0E
	MSX2 Machine = 0x0F
	// ZX81 subtargets, the only ones that change their machine's base address.
	SubZX81_16K   = "16K"
	SubZX81_SD81B = "SD81B"
)

// MachineData holds the DDB traits that depend on the machine.
type MachineData struct {
	// name is the canonical target name, as accepted by drf and drb.
	name string

	// baseAddress is the load address in the machine's address space. DDB
	// pointers are absolute, not file offsets:
	//
	//	file_offset = ddb_pointer - baseAddress
	//
	// Non-zero bases belong to machines whose low memory is taken by the ROM,
	// the system area or the interpreter itself.
	//
	// Careful: on ZX81 this depends on the subtarget. Use the baseAddress
	// method instead of reading this field.
	baseAddress uint16

	// bigEndian reports that DDB words are stored high byte first. Only the
	// 68000 machines.
	bigEndian bool

	// wordAligned reports that sections and messages are padded to an even
	// address with a zero byte.
	wordAligned bool
}

var machines = map[Machine]MachineData{
	PC:    {name: "PC", baseAddress: 0x0000, wordAligned: true},
	ZX:    {name: "ZX", baseAddress: 0x8400},
	C64:   {name: "C64", baseAddress: 0x3880},
	CPC:   {name: "CPC", baseAddress: 0x2880},
	MSX:   {name: "MSX", baseAddress: 0x0100},
	ST:    {name: "ST", baseAddress: 0x0000, bigEndian: true, wordAligned: true},
	AMIGA: {name: "AMIGA", baseAddress: 0x0000, bigEndian: true, wordAligned: true},
	PCW:   {name: "PCW", baseAddress: 0x0100},
	// The real base depends on the subtarget; see the BaseAddress method.
	ZX81: {name: "ZX81", baseAddress: 0x0000},
	CPM:  {name: "CPM", baseAddress: 0x2000},
	// NEXTDAAD only exists on the origin/nextdaad branch of the DRC fork.
	NEXT: {name: "NEXTDAAD", baseAddress: 0x0000},
	HTML: {name: "HTML", baseAddress: 0x0000, wordAligned: true},
	CP4:  {name: "CP4", baseAddress: 0x7080},
	// MSX2 is absent from the non-zero base list in drb.php, so it falls
	// through to the default of 0. Only MSX uses 0x0100.
	MSX2: {name: "MSX2", baseAddress: 0x0000},
}

// ID returns the machine identifier stored in the high nibble of byte 0x01 of
// the DDB header.
func (m Machine) ID() byte { return byte(m) }

// Valid reports whether the value maps to an assigned machine. Codes 0x09 and
// 0x0A are not assigned.
func (m Machine) Valid() bool {
	_, ok := machines[m]

	return ok
}

// Data returns the machine traits. For an unassigned machine it returns the
// zero MachineData.
func (m Machine) Data() MachineData { return machines[m] }

// String returns the canonical target name.
func (m Machine) String() string {
	if d, ok := machines[m]; ok {
		return d.name
	}

	return fmt.Sprintf("Machine(0x%02X)", byte(m))
}

// BaseAddress returns the load address in the machine's address space.
//
// The subtarget only matters for ZX81, the sole machine whose base depends on
// it: SD81B loads at 0x8400 and 16K at 0x0000. Every other machine ignores it.
func (m Machine) BaseAddress(subtarget string) uint16 {
	if m == ZX81 && strings.EqualFold(subtarget, SubZX81_SD81B) {
		return 0x8400
	}

	return machines[m].baseAddress
}

// Parse turns a target name into a Machine, case-insensitively. It accepts
// "NEXT" as an alias for "NEXTDAAD".
func Parse(s string) (Machine, error) {
	if strings.EqualFold(s, "NEXT") {
		return NEXT, nil
	}

	for m, d := range machines {
		if strings.EqualFold(s, d.name) {
			return m, nil
		}
	}

	return 0, fmt.Errorf("unknown target: %q", s)
}

// FromHeaderByte extracts the machine from byte 0x01 of a DDB header, which
// encodes the identifier in the high nibble and the language in bit 0.
func FromHeaderByte(b byte) (Machine, error) {
	m := Machine(b >> 4)
	if !m.Valid() {
		return 0, fmt.Errorf("unassigned machine identifier: 0x%02X", byte(m))
	}

	return m, nil
}

func (m Machine) Name() string {
	return machines[m].name
}

func (m Machine) IsBigEndian() bool {
	return machines[m].bigEndian
}

func (m Machine) IsWordAligned() bool {
	return machines[m].wordAligned
}
