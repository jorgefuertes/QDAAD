package decompiler

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
)

// Turning a version 1 database into a source a version 3 interpreter will run.
//
// This is not the same job as decompiling, and the difference is worth being
// plain about. Decompiling says what is in the binary; this says what the
// binary would have to be to behave the same under a newer interpreter. They
// disagree, so it is a flag and never the default: without it the source is a
// faithful account, and recompiling it reproduces the database it came from.
//
// What changes, and why, is written out in docs/DAAD_v2_v3_Research/07-daad-v3.md
// section 11. In short, DAAD 1 kept the game loop in the interpreter and
// version 3 keeps it in the database, so the old tables move out of the way and
// a loop is written for them; and three condacts behaved differently.
//
// It is done to the tables and the texts rather than to the source text. The
// difference is not stylistic: the same conversion written as regular
// expressions over the emitted source missed every PROCESS and every DESC that
// sits on an entry's header line — "> EX _ PROCESS 4" — because the obvious
// pattern expects a condact to start its line. Seven calls in La Aventura
// Original pointed at the wrong table and nothing complained, because the
// number was still a valid one.

// Opcodes this conversion has to know by name.
const (
	opEQ      = 13
	opDESC    = 19
	opQUIT    = 20
	opEND     = 21
	opCLS     = 29
	opDISPLAY = 28
	opDPRINT  = 27
	opSYSMESS = 54
	opPARSE   = 73
	opPICTURE = 84
	opPROCESS = 75
	opSKIP    = 116
	opRESTART = 117
)

// The system messages the endgame report is built out of. They are the
// adventure's own, in the slots AD.EXE named when it printed the report itself:
// 17 before the number, then 18, 19 for the plural and 20 to close. In La
// Aventura Original they hold "Has dado ", " orden", "es" and ".".
const (
	sysTurnsBefore = 17
	sysTurnsNoun   = 18
	sysTurnsPlural = 19
	sysTurnsAfter  = 20
)

// parseLiteral is the operand that makes PARSE read the phrase the player put
// in quotes rather than ask for a new line.
const parseLiteral = 1

// turnCounter is the low half of the pair the interpreter counts turns in, and
// it is the same pair in every version. AD.EXE — the interpreter shipped beside
// the two databases of La Aventura Original — does it in one instruction, once
// per turn, at 0x16b of its code segment:
//
//	16b: ff 06 49 01    inc word [0x0149]
//
// Its flags start at DS:0x12a (the indirection at 0x264 loads that address), so
// 0x0149 is flag 31 and the word is the pair 31/32. The same routine reads it
// back to print the endgame report, with the singular and the plural:
//
//	466: 8b 3e 49 01    mov di,[0x0149]   the count
//	46a: e8 85 11       call 0x15f2       printed as a 16-bit number
//	46d: 8b 0e 49 01    mov cx,[0x0149]
//	474: 49             dec cx            one turn or more than one
//	48f: a0 48 01       mov al,[0x0148]   and flag 30 is the score
const turnCounter = 31

// Where version 3 expects the loop to live. Tables 0, 1 and 2 are the loop
// itself and the parser's replies; 6 is the initialisation. The adventure's own
// tables go anywhere else.
const (
	v3MainLoop  = 0
	v3InputLoop = 1
	v3Parser    = 2
	v3Init      = 6
)

// v3Reserved are the tables the written loop occupies, in order, for the
// comment that tells a reader where they went.
var v3Reserved = []int{v3MainLoop, v3InputLoop, v3Parser, v3Init}

// movedTable says where a version 1 table goes.
//
// The first three have a meaning and move to where version 3 keeps that
// meaning: the responses, what runs after the description, and what runs once a
// turn. They are PAW's three tables, which DAAD 1 inherited. Everything after
// them is a subprocess with no meaning of its own, so it only has to get out of
// the way of the four the loop takes.
func movedTable(n int) int {
	switch n {
	case 0:
		return 5 // the response table
	case 1:
		return 3 // after the location description
	case 2:
		return 4 // once a turn
	default:
		return n + 4
	}
}

