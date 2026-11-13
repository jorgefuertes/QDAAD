package vm

// What the machine will tell about itself while it runs.
//
// A Watcher is told what changes, which is what a trace is made of. These are
// the other half: what something is right now, asked for rather than reported.
// A debugger stopped at a prompt wants this, and so does a test asserting that
// the player ended up in the room the walkthrough says — a question nothing in
// the transcript answers, because an adventure prints the room's description
// and never its number.
//
// They are all reads. They are called from inside a Console call, which is the
// machine's own goroutine with the machine halfway through a turn, so none of
// them may touch anything: observing a run that changes the run is not
// observing the run. That is the same rule Watcher is held to.

// Flag gives what a flag holds. Numbers outside a byte wrap, as everywhere
// else in the format.
func (m *Machine) Flag(number int) byte { return m.flags[number&0xFF] }

// Picture gives the last illustration the adventure asked for, and whether it
// ever asked.
//
// Nothing here draws it. What PICTURE says, though, is which illustration goes
// with what is being described, and that is a thing an adventure gets wrong in
// a way no amount of reading the text will show.
func (m *Machine) Picture() (number byte, asked bool) { return m.picture, m.pictured }

// ObjectAt gives where an object is, and whether there is such an object. The
// four markers that stand where a location would — LocCarried and its
// neighbours — come back as they are.
func (m *Machine) ObjectAt(object int) (where byte, exists bool) {
	if object < 0 || object >= len(m.objects) {
		return 0, false
	}

	return m.objects[object].location, true
}

// Objects is how many the database has, so that whoever asks about all of them
// knows where to stop.
func (m *Machine) Objects() int { return len(m.objects) }

// Ended says whether the game is over, which is QUIT or the input running out
// rather than any process returning. See Run.
func (m *Machine) Ended() bool { return m.finished }

// SetSeed fixes the dice, and is the one thing here that changes anything.
//
// CHANCE and RANDOM are what make an adventure feel alive and what make a
// walkthrough of one impossible to trust: the same orders in the same rooms do
// not give the same game twice, so a test over an adventure with dice in it
// fails on its own schedule and every failure has to be re-run to find out
// whether it was real. A seed is what turns that back into a test.
//
// It is not undone by a RESTART. The seed belongs to whoever asked for it, and
// the adventure asking to begin again is not that.
func (m *Machine) SetSeed(seed int64) { m.random = newDice(seed) }

// SetXMessages hands over the .XMB, the file an adventure keeps the texts in
// that would not fit in its database.
//
// It is given rather than found because a database does not say where it is,
// or even that there is one: the condact carries an offset into a file whose
// name is the interpreter's own business. On a +3 the interpreter opens
// "0.XMB"; here it is whoever started the machine who knows, which is qrun.
//
// Nothing breaks without it until an XMES runs, and then it says so rather
// than printing nothing: an adventure silently missing half its text is worse
// than one that stops.
func (m *Machine) SetXMessages(xmb []byte) { m.texts.xmessages = xmb }
