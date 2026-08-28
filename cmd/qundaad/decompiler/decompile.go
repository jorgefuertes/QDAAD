package decompiler

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/jorgefuertes/QDAAD/internal/media"
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
	processesFile   = "processes.sce"
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

// database is one database to decompile, with the name to file it under and a
// label naming where it came from.
type database struct {
	dir    string // empty when the input was a database on its own
	source string
	reader *Reader
}

// blob is somewhere a database might be hiding, and what to call that place.
type blob struct {
	where string
	data  []byte
}

// Decompile writes the source of a database into outputDir.
//
// The input can be the database itself or a disk image holding several. Every
// machine but the Commodore shipped its databases as ordinary files on the
// disk, so an image is opened, walked, and each database inside it decompiled
// into a directory of its own.
func Decompile(inputFile, outputDir string) error {
	databases, err := databasesIn(inputFile)
	if err != nil {
		return err
	}

	for _, db := range databases {
		if err := decompileOne(db, filepath.Join(outputDir, db.dir)); err != nil {
			return err
		}
	}

	return nil
}

// databasesIn works out what there is to decompile behind a path.
func databasesIn(path string) ([]database, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	// A database on its own is the common case, and it reads as one straight
	// away. Only if it does not is the file worth trying as an image.
	if r, ok := readerAt(data, 0); ok {
		return []database{{source: filepath.Base(path), reader: r}}, nil
	}

	volume, err := media.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s is neither a database nor an image we can read: %w", path, err)
	}

	image := filepath.Base(path)

	// A disk formatted for its own loader has no filesystem and returns an
	// error here. That is not fatal: the sectors are still readable, and the
	// search below is what such a disk needs anyway.
	files, filesErr := volume.Files()

	// Where a database was shipped as a file, the name it was given is the one
	// to file its source under.
	var found []database

	for _, f := range files {
		name := filepath.Base(f.Name)

		extension := filepath.Ext(name)
		if !strings.EqualFold(extension, ".ddb") {
			continue
		}

		r, ok := readerAt(f.Data, 0)
		if !ok {
			return nil, fmt.Errorf("%s in %s does not read as a database", f.Name, image)
		}

		found = append(found, database{
			// Names come off the disk as the machine wrote them, upper case on
			// some and lower on others. One case keeps the output comparable.
			dir:    strings.ToLower(strings.TrimSuffix(name, extension)),
			source: fmt.Sprintf("%s (from %s)", f.Name, image),
			reader: r,
		})
	}

	if len(found) > 0 {
		return found, nil
	}

	// Nothing named like a database. This is where the eight-bit editions land,
	// none of which shipped one as a file, so there is nothing left but to
	// search: inside every file if the disk has any, and the sectors themselves
	// if it has not.
	var blobs []blob

	for _, f := range files {
		blobs = append(blobs, blob{where: f.Name, data: f.Data})
	}

	// Only when the volume gave up nothing to look inside is the image itself
	// worth searching. Doing both would find the same database twice, once in a
	// file and once again in the sectors that hold it.
	if img, ok := volume.(media.Image); ok && len(blobs) == 0 {
		blobs = append(blobs, blob{where: "the contents", data: img.Payload()})
	}

	if len(blobs) == 0 {
		// Either the volume would not give up its files, or it gave up none.
		if filesErr != nil {
			return nil, fmt.Errorf("%s: %w", path, filesErr)
		}

		return nil, fmt.Errorf("%s: %s, and it is empty", path, volume.Format())
	}

	for _, b := range blobs {
		for _, r := range FindDatabases(b.data) {
			found = append(found, database{
				source: fmt.Sprintf("%s at %#x (from %s)", b.where, r.Offset(), image),
				reader: r,
			})
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("%s: %s, and no database is to be found in it", path, volume.Format())
	}

	// Nothing names these. One on its own goes where it was asked to go, the
	// same as a database in a file of its own — a tape holding a single part
	// should not bury it under a directory saying which part of itself it is.
	// Several are numbered in the order they lie, which for a multi-part
	// adventure is the order of its parts.
	if len(found) > 1 {
		for i := range found {
			found[i].dir = fmt.Sprintf("part%d", i+1)
		}
	}

	return found, nil
}

func decompileOne(db database, outputDir string) error {
	fmt.Printf("Decompiling %q to %q\n", db.source, outputDir)

	startTime := time.Now()
	reader := db.reader

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
		{processesFile, writeProcesses},
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

	main := writeMain(reader, db.source, names)
	if err := os.WriteFile(filepath.Join(outputDir, mainFile), []byte(main), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", mainFile, err)
	}

	report(reader, startTime)

	return nil
}

