package legacy

// DirectiveID names one of the compiler directives.
//
// They split in two, and getting the split wrong is expensive. The first group
// is the preprocessor's: it resolves them and they never reach the parser. The
// second group only looks like preprocessing — those directives emit bytes into
// the condact block being read, so they are part of the grammar and have to
// arrive intact.
type DirectiveID uint8

const (
	// Resolved and consumed by the preprocessor.
	DirDefine DirectiveID = iota
	DirIfdef
	DirIfndef
	DirEndif
	DirElse
	DirEcho
	DirExtern
	DirInt
	DirSfx
	DirDebug
	DirClassic

	// Passed through to the parser.
	DirDB
	DirDW
	DirHex
	DirIncbin
	DirUserptr
)

// The names, with the synonyms the compiler of reference accepts: #if is #ifdef
// and #defb/#defw are #db/#dw, so they collapse onto the same value.
var directiveNames = map[string]DirectiveID{
	"#define":  DirDefine,
	"#ifdef":   DirIfdef,
	"#if":      DirIfdef,
	"#ifndef":  DirIfndef,
	"#endif":   DirEndif,
	"#else":    DirElse,
	"#echo":    DirEcho,
	"#extern":  DirExtern,
	"#int":     DirInt,
	"#sfx":     DirSfx,
	"#debug":   DirDebug,
	"#classic": DirClassic,
	"#db":      DirDB,
	"#defb":    DirDB,
	"#dw":      DirDW,
	"#defw":    DirDW,
	"#hex":     DirHex,
	"#incbin":  DirIncbin,
	"#userptr": DirUserptr,
}

// forParser says whether a directive belongs to the grammar rather than to the
// preprocessor.
func (d DirectiveID) forParser() bool {
	return d >= DirDB
}

// The name to report a directive by. It has to be spelled out rather than
// searched for in the table above, which holds the synonyms too and would name
// #ifdef as #if half the time.
var directiveCanonical = [...]string{
	DirDefine: "#define", DirIfdef: "#ifdef", DirIfndef: "#ifndef",
	DirEndif: "#endif", DirElse: "#else", DirEcho: "#echo",
	DirExtern: "#extern", DirInt: "#int", DirSfx: "#sfx",
	DirDebug: "#debug", DirClassic: "#classic",
	DirDB: "#db", DirDW: "#dw", DirHex: "#hex",
	DirIncbin: "#incbin", DirUserptr: "#userptr",
}

func (d DirectiveID) String() string {
	if int(d) < len(directiveCanonical) {
		return directiveCanonical[d]
	}

	return "#?"
}

func lookupDirective(text string) (DirectiveID, bool) {
	id, found := directiveNames[text]

	return id, found
}
