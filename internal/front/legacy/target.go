package legacy

import (
	"fmt"
	"strings"
	"time"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
)

// Options is what a compilation needs to know beyond the source itself.
//
// None of it is optional in the sense of "nice to have". The machine and the
// subtarget reach the source as symbols, and a source reads them: it picks a
// branch on BIT8, it works a width out of COLS, it asks whether it is being
// built for a Next. Compiling the same file for two machines is meant to give
// two different databases, and that is the whole point of the mechanism.
type Options struct {
	// Machine is the target. Its name is the one the compiler of reference
	// accepts, and it arrives as a symbol worth 1.
	Machine arch.Machine

	// Subtarget narrows the machine down, where it has variants: a Spectrum is
	// 48K or PLUS3 or NEXT, a PC is VGA256 or TEXT. Empty where the machine has
	// none. It arrives as two symbols, its own name and MODE_ before it.
	Subtarget string

	// Language is the one the interpreter speaks.
	Language language.Language

	// Version is the format to emit, 2 or 3. It arrives as the V2 or V3 symbol.
	Version uint8

	// Date is what the YEARHIGH, YEARLOW, MONTH and DAY symbols are worth.
	//
	// The zero value means now, which is what the compiler of reference always
	// does — and which makes a build unrepeatable, because the same source on
	// two days gives two different databases. Setting it is how a caller gets
	// that repeatability back, and it is what lets two compilers be compared
	// without the clock separating them.
	Date time.Time

	// Extra are the symbols given on the command line. Note the value: the
	// compiler of reference numbers them by their position, starting at one,
	// rather than giving them all the 1 that every other flag symbol gets.
	Extra []string
}

// Validate checks the options against each other before anything is read.
func (o Options) Validate() error {
	if !o.Machine.Valid() {
		return fmt.Errorf("0x%02X is not a machine", o.Machine.ID())
	}

	if o.Version != 2 && o.Version != 3 {
		return fmt.Errorf("version %d does not exist, it is 2 or 3", o.Version)
	}

	return validSubtarget(o.Machine, o.Subtarget)
}

// when is the date the symbols are worth, resolving the zero value to now.
func (o Options) when() time.Time {
	if o.Date.IsZero() {
		return time.Now()
	}

	return o.Date
}

// subtargets lists the variants each machine accepts. A machine that is absent
// takes none, and one that is present takes nothing else.
//
// The names are the compiler of reference's, upper case and all, because they
// reach the source as symbols and a source spells them the way it spells them.
var subtargets = map[arch.Machine][]string{
	arch.ZX:   {"48K", "128K", "PLUS3", "ESXDOS", "NEXT", "UNO"},
	arch.ZX81: {arch.SubZX81_16K, arch.SubZX81_SD81B},
	arch.PC:   {"TEXT", "CGA", "EGA", "VGA", "VGA256"},
	arch.MSX2: {
		"5_6", "5_8", "6_6", "6_8", "7_6", "7_8",
		"8_6", "8_8", "10_6", "10_8", "12_6", "12_8",
	},
}

func validSubtarget(m arch.Machine, sub string) error {
	allowed := subtargets[m]

	if sub == "" {
		// A machine with variants still compiles without naming one: the
		// compiler of reference takes the subtarget as an optional argument and
		// falls back to the conservative width. What it will not do is accept a
		// name it does not know.
		return nil
	}

	for _, name := range allowed {
		if strings.EqualFold(sub, name) {
			return nil
		}
	}

	if len(allowed) == 0 {
		return fmt.Errorf("%s has no subtargets, so %q is not one of them", m, sub)
	}

	return fmt.Errorf("%q is not a subtarget of %s, which takes %s",
		sub, m, strings.Join(allowed, ", "))
}

// cols is the width of the text screen in characters, and rows its height.
//
// Both are taken from drf.pas:60-137 rather than worked out, because there is
// nothing to work out: they are a list of machines and numbers, with a
// conservative guess where the machine is not on the list. A source uses them
// to lay text out, so being wrong here moves words on the screen.
func cols(m arch.Machine, sub string) int {
	switch m {
	case arch.PC:
		if strings.EqualFold(sub, "TEXT") {
			return 80
		}

		return 53

	case arch.MSX2:
		return msx2Cols(sub)

	case arch.ZX81:
		if strings.EqualFold(sub, arch.SubZX81_16K) {
			return 32
		}

		return 42

	case arch.C64, arch.CP4, arch.CPC:
		return 40

	case arch.CPM:
		return 80

	case arch.HTML, arch.ST, arch.AMIGA:
		return 53

	case arch.PCW:
		return 90
	}

	// ZX, MSX and anything unlisted.
	return 42
}

// The MSX2 subtargets are a screen mode and a character width joined by an
// underscore, and the width of the screen follows from the pair.
func msx2Cols(sub string) int {
	widths := map[string]int{
		"5_6": 42, "5_8": 32,
		"6_6": 85, "6_8": 64,
		"7_6": 85, "7_8": 64,
		"8_6": 42, "8_8": 32,
		"10_6": 42, "10_8": 32,
		"12_6": 42, "12_8": 32,
	}

	if n, known := widths[strings.ToUpper(sub)]; known {
		return n
	}

	return 42
}

func rows(m arch.Machine) int {
	switch m {
	case arch.PCW:
		return 32
	case arch.MSX2:
		return 26
	case arch.ZX81, arch.ZX, arch.MSX:
		return 24
	}

	return 25
}
