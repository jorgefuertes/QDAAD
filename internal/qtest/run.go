package qtest

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/vm"
)

// Options is what a run needs besides a script and a database.
type Options struct {
	// Report is where the run is written. A nil one says nothing, which is
	// what a test that only wants the verdict asks for.
	Report *Report

	// Seed fixes the dice and wins over the script's own SEED, because it
	// comes from the person running it.
	Seed    int64
	HasSeed bool

	// Show echoes what the adventure says as it says it. A test run is about
	// the steps, so it is off by default and the transcript is where the words
	// go.
	Show bool

	// Transcript is everything the adventure said, kept whatever happens.
	Transcript io.Writer

	// Trace is where a condact trace goes, if one was asked for. A script with
	// TRACE orders in it starts with the trace off and turns it on itself;
	// without them, a trace asked for from outside covers the whole run.
	Trace vm.Watcher

	// Breakpoint is the console a BREAKPOINT hands the adventure to, which is
	// the one a person plays on. Handing it over whole rather than imitating a
	// terminal is what makes the clock and the pauses behave for them exactly
	// as they do when there is no script at all.
	//
	// Without one there is nobody to hand it to, and a BREAKPOINT says so and
	// carries on — a script with one left in it still runs in a suite.
	Breakpoint vm.Console
}

// Resume is the line that ends a BREAKPOINT and gives the script the adventure
// back. It is never handed to the machine.
const Resume = "CONTINUE TEST"

// ErrNoDatabase is what a script that neither names a database nor is given
// one gets. Playing nothing quietly and reporting that no step failed would be
// the worst answer there is.
var ErrNoDatabase = errors.New("no database to play: give one with --input or name it with GAME")

// Play runs a script against a database and gives back whether it held.
//
// An error here is a test that failed, not a program that broke: the one thing
// a caller has to do with it is exit non-zero.
func Play(db *ddbfile.Reader, script *Script, o Options) error {
	if o.Report == nil {
		o.Report = NewReport(io.Discard)
	}

	r := &Runner{
		script:  script,
		report:  o.Report,
		opts:    o,
		tracing: o.Trace != nil && !script.tracing,
	}

	machine, err := vm.New(db, r, r)
	if err != nil {
		return err
	}

	r.machine = machine

	// The dice are set before the first turn and not from a step, because by
	// the time a step runs the adventure has already started and whatever
	// CHANCE is in its opening has already been rolled.
	switch {
	case o.HasSeed:
		machine.SetSeed(o.Seed)
	case script.HasSeed:
		machine.SetSeed(script.Seed)
	}

	o.Report.Title(script.Name)

	// The widest name is known before anything runs, which is what lets every
	// badge land in the same column instead of chasing the end of each line.
	o.Report.Align(script.widest())

	if err := machine.Run(); err != nil {
		o.Report.Broke(err)
	} else {
		// What is left of the script still has the last of what was said to
		// look at, and an ASSERT ENDED is the whole point of being here.
		r.pump(atEnd, 0)
	}

	o.Report.Summary()

	return o.Report.Err()
}

// Runner plays a script. It is the machine's console, its clock and its
// watcher, which is everything the machine asks of the world.
type Runner struct {
	script  *Script
	report  *Report
	opts    Options
	machine *vm.Machine

	// at is the step being run and waited is how long it has been waiting.
	// Only an EXPECT or a WAIT ever leaves anything in waited, and it is the
	// only thing a half-finished step remembers.
	at     int
	waited time.Duration

	// typed is the keyboard buffer, which only a KEY ever leaves anything in:
	// an INPUT types its text and presses return in the same breath, so it
	// empties the buffer again on its way out. line is what was handed over.
	typed strings.Builder
	line  string

	// said is what the adventure has said that no EXPECT has matched yet.
	// Matching eats it up to the end of the match, which is what makes the
	// order things are said in part of the test.
	said string

	// order is the last thing the script gave the adventure, as it was
	// written, and since is what the adventure said back. Nothing eats since:
	// it is emptied when the next order goes in and that is all.
	//
	// It is the pair somebody writing a script from scratch is after. What
	// they want is not everything still unmatched — which is the leftovers of
	// however many commands ago — but the answer to the command they just
	// typed, so they can lift the phrase out of it and put it in an EXPECT.
	order string
	since string

	// stopped is a failure having happened. Nothing else runs after one: a
	// walkthrough that is one answer behind answers every remaining step in
	// the wrong room and reports nonsense in ten places instead of the truth
	// in one.
	stopped bool

	// manual is a BREAKPOINT being in force: the adventure belongs to whoever
	// is at the terminal and the script does not advance at all, however many
	// turns they take. Only Resume gives it back.
	manual bool

	tracing bool
}