// v3Conversion is what this database needs, worked out once.
type v3Conversion struct {
	// tables is how many there are once the loop has its four.
	tables int
	// sysmess maps a system message the interpreter reserves to the free slot
	// its text was moved to. Empty when the adventure used none of them.
	sysmess map[int]int
	// named is every flag the database mentions anywhere outside a DPRINT, which
	// is what says whether a DPRINT's operand holds anything. Filled from the
	// old tables before any of them is rewritten.
	named map[byte]bool
	// report is the table holding the endgame report, added after the
	// adventure's own. Zero when the database has nothing that needs it.
	report int
}

// reservedSysmess are the system messages a version 3 interpreter prints by
// itself, as separators, and which an adventure of 1989 may well be using for
// its own text.
//
// In the official template all four are "#n" and nobody has ever noticed them.
// In La Aventura Original number 3 is "ir al " and number 4 is ". ", two pieces
// of the sentence "No puedes ir al Este. " that its response table builds by
// hand — and under version 3 they appeared on screen every turn, by themselves,
// looking like anything but a system message.
var reservedSysmess = []int{2, 3, 4, 5}

// freeSysmess are the slots the official template leaves empty, which makes
// them the only safe places to move a reserved message's text to.
//
// There are three, and they are not a guess: the template marks them itself.
// Every other slot it leaves blank carries a comment saying what the
// interpreter keeps there, and both the version 1 Spanish template
// (daad/Deprecated/SCE/SPANISH.SCE) and the version 3 one
// (daad-ready/ASSETS/TEMPLATES/BLANK_ES.DSF) agree on which three:
//
//	/11 ;*Libre      /17 ;*Has jugado    /20 ;*.[CR]
//	/14 ;*Libre      /18 ;*\sturno       /21 ;*Ha resuelto un
//	/34 ;*Libre      /19 ;*s             /22 ;*%[CR]
//
// Three are not always enough — the PC edition of La Aventura Original fills
// all four reserved slots — so the two the report does not use follow them.
// Never 17 to 20: those four are the report this conversion writes, and text
// moved there would come out in the middle of "Has dado 5 ordenes.".
//
// Picking merely "a slot this adventure leaves empty" is not enough either: the
// template fills 29 with "Ya llevo puesto _.", so moving text there would trade
// one message appearing uninvited for another.
var freeSysmess = []int{34, 11, 14, 22, 21}

// planV3 works out what this database needs converting.
func planV3(r *ddbfile.Reader) (*v3Conversion, error) {
	all, err := r.Processes()
	if err != nil {
		return nil, err
	}

	named := flagsNamed(all)

	plan := &v3Conversion{
		tables:  len(all) + len(v3Reserved),
		sysmess: map[int]int{},
		named:   named,
	}

	// The endgame report needs a table of its own, and it goes after every
	// table the adventure has so that none of theirs is renumbered.
	if hasDeadDPRINT(all, named) {
		plan.report = plan.tables
		plan.tables++
	}

	texts, err := r.SystemMessages()
	if err != nil {
		return nil, err
	}

	free := slices.Clone(freeSysmess)

	for _, slot := range reservedSysmess {
		// A slot the adventure left empty is already what the interpreter
		// wants, near enough, and needs no home of its own.
		if slot >= len(texts) || texts[slot] == "" {
			continue
		}

		var to int

		for {
			if len(free) == 0 {
				return nil, fmt.Errorf(
					"system message %d holds %q and there is no free slot to move it to",
					slot, texts[slot])
			}

			to, free = free[0], free[1:]
			if to < len(texts) && texts[to] == "" {
				break
			}
		}

		plan.sysmess[slot] = to
	}

	return plan, nil
}

// convertSysmess rewrites the system message table: the reserved slots get the
// newline the interpreter expects to print, and their text moves to the slots
// the plan found for it.
func (c *v3Conversion) convertSysmess(texts []string) []string {
	if len(c.sysmess) == 0 {
		return texts
	}

	out := slices.Clone(texts)

	for from, to := range c.sysmess {
		out[to] = out[from]
		out[from] = "#n"
	}

	return out
}

