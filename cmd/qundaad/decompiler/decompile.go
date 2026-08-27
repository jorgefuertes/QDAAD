package decompiler

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The source is split one section per file, joined by #INCLUDE from a main
// file. That is how DAAD sources were organized in the first place: the 1991
// manual warns about "including a TOK file which contains an extra /TOK",
// which only makes sense if splitting was the normal practice.
const (
	mainFile        = "game.sce"
	tokensFile      = "tokens.sce"
	vocabFile       = "vocabulary.sce"
	sysMessFile     = "sysmess.sce"
	messagesFile    = "messages.sce"
	objTextFile     = "object-text.sce"
	locTextFile     = "location-text.sce"
	connectionsFile = "connections.sce"
	objectsFile     = "objects.sce"
)

// Word types, in the spelling the compiler expects. Note it is CONJUGATION,
// not conjunction (1991 manual, section 5.1).
var wordKinds = [...]string{
	"VERB", "ADVERB", "NOUN", "ADJECTIVE", "PREPOSITION", "CONJUGATION", "PRONOUN",
}

var machineNames = map[byte]string{
	0: "IBM PC", 1: "ZX Spectrum", 2: "Commodore 64", 3: "Amstrad CPC",
	4: "MSX", 5: "Atari ST", 6: "Commodore Amiga", 7: "Amstrad PCW",
}

var languageNames = map[byte]string{0: "English", 1: "Spanish"}

// Decompile reads a DDB and writes the source files into outputDir.
func Decompile(inputFile, outputDir string) error {
	fmt.Printf("Decompiling %q to %q\n", inputFile, outputDir)
	startTime := time.Now()

	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inputFile, err)
	}

	reader, err := NewReader(data)
	if err != nil {
		return fmt.Errorf("%s: %w", inputFile, err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", outputDir, err)
	}

	// The order is the one the compiler expects, asnd it is not arbitrary: the
	// tokens have to be known before any text refers to them.
	sections := []struct {
		name  string
		write func(*Reader) (string, error)
	}{
		{tokensFile, writeTokens},
		{vocabFile, writeVocabulary},
		{sysMessFile, textSection("/STX", "System messages", ptrSysMessageText, (*Reader).NumSysMessages)},
		{messagesFile, textSection("/MTX", "User messages", ptrMessageText, (*Reader).NumMessages)},
		{objTextFile, textSection("/OTX", "Object descriptions", ptrObjectText, (*Reader).NumObjects)},
		{locTextFile, textSection("/LTX", "Location descriptions", ptrLocationText, (*Reader).NumLocations)},
		{connectionsFile, writeConnections},
		{objectsFile, writeObjects},
	}

	names := make([]string, 0, len(sections))

	for _, s := range sections {
		body, err := s.write(reader)
		if err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}

		if err := os.WriteFile(filepath.Join(outputDir, s.name), []byte(body), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", s.name, err)
		}

		names = append(names, s.name)
	}

	main := writeMain(reader, inputFile, names)
	if err := os.WriteFile(filepath.Join(outputDir, mainFile), []byte(main), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", mainFile, err)
	}

	report(reader, startTime)

	return nil
}

