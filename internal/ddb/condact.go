package ddb

// The condact catalogue, transcribed from the canonical table of the reference
// compiler (work/DRC/src/UCondacts.pas:25-172) with the DAAD v3 changes already
// applied (ApplyV3Changes, UCondacts.pas:277-290).
//
// We only ever target v3, so this is the one and only table.
//
// The number of parameters is NOT stored in the DDB: it is deduced from the
// opcode. A database is therefore impossible to walk without this table, and
// any mismatch shifts the reading of the rest of the block.

type Opcode uint8

const (
	// NUM_CONDACTS is the opcode space: bits 0 to 6, as bit 7 of the opcode
	// carries the indirection of the first parameter.
	NUM_CONDACTS Opcode = 128
	// NUM_FAKE_CONDACTS counts the pseudo-condacts. They are written by the
	// author but never reach the binary as such: the compiler translates them.
	NUM_FAKE_CONDACTS Opcode = 16

	MAX_CONDACT_PARAMS = 3
)

// ParamKind is the declared type of a condact operand. Taken from the
// TParamType enumeration (UCondacts.pas:8-13).
type ParamKind uint8

const (
	// ParamNone is an unused slot. PAUSE declares one parameter of this kind:
	// the byte is there, but no type constrains it.
	ParamNone             ParamKind = iota
	ParamLocation                   // locno: 0 to the number of locations
	ParamLocationOrMarker           // locno_: also accepts the 252-255 markers
	ParamObject                     // objno
	ParamFlag                       // flagno: 0-255, unchecked
	ParamSysMessage                 // sysno
	ParamMessage                    // mesno
	ParamProcess                    // procno
	ParamValue                      // value: a free byte
	ParamPercent                    // 1-99
	ParamVerb                       // vocabularyVerb
	ParamNoun                       // vocabularyNoun
	ParamPreposition                // vocabularyPrep
	ParamAdverb                     // vocabularyAdverb
	ParamAdjective                  // vocabularyAdjective
	ParamSkip                       // a signed entry delta, never a byte offset
	ParamString                     // a literal string, only on pseudo-condacts
	ParamWindow                     // 0-7
	ParamBit                        // 0-15: declared upstream but used by no condact
)

// CondactDef describes one condact. An empty Name marks a reserved opcode,
// which has no effect and must not be emitted.
type CondactDef struct {
	Name   string
	Params []ParamKind
	// Condition marks a condact that can fail and abandon the entry. It is the
	// CanBeJump of the reference table.
	Condition bool
	// Terminal marks a condact that ends the block by itself, so the emitter
	// can save the 0xFF terminator.
	Terminal bool
	// Fake marks a pseudo-condact: the author writes it, the compiler
	// translates it, and it never reaches the binary with this opcode.
	Fake bool
}

// NumParams is the arity of the condact.
func (cd CondactDef) NumParams() int {
	return len(cd.Params)
}

var (
	none      = []ParamKind{}
	locno     = []ParamKind{ParamLocation}
	objno     = []ParamKind{ParamObject}
	flagno    = []ParamKind{ParamFlag}
	valueP    = []ParamKind{ParamValue}
	value2    = []ParamKind{ParamValue, ParamValue}
	flagValue = []ParamKind{ParamFlag, ParamValue}
	flag2     = []ParamKind{ParamFlag, ParamFlag}
	obj2      = []ParamKind{ParamObject, ParamObject}
	objLocM   = []ParamKind{ParamObject, ParamLocationOrMarker}
	objLoc    = []ParamKind{ParamObject, ParamLocation}
	objFlag   = []ParamKind{ParamObject, ParamFlag}
	locnoM    = []ParamKind{ParamLocationOrMarker}
	stringP   = []ParamKind{ParamString}
)

