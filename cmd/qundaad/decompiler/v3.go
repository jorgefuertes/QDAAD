package decompiler

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
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
	opDESC    = 19
	opQUIT    = 20
	opEND     = 21
	opCLS     = 29
	opDISPLAY = 28
	opSYSMESS = 54
	opPICTURE = 84
	opPROCESS = 75
	opRESTART = 117
)

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

// freeSysmess are slots the official template leaves empty in every language,
// which makes them the only safe places to move a reserved message's text to.
//
// Picking merely "a slot this adventure leaves empty" is not enough: the
// template fills 29 with "Ya llevo puesto _.", so moving text there would trade
// one message appearing uninvited for another.
var freeSysmess = []int{34, 22, 21, 20, 19, 18, 17, 14, 11}

// planV3 works out what this database needs converting.
func planV3(r *ddbfile.Reader) (*v3Conversion, error) {
	all, err := r.Processes()
	if err != nil {
		return nil, err
	}

	plan := &v3Conversion{
		tables:  len(all) + len(v3Reserved),
		sysmess: map[int]int{},
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

	return out
}

// convertCondacts rewrites one block.
func (c *v3Conversion) convertCondacts(in []ddbfile.Condact) []ddbfile.Condact {
	out := make([]ddbfile.Condact, 0, len(in))

	for i, condact := range in {
		switch {
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

		// QUIT ended the game in DAAD 1. In version 3 it only asks, and END is
		// what ends it — so an adventure of 1989 has no END anywhere and
		// confirming "quit" did nothing at all.
		case condact.Opcode == opQUIT && !followedBy(in, i, opEND):
			out = append(out,
				ddbfile.Condact{Opcode: opQUIT},
				ddbfile.Condact{Opcode: opEND})

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
	return func(_ *ddbfile.Reader) (string, error) {
		return writeV3Loop(plan)
	}
}

func writeV3Loop(plan *v3Conversion) (string, error) {
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

	sb.WriteString(v3LoopBody)

	return sb.String(), nil
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
                WINSIZE 25 127

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

> _       _     WINDOW  1
                WINAT   0 0
                WINSIZE 25 128
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
// version 3 interpreter. It is only ever right for a format 1 database: the
// later ones already keep their loop where version 3 looks for it.
func convertToV3(r *ddbfile.Reader) (*v3Conversion, error) {
	if !r.OldFormat() {
		return nil, fmt.Errorf("this database is format %d and needs no converting", r.Version())
	}

	plan, err := planV3(r)
	if err != nil {
		return nil, err
	}

	r.Convert(plan)

	return plan, nil
}