// banner describes where the source came from, so a reader of the output knows
// what was deduced rather than read.
func banner(r *Reader, inputFile string) string {
	var sb strings.Builder

	line := strings.Repeat("-", 74)
	add := func(f string, a ...any) { fmt.Fprintf(&sb, "; "+f+"\n", a...) }

	add("%s", line)
	add("Decompiled by QUnDAAD from %s", filepath.Base(inputFile))
	add("Queru's DAAD decompiler v%s", VERSION)
	add("%s", line)
	add("DAAD version   : %d", r.Version())
	add("Machine        : %s (%d)", nameOr(machineNames, r.Machine()), r.Machine())
	add("Language       : %s (%d)", nameOr(languageNames, r.Language()), r.Language())
	add("Null word      : %c", r.NullWord())
	// The database holds DAAD codes and the sources of the time were Latin-1.
	// This output is UTF-8 so it can be read and edited on a modern machine;
	// converting back is the emitter's job.
	add("Encoding       : UTF-8   (the original sources were ISO-8859-1)")
	add("Header size    : %d bytes   (deduced, not declared)", r.HeaderSize())
	add("Endianness     : %s   (deduced, not declared)", endianName(r.BigEndian()))
	add("Declared length: %d bytes", r.DeclaredLength())
	add("%s", line)
	add("Objects        : %d", r.NumObjects())
	add("Locations      : %d", r.NumLocations())
	add("User messages  : %d", r.NumMessages())
	add("System messages: %d", r.NumSysMessages())
	add("Processes      : %d", r.NumProcesses())
	add("%s", line)

	return sb.String()
}

func writeMain(r *Reader, inputFile string, includes []string) string {
	var sb strings.Builder

	sb.WriteString(banner(r, inputFile))
	sb.WriteString("\n/CTL\n")
	sb.WriteString(string(rune(r.NullWord())))
	sb.WriteString("\n\n")

	for _, name := range includes {
		fmt.Fprintf(&sb, "#INCLUDE %s\n", name)
	}

	return sb.String()
}