// Processes, Texts and SystemMessages are what ddbfile.Conversion asks for.
func (c *v3Conversion) Processes(all [][]ddbfile.Entry) [][]ddbfile.Entry {
	return c.convertProcesses(all)
}

func (c *v3Conversion) Texts(texts []string) []string { return c.convertTexts(texts) }

func (c *v3Conversion) SystemMessages(texts []string) []string { return c.convertSysmess(texts) }

// convertProcesses moves every table to where version 3 expects it and rewrites
// the condacts that do not mean the same thing any more.
func (c *v3Conversion) convertProcesses(old [][]ddbfile.Entry) [][]ddbfile.Entry {
	// Before anything is rewritten, because what a DPRINT means depends on what
	// the rest of the tables do with the flag it names. planV3 has worked this
	// out already; a conversion built by hand, as the tests do, has not.
	if c.named == nil {
		c.named = flagsNamed(old)
	}

	out := make([][]ddbfile.Entry, c.tables)

	for number, entries := range old {
		moved := make([]ddbfile.Entry, 0, len(entries))

		for _, e := range entries {
			moved = append(moved, ddbfile.Entry{
				Verb: e.Verb, Noun: e.Noun,
				Condacts: c.convertCondacts(e.Condacts),
			})
		}

		out[movedTable(number)] = moved
	}

	if c.report > 0 {
		out[c.report] = endgameReport()
	}

	return out
}

// flagsNamed is every flag number the tables mention, except the ones a DPRINT
// names. The exception is the point of it: a DPRINT is the only condact here
// whose operand is being judged, so counting its own mention would make every
// flag look used.
//
// Both ways a condact can name a flag are counted. The declared kind is one,
// and the indirection is the other: bit 7 of the opcode makes the first operand
// a flag number whatever the table says that operand is.
func flagsNamed(all [][]ddbfile.Entry) map[byte]bool {
	named := map[byte]bool{}

	for _, entries := range all {
		for _, e := range entries {
			for _, condact := range e.Condacts {
				if condact.Opcode == opDPRINT {
					continue
				}

				if condact.Indirect && len(condact.Params) > 0 {
					named[condact.Params[0]] = true
				}

				def, found := ddb.LookupCondact(ddb.Opcode(condact.Opcode))
				if !found {
					continue
				}

				for i, kind := range def.Params {
					if kind == ddb.ParamFlag && i < len(condact.Params) {
						named[condact.Params[i]] = true
					}
				}
			}
		}
	}

	return named
}

// deadPair says whether a DPRINT of this flag can only ever print a zero:
// nothing else in the database names either half of the pair, and no
// interpreter writes them either.
//
// That last part is what makes this safe to act on. In version 1 the flags
// between 4 and 19 and from 21 up to the system block are the author's to use —
// AD.EXE never once refers to flag 21 or 22, the way it refers to 30, 31 and 32
// — so a version 1 adventure counting something of its own in 21/22 is doing
// nothing wrong, and this leaves it alone because the condacts that count it
// name the flag.
func deadPair(named map[byte]bool, flag byte) bool {
	// The manual says DPRINT 255 is meaningless, and a pair needs a second half.
	if flag == 255 || flag == turnCounter {
		return false
	}

	return !named[flag] && !named[flag+1]
}

func (c *v3Conversion) deadPair(flag byte) bool { return deadPair(c.named, flag) }

// reportAt says whether this condact of a block is one the endgame report
// replaces: a DPRINT of a pair nothing fills, in a conversion that has a table
// to put the report in.
func (c *v3Conversion) reportAt(block []ddbfile.Condact, at int) bool {
	if c.report == 0 || at < 0 || at >= len(block) {
		return false
	}

	one := block[at]

	return one.Opcode == opDPRINT && len(one.Params) == 1 && !one.Indirect &&
		c.deadPair(one.Params[0])
}

