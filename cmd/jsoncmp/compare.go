package main

import (
	"fmt"
	"sort"
	"strings"
)

// Report is what the comparison found, grouped by section so that a failure
// says where to look rather than just that something is wrong.
type Report struct {
	sections []sectionReport
}

type sectionReport struct {
	name     string
	problems []string
}

func (r *Report) Empty() bool { return len(r.sections) == 0 }

func (r *Report) String() string {
	var b strings.Builder

	for _, s := range r.sections {
		fmt.Fprintf(&b, "%s: %d difference(s)\n", s.name, len(s.problems))

		// A section that has gone badly wrong produces hundreds of lines and
		// none of them after the first few help. The count above is the whole
		// truth; this is the part a person reads.
		shown := s.problems
		if len(shown) > 10 {
			shown = shown[:10]
		}

		for _, p := range shown {
			fmt.Fprintf(&b, "  %s\n", p)
		}

		if len(s.problems) > len(shown) {
			fmt.Fprintf(&b, "  ... and %d more\n", len(s.problems)-len(shown))
		}
	}

	return b.String()
}

func (r *Report) add(name string, problems []string) {
	if len(problems) == 0 {
		return
	}

	r.sections = append(r.sections, sectionReport{name: name, problems: problems})
}

// Compare holds the two documents against each other.
//
// Two sections are left out on purpose. The symbols are not compared because
// they are an input rather than an output — the two are told the same target
// and both seed from it, so a difference there is a fault in the seeding and
// the target-symbol tests already cover that. The settings are not compared
// because half of them come from flags this script does not pass.
//
// xmessages and other_strings are skipped while neither side is asked to fill
// them: comparing two empty lists proves nothing and a false pass is worse
// than no check.
func Compare(theirs, ours *document) *Report {
	var r Report

	r.add("vocabulary", compareVocabulary(theirs.Vocabulary, ours.Vocabulary))
	r.add("sysmess", compareTexts(theirs.Sysmess, ours.Sysmess))
	r.add("messages", compareTexts(theirs.Messages, ours.Messages))
	r.add("locations", compareTexts(theirs.Locations, ours.Locations))
	r.add("objects", compareTexts(theirs.Objects, ours.Objects))
	r.add("object_data", compareObjectData(theirs.ObjectData, ours.ObjectData))
	r.add("connections", compareConnections(theirs.Connections, ours.Connections))
	r.add("externs", compareExterns(theirs.Externs, ours.Externs))
	r.add("processes", compareProcesses(theirs, ours))

	return &r
}

// compareVocabulary keys on the word and its kind rather than on its position.
//
// The two write the list in different orders and neither order means anything:
// what matters is that a given word has the same value and the same kind on
// both sides. A word appearing twice with different kinds is legal — a noun
// and a verb can share a spelling — which is why the key is the pair.
// Both sides are uppercased before the key is built, which settles the one
// thing they spell differently. DAAD's low character set has no uppercase Á,
// so an accented capital and its lowercase self are the same code point;
// theirs writes the lowercase one and ours keeps the letter as the author
// typed it, because we hold that the folding belongs to the emitter. Upper
// case is where the two meet.
func compareVocabulary(theirs, ours []vocWord) []string {
	index := func(words []vocWord) map[string]int {
		out := make(map[string]int, len(words))
		for _, w := range words {
			out[fmt.Sprintf("%s/%d", strings.ToUpper(w.VocWord), w.VocType)] = w.Value
		}

		return out
	}

	return compareMaps("word", index(theirs), index(ours),
		func(v int) string { return fmt.Sprintf("value %d", v) })
}

// compareTexts holds two text tables against each other by number.
//
// The number is the key here and not the text, because a text table is indexed
// by position in the binary: message 7 has to be message 7 on both sides or
// every condact that shows it shows the wrong thing.
// Their side arrives with the character set already resolved and ours does
// not, so ours is put through the same conversion first. See charset.go.
func compareTexts(theirs, ours []text) []string {
	index := func(texts []text, encode bool) map[string]string {
		out := make(map[string]string, len(texts))

		for _, t := range texts {
			body := t.Text
			if encode {
				body = daadEncode(body)
			}

			out[fmt.Sprint(t.Value)] = strings.TrimRight(body, " \t")
		}

		return out
	}

	return compareMaps("entry", index(theirs, false), index(ours, true),
		func(s string) string { return fmt.Sprintf("%q", s) })
}

func compareObjectData(theirs, ours []objectData) []string {
	index := func(objects []objectData) map[string]objectData {
		out := make(map[string]objectData, len(objects))
		for _, o := range objects {
			out[fmt.Sprint(o.Value)] = o
		}

		return out
	}

	return compareMaps("object", index(theirs), index(ours),
		func(o objectData) string { return o.String() })
}

