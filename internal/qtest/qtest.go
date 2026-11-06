// Package qtest runs a test script against an adventure.
//
// A walkthrough is not code. It is a list of what to type and what should come
// back, written while playing, and the place for it is a file somebody can edit
// with the game open in the other window — not a Go test that has to be
// compiled to be read. That file is a .qtest, and this is what runs it.
//
// The shape is expect's: say what should appear, type, say what should appear
// next. What the adventure has said and no EXPECT has matched yet is a buffer,
// and matching consumes it up to the end of the match, so the order things are
// said in is part of what is being tested.
//
// One order is one command: INPUT types its words and presses return in the
// same breath. KEY is what types without sending, for the one place that wants
// it — a yes-or-no question answered with a single letter.
//
// # The clock
//
// An adventure that has set TIME gives the player a few seconds to answer, and
// when they run out the tables run on without an order — "El tiempo pasa... ¡Y
// no vuelve!", or the dwarf walking into the room. That is half of what an
// adventure does and it has to be testable, so the seconds in this language
// are seconds, and a step that waits waits.
//
// Whoever runs out first decides. An EXPECT given five seconds against an
// adventure that allows forty-seven gives up first: the step fails, and
// because the adventure's own timeout never fired, the game is exactly where
// it was. Given sixty it is the other way round: the timeout fires while the
// step is still waiting, the runtime runs its own turn for its own reasons,
// and what it prints may be the very thing being waited for.
//
// What is never done is to report a timeout that did not happen. It would cost
// nothing and it would be a lie with consequences — the adventure would run a
// turn it would never have run and say something a player would never have
// seen, which is observing a run by altering it.
//
// When the adventure has no clock running on the question, there is nothing to
// wait for at all: it is holding out for an order and will say nothing more
// until it gets one. An EXPECT then fails at once and says so, rather than
// spending real seconds to learn nothing.
//
// Nothing here changes what the adventure does. The machine is not told that
// it is being tested and has no way of finding out: what the script hands it
// is a line, a keypress or a player who really did say nothing for that long,
// which is all a person hands it too. Where the clock runs out, the runtime is
// the one running the turn — it set the TIME and its own timeout expired — and
// the console only reported that nobody answered. Where the adventure pauses,
// the script has to press something, because deciding that a pause does not
// count would be the harness changing the game.
//
// # How a script and a machine take turns
//
// The machine is in charge and the script is what it pumps. There are no
// goroutines and no channels here: every step runs inside one of the three
// calls the machine makes of a console — Input, InputWithin, AnyKey — plus one
// last pass once the game is over. That is what makes a run repeatable, and it
// is the same arrangement internal/vm's own play tests use.
//
// Only one step can be left halfway, and it is the one that matters: an EXPECT
// spending the adventure's clock returns to the machine, keeps its place, and
// carries on from the same step the next time it is asked for a line.
package qtest
