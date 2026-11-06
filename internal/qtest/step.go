package qtest

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Place is where a step was written. It is the first thing a failure has to
// say, so it travels with every step rather than being worked out afterwards.
type Place struct {
	File string
	Line int
	Col  int
}

func (p Place) String() string { return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Col) }

// op is what a step does.
type op int

const (
	opExpect    op = iota + 1 // wait for text
	opExpectNot               // text that must not have been said
	opKey                     // a keypress
	opInput                   // characters typed, with no return
	opWait                    // let the adventure's clock run
	opDebug                   // print something about the machine
	opAssert                  // check something about the machine
	opTrace                   // turn the condact trace on or off
	opBreak                   // hand the adventure to whoever is at the terminal
	opStep                    // open a named step
	opEnd                     // close it, which is what prints OK
)

// ask is what a DEBUG wants printed.
type ask int

const (
	askFlag ask = iota + 1
	askFlags
	askFlagsSet
	askAt
	askPicture
	askObjects
	askSaid
)

// check is what an ASSERT looks at.
type check int

const (
	chkFlag check = iota + 1
	chkAt
	chkPicture
	chkScore
	chkTurns
	chkCarried
	chkWorn
	chkHere
	chkPresent
	chkObject
	chkEnded
)

// cmp is how an ASSERT compares.
type cmp int

const (
	cmpEq cmp = iota
	cmpNot
	cmpGt
	cmpLt
)

func (c cmp) String() string {
	switch c {
	case cmpNot:
		return "!="
	case cmpGt:
		return ">"
	case cmpLt:
		return "<"
	default:
		return "="
	}
}

func (c cmp) holds(got, want int) bool {
	switch c {
	case cmpNot:
		return got != want
	case cmpGt:
		return got > want
	case cmpLt:
		return got < want
	default:
		return got == want
	}
}

// step is one order of a script.
//
// It is one struct with a kind rather than a type per order on purpose: the
// language is a dozen orders and all the interesting behaviour is in how the
// machine and the script take turns, which lives in one switch in run.go.
// Spreading twelve one-line methods across a file would hide it.
//
// Nothing here is written to while a script runs. A step that has to remember
// something — an EXPECT spending the clock — keeps it on the Runner, so the
// same script can be played twice.
type step struct {
	op op
	at Place

	// source is the line as it was written, which is what a failure shows. It
	// is better than anything this could reconstruct: it is what the author
	// will be looking at.
	source string

	// text is the words of an EXPECT or an INPUT, or the name of a STEP. An
	// INPUT's are typed and handed over together: one order is one command.
	text string

	// secs is how long an EXPECT or a WAIT waits, and timed says the author
	// wrote the number rather than leaving it to the script's TIMEOUT. See
	// resolve.
	secs  int
	timed bool

	// typed is what a KEY adds to the keyboard buffer, and enter says it is
	// the one that hands the line over instead. A KEY is the only way to put
	// something in that buffer and leave it there, which is what answering a
	// yes-or-no question a letter at a time needs.
	typed string
	enter bool

	// What a DEBUG asks for and what an ASSERT checks, with the numbers they
	// name: num is a flag or an object, val is what it should hold.
	ask    ask
	check  check
	num    int
	val    int
	cmp    cmp
	hasVal bool

	// on is TRACE ON rather than OFF.
	on bool
}

// Script is a parsed test.
type Script struct {
	// Name is what TEST called it, for the report.
	Name string

	// Game is the database the script wants played, if it said. A database
	// given from outside wins: the file is a convenience for running a script
	// on its own, not a claim on what it has to be run against.
	Game string

	// Seed fixes the dice. It is not a step, because by the time the first step
	// runs the adventure has already started and whatever CHANCE it has in its
	// opening has already been rolled.
	Seed    int64
	HasSeed bool

	// timeout is how long an EXPECT that does not say waits, which TIMEOUT
	// sets and defaultSeconds decides when nothing does.
	timeout int

	steps []step

	// tracing says the script turns the condact trace on and off itself, so a
	// trace asked for from the command line should start off and wait to be
	// told.
	tracing bool
}

// newScript is an empty script with the waiting it would do if nothing said
// otherwise.
func newScript() *Script { return &Script{timeout: defaultSeconds} }

// Steps is how many orders the script has, which is all anybody outside needs
// to know about them.
func (s *Script) Steps() int { return len(s.steps) }

// widest is the longest STEP name in the script, which the report needs before
// it draws the first one so that the verdicts line up.
func (s *Script) widest() int {
	most := 0

	for _, step := range s.steps {
		if step.op != opStep {
			continue
		}

		// The widest line of the name, because a name may hold a break and the
		// column has to clear the longest of its lines.
		for line := range strings.SplitSeq(step.text, "\n") {
			if n := utf8.RuneCountInString(line); n > most {
				most = n
			}
		}
	}

	return most
}