// banner describes where the source came from, so a reader of the output knows
// what was deduced rather than read.
func banner(r *Reader, source string) string {
	var sb strings.Builder

	line := strings.Repeat("-", 74)
	add := func(f string, a ...any) { fmt.Fprintf(&sb, "; "+f+"\n", a...) }

	add("%s", line)
	add("Decompiled by QUnDAAD from %s", source)
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

	// A database found inside a program or on a disk holds addresses, not
	// offsets, so where it was laid has to be worked out to read it at all.
	if r.Base() != 0 {
		add("Found at       : %#x, laid at address %#04x   (deduced, not declared)",
			r.Offset(), r.Base())
	}
	add("%s", line)
	add("Objects        : %d", r.NumObjects())
	add("Locations      : %d", r.NumLocations())
	add("User messages  : %d", r.NumMessages())
	add("System messages: %d", r.NumSysMessages())
	add("Processes      : %d", r.NumProcesses())
	add("%s", line)

	return sb.String()
}

func writeMain(r *Reader, source string, includes []string) string {
	var sb strings.Builder

	sb.WriteString(banner(r, source))
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

	// Compiled without a compression table, so the source gets no /TOK either:
	// writing an empty one would not compile back to the same database.
	if len(r.Tokens()) == 0 {
		sb.WriteString("; This database was compiled without a compression table, so there is\n")
		sb.WriteString("; no /TOK section. Every text stands on its own.\n")

		return sb.String(), nil
	}

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

// processRole describes what a table is for. Only table 0 has a role the 1991
// manual fixes; the rest are sub-processes, and what each one does is a
// decision of whoever wrote the adventure. Rather than guess at a template,
// the caller notes which tables reach this one.
func processRole(number int, calledFrom []int) string {
	if number == 0 {
		return "MAIN LOOP. The interpreter enters here after initialising, with an\n" +
			"; empty logical sentence. Falling off the end or reaching DONE returns\n" +
			"; to the operating system rather than to a calling table."
	}

	if len(calledFrom) == 0 {
		return "SUB-PROCESS. No PROCESS condact in this database reaches it: it is\n" +
			"; either dead code or entered some other way."
	}

	callers := make([]string, 0, len(calledFrom))
	for _, c := range calledFrom {
		callers = append(callers, fmt.Sprint(c))
	}

	return "SUB-PROCESS, called from table " + strings.Join(callers, ", ") + "."
}

// callers maps every process to the tables that invoke it with PROCESS. The
// parameter is only meaningful when it is not indirect: an indirect call names
// a flag, and where it lands is only known at run time.
func callers(all [][]Entry) map[int][]int {
	const opProcess = 75

	out := map[int][]int{}

	for from, entries := range all {
		for _, e := range entries {
			for _, c := range e.Condacts {
				if c.Opcode != opProcess || c.Indirect || len(c.Params) == 0 {
					continue
				}

				to := int(c.Params[0])
				if !slices.Contains(out[to], from) {
					out[to] = append(out[to], from)
				}
			}
		}
	}

	return out
}

func writeProcesses(r *Reader) (string, error) {
	all, err := r.Processes()
	if err != nil {
		return "", err
	}

	from := callers(all)
	null := string(rune(r.NullWord()))

	var sb strings.Builder

	sb.WriteString("; The process tables: the program of the adventure. Each entry names a\n")
	sb.WriteString("; verb and a noun, and the condacts below run when the player's sentence\n")
	sb.WriteString("; matches them. The null word matches anything.\n")
	sb.WriteString(";\n")
	sb.WriteString("; Skip distances are written as numbers of entries, not as labels.\n\n")

	if at := r.UnreadableProcess(); at >= 0 {
		fmt.Fprintf(&sb, "; INCOMPLETE. The database declares %d process tables, and the %d\n",
			r.NumProcesses(), at)
		sb.WriteString("; below are all that could be read: the pointer to the next one is not\n")
		sb.WriteString("; an address. Recompiling this source will not reproduce the database\n")
		sb.WriteString("; it came from.\n\n")
	}

	for number, entries := range all {
		fmt.Fprintf(&sb, "; %s\n", strings.Repeat("-", 70))
		fmt.Fprintf(&sb, "; TABLE %d - %s\n", number, processRole(number, from[number]))
		fmt.Fprintf(&sb, "; %s\n", strings.Repeat("-", 70))
		fmt.Fprintf(&sb, "/PRO %d\n\n", number)

		for _, e := range entries {
			if err := writeEntry(&sb, r, e, null); err != nil {
				return "", fmt.Errorf("table %d: %w", number, err)
			}
		}
	}

	return sb.String(), nil
}

func writeEntry(sb *strings.Builder, r *Reader, e Entry, null string) error {
	verb, err := r.entryWord(e.Verb, null)
	if err != nil {
		return err
	}

	noun, err := r.entryWord(e.Noun, null)
	if err != nil {
		return err
	}

	head := fmt.Sprintf("%-8s %-8s ", verb, noun)

	for _, c := range e.Condacts {
		text, err := r.condactText(c)
		if err != nil {
			return err
		}

		sb.WriteString(head)
		sb.WriteString(text)
		sb.WriteString("\n")

		head = strings.Repeat(" ", 18) // the condacts after the first line up
	}

	sb.WriteString("\n")

	return nil
}

// entryWord names the verb or the noun of an entry. An entry matches on the
// word value, so any synonym would do; the first one is used.
func (r *Reader) entryWord(value byte, null string) (string, error) {
	if value == NoWord {
		return null, nil
	}

	word, found := r.WordFor(value, kindVerb, kindNoun)
	if !found {
		return "", fmt.Errorf("entry uses word value %d, which is in no vocabulary entry", value)
	}

	return word, nil
}

// condactText writes one condact with its parameters.
func (r *Reader) condactText(c Condact) (string, error) {
	def, found := r.Condact(c.Opcode)
	if !found {
		return "", fmt.Errorf("opcode %d is not a condact", c.Opcode)
	}

	out := def.Name

	for i, value := range c.Params {
		out += " "

		// Only the first parameter can be indirect, and then it is a flag
		// number rather than a value of its declared type.
		if i == 0 && c.Indirect {
			out += "@" + fmt.Sprint(value)

			continue
		}

		out += r.paramText(def.Params[i], value)
	}

	// Where the sources disagree on how many bytes a condact takes, say so on
	// the line itself: reading one too many shifts everything after it, so a
	// wrong guess here is not a local mistake.
	if note, uncertain := uncertainArity[c.Opcode]; uncertain && r.OldFormat() {
		out += "   ; TODO " + note
	}

	return out, nil
}

// paramText renders a parameter. Those naming a vocabulary word are written as
// the word; the rest stay numbers, which is what the source holds anyway.
func (r *Reader) paramText(kind ddb.ParamKind, value byte) string {
	vocabulary := map[ddb.ParamKind]byte{
		ddb.ParamVerb:        kindVerb,
		ddb.ParamNoun:        kindNoun,
		ddb.ParamAdjective:   kindAdjective,
		ddb.ParamAdverb:      1,
		ddb.ParamPreposition: 4,
	}

	wordKind, isWord := vocabulary[kind]
	if !isWord {
		return fmt.Sprint(value)
	}

	if value == NoWord {
		return string(rune(r.NullWord()))
	}

	if word, found := r.WordFor(value, wordKind); found {
		return word
	}

	return fmt.Sprint(value)
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
				// Emitting a line the compiler would read as anything but text
				// gives a source that compiles into something else.
				if readsAsMarkup(line) {
					return "", fmt.Errorf("%s entry %d has a line reading %q, which the "+
						"compiler would take for markup rather than text", tag, i, line)
				}

				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}

		return sb.String(), nil
	}
}

