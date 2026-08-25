package ddb

import "github.com/jorgefuertes/QDAAD/internal/qderror"

// MAX_PROCESS_ID is the last usable process number: the header stores the
// process count in a single byte, so 255 processes fit and they are numbered
// from 0. See docs/DAAD_v2_v3_Research/15-limites.md.
const MAX_PROCESS_ID ID = 254

// Param is one condact operand. Indirect is the "@" of the source: the value
// is then a flag number, and the interpreter uses the flag's content instead.
//
// How indirection reaches the binary is not stored here. In v2 it is bit 7 of
// the opcode; in v3 the second parameter needs a whole INDIR condact in front.
// Both are the emitter's business.
type Param struct {
	Value    ID
	Indirect bool
}

// Condact is one instruction. The number of parameters is not stored in the
// DDB: it is deduced from the opcode through the condact table.
type Condact struct {
	Opcode Opcode
	Params []Param
}

// Entry is a "> verb noun" line of the source with its condact block.
// NO_WORD_ID in Verb or Noun is the "_" wildcard: it matches anything.
type Entry struct {
	Verb     ID
	Noun     ID
	Condacts []Condact
}

// Process is one process table: a numbered list of entries. The interpreter
// has no game loop of its own — it stacks process 0 and runs whatever is
// there — so this is where the adventure's program lives.
type Process struct {
	ID      ID
	LabelID ID16
	Entries []Entry
}

type Processes []Process

func NewProcessStore() Processes {
	return Processes{}
}

// Add opens a new process and returns its number. Numbering is sequential and
// dense because the process table of a DDB is an array indexed by that number:
// a gap would have to be materialized as an empty process anyway.
func (ps *Processes) Add(labelID ID16) (ID, error) {
	if len(*ps) > MAX_PROCESS_ID.Int() {
		return 0, qderror.ErrProcessStoreIsFull
	}

	id := ID(len(*ps))
	*ps = append(*ps, Process{ID: id, LabelID: labelID})

	return id, nil
}

// AddEntry appends an entry to the last process. Building is sequential: there
// is no random access, and nothing hands out pointers into the store, so no
// later append can invalidate anything.
func (ps *Processes) AddEntry(verb, noun ID) error {
	if len(*ps) == 0 {
		return qderror.ErrProcessNoProcess
	}

	p := &(*ps)[len(*ps)-1]
	p.Entries = append(p.Entries, Entry{Verb: verb, Noun: noun})

	return nil
}

// AddCondact appends a condact to the last entry of the last process, checking
// that the opcode exists and that the parameters match its arity.
func (ps *Processes) AddCondact(op Opcode, params ...Param) error {
	if len(*ps) == 0 {
		return qderror.ErrProcessNoProcess
	}

	p := &(*ps)[len(*ps)-1]
	if len(p.Entries) == 0 {
		return qderror.ErrProcessNoEntry
	}

	def, found := LookupCondact(op)
	if !found {
		return qderror.ErrInvalidOpcode
	}

	if len(params) != def.NumParams() {
		return qderror.ErrCondactParamCount
	}

	// The parameters are copied: a variadic slice called with "params..." is
	// the caller's own array, with no copy in between. Appending to a nil slice
	// allocates a fresh one.
	var stored []Param

	stored = append(stored, params...)

	e := &p.Entries[len(p.Entries)-1]
	e.Condacts = append(e.Condacts, Condact{Opcode: op, Params: stored})

	return nil
}