// hasDeadDPRINT says whether the database prints a pair nothing fills, which is
// what decides that a table for the endgame report is worth having.
func hasDeadDPRINT(all [][]ddbfile.Entry, named map[byte]bool) bool {
	for _, entries := range all {
		for _, e := range entries {
			for _, c := range e.Condacts {
				if c.Opcode == opDPRINT && len(c.Params) == 1 && !c.Indirect &&
					deadPair(named, c.Params[0]) {
					return true
				}
			}
		}
	}

	return false
}

// endgameReport is the report the version 1 interpreter printed by itself and
// version 3 does not: how many turns the game took, in a sentence.
//
// It goes in a table of its own because the plural needs a condition, and a
// condition inside DAAD is an entry: SKIP steps over whole entries, so there is
// no branching within one block. The table is added after the adventure's own
// rather than among them, so that nothing is renumbered.
//
// The shape is AD.EXE's, which built the sentence out of four system messages
// and chose the singular by counting down from the turn count:
//
//	466: 8b 3e 49 01    mov di,[0x0149]   the count, printed as 16 bits
//	471: be 12 00       mov si,0x12       message 18
//	474: 49             dec cx            one turn, or more than one
//	475: 74 06          je 0x47d          one: no "es"
//
// The score half of that routine is not here: it reads flag 30, which these
// adventures never set, and prints it between two system messages that La
// Aventura Original leaves empty.
func endgameReport() []ddbfile.Entry {
	sysmess := func(n byte) ddbfile.Condact {
		return ddbfile.Condact{Opcode: opSYSMESS, Params: []byte{n}}
	}

	return []ddbfile.Entry{
		{
			Verb: ddbfile.NoWord, Noun: ddbfile.NoWord,
			Condacts: []ddbfile.Condact{
				sysmess(sysTurnsBefore),
				{Opcode: opDPRINT, Params: []byte{turnCounter}},
			},
		},
		// One turn exactly: both halves of the pair say so, and the noun goes
		// in without its plural.
		{
			Verb: ddbfile.NoWord, Noun: ddbfile.NoWord,
			Condacts: []ddbfile.Condact{
				{Opcode: opEQ, Params: []byte{turnCounter + 1, 0}},
				{Opcode: opEQ, Params: []byte{turnCounter, 1}},
				sysmess(sysTurnsNoun),
				sysmess(sysTurnsAfter),
				{Opcode: opSKIP, Params: []byte{1}},
			},
		},
		{
			Verb: ddbfile.NoWord, Noun: ddbfile.NoWord,
			Condacts: []ddbfile.Condact{
				sysmess(sysTurnsNoun),
				sysmess(sysTurnsPlural),
				sysmess(sysTurnsAfter),
			},
		},
	}
}