// Print is the adventure speaking.
func (r *Runner) Print(text string) {
	r.said += text
	r.since += text

	if r.opts.Transcript != nil {
		_, _ = io.WriteString(r.opts.Transcript, text)
	}

	// Inside a breakpoint it goes to the person whatever --show says: they are
	// playing, and playing blind is not playing.
	if r.manual {
		r.opts.Breakpoint.Print(text)

		return
	}

	if r.opts.Show {
		r.report.Said(text)
	}
}

// Input is the adventure asking for an order with no clock on it.
func (r *Runner) Input() (string, bool) {
	for {
		switch r.pump(atLine, 0) {
		case outLine:
			return r.line, true

		case outManual:
			typed, more := r.opts.Breakpoint.Input()

			line, more, back := r.played(typed, more)
			if back {
				continue
			}

			return line, more

		default:
			// The script having nothing more to say ends the game, which is
			// how a script that does not play to the end still finishes.
			return "", false
		}
	}
}

// InputWithin is the same with the adventure's TIME on it, and it is what
// makes the seconds in this language mean anything. See the package comment.
func (r *Runner) InputWithin(limit time.Duration) (line string, more, expired bool) {
	for {
		switch r.pump(atLine, limit) {
		case outLine:
			return r.line, true, false

		case outClock:
			return "", true, true

		case outManual:
			// The clock is theirs too: a person at a breakpoint gets the
			// seconds the adventure gives a player, and times out like one.
			typed, more, expired := r.listen(limit)

			// And an expiry is not a line. Handing the machine an empty one
			// instead loses the only thing it needed to know — that nobody
			// answered in time — so it answers "Lo siento, no entiendo", which
			// is what it says to somebody who typed nothing, rather than its
			// "El tiempo pasa... ¡Y no vuelve!". Resume is not looked for here
			// either: nothing was typed, so there is nothing to look in.
			if expired {
				return "", true, true
			}

			line, more, back := r.played(typed, more)
			if back {
				continue
			}

			return line, more, false

		default:
			return "", false, false
		}
	}
}

// listen takes a line from whoever is at the terminal during a breakpoint,
// with the adventure's clock on it where their console has one.
//
// A console that cannot be told to stop waiting never times out, which is not
// a failure: it is a console with no clock worth the name, and vm.Impatient
// exists because most of them are. What it never does is invent an expiry, nor
// swallow a real one.
func (r *Runner) listen(limit time.Duration) (line string, more, expired bool) {
	if waits, ok := r.opts.Breakpoint.(vm.Impatient); ok {
		return waits.InputWithin(limit)
	}

	line, more = r.opts.Breakpoint.Input()

	return line, more, false
}

// played takes one line from whoever is at the terminal during a breakpoint,
// and watches it for the one line that is not for the adventure.
//
// back says the script has the adventure again, and the line is then nobody's:
// Resume is read by the harness and never handed on, which is the whole of
// what makes it a word of this language rather than a command of the game.
func (r *Runner) played(typed string, said bool) (line string, more, back bool) {
	// Running out of input is the end of the session and the end of the game,
	// the same as it is for a person playing without a script.
	if !said {
		return "", false, false
	}

	line = typed

	if strings.EqualFold(strings.Join(strings.Fields(line), " "), Resume) {
		r.manual = false
		r.report.Resumed()

		return "", false, true
	}

	return line, true, false
}

// AnyKey is a pause, and it waits, because the adventure said to wait.
//
// A script that has nothing to press fails here rather than being let through.
// Letting it through would be the harness deciding that this pause does not
// count, and a pause is the adventure's own behaviour: how many there are and
// where they fall is part of what a test of it is for. A walkthrough that
// stops working the day somebody adds one is a walkthrough that noticed.
//
// It cannot hang. A script with no key left where a key is wanted is a failure
// with the line it got to, and a script that has run out altogether returns —
// the test is over, and what the adventure does afterwards is nobody's
// business.
func (r *Runner) AnyKey() {
	if r.pump(atKey, 0) == outManual {
		// Theirs as well, and the one place Resume is not watched for: a pause
		// wants a keypress and this language has no way to read a line out of
		// one. They press something and go on playing; the way out is at the
		// next prompt.
		r.opts.Breakpoint.AnyKey()
	}
}