func writeTokens(r *Reader) (string, error) {
	var sb strings.Builder

	sb.WriteString("; ---------------------------------------------------------------------\n")
	sb.WriteString("; GENERATED FILE - DO NOT EDIT BY HAND\n")
	sb.WriteString("; ---------------------------------------------------------------------\n")
	sb.WriteString("; The compression table. Nobody writes one of these: the 1991 compiler\n")
	sb.WriteString("; is single pass, so it cannot pick the tokens itself -- that would take\n")
	sb.WriteString("; reading every text first to count what repeats. Authors included a\n")
	sb.WriteString("; table shipped with DAAD, one per language, or produced their own with\n")
	sb.WriteString("; the FINDTOK utility from a scan file the compiler wrote with -t.\n")
	sb.WriteString(";\n")
	sb.WriteString("; It is reproduced here byte for byte so that recompiling yields the\n")
	sb.WriteString("; same database: a different table compresses the texts differently.\n")
	sb.WriteString("; Editing it changes every text that refers to it.\n")
	sb.WriteString("; ---------------------------------------------------------------------\n")
	sb.WriteString("; Every text may refer to these, so the section comes before any text.\n")
	sb.WriteString("; Spaces are written as the null word character.\n")
	sb.WriteString("; The first entry is a placeholder no text refers to: every interpreter\n")
	sb.WriteString("; skips it, but it has to be there and be one byte long.\n\n")
	sb.WriteString("/TOK\n")
	sb.WriteString(tokenSource(r.Placeholder(), r.NullWord()))
	sb.WriteString("\n")

	for _, t := range r.Tokens() {
		sb.WriteString(tokenSource(t, r.NullWord()))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// Value ranges the interpreter treats specially. The thresholds are the ones
// the 1991 manual gives (section 5.1, notes f and g): the modern compiler moved
// the convertible one up to 40.
const (
	lastMovementValue    = 13
	lastConvertibleValue = 19
)

func writeVocabulary(r *Reader) (string, error) {
	entries := r.Vocabulary()

	// The database keeps the vocabulary alphabetically, which is an artefact of
	// the tree the compiler builds, not a requirement: it sorts on its own when
	// emitting, so any order recompiles to the same bytes. Sorting by value puts
	// synonyms together, which is what a reader of this file is after.
	sorted := slices.Clone(entries)
	slices.SortStableFunc(sorted, func(a, b VocabularyEntry) int {
		if a.Value != b.Value {
			return int(a.Value) - int(b.Value)
		}

		return int(a.Kind) - int(b.Kind)
	})

	var sb strings.Builder

	sb.WriteString("; Vocabulary, ordered by word value. Words that share a value AND a\n")
	sb.WriteString("; type are synonyms: they are grouped together below.\n")
	sb.WriteString(";\n")
	sb.WriteString("; Only the first five characters of a word count, and a value may be\n")
	sb.WriteString("; reused across types -- the same number can be a verb and a noun.\n\n")
	sb.WriteString("/VOC\n")

	var previous *VocabularyEntry

	for i := range sorted {
		e := sorted[i]

		if note := rangeNote(previous, e); note != "" {
			sb.WriteString(note)
		} else if previous != nil && (previous.Value != e.Value || previous.Kind != e.Kind) {
			sb.WriteString("\n")
		}

		kind := "?"
		if int(e.Kind) < len(wordKinds) {
			kind = wordKinds[e.Kind]
		}

		fmt.Fprintf(&sb, "%-8s %-4d %s\n", e.Word, e.Value, kind)

		previous = &sorted[i]
	}

	return sb.String(), nil
}

// Word types, as the database numbers them.
const (
	kindVerb      = 0
	kindNoun      = 2
	kindAdjective = 3
)

func writeConnections(r *Reader) (string, error) {
	all, err := r.Connections()
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("; Connections: where each movement word takes the player from each\n")
	sb.WriteString("; location. Every location has an entry even when it has no exits.\n")
	sb.WriteString(";\n")
	sb.WriteString("; The word has to be a verb, or a noun the parser can turn into one,\n")
	sb.WriteString("; and only the last verb typed causes movement.\n\n")
	sb.WriteString("/CON\n")

	for location, moves := range all {
		fmt.Fprintf(&sb, "/%d\n", location)

		for _, m := range moves {
			// A movement is "a Verb (or Noun < 20)", and this adventure declares
			// its directions as nouns, so both types are worth trying.
			word, found := r.WordFor(m.Word, kindVerb, kindNoun)
			if !found {
				return "", fmt.Errorf("location %d moves on word value %d, "+
					"which is in no vocabulary entry", location, m.Word)
			}

			fmt.Fprintf(&sb, "%-8s %d\n", word, m.To)
		}
	}

	return sb.String(), nil
}

// Initial locations that are not locations at all.
const (
	locNotCreated = 252
	locWorn       = 253
	locCarried    = 254
)

func writeObjects(r *Reader) (string, error) {
	objects, err := r.Objects()
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("; Objects: where each one starts, what it weighs, whether it is a\n")
	sb.WriteString("; container or can be worn, and the words that name it.\n")
	sb.WriteString(";\n")
	sb.WriteString("; Object 0 is the light source, by convention of the interpreter.\n")

	if r.HasObjectAttributes() {
		sb.WriteString(";\n")
		sb.WriteString("; WARNING: this database carries the extra attribute bits that version\n")
		sb.WriteString("; 2 added, and the 1991 source syntax has no column for them. They are\n")
		sb.WriteString("; NOT written below and would be lost on a recompile.\n")
	}

	sb.WriteString("\n/OBJ\n")
	sb.WriteString(";obj  starts.at    weight  cont  worn  noun      adjective\n")

	null := string(rune(r.NullWord()))

	flag := func(on bool) string {
		if on {
			return "Y"
		}

		return null
	}

	for i, o := range objects {
		noun, err := r.objectWord(o.Noun, kindNoun, i, "noun")
		if err != nil {
			return "", err
		}

		adjective, err := r.objectWord(o.Adjective, kindAdjective, i, "adjective")
		if err != nil {
			return "", err
		}

		fmt.Fprintf(&sb, "/%-4d %-12s %-7d %-5s %-5s %-9s %s\n",
			i, initialLocation(o.InitialLocation, null), o.Weight,
			flag(o.Container), flag(o.Wearable), noun, adjective)
	}

	return sb.String(), nil
}

// initialLocation names the three values that are not a location number. HERE
// is not among them: an object cannot start where the player happens to be.
func initialLocation(value byte, null string) string {
	switch value {
	case locNotCreated:
		return null
	case locWorn:
		return "WORN"
	case locCarried:
		return "CARRIED"
	}

	return fmt.Sprint(value)
}

// objectWord names the noun or the adjective of an object, or the null word
// when it has none.
func (r *Reader) objectWord(value, kind byte, object int, field string) (string, error) {
	if value == NoWord {
		return string(rune(r.NullWord())), nil
	}

	word, found := r.WordFor(value, kind)
	if !found {
		return "", fmt.Errorf("object %d has %s value %d, which is in no vocabulary entry",
			object, field, value)
	}

	return word, nil
}

// rangeNote heads the value ranges the interpreter gives a meaning of its own,
// so that a reader can see where they begin without knowing the conventions.
// It returns the empty string when the entry does not open a range.
func rangeNote(previous *VocabularyEntry, e VocabularyEntry) string {
	crosses := func(limit byte) bool {
		return (previous == nil || previous.Value <= limit) && e.Value > limit
	}

	switch {
	case previous == nil && e.Value <= lastMovementValue:
		return "; Movement words: value under " + fmt.Sprint(lastMovementValue+1) +
			". These are the ones /CON uses.\n"
	case crosses(lastMovementValue):
		return "\n; Nouns under " + fmt.Sprint(lastConvertibleValue+1) +
			" can act as verbs when the player types none.\n"
	case crosses(lastConvertibleValue):
		return "\n; Ordinary words from here on.\n"
	}

	return ""
}

// textSection builds the writer for one of the four text tables. They all share
// the same shape: a "/n" line per entry followed by its text.
func textSection(tag, title string, pointer int, count func(*Reader) int) func(*Reader) (string, error) {
	return func(r *Reader) (string, error) {
		texts, err := r.TextTable(pointer, count(r))
		if err != nil {
			return "", err
		}

		var sb strings.Builder

		fmt.Fprintf(&sb, "; %s. Numbering must stay consecutive from zero.\n", title)
		sb.WriteString("; A line break inside a text is written as #n, so that no entry\n")
		sb.WriteString("; can be mistaken for the start of the next one.\n\n")
		sb.WriteString(tag)
		sb.WriteString("\n")

		for i, text := range texts {
			fmt.Fprintf(&sb, "/%d\n", i)

			for _, line := range sourceLines(text) {
				// A line starting with "/" would read as the next entry.
				// Emitting one silently would produce a source that compiles
				// into something else.
				if strings.HasPrefix(line, "/") {
					return "", fmt.Errorf("%s entry %d has a line starting with '/', "+
						"which cannot be written as source", tag, i)
				}

				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}

		return sb.String(), nil
	}
}

func report(r *Reader, startTime time.Time) {
	entries := r.Vocabulary()

	fmt.Printf("  DAAD v%d, %s, %s\n",
		r.Version(), nameOr(machineNames, r.Machine()), nameOr(languageNames, r.Language()))
	fmt.Printf("  %s, %d-byte header (deduced)\n", endianName(r.BigEndian()), r.HeaderSize())
	fmt.Printf("  %d tokens, %d vocabulary entries\n", len(r.Tokens()), len(entries))
	fmt.Printf("  %d objects, %d locations\n", r.NumObjects(), r.NumLocations())
	fmt.Printf("  %d user messages, %d system messages\n", r.NumMessages(), r.NumSysMessages())
	fmt.Printf("  %d processes\n", r.NumProcesses())
	fmt.Printf("  %d bytes total, %d bytes declared\n", len(r.data), r.DeclaredLength())
	fmt.Printf("  Decompilation took %s\n", time.Since(startTime))
}

func nameOr(m map[byte]string, k byte) string {
	if v, ok := m[k]; ok {
		return v
	}

	return "unknown"
}

func endianName(big bool) string {
	if big {
		return "big-endian"
	}

	return "little-endian"
}