// convertCondacts rewrites one block.
func (c *v3Conversion) convertCondacts(in []ddbfile.Condact) []ddbfile.Condact {
	out := make([]ddbfile.Condact, 0, len(in))
	eaten := -1

	for i, condact := range in {
		if i == eaten {
			continue
		}

		switch {
		// QUIT with the report right after it, which is how both adventures of
		// 1989 end: the report has to print before END, because END ends the
		// game and nothing after it runs. Taken as a pair so that the order
		// comes out the way the player of 1989 saw it — the count, and then
		// "¿Hace otra partidita?".
		case condact.Opcode == opQUIT && c.reportAt(in, i+1):
			out = append(out,
				ddbfile.Condact{Opcode: opQUIT},
				ddbfile.Condact{Opcode: opPROCESS, Params: []byte{byte(c.report)}},
				ddbfile.Condact{Opcode: opEND})

			eaten = i + 1
		// A PROCESS names a table, and the tables have moved.
		case condact.Opcode == opPROCESS && len(condact.Params) == 1 && !condact.Indirect:
			out = append(out, ddbfile.Condact{
				Opcode: opPROCESS,
				Params: []byte{byte(movedTable(int(condact.Params[0])))},
			})

		// A SYSMESS may name a slot whose text moved.
		case condact.Opcode == opSYSMESS && len(condact.Params) == 1 && !condact.Indirect:
			slot := int(condact.Params[0])
			if to, moved := c.sysmess[slot]; moved {
				slot = to
			}

			out = append(out, ddbfile.Condact{Opcode: opSYSMESS, Params: []byte{byte(slot)}})

		// A DESC with no operand described the player's location and ended the
		// turn there, the way PAW did. Version 3's DESC only describes, and the
		// entry carries on into the next one — which is why La Aventura
		// Original announced "No puedes ir al Este." immediately after going
		// east.
		//
		// The version 3 way of saying "describe here and end the turn" is not
		// DESC with a DONE after it, which would print the description a step
		// too early. It is RESTART, because RESTART restarts table 0 and table
		// 0 is what describes. CLS first, which is what the old DESC did to the
		// text window.
		case condact.Opcode == opDESC && len(condact.Params) == 0:
			out = append(out,
				ddbfile.Condact{Opcode: opCLS},
				ddbfile.Condact{Opcode: opRESTART})

		// PARSE carried no operand in DAAD 1, and version 3 gives it one that
		// chooses between two different things: 0 asks the player for a line,
		// 1 reads the phrase they already put in quotes.
		//
		// The 1 is the one an adventure of 1989 means. Talking to somebody in
		// La Aventura Original is DECIR "SIGUEME" — the quotes are in the game's
		// own instructions and in every solution ever written for it — and that
		// only works if the quoted phrase is what the conversation parses.
		// Converted to 0 the phrase is thrown away and the elf's table eats the
		// player's *next* command instead, so the answer arrives one turn late
		// and attached to the wrong order.
		//
		// This is read from how the adventure is played rather than from a
		// disassembly of its interpreter, which is a weaker kind of evidence
		// and worth saying so. What makes it hard to doubt is that the other
		// reading makes the published solution impossible.
		case condact.Opcode == opPARSE && len(condact.Params) == 0:
			out = append(out, ddbfile.Condact{
				Opcode: opPARSE,
				Params: []byte{parseLiteral},
			})

		// QUIT ended the game in DAAD 1. In version 3 it only asks, and END is
		// what ends it — so an adventure of 1989 has no END anywhere and
		// confirming "quit" did nothing at all.
		case condact.Opcode == opQUIT && !followedBy(in, i, opEND):
			out = append(out,
				ddbfile.Condact{Opcode: opQUIT},
				ddbfile.Condact{Opcode: opEND})

		// PICTURE drew in DAAD 1. In version 3 it only finds the image and gets
		// it ready, and DISPLAY 0 is what puts it on the screen.
		//
		// So a version 1 database asks for pictures that never arrive. La
		// Aventura Original has five PICTUREs and not one DISPLAY, which is the
		// proof that the old one drew: the frame around its opening text is
		// picture 44, and the grating at location 26 is pictures 46, 47 and 48,
		// one per state of the padlock as it opens. Converted literally, the
		// opening text comes up bare and the padlock never changes.
		//
		// The adventure's own shape says where the DISPLAY goes. Every one of
		// the five sits between a WINDOW 0 and a WINDOW 1 — into the graphics
		// window and straight back out — so the drawing happened there and then,
		// and that is where this puts it.
		//
		// One difference comes with it: version 3's PICTURE is a condition and
		// fails when there is no such image, taking the rest of the entry with
		// it, where the old one would have carried on. All five of these exist.
		case condact.Opcode == opPICTURE && !followedBy(in, i, opDISPLAY):
			out = append(out,
				condact,
				ddbfile.Condact{Opcode: opDISPLAY, Params: []byte{0}})

		// A DPRINT of a pair nothing fills prints a zero, and what it meant was
		// the endgame report.
		//
		// This one is a repair and not a translation, so it is worth separating
		// what is measured from what is judged. Measured: La Aventura Original
		// and El Jabato both say DPRINT 21, five times between them, and neither
		// ever writes flag 21 or 22 anywhere. Their interpreter does not write
		// them either — see turnCounter above, where AD.EXE counts turns in the
		// pair 31/32 and never once refers to 0x13f or 0x140. So that DPRINT
		// printed a bare zero in 1989 exactly as it does now: the adventures
		// have a typo, one digit wide, and the three later version 2 adventures
		// of the same house — Cozumel, Chichén Itzá and Los Templos Sagrados —
		// write DPRINT 31 in the same place, which is the flag they all meant.
		//
		// Measured too: what the player of 1989 read there was not a number on
		// its own but a sentence, "Has dado 5 ordenes.", and the interpreter
		// built it. Version 3's does not, so the report goes into a table of
		// the conversion's own — see endgameReport — and this calls it.
		//
		// Judged: that the sentence is better than the bare zero the typo gave.
		// It is a deliberate difference from the original and the only one here
		// that the binary does not ask for.
		//
		// The guard is what keeps it from being a blind substitution. In version
		// 1 those flags belong to the author, so an adventure that really does
		// keep a 16-bit number of its own in 21/22 names them in the condacts
		// that count it, and then this leaves the DPRINT alone.
		case c.reportAt(in, i):
			out = append(out, ddbfile.Condact{
				Opcode: opPROCESS,
				Params: []byte{byte(c.report)},
			})

		default:
			out = append(out, condact)
		}
	}

	return out
}