// moment is what the machine is stopped for, which decides what the orders
// that type can mean and whether the clock can be let out.
type moment int

const (
	atLine moment = iota // waiting for an order
	atKey                // waiting for a keypress
	atEnd                // the game is over
)

// outcome is what the script gave the machine.
type outcome int

const (
	outLine   outcome = iota // a line was typed and is in r.line
	outKey                   // a key was pressed
	outClock                 // let the adventure's clock run out
	outManual                // a BREAKPOINT: ask the person, not the script
	outOver                  // the script has nothing more to give
)

// pump runs the script forward until it has something for the machine.
//
// It is the whole of how a script and a machine take turns. Everything the
// language does happens in here, in the one place where what the machine is
// waiting for is known.
//
//nolint:gocognit,gocyclo,cyclop,funlen // the turn-taking is one thing and reads as one
func (r *Runner) pump(where moment, clock time.Duration) outcome {
	for {
		// Inside a breakpoint the script is not consulted at all, however many
		// turns the person takes. See Options.Breakpoint.
		if r.manual {
			return outManual
		}

		if r.stopped || r.at >= len(r.script.steps) {
			return outOver
		}

		s := &r.script.steps[r.at]

		switch s.op {
		case opStep:
			r.report.Step(s.text)

			r.next()

		case opEnd:
			// Getting here is the proof: everything in the step held.
			r.report.Pass()

			r.next()

		case opTrace:
			r.tracing = s.on

			r.next()

		case opBreak:
			if r.opts.Breakpoint == nil {
				r.report.Note(s.at, "nobody is at a terminal, so there is nobody to hand it to")
				r.next()

				continue
			}

			// The step is spent on the way in: what brings the script back is
			// the line, not the step, and coming back has to land on the one
			// after this.
			r.manual = true
			r.report.Broken(s.at)

			r.next()

			return outManual

		case opDebug:
			r.describe(s)
			r.next()

		case opAssert:
			if !r.verify(s) {
				return outOver
			}

			r.next()

		case opExpectNot:
			if _, end := indexFold(r.said, s.text); end >= 0 {
				r.fail(s, "it said that, and this is the line saying it should not have")

				return outOver
			}

			r.next()

		case opExpect:
			if r.hear(s.text) {
				r.next()

				continue
			}

			if where == atKey || where == atEnd || clock == 0 {
				r.fail(s, "it has not said that, and "+nothingMore(where))

				return outOver
			}

			// The adventure has a clock on this question, so waiting is a
			// thing that can be done, and it is done for real.
			//
			// Whoever runs out first decides. If the adventure's own timeout
			// is shorter than what is left of the wait, it fires while the
			// step is still waiting: the runtime runs its own turn, for its
			// own reasons, and what it prints may be the very thing being
			// waited for. If the step gives up first, nothing runs at all —
			// the adventure is still holding out for an order and never got
			// one.
			//
			// What is never done is to say "expired" without the time having
			// passed. That is the harness telling the runtime something untrue
			// in order to see what it would do, and it is how observing a run
			// ends up altering it.
			left := time.Duration(s.secs)*time.Second - r.waited

			if left >= clock {
				r.waited += clock
				pause(clock)

				return outClock
			}

			pause(left)
			r.fail(s, fmt.Sprintf("it did not say that in %s", plural(s.secs, "second")))

			return outOver

		case opWait:
			switch {
			case where == atKey:
				// A pause has no clock on it: nothing is going to happen until
				// a key is pressed, so there is nothing to wait for.
				r.fail(s, "nothing to wait for: "+nothingMore(where))

				return outOver

			case where == atEnd || clock == 0:
				r.report.Note(s.at, "nothing to wait for: "+nothingMore(where))
				r.next()

				continue
			}

			// The same race as an EXPECT's, and this is the order that is
			// about it: what it asks for is that the time pass, so that
			// whatever the adventure does when nobody answers, it does.
			left := time.Duration(s.secs)*time.Second - r.waited

			if left >= clock {
				r.waited += clock
				pause(clock)

				return outClock
			}

			pause(left)
			r.next()

		case opInput:
			switch where {
			case atKey, atEnd:
				r.fail(s, "there is nowhere to type: "+nothingMore(where))

				return outOver

			default:
				// Into the buffer and then over, rather than instead of it: a
				// KEY in front of an INPUT still belongs to the same line, so
				// KEY S followed by INPUT alir is one order reading "Salir".
				r.typed.WriteString(s.text)

				r.line = r.typed.String()
				r.typed.Reset()

				r.gave(s)
				r.next()

				return outLine
			}

		case opKey:
			switch where {
			case atKey:
				// A pause takes whichever key comes next, exactly as a real
				// keyboard does, and never asks which one it was.
				r.gave(s)
				r.next()

				return outKey

			case atEnd:
				r.fail(s, "the adventure had already ended, so there was nothing to press a key at")

				return outOver

			default:
				if !s.enter {
					r.typed.WriteString(s.typed)
					r.next()

					continue
				}

				r.line = r.typed.String()
				r.typed.Reset()

				r.gave(s)
				r.next()

				return outLine
			}
		}
	}
}

