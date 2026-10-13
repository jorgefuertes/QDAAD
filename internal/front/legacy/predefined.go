package legacy

import (
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// predefined seeds the symbol table before a line of source is read.
//
// The compiler of reference does the same, in drf.pas:231-292, and it matters
// that these arrive as ordinary symbols rather than as names some section knows
// about: a symbol is worth a value **anywhere a value is expected** — a condact
// parameter, an arithmetic expression between quotes, the value of another
// #define. Treating them as a special case of /OBJ would make "PLACE 5 CARRIED"
// fail, and it is good code.
//
// It also means #ifdef "CARRIED" is true, which is what happens there too, and
// it is why the target has to be known before the source is read: half of what
// follows depends on it, and a source branches on it.
//
// Two families are missing here because they cannot be known yet. The counts
// the source itself produces — NUM_OBJECTS and its four companions — and the
// _VOC_ of every vocabulary word are added by the parser as it reaches them.
// See docs/DAAD_v2_v3_Research/12-formato-dsf.md §8 for the whole list.
func predefined(opts Options) Symbols {
	s := Symbols{
		// Where an object can be that is not a location.
		"NOT_CREATED": int(ddb.LOC_NOT_CREATED),
		"NON_CREATED": int(ddb.LOC_NOT_CREATED), // the spelling drf also accepts
		"WORN":        int(ddb.LOC_WORN),
		"CARRIED":     int(ddb.LOC_CARRIED),
		"HERE":        int(ddb.LOC_HERE),

		// What the SFX condact's first operand can be.
		"PLAYSFX":   1,
		"PLAYSFXL":  2,
		"PLAYSFXF":  3,
		"PLAYSFXFL": 4,
		"STOPSFX":   5,
		"PLAYDRO":   6,
		"PLAYDROL":  7,
		"STOPDRO":   8,
		"PLAYFLI":   9,
		"PLAYFLIL":  10,

		// What the MOUSE condact's first operand can be.
		"RESETMS":   0,
		"SHOWMS":    1,
		"HIDEMS":    2,
		"GETMS":     3,
		"GETFINEMS": 4,
		"POINTERMS": 5,
		"DELTAXMS":  6,
		"DELTAYMS":  7,
	}

	s.addTarget(opts)
	s.addScreen(opts)
	s.addDate(opts)
	s.addExtra(opts)

	if opts.Version >= 3 {
		s["V3"] = 1
	} else {
		s["V2"] = 1
	}

	return s
}

// addTarget names what is being compiled for.
//
// The subtarget arrives twice, once bare and once with MODE_ in front, and both
// spellings appear in real sources. The machine's width arrives as a symbol
// that exists or does not, never as a number, so a source tests it with #ifdef.
func (s Symbols) addTarget(opts Options) {
	s[opts.Machine.Name()] = 1

	if opts.Subtarget != "" {
		sub := strings.ToUpper(opts.Subtarget)
		s[sub] = 1
		s["MODE_"+sub] = 1
	}

	switch opts.Machine.Bits() {
	case 8:
		s["BIT8"] = 1
	case 16:
		s["BIT16"] = 1
	}
}

func (s Symbols) addScreen(opts Options) {
	if n := cols(opts.Machine, opts.Subtarget); n != 0 {
		s["COLS"] = n
	}

	s["ROWS"] = rows(opts.Machine)
}

// addDate splits the day into the four symbols a source can read it through.
// The year comes in two halves because neither fits in a byte on its own.
func (s Symbols) addDate(opts Options) {
	when := opts.when()

	s["YEARHIGH"] = when.Year() / 100
	s["YEARLOW"] = when.Year() % 100
	s["MONTH"] = int(when.Month())
	s["DAY"] = when.Day()
}

// addExtra takes the symbols given on the command line.
//
// Their value is their position, counting from one, which is worth saying out
// loud because it looks like a bug and is not: it is what drf.pas:288-296 does,
// so -DFOO,BAR leaves FOO worth 1 and BAR worth 2. A source that only tests
// them with #ifdef never notices; one that uses them as a number does.
func (s Symbols) addExtra(opts Options) {
	for i, name := range opts.Extra {
		if name == "" {
			continue
		}

		s[name] = i + 1
	}
}