// followedBy says whether the next condact of a block is this opcode, so that a
// database which already does the version 3 thing is left alone.
func followedBy(block []ddbfile.Condact, at int, opcode byte) bool {
	return at+1 < len(block) && block[at+1].Opcode == opcode
}

// writeV3Loop writes the game loop that a version 1 database does not have.
//
// It is the one part of the conversion that is not a translation. The rest says
// in version 3 terms what the binary already said; this says something the
// binary never said, because in 1989 it lived in the interpreter's own code.
//
// So it goes in a file of its own rather than being folded into the tables, and
// every line of it that is a guess says so. It is the official template's loop
// with the graphics taken out, and the places where the template knows
// something about its own game — which flag means darkness, which object is the
// lamp, how much the player can carry — are places where this knows nothing.
func v3LoopSection(plan *v3Conversion) func(*ddbfile.Reader) (string, error) {
	// The machine comes from the database rather than from a flag: a loop
	// sized for the wrong screen is a loop that writes off the edge of it.
	return func(r *ddbfile.Reader) (string, error) {
		return writeV3Loop(plan, arch.Machine(r.Machine()))
	}
}

func writeV3Loop(plan *v3Conversion, machine arch.Machine) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("no version 3 conversion is in effect")
	}

	var sb strings.Builder

	sb.WriteString(`; El bucle del juego, que una base de datos de formato 1 no trae.
;
; En DAAD 1 el bucle estaba en el código del intérprete: describir, pedir la
; orden, parsearla, ejecutar la tabla de respuestas, mover al jugador. El
; intérprete v3 no tiene bucle propio —apila el proceso 0 y ejecuta lo que
; encuentre—, así que hay que escribirlo, y esto es el de la plantilla oficial
; con los gráficos quitados.
;
; ATENCIÓN: esto no es una traducción de nada. Lo demás que emite la conversión
; dice en términos de v3 lo que el binario ya decía; esto dice algo que el
; binario nunca dijo. Las líneas marcadas con CONJETURA son valores que la
; plantilla sabe de su propio juego y que de esta aventura no se saben.
;
; Dónde ha ido cada tabla de la aventura:
;
`)

	for _, old := range []int{0, 1, 2} {
		fmt.Fprintf(&sb, ";   tabla %d  ->  %d   %s\n", old, movedTable(old), tableRole(old))
	}

	fmt.Fprintf(&sb, ";   tabla 3 en adelante  ->  %d en adelante\n;\n", movedTable(3))

	if len(plan.sysmess) > 0 {
		sb.WriteString("; Y qué mensajes de sistema se han movido, porque el intérprete v3 los\n")
		sb.WriteString("; imprime por su cuenta como separadores:\n;\n")

		for _, from := range reservedSysmess {
			if to, moved := plan.sysmess[from]; moved {
				fmt.Fprintf(&sb, ";   mensaje %d  ->  %d\n", from, to)
			}
		}

		sb.WriteString(";\n")
	}

	sb.WriteString(strings.ReplaceAll(v3LoopBody, windowToken, windowSize(machine)))

	return sb.String(), nil
}