// gave records that the script has just handed the adventure something, so
// that whatever it says next is known to be the answer to it. See the order
// and since fields.
func (r *Runner) gave(s *step) {
	// The source line with its padding squeezed: it is a label now, and the
	// spaces that line an order up in a file are not part of what it says.
	r.order = strings.Join(strings.Fields(s.source), " ")
	r.since = ""
}

// next moves to the step after this one. The waiting belongs to the step that
// was doing it, so it goes with it.
func (r *Runner) next() {
	r.at++
	r.waited = 0
}

// pause is the waiting, and it is the real thing.
//
// Simulating it would cost nothing and would be a lie with consequences: the
// only way to tell a runtime that a timeout expired without the time having
// passed is to say so, and then the adventure runs a turn it would not have
// run and says something it would not have said. A script that waits pays for
// it in seconds, the same as a player who sits still.
func pause(d time.Duration) {
	if d > 0 {
		time.Sleep(d)
	}
}

// hear looks for what an EXPECT wants and swallows everything up to the end of
// it, so that the next EXPECT can only match later text.
func (r *Runner) hear(text string) bool {
	_, end := indexFold(r.said, text)
	if end < 0 {
		return false
	}

	r.said = r.said[end:]

	return true
}

// fail records the first failure and stops the run.
//
// Which step it belongs to is not passed along, because the report has already
// said: the name is on the line the badge is about to close.
func (r *Runner) fail(s *step, why string) {
	r.stopped = true

	r.report.Fail(s.at, s.source, why, Heard{
		Order:     r.order,
		After:     r.since,
		Unmatched: r.said,
	})
}

// nothingMore is why nothing else is going to be said, which is the difference
// between a test that is wrong and a test that is only early.
func nothingMore(where moment) string {
	switch where {
	case atKey:
		return "the adventure is waiting for a keypress, and no clock runs during a pause: " +
			"it will say nothing more until a KEY is given"
	case atEnd:
		return "the adventure has ended and will say nothing more"
	}

	// Not "no TIME set": the yes-no questions QUIT and END ask are read with
	// no clock on them whatever the adventure's TIME is, and this is the
	// sentence a script sitting at one of those needs to read.
	return "the adventure is waiting for something to be typed with no clock running on it, " +
		"so it will say nothing more until it is answered"
}

func plural(n int, thing string) string {
	if n == 1 {
		return "1 " + thing
	}

	return fmt.Sprintf("%d %ss", n, thing)
}

// describe answers a DEBUG.
func (r *Runner) describe(s *step) {
	switch s.ask {
	case askFlag:
		r.report.Debug(fmt.Sprintf("FLAG_%d=%d", s.num, r.machine.Flag(s.num)))

	case askFlags, askFlagsSet:
		const flags = 256

		for n := range flags {
			held := r.machine.Flag(n)
			if s.ask == askFlagsSet && held == 0 {
				continue
			}

			r.report.Debug(fmt.Sprintf("FLAG_%d=%d", n, held))
		}

	case askAt:
		r.report.Debug(fmt.Sprintf("AT=%d", r.machine.Flag(vm.FlagPlayer)))

	case askPicture:
		if number, asked := r.machine.Picture(); asked {
			r.report.Debug(fmt.Sprintf("PICTURE=%d", number))
		} else {
			r.report.Debug("PICTURE=none")
		}

	case askObjects:
		for n := range r.machine.Objects() {
			where, _ := r.machine.ObjectAt(n)
			r.report.Debug(fmt.Sprintf("OBJ_%d=%d", n, where))
		}

	case askSaid:
		r.report.Said(r.said)
	}
}

