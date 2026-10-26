package vm

import "github.com/jorgefuertes/QDAAD/internal/ddbfile"

// The process machine, which is processPROC of daad_condacts.c:175.
//
// A process is a list of entries; an entry is a verb, a noun and a block of
// condacts. Running one means: execute the current entry's condacts until they
// run out or a condition fails, then walk forward to the next entry whose verb
// and noun match the sentence, and do it again. When the entries run out the
// process returns to whoever called it.
//
// Two things make that less simple than it sounds, and both are in here rather
// than hidden in a condact:
//
//   - An entry stops at the first condition that fails, but the process goes
//     on to the next entry. That is what "continuing" tracks, and it is the
//     difference between a failed condition and a finished game.
//   - A DOALL turns the end of the table into a loop. Running off the end with
//     one active does not return: it goes back to the condact after the DOALL
//     with the next object, and only returns when the objects run out.
func (m *Machine) runStack() error {
	m.done, m.lastDone = false, m.done

	for len(m.stack) > 0 && !m.finished && !m.restart {
		f := m.top()
		entries := m.entriesOf(f.process)

		if !f.running {
			m.findEntry(entries)

			continue
		}

		if f.entry < 0 || f.entry >= len(entries) {
			f.running = false

			continue
		}

		m.continuing = true

		for m.continuing && f.condact < len(entries[f.entry].Condacts) {
			c := entries[f.entry].Condacts[f.condact]
			f.condact++

			m.watch.Condact(f.process, f.entry, c)

			if err := m.exec(c); err != nil {
				return err
			}

			// A condact that pushed a process, jumped, or ended the game has
			// moved the ground under this loop, so everything is read again.
			if m.finished || m.restart || !m.holds(f) || !f.running {
				break
			}
		}

		// This entry is done with, one way or another. The next turn of the
		// loop looks for another that matches.
		if !m.finished && !m.restart && m.holds(f) {
			f.running = false
		}
	}

	return nil
}

// findEntry steps forward to the next entry that answers the sentence, which
// is the do-while at the bottom of processPROC. Running out of entries either
// goes round a DOALL again or returns from the process.
func (m *Machine) findEntry(entries []ddbfile.Entry) {
	f := m.top()

	for {
		f.entry++
		f.condact = 0

		if f.entry >= len(entries) {
			m.loopOrPop(entries)

			return
		}

		if m.matches(entries[f.entry]) {
			f.running = true

			return
		}
	}
}

// entriesOf is the entries of a process, or nothing when the number names no
// process. A PROCESS into thin air returns rather than crashing, which is what
// an interpreter of the period did by accident and what this does on purpose.
func (m *Machine) entriesOf(process int) []ddbfile.Entry {
	if process < 0 || process >= len(m.processes) {
		return nil
	}

	return m.processes[process]
}

// matches is whether an entry answers the sentence in the flags. The null word
// in either slot matches anything, which is what "> _ _" means in a source.
func (m *Machine) matches(e ddbfile.Entry) bool {
	return (e.Verb == NoWord || e.Verb == m.flags[FlagVerb]) &&
		(e.Noun == NoWord || e.Noun == m.flags[FlagNoun1])
}

// loopOrPop is what happens at the end of a process's entries: either a DOALL
// goes round again, or the process returns.
//
// It reports whether the process returned, which the caller needs because the
// two outcomes leave the stack in different shapes.
func (m *Machine) loopOrPop(entries []ddbfile.Entry) bool {
	f := m.top()

	if f.doallCondact >= 0 && m.doallNext() {
		return false
	}

	m.pop()

	return true
}
