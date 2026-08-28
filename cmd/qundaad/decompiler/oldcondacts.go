package decompiler

import "github.com/jorgefuertes/QDAAD/internal/ddb"

// Four opcodes meant something else before version 2 reassigned them. In the
// old format 56 to 59 hold the COPY family, each taking two parameters; the
// version 2 table this project compiles for puts SETCO, SPACE, HASAT and
// HASNAT there, with one parameter or none.
//
// The arity is the part that matters, and it is proven: with one parameter the
// walk over La Aventura Original swallows whole blocks and yields impossible
// operands such as "LT 255 49" in a game of 49 locations. With two, the 253
// blocks of part one and the 357 of part two decode without a single
// contradiction between overlapping blocks.
//
// The names come from the undaad and unDRC decompilers, which are secondary
// sources: the 1991 manual documents version 2, where these opcodes already
// mean something else, so there is no primary source naming them. A complete
// and consecutive COPY family is too neat to be a coincidence, but it is worth
// remembering where it comes from.
var oldCondacts = map[byte]ddb.CondactDef{
	56: {Name: "COPYOF", Params: []ddb.ParamKind{ddb.ParamObject, ddb.ParamFlag}},
	57: {Name: "COPYOO", Params: []ddb.ParamKind{ddb.ParamObject, ddb.ParamObject}},
	58: {Name: "COPYFO", Params: []ddb.ParamKind{ddb.ParamFlag, ddb.ParamObject}},
	59: {Name: "COPYFF", Params: []ddb.ParamKind{ddb.ParamFlag, ddb.ParamFlag}},
}

// TODO: settle the arity of CALL and MOUSE in old databases.
//
// The 1991 manual writes them with a single operand, "CALL address" and
// "MOUSE option", and both existing decompilers read one byte. The table this
// project compiles for gives them two, and for CALL that is the reading that
// makes sense: an address is 16 bits and a DAAD parameter is a byte, so it
// takes two of them. Read as one, the only CALL in La Aventura Original comes
// out as "CALL 3", a call to address 3.
//
// The overlap check cannot decide it. Each appears exactly once per database,
// in blocks nothing else shares, so neither reading contradicts anything.
//
// Two parameters are kept for now, which is what CALL needs. Settling it takes
// evidence this adventure cannot give: an interpreter of the period, or a game
// that uses either condact more than once. Other platforms and other
// adventures should clear it up.
var uncertainArity = map[byte]string{
	101: "CALL: the manual writes one operand, this reads two. See oldcondacts.go",
	86:  "MOUSE: the manual writes one operand, this reads two. See oldcondacts.go",
}

// OldFormat reports whether this is a pre-version-2 database.
//
// Nothing in the file says so outright: the version byte is unreliable across
// the intermediate builds of the era. What does say so is the shape of the
// header, since version 2 inserted the pointer to the extra object attributes.
// The undaad decompiler decides the same way, by checking whether the word at
// offset 30 is the length of the file.
func (r *Reader) OldFormat() bool {
	return r.headerSize == headerSizeV1
}

// Condact resolves an opcode against the table this database belongs to.
func (r *Reader) Condact(opcode byte) (ddb.CondactDef, bool) {
	if r.OldFormat() {
		if def, found := oldCondacts[opcode]; found {
			return def, true
		}
	}

	return ddb.LookupCondact(ddb.Opcode(opcode))
}
