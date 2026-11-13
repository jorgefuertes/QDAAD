package arch

import (
	"fmt"
	"slices"
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

	// textRows and textCols are how much text the screen holds, which is what
	// a game loop has to tell the interpreter when it sets up its window.
	//
	// They are the screen in pixels divided by the font, and the font is not a
	// guess: DAAD's charsets are eight rows tall and six pixels wide, and the
	// six can be measured rather than read anywhere. The letters of every
	// charset in the corpus — ours, DAAD's own for the Amiga, La Aventura
	// Original's, El Jabato's for the PC — use bits 7 down to 2 and never the
	// two lowest.
	//
	// Two machines confirm the arithmetic from opposite directions. A Spectrum
	// is 256 by 192, which gives 42 by 24, and 42 by 24 is exactly what its
	// interpreter clamps a window to at 0x76F7. A PC is 320 by 200, which
	// gives 53 by 25, and 53 by 25 is what the research notes record for it.
	//
	// Zero means we have not established them for that machine, and a caller
	// then has to fall back on asking for more than the screen holds and
	// trusting the interpreter to clamp. That is what every DAAD source does —
	// "WINSIZE 25 127", with a comment saying maximum size — and it works
	// because the interpreters do clamp. It is still a bet, and a declared
	// height of 25 on a 24-row screen is the kind of bet that comes back as
	// text wrapping to the top of the window instead of scrolling.
	textRows, textCols uint8

	// bigEndian reports that DDB words are stored high byte first. Only the
	// 68000 machines.
	bigEndian bool

	// wordAligned reports that sections and messages are padded to an even
	// address with a zero byte.
	wordAligned bool

	// bits is the width of the machine, 8 or 16, and zero where the question
	// does not apply. A source asks about it through the BIT8 and BIT16
	// symbols, and picks a different branch for each.
	//
	// HTML is the zero: it runs wherever the browser runs, so the compiler of
	// reference gives it neither symbol, and a source that only handles the two
	// it knows will fail to compile for it. That is deliberate there and it is
	// kept here.
	bits int
}

var machines = map[Machine]MachineData{
	PC:    {name: "PC", baseAddress: 0x0000, wordAligned: true, bits: 16, textRows: 25, textCols: 53},
	ZX:    {name: "ZX", baseAddress: 0x8400, bits: 8, textRows: 24, textCols: 42},
	C64:   {name: "C64", baseAddress: 0x3880, bits: 8},
	CPC:   {name: "CPC", baseAddress: 0x2880, bits: 8},
	MSX:   {name: "MSX", baseAddress: 0x0100, bits: 8},
	ST:    {name: "ST", baseAddress: 0x0000, bigEndian: true, wordAligned: true, bits: 16},
	AMIGA: {name: "AMIGA", baseAddress: 0x0000, bigEndian: true, wordAligned: true, bits: 16},
	PCW:   {name: "PCW", baseAddress: 0x0100, bits: 8},
	// The real base depends on the subtarget; see the BaseAddress method.
	ZX81: {name: "ZX81", baseAddress: 0x0000, bits: 8},
	CPM:  {name: "CPM", baseAddress: 0x2000, bits: 8},
	// NEXTDAAD only exists on the origin/nextdaad branch of the DRC fork, and
	// the mainline compiler has no target of this name — a Next is reached
	// there as the ZX target with the NEXT subtarget. The width is the Z80N's.
	NEXT: {name: "NEXTDAAD", baseAddress: 0x0000, bits: 8},
	HTML: {name: "HTML", baseAddress: 0x0000, wordAligned: true},
	CP4:  {name: "CP4", baseAddress: 0x7080, bits: 8},
	// MSX2 is absent from the non-zero base list in drb.php, so it falls
	// through to the default of 0. Only MSX uses 0x0100.
	MSX2: {name: "MSX2", baseAddress: 0x0000, bits: 8},
}

func MachineNames() []string {
	names := make([]string, 0, len(machines))
	for _, d := range machines {
		names = append(names, d.name)
	}

	return names
}

func MachineHumanList() string {
	names := MachineNames()
	slices.Sort(names)

	for i, n := range names {
		names[i] = strings.ToLower(n)
	}

	if len(names) == 0 {
		return ""
	}

	if len(names) == 1 {
		return names[0]
	}

	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
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
// TextSize is how much text the screen holds, and whether we know.
//
// A game loop has to tell the interpreter how big its text window is, and the
// idiom every DAAD source uses is to ask for more than the screen holds and let
// the interpreter clamp — "WINSIZE 25 127", with a comment saying maximum size.
// The interpreters do clamp, so it works, and it is still worth not doing: a
// window declared 25 rows tall on a 24-row Spectrum puts its first line on a
// row that is not there, and the lines after it wrap round to the top. What
// that looks like is an adventure whose descriptions start in the middle of a
// sentence.
func (m Machine) TextSize() (rows, cols int, known bool) {
	data, found := machines[m]
	if !found || data.textRows == 0 || data.textCols == 0 {
		return 0, 0, false
	}

	return int(data.textRows), int(data.textCols), true
}

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

// Bits is the width of the machine, 8 or 16, and zero for the one where the
// question does not apply. See the bits field.
func (m Machine) Bits() int {
	return machines[m].bits
}
