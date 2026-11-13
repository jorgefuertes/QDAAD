package ddbfile

// The two condacts that load and show an illustration.
const (
	opDISPLAY = 28
	opPICTURE = 84
)

// withoutPictures drops the condacts that load and show an illustration.
//
// Nothing to do with any version: version 3 draws pictures perfectly well. It
// is for a build that has none, and then it matters — a PICTURE that finds no
// image still leaves the graphics window where it put it, and the text comes
// out with the edges of it showing.
func withoutPictures(all [][]Entry) [][]Entry {
	out := make([][]Entry, len(all))

	for number, entries := range all {
		kept := make([]Entry, 0, len(entries))

		for _, e := range entries {
			condacts := make([]Condact, 0, len(e.Condacts))

			for _, c := range e.Condacts {
				if c.Opcode == opPICTURE || c.Opcode == opDISPLAY {
					continue
				}

				condacts = append(condacts, c)
			}

			kept = append(kept, Entry{Verb: e.Verb, Noun: e.Noun, Condacts: condacts})
		}

		out[number] = kept
	}

	return out
}