// windowToken stands where the loop sizes its text window, so that the two
// places it does so cannot drift apart.
const windowToken = "@VENTANA@"

// windowSize is the WINSIZE operands for a machine's whole screen.
//
// Every DAAD source asks for more than the screen holds and lets the
// interpreter clamp — "WINSIZE 25 127", with a comment saying maximum size —
// and that works, because the interpreters do clamp. It is a bet all the same,
// and on a Spectrum it is lost: a window declared 25 rows tall on a 24-row
// screen puts its first line on a row that is not there, and the lines after
// it wrap round to the top. La Aventura Original opens with "Este hay una
// cabaña de ladrillos rodeada de árboles", because "Estás en el bosque. Muy
// cerca hacia el" went to the row below the bottom of the screen.
//
// Where the size is known it is written down. Where it is not, the old idiom
// stays: asking for too much and being clamped still beats inventing a number.
func windowSize(machine arch.Machine) string {
	rows, cols, known := machine.TextSize()
	if !known {
		return "25 127   ; más de lo que cabe, para que el intérprete lo recorte"
	}

	return fmt.Sprintf("%d %d   ; la pantalla entera de un %s", rows, cols, machine)
}

// tableRole names what version 1 kept in each of its first three tables.
func tableRole(n int) string {
	switch n {
	case 0:
		return "la tabla de respuestas"
	case 1:
		return "tras describir la localidad"
	case 2:
		return "cada turno"
	default:
		return "subproceso"
	}
}

// v3LoopBody is the loop itself. The flag numbers are the ones every DAAD
// interpreter fixes — 28 darkness, 31 and 32 the turn counter, 33 the verb, 38
// the player's location, 49 the timeout flags — so those are not guesses. What
// is written beside CONJETURA is.
const v3LoopBody = `
/PRO 0

> _       _     AT      0
                PROCESS 6               ; inicialización

> _       _     WINDOW  1
                WINAT   0 0
                WINSIZE @VENTANA@

; CONJETURA: que el flag 0 marca la oscuridad y el objeto 0 es la lámpara. Es
; lo que hace la plantilla oficial; esta aventura puede llevarlo en otro sitio.
> _       _     CLEAR   28
                NOTZERO 0
                ABSENT  0
                SET     28

> _       _     NOTZERO 28
                SYSMESS 0

> _       _     ZERO    28
                DESC    @38

> _       _     PROCESS 3

> _       _     PROCESS 1

/PRO 1

> _       _     PROCESS 4

; La ventana vuelve a ser la nuestra antes de preguntar, porque las tablas de
; la aventura cambian de ventana y no todas la devuelven. La Aventura Original
; deja puesta la 5 —"WINDOW 5 / WINAT 0 5 / WINSIZE 25 36"— y el prompt salía
; en ella: en la columna 5 y en la fila 0, encima de la primera línea de la
; descripción, que se perdía entera.
;
; Sin WINAT ni WINSIZE a propósito. Cada ventana lleva su propio cursor, así
; que seleccionarla basta para seguir donde la descripción lo dejó; fijarla
; otra vez lo devolvería a 0,0, que es exactamente el defecto que esto arregla.
> _       _     WINDOW  1

; Una frase sin verbo deja fVerb a un valor que las entradas comodín de una
; tabla de respuestas de DAAD 1 toman por una dirección, porque comprueban
; "LT 33 14". El 255 es lo que el intérprete de 1989 dejaba ahí, y hace falta a
; los dos lados del PARSE: antes porque si el parser no lo toca queda lo del
; turno anterior, y después porque cuando no hay verbo lo pone a cero.
> _       _     LET     33 255

> _       _     PARSE   0
                PROCESS 2
                REDO

> _       _     ZERO    33
                LET     33 255

> _       _     EQ      31 255
                PLUS    32 1
                CLEAR   31
                SKIP    1

> _       _     PLUS    31 1

> _       _     PROCESS 5
                ISDONE
                REDO

> _       _     MOVE    38
                CLS
                RESTART

> _       _     NEWTEXT
                LT      33 14
                SYSMESS 7
                REDO

> _       _     SYSMESS 8
                REDO

/PRO 2

> _       _     HASAT   49
                SYSMESS 35
                DONE

> _       _     SYSMESS 6

/PRO 6

; La ventana gráfica se queda sin superficie, porque aquí no hay ilustraciones:
; la versión 3 no dibuja vectoriales, que es lo único que trae una aventura de
; Spectrum de 1989. Sin esto, la ventana 0 conserva el tamaño que le da el
; intérprete y encima de la nuestra, así que un "WINDOW 0 / CLS" de la aventura
; —La Aventura Original lo hace al entrar en la depresión de 7 metros— borra la
; primera línea de la descripción que acabamos de escribir.
;
; La plantilla oficial no lo padece porque pone su texto en "WINAT 14 0", debajo
; del hueco de la imagen. Nosotros no podemos permitírnoslo: serían 14 filas de
; 24 perdidas para una imagen que no va a existir.
> _       _     WINDOW  0
                WINAT   0 0
                WINSIZE 0 0

> _       _     WINDOW  1
                WINAT   0 0
                WINSIZE @VENTANA@
                CLS
                SET     255

$limpia
> _       _     MINUS   255 1
                NOTEQ   255 29
                CLEAR   @255

> _       _     NOTZERO 255
                SKIP    $limpia

; CONJETURA: la fuerza y cuánto se puede llevar. Son los valores por defecto de
; la plantilla, no los de esta aventura.
> _       _     RESET
                LET     52 10
                LET     37 4
                SET     46
                SET     47
                LET     1 0
`