// verify answers an ASSERT, and says what it found rather than only that it
// was not what was wanted.
func (r *Runner) verify(s *step) bool {
	switch s.check {
	case chkEnded:
		if r.machine.Ended() {
			return true
		}

		r.fail(s, "the adventure is still going")

		return false

	case chkCarried, chkWorn, chkHere, chkPresent:
		return r.verifyWhere(s)

	case chkObject:
		where, exists := r.machine.ObjectAt(s.num)
		if !exists {
			r.fail(s, fmt.Sprintf("there is no object %d in this database", s.num))

			return false
		}

		return r.compare(s, fmt.Sprintf("OBJ_%d", s.num), int(where))

	case chkPicture:
		number, asked := r.machine.Picture()
		if !asked {
			r.fail(s, "the adventure has not asked for a picture at all")

			return false
		}

		return r.compare(s, "PICTURE", int(number))

	case chkFlag:
		return r.compare(s, fmt.Sprintf("FLAG_%d", s.num), int(r.machine.Flag(s.num)))

	case chkAt:
		return r.compare(s, "AT", int(r.machine.Flag(vm.FlagPlayer)))

	case chkScore:
		return r.compare(s, "SCORE", int(r.machine.Flag(vm.FlagScore)))

	case chkTurns:
		// Two bytes, little-endian, which is how the format counts anything
		// that can go past 255.
		turns := int(r.machine.Flag(vm.FlagTurns)) + 256*int(r.machine.Flag(vm.FlagTurns+1))

		return r.compare(s, "TURNS", turns)
	}

	return true
}

func (r *Runner) compare(s *step, name string, got int) bool {
	if s.cmp.holds(got, s.val) {
		return true
	}

	why := fmt.Sprintf("%s is %d, not %d", name, got, s.val)
	if s.cmp != cmpEq {
		why = fmt.Sprintf("%s is %d, and %s %s %d was wanted", name, got, name, s.cmp, s.val)
	}

	r.fail(s, why)

	return false
}

// verifyWhere checks where an object is, in the four words a player would use
// for it.
func (r *Runner) verifyWhere(s *step) bool {
	where, exists := r.machine.ObjectAt(s.num)
	if !exists {
		r.fail(s, fmt.Sprintf("there is no object %d in this database", s.num))

		return false
	}

	player := r.machine.Flag(vm.FlagPlayer)

	var (
		held  bool
		meant string
	)

	switch s.check {
	case chkCarried:
		held, meant = where == vm.LocCarried, "carried"
	case chkWorn:
		held, meant = where == vm.LocWorn, "worn"
	case chkHere:
		held, meant = where == player, "here"
	default:
		held, meant = where == vm.LocCarried || where == vm.LocWorn || where == player, "present"
	}

	if held {
		return true
	}

	r.fail(s, fmt.Sprintf("object %d is not %s: it is at %s", s.num, meant, located(where)))

	return false
}

// located reads a location out, because three of the numbers are not places.
func located(where byte) string {
	switch where {
	case vm.LocCarried:
		return "254, carried"
	case vm.LocWorn:
		return "253, worn"
	case vm.LocNotCreated:
		return "252, nowhere yet"
	default:
		return fmt.Sprint(where)
	}
}

// The watcher, which is the condact trace and nothing else. It is gated here
// rather than by swapping watchers, because the machine is given one for the
// whole run and TRACE wants to change its mind halfway through.

func (r *Runner) traced() vm.Watcher {
	if r.tracing && r.opts.Trace != nil {
		return r.opts.Trace
	}

	return vm.NoWatcher{}
}

func (r *Runner) Condact(process, entry int, c ddbfile.Condact) {
	r.traced().Condact(process, entry, c)
}

func (r *Runner) Flag(number int, was, now byte)   { r.traced().Flag(number, was, now) }
func (r *Runner) Process(number int, running bool) { r.traced().Process(number, running) }
func (r *Runner) Moved(object int, from, to byte)  { r.traced().Moved(object, from, to) }