// condacts is the table, indexed by opcode.
//
// The index of every row is written out on purpose: an opcode is then code and
// not a comment that could quietly start lying, and a repeated one does not
// compile ("duplicate index in array or slice literal"). Two opcodes are simply
// missing, and a lookup rejects them: 120, where the backend emits the
// translated XMES but which nobody writes, and 137, the deprecated XUNDONE.
var condacts = [NUM_CONDACTS + NUM_FAKE_CONDACTS]CondactDef{
	0:   {Name: "AT", Params: locno, Condition: true},
	1:   {Name: "NOTAT", Params: locno, Condition: true},
	2:   {Name: "ATGT", Params: locno, Condition: true},
	3:   {Name: "ATLT", Params: locno, Condition: true},
	4:   {Name: "PRESENT", Params: objno, Condition: true},
	5:   {Name: "ABSENT", Params: objno, Condition: true},
	6:   {Name: "WORN", Params: objno, Condition: true},
	7:   {Name: "NOTWORN", Params: objno, Condition: true},
	8:   {Name: "CARRIED", Params: objno, Condition: true},
	9:   {Name: "NOTCARR", Params: objno, Condition: true},
	10:  {Name: "CHANCE", Params: []ParamKind{ParamPercent}, Condition: true},
	11:  {Name: "ZERO", Params: flagno, Condition: true},
	12:  {Name: "NOTZERO", Params: flagno, Condition: true},
	13:  {Name: "EQ", Params: flagValue, Condition: true},
	14:  {Name: "GT", Params: flagValue, Condition: true},
	15:  {Name: "LT", Params: flagValue, Condition: true},
	16:  {Name: "ADJECT1", Params: []ParamKind{ParamAdjective}, Condition: true},
	17:  {Name: "ADVERB", Params: []ParamKind{ParamAdverb}, Condition: true},
	18:  {Name: "SFX", Params: value2},
	19:  {Name: "DESC", Params: locno},
	20:  {Name: "QUIT", Params: none},
	21:  {Name: "END", Params: none},
	22:  {Name: "DONE", Params: none, Terminal: true},
	23:  {Name: "OK", Params: none, Terminal: true},
	24:  {Name: "ANYKEY", Params: none},
	25:  {Name: "SAVE", Params: valueP},
	26:  {Name: "LOAD", Params: valueP},
	27:  {Name: "DPRINT", Params: flagno},
	28:  {Name: "DISPLAY", Params: valueP},
	29:  {Name: "CLS", Params: none},
	30:  {Name: "DROPALL", Params: none},
	31:  {Name: "AUTOG", Params: none},
	32:  {Name: "AUTOD", Params: none},
	33:  {Name: "AUTOW", Params: none},
	34:  {Name: "AUTOR", Params: none},
	35:  {Name: "PAUSE", Params: []ParamKind{ParamNone}}, // one untyped byte
	36:  {Name: "SYNONYM", Params: []ParamKind{ParamVerb, ParamNoun}},
	37:  {Name: "GOTO", Params: locno},
	38:  {Name: "MESSAGE", Params: []ParamKind{ParamMessage}},
	39:  {Name: "REMOVE", Params: objno},
	40:  {Name: "GET", Params: objno},
	41:  {Name: "DROP", Params: objno},
	42:  {Name: "WEAR", Params: objno},
	43:  {Name: "DESTROY", Params: objno},
	44:  {Name: "CREATE", Params: objno},
	45:  {Name: "SWAP", Params: obj2},
	46:  {Name: "PLACE", Params: objLocM},
	47:  {Name: "SET", Params: flagno},
	48:  {Name: "CLEAR", Params: flagno},
	49:  {Name: "PLUS", Params: flagValue},
	50:  {Name: "MINUS", Params: flagValue},
	51:  {Name: "LET", Params: flagValue},
	52:  {Name: "NEWLINE", Params: none},
	53:  {Name: "PRINT", Params: flagno},
	54:  {Name: "SYSMESS", Params: []ParamKind{ParamSysMessage}},
	55:  {Name: "ISAT", Params: objLocM, Condition: true},
	56:  {Name: "SETCO", Params: objno},
	57:  {Name: "SPACE", Params: none},
	58:  {Name: "HASAT", Params: valueP, Condition: true},
	59:  {Name: "HASNAT", Params: valueP, Condition: true},
	60:  {Name: "LISTOBJ", Params: none},
	61:  {Name: "EXTERN", Params: value2}, // see the irregular arities below
	62:  {Name: "RAMSAVE", Params: none},
	63:  {Name: "RAMLOAD", Params: flagno},
	64:  {Name: "BEEP", Params: value2},
	65:  {Name: "PAPER", Params: valueP},
	66:  {Name: "INK", Params: valueP},
	67:  {Name: "BORDER", Params: valueP},
	68:  {Name: "PREP", Params: []ParamKind{ParamPreposition}, Condition: true},
	69:  {Name: "NOUN2", Params: []ParamKind{ParamNoun}, Condition: true},
	70:  {Name: "ADJECT2", Params: []ParamKind{ParamAdjective}, Condition: true},
	71:  {Name: "ADD", Params: flag2},
	72:  {Name: "SUB", Params: flag2},
	73:  {Name: "PARSE", Params: valueP},
	74:  {Name: "LISTAT", Params: locnoM},
	75:  {Name: "PROCESS", Params: []ParamKind{ParamProcess}},
	76:  {Name: "SAME", Params: flag2, Condition: true},
	77:  {Name: "MES", Params: []ParamKind{ParamMessage}},
	78:  {Name: "WINDOW", Params: []ParamKind{ParamWindow}},
	79:  {Name: "NOTEQ", Params: flagValue, Condition: true},
	80:  {Name: "NOTSAME", Params: flag2, Condition: true},
	81:  {Name: "MODE", Params: valueP},
	82:  {Name: "WINAT", Params: value2},
	83:  {Name: "TIME", Params: value2},
	84:  {Name: "PICTURE", Params: valueP},
	85:  {Name: "DOALL", Params: locnoM},
	86:  {Name: "MOUSE", Params: value2},
	87:  {Name: "GFX", Params: value2},
	88:  {Name: "ISNOTAT", Params: objLocM, Condition: true},
	89:  {Name: "WEIGH", Params: objFlag},
	90:  {Name: "PUTIN", Params: objLoc},
	91:  {Name: "TAKEOUT", Params: objLoc},
	92:  {Name: "NEWTEXT", Params: none},
	93:  {Name: "ABILITY", Params: value2},
	94:  {Name: "WEIGHT", Params: flagno},
	95:  {Name: "RANDOM", Params: flagno},
	96:  {Name: "INPUT", Params: value2}, // see the irregular arities below
	97:  {Name: "SAVEAT", Params: none},
	98:  {Name: "BACKAT", Params: none},
	99:  {Name: "PRINTAT", Params: value2},
	100: {Name: "WHATO", Params: none},
	101: {Name: "CALL", Params: value2},
	102: {Name: "PUTO", Params: locnoM},
	103: {Name: "NOTDONE", Params: none, Terminal: true},
	104: {Name: "AUTOP", Params: locno},
	105: {Name: "AUTOT", Params: locno},
	106: {Name: "MOVE", Params: flagno},
	107: {Name: "WINSIZE", Params: value2},
	108: {Name: "REDO", Params: none, Terminal: true},
	109: {Name: "CENTRE", Params: none},
	110: {Name: "EXIT", Params: valueP},
	111: {Name: "INKEY", Params: none},
	112: {Name: "BIGGER", Params: flag2, Condition: true},
	113: {Name: "SMALLER", Params: flag2, Condition: true},
	114: {Name: "ISDONE", Params: none, Condition: true},
	115: {Name: "ISNDONE", Params: none, Condition: true},
	116: {Name: "SKIP", Params: []ParamKind{ParamSkip}, Terminal: true},
	117: {Name: "RESTART", Params: none, Terminal: true},
	118: {Name: "TAB", Params: valueP},
	119: {Name: "COPYOF", Params: objFlag},
	// 120: reserved. The backend emits the translated XMES here
	121: {Name: "COPYOO", Params: obj2},
	122: {Name: "INDIR", Params: valueP}, // generated by the compiler, never written by the author
	123: {Name: "COPYFO", Params: []ParamKind{ParamFlag, ParamObject}},
	124: {Name: "SETAT", Params: value2},
	125: {Name: "COPYFF", Params: flag2},
	126: {Name: "COPYBF", Params: flag2},
	127: {Name: "RESET", Params: none},

	// Pseudo-condacts. The author writes them and the compiler translates them
	// into real opcodes, EXTERN calls or nothing at all.
	128: {Name: "XMES", Params: stringP, Fake: true},
	129: {Name: "XMESSAGE", Params: stringP, Fake: true},
	130: {Name: "XPICTURE", Params: valueP, Fake: true},
	131: {Name: "XSAVE", Params: valueP, Fake: true},
	132: {Name: "XLOAD", Params: valueP, Fake: true},
	133: {Name: "XPART", Params: valueP, Fake: true},
	134: {Name: "XPLAY", Params: stringP, Fake: true},
	135: {Name: "XBEEP", Params: value2, Fake: true},
	136: {Name: "XSPLITSCR", Params: valueP, Fake: true},
	// 137: was XUNDONE, deprecated in v3 (drb.php:878)
	138: {Name: "XNEXTCLS", Params: none, Fake: true},
	139: {Name: "XNEXTRST", Params: none, Fake: true},
	140: {Name: "XSPEED", Params: valueP, Fake: true},
	141: {Name: "PENDINGSKIP", Params: valueP, Fake: true}, // compiler internal
	142: {Name: "XDATA", Params: stringP, Fake: true},
	143: {Name: "GETKEY", Params: none, Fake: true}, // emitted as PAUSE 0
}

// LookupCondact returns the definition of an opcode.
//
// Reserved opcodes report false: they have no effect and emitting one would be
// a bug.
func LookupCondact(op Opcode) (CondactDef, bool) {
	if op.Int() >= len(condacts) {
		return CondactDef{}, false
	}

	def := condacts[op]
	if def.Name == "" {
		return CondactDef{}, false
	}

	return def, true
}

// Int is a convenience for indexing and comparing.
func (op Opcode) Int() int {
	return int(op)
}

// Irregular arities, documented here because the table cannot express them and
// an emitter will have to special-case each one:
//
//   - EXTERN (61) with a second parameter of 3 consumes THREE parameter bytes,
//     "offsetLo, 3, offsetHi": it is the Maluva call that v2 uses for XMES.
//   - SFX (18) with a second parameter of 3 or 4 eats one extra byte from the
//     stream with the sample rate, written as a #defb in the source.
//   - INPUT (96) is declared with 21 parameters in the interpreters' tables
//     (PCDAAD/condacts.pas:255). It is a sentinel that triggers special
//     handling, not a real arity.