// compareConnections keys on where the exit starts and which word takes it,
// because that pair is what the interpreter looks an exit up by. Two exits
// from the same place through the same word cannot both exist.
func compareConnections(theirs, ours []connection) []string {
	index := func(conns []connection) map[string]int {
		out := make(map[string]int, len(conns))
		for _, c := range conns {
			out[fmt.Sprintf("from %d by %d", c.FromLoc, c.Direction)] = c.ToLoc
		}

		return out
	}

	return compareMaps("exit", index(theirs), index(ours),
		func(to int) string { return fmt.Sprintf("goes to %d", to) })
}

func compareExterns(theirs, ours []extern) []string {
	index := func(list []extern) map[string]string {
		out := make(map[string]string, len(list))
		for _, e := range list {
			out[e.FilePath] = e.FilePath
		}

		return out
	}

	return compareMaps("extern", index(theirs), index(ours),
		func(s string) string { return s })
}

// compareProcesses walks the program, which is where a front end most easily
// goes wrong and least visibly.
//
// Processes are keyed by number and entries by position within their process:
// entries are tried in order and the order is the meaning, so an entry that
// moved is a difference even when the same entries are all present.
func compareProcesses(theirs, ours *document) []string {
	var problems []string

	theirsByID := indexProcesses(theirs.Processes)
	oursByID := indexProcesses(ours.Processes)

	for _, id := range sortedKeys(theirsByID) {
		mine, present := oursByID[id]
		if !present {
			problems = append(problems, fmt.Sprintf("process %s is missing from ours", id))

			continue
		}

		problems = append(problems, compareEntries(id, theirsByID[id], mine)...)
	}

	for _, id := range sortedKeys(oursByID) {
		if _, present := theirsByID[id]; !present {
			problems = append(problems, fmt.Sprintf("process %s is only in ours", id))
		}
	}

	return problems
}

func indexProcesses(processes []process) map[string]process {
	out := make(map[string]process, len(processes))
	for _, p := range processes {
		out[fmt.Sprintf("%03d", p.Value)] = p
	}

	return out
}

func compareEntries(id string, theirs, ours process) []string {
	var problems []string

	if len(theirs.Entries) != len(ours.Entries) {
		return []string{fmt.Sprintf("process %s has %d entries and ours has %d",
			id, len(theirs.Entries), len(ours.Entries))}
	}

	for i := range theirs.Entries {
		a, b := theirs.Entries[i], ours.Entries[i]

		where := fmt.Sprintf("process %s entry %d", id, i)

		if a.Verb != b.Verb || a.Noun != b.Noun {
			problems = append(problems, fmt.Sprintf("%s is %d %d and ours is %d %d",
				where, a.Verb, a.Noun, b.Verb, b.Noun))

			continue
		}

		problems = append(problems, compareCondacts(where, a, b)...)
	}

	return problems
}

func compareCondacts(where string, theirs, ours entry) []string {
	mine := make([]condact, 0, len(ours.Condacts))
	for _, raw := range ours.Condacts {
		mine = append(mine, readCondact(raw))
	}

	yours := make([]condact, 0, len(theirs.Condacts))
	for _, raw := range theirs.Condacts {
		yours = append(yours, readCondact(raw))
	}

	if len(yours) != len(mine) {
		return []string{fmt.Sprintf("%s has %d condacts and ours has %d",
			where, len(yours), len(mine))}
	}

	var problems []string

	for i := range yours {
		if yours[i].String() != mine[i].String() {
			problems = append(problems, fmt.Sprintf("%s condact %d is %q and ours is %q",
				where, i, yours[i].String(), mine[i].String()))
		}
	}

	return problems
}

// compareMaps is the shape every section boils down to: two tables keyed the
// same way, and a report of what is in one and not the other or differs.
func compareMaps[V comparable](what string, theirs, ours map[string]V, show func(V) string) []string {
	var problems []string

	for _, key := range sortedKeys(theirs) {
		mine, present := ours[key]
		if !present {
			problems = append(problems, fmt.Sprintf("%s %s is missing from ours", what, key))

			continue
		}

		if mine != theirs[key] {
			problems = append(problems, fmt.Sprintf("%s %s is %s and ours is %s",
				what, key, show(theirs[key]), show(mine)))
		}
	}

	for _, key := range sortedKeys(ours) {
		if _, present := theirs[key]; !present {
			problems = append(problems, fmt.Sprintf("%s %s is only in ours", what, key))
		}
	}

	sort.Strings(problems)

	return problems
}