// The section markers the compiler's tokenizer recognises, from the rules in
// DSF.l. It matches the whole word, not the slash, which is why a line holding
// only a slash is text and not markup: system message 17 of Cozumel is exactly
// that, one slash and nothing else.
var sectionMarkers = [...]string{
	"/CON", "/CTL", "/END", "/LTX", "/MTX", "/OBJ", "/OTX", "/PRO", "/STX", "/VOC", "/TOK",
}

// readsAsMarkup says whether the compiler would take a line of text for
// something other than text.
//
// Two things it would: a section marker, and the "/12" that numbers an entry
// within a text section, lexed as \/[0-9]+.
func readsAsMarkup(line string) bool {
	if !strings.HasPrefix(line, "/") {
		return false
	}

	if len(line) > 1 && line[1] >= '0' && line[1] <= '9' {
		return true
	}

	for _, marker := range sectionMarkers {
		if strings.HasPrefix(strings.ToUpper(line), marker) {
			return true
		}
	}

	return false
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
	if r.Base() == 0 {
		fmt.Printf("  %d bytes total, %d bytes declared\n", len(r.data), r.DeclaredLength())
	} else {
		// One found inside something has no size of its own: what follows it
		// belongs to whatever it was buried in.
		fmt.Printf("  %d bytes declared, found at %#x, laid at address %#04x\n",
			r.DeclaredLength(), r.Offset(), r.Base())
	}
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