// convertText takes the spaces out of a run of graphics mode.
//
// #g and #t turn graphics mode on and off, and in it every byte is drawn with
// the glyph from the top half of the font. An adventure of 1989 wrote its
// titles that way, spaces included, counting on its own font having nothing in
// the slot a space lands on — 0xA0.
//
// AD8x6.CHR, which is the font a version 3 interpreter loads and the only one
// whose top half is filled in at all, has an Á there. So every space of the
// title came out as "Á". The fix is not to ask for a space in graphics mode:
// turn it off before and back on after.
func convertText(text string) string {
	var sb strings.Builder

	rest := text

	for {
		open := strings.Index(rest, "#g")
		if open < 0 {
			sb.WriteString(rest)

			return sb.String()
		}

		close := strings.Index(rest[open:], "#t")
		if close < 0 {
			sb.WriteString(rest)

			return sb.String()
		}

		close += open

		sb.WriteString(rest[:open])
		sb.WriteString("#g")
		sb.WriteString(strings.ReplaceAll(rest[open+2:close], " ", "#t #g"))
		sb.WriteString("#t")

		rest = rest[close+2:]
	}
}

// convertTexts applies convertText to a whole table.
func (c *v3Conversion) convertTexts(texts []string) []string {
	out := make([]string, len(texts))
	for i, text := range texts {
		out[i] = convertText(text)
	}

	return out
}

// convertToV3 makes everything the reader hands out come back converted for a
// version 3 interpreter, and gives back nothing at all when there is nothing to
// do.
//
// A format 2 or 3 database already keeps its loop where version 3 looks for it,
// so asking for the conversion is answered by not doing one. That is a nil plan
// and no error on purpose: --v3 asks for a source a version 3 interpreter will
// run, and for these the plain decompilation already is one. Refusing instead
// made the flag unusable over a mixed corpus — the five adventures are two
// formats, and a harness that walks all of them had to know which before it
// could ask.
func convertToV3(r *ddbfile.Reader) (*v3Conversion, error) {
	if !r.OldFormat() {
		return nil, nil
	}

	plan, err := planV3(r)
	if err != nil {
		return nil, err
	}

	r.Convert(plan)

	return plan, nil
}
