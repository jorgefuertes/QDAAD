// Package back turns a database into the bytes a machine loads.
//
// It is the second half of the compiler and the mirror of the decompiler's
// reader: where that one walks a DDB and builds a source, this one walks the
// structures a source produced and lays the DDB out. The format is in
// docs/DAAD_v2_v3_Research/02-formato-ddb.md and 03-secciones.md.
//
// Three things decide the bytes and none of them is in the database: the
// address the machine loads it at, the byte order, and whether sections are
// padded to an even address. All three come from the target.
package back

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/charset"
	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
)

// Options are what the target decides, which the database does not carry.
type Options struct {
	Machine   arch.Machine
	Subtarget string
	Language  language.Language
	Version   uint8
}

// The shape of the header, which is the same in every version 2 database.
const (
	// counters is the version byte, the machine and language byte, the null
	// word, and the five counts.
	counters = 8
	// pointers are the twelve section addresses and the end address after
	// them, written once the sections are laid and their addresses known.
	pointers = 13
	// headerSize is the two together.
	headerSize = counters + 2*pointers
	// externVectors follow the header: three are used and the rest are there.
	externVectors = 13

	// obfuscate is applied to every text byte and every vocabulary character.
	// It is not encryption, it is what the interpreter undoes as it reads.
	obfuscate = 0xFF

	endOfText       = 0x0A
	endOfList       = 0xFF
	endOfProcess    = 0x00
	wordLen         = 5
	maxObjectWeight = 0x3F
	containerBit    = 0x40
	wearableBit     = 0x80
	indirectBit     = 0x80
)

// Link lays a database out as the bytes a machine loads.
func Link(db *ddb.DDB, opts Options) ([]byte, error) {
	if !opts.Machine.Valid() {
		return nil, fmt.Errorf("%s is not a machine this can build for", opts.Machine)
	}

	l := &linker{
		db:    db,
		opts:  opts,
		base:  int(opts.Machine.BaseAddress(opts.Subtarget)),
		big:   opts.Machine.IsBigEndian(),
		align: opts.Machine.IsWordAligned(),
	}

	if err := l.run(); err != nil {
		return nil, err
	}

	return l.out, nil
}

type linker struct {
	db   *ddb.DDB
	opts Options

	out   []byte
	base  int
	big   bool
	align bool

	// where each section ended up, in the order the header wants them.
	at [pointers]int
}

// The header slots, in the order they are written. The names are the
// decompiler's, so that the two halves can be read side by side.
const (
	ptrTokens = iota
	ptrProcesses
	ptrObjectText
	ptrLocationText
	ptrMessageText
	ptrSysMessageText
	ptrConnections
	ptrVocabulary
	ptrObjectLocation
	ptrObjectName
	ptrObjectWeight
	ptrObjectAttrs
	ptrEndAddress
)

func (l *linker) run() error {
	l.header()

	// The order the sections go in is not free: the object texts come first on
	// purpose, so that they are the ones in RAM where a target can page the
	// rest out. See drb.php, which lays them the same way.
	steps := []func() error{
		l.externs,
		l.objectTexts,
		l.objectNames,
		l.objectWeights,
		l.objectAttributes,
		l.objectLocations,
		l.vocabulary,
		l.tokens,
		l.systemMessages,
		l.userMessages,
		l.locationTexts,
		l.connections,
		l.processes,
	}

	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}

	l.at[ptrEndAddress] = l.here()

	return l.fillHeader()
}

// here is the address the next byte will be laid at.
func (l *linker) here() int { return l.base + len(l.out) }

func (l *linker) writeByte(b byte) { l.out = append(l.out, b) }

func (l *linker) writeWord(v int) {
	if l.big {
		l.out = append(l.out, byte(v>>8), byte(v))

		return
	}

	l.out = append(l.out, byte(v), byte(v>>8))
}

// pad takes the next section to an even address where the machine wants its
// words aligned. Where it does not, every section follows the one before it.
func (l *linker) pad() {
	if l.align && l.here()%2 != 0 {
		l.writeByte(0)
	}
}

// header writes what is known now and leaves room for what is not.
//
// Every address in it is a section that has not been laid yet, so the pointers
// go in as zeros and fillHeader comes back for them.
func (l *linker) header() {
	l.writeByte(l.opts.Version)
	l.writeByte(l.opts.Machine.ID()<<4 | l.languageBit())
	l.writeByte(l.db.NullWord)
	l.writeByte(byte(l.db.Objects().Len()))
	l.writeByte(byte(l.db.Locations().Len()))
	l.writeByte(byte(l.count(ddb.UserMessage)))
	l.writeByte(byte(l.count(ddb.SystemMessage)))
	l.writeByte(byte(l.db.Processes().Len()))

	for range pointers {
		l.writeWord(0)
	}

	for range externVectors {
		l.writeWord(0)
	}
}

// languageBit is the low nibble of byte 1, and it is one bit wide: an
// interpreter only needs to know whether the parser is the Spanish one.
//
// Our own numbering is not this one — it has an undefined at zero — so the two
// are mapped rather than cast. Portuguese shares the Spanish parser where DAAD
// READY builds it, which is the same rule at drb.php:1840.
func (l *linker) languageBit() byte {
	if l.opts.Language == language.LanguageSpanish {
		return 1
	}

	return 0
}

func (l *linker) fillHeader() error {
	if l.at[ptrEndAddress] > 0xFFFF {
		return fmt.Errorf("the database ends at %d, past the 64 KiB a machine can address",
			l.at[ptrEndAddress])
	}

	saved := l.out
	l.out = l.out[:counters]

	for _, address := range l.at {
		l.writeWord(address)
	}

	l.out = saved

	return nil
}

func (l *linker) count(kind ddb.MessageKind) int {
	messages, _ := l.db.Messages().GetAllMessages(kind)

	return len(messages)
}

// externs are the binaries a source asked to be linked in. Nothing writes them
// yet: a source that names one is refused rather than quietly built without it.
func (l *linker) externs() error {
	return nil
}

// texts lays one of the four text tables: every text in turn, then a table of
// where each one went.
//
// The text is obfuscated and ends with a newline, which is the one byte that
// can never be part of it. The lookup table is what the interpreter indexes,
// so its address is the one the header carries, not the texts'.
func (l *linker) texts(slot int, bodies []string) error {
	offsets := make([]int, len(bodies))

	for i, body := range bodies {
		l.pad()

		offsets[i] = l.here()

		encoded, err := charset.EncodeBytes(body)
		if err != nil {
			return fmt.Errorf("text %d: %w", i, err)
		}

		for _, b := range encoded {
			l.writeByte(b ^ obfuscate)
		}

		l.writeByte(endOfText ^ obfuscate)
	}

	l.pad()
	l.at[slot] = l.here()

	for _, offset := range offsets {
		l.writeWord(offset)
	}

	return nil
}

func (l *linker) objectTexts() error {
	objects := l.db.Objects()
	bodies := make([]string, 0, objects.Len())

	for i := range objects.Len() {
		obj, err := objects.Get(ddb.ID(i))
		if err != nil {
			return fmt.Errorf("object %d: %w", i, err)
		}

		bodies = append(bodies, obj.Description)
	}

	return l.texts(ptrObjectText, bodies)
}

func (l *linker) locationTexts() error {
	locations := l.db.Locations()
	bodies := make([]string, 0, locations.Len())

	for i := range locations.Len() {
		loc, err := locations.Get(ddb.ID(i))
		if err != nil {
			return fmt.Errorf("location %d: %w", i, err)
		}

		bodies = append(bodies, loc.Description)
	}

	return l.texts(ptrLocationText, bodies)
}

func (l *linker) systemMessages() error {
	return l.texts(ptrSysMessageText, l.messages(ddb.SystemMessage))
}

func (l *linker) userMessages() error {
	return l.texts(ptrMessageText, l.messages(ddb.UserMessage))
}

func (l *linker) messages(kind ddb.MessageKind) []string {
	stored, _ := l.db.Messages().GetAllMessages(kind)

	out := make([]string, len(stored))
	for _, m := range stored {
		out[m.ID.Int()] = m.Content
	}

	return out
}

// objectNames is the noun and the adjective of each object, which is how the
// parser decides that "coger la llave vieja" means this one.
func (l *linker) objectNames() error {
	l.pad()
	l.at[ptrObjectName] = l.here()

	return l.eachObject(func(obj *ddb.Object) error {
		l.writeByte(byte(obj.Name))
		l.writeByte(byte(obj.Adjective))

		return nil
	})
}

// objectWeights packs the weight and the two flags that share its byte: 6 bits
// of weight, then whether the object holds things and whether it is worn.
func (l *linker) objectWeights() error {
	l.at[ptrObjectWeight] = l.here()

	return l.eachObject(func(obj *ddb.Object) error {
		if obj.Weight > maxObjectWeight {
			return fmt.Errorf("weight %d does not fit in the 6 bits it has", obj.Weight)
		}

		b := obj.Weight & maxObjectWeight

		if obj.Container {
			b |= containerBit
		}

		if obj.Wearable {
			b |= wearableBit
		}

		l.writeByte(b)

		return nil
	})
}

// objectAttributes is the word of sixteen flags version 2 added. The leftmost
// column of an /OBJ line is the highest bit, which is what makes the line read
// like the number written out.
func (l *linker) objectAttributes() error {
	l.pad()
	l.at[ptrObjectAttrs] = l.here()

	return l.eachObject(func(obj *ddb.Object) error {
		packed := 0

		for i, on := range obj.Flags {
			if on {
				packed |= 1 << (len(obj.Flags) - 1 - i)
			}
		}

		l.writeWord(packed)

		return nil
	})
}

// objectLocations is where each object starts the game, and it is the one
// object table with a terminator after it.
func (l *linker) objectLocations() error {
	l.at[ptrObjectLocation] = l.here()

	if err := l.eachObject(func(obj *ddb.Object) error {
		l.writeByte(byte(obj.InitLocation))

		return nil
	}); err != nil {
		return err
	}

	l.writeByte(endOfList)

	return nil
}

func (l *linker) eachObject(write func(*ddb.Object) error) error {
	objects := l.db.Objects()

	for i := range objects.Len() {
		obj, err := objects.Get(ddb.ID(i))
		if err != nil {
			return fmt.Errorf("object %d: %w", i, err)
		}

		if err := write(obj); err != nil {
			return fmt.Errorf("object %d: %w", i, err)
		}
	}

	return nil
}

// vocabulary is seven bytes a word: five characters, the value and the kind.
//
// A word is cut or padded to five because that is all the parser compares, and
// it goes in uppercase and obfuscated. One synonym is one entry: the database
// groups them under a value and the binary does not.
func (l *linker) vocabulary() error {
	l.pad()
	l.at[ptrVocabulary] = l.here()

	for _, entry := range l.vocabularyOrder() {
		encoded, err := vocabularyWord(entry.word)
		if err != nil {
			return fmt.Errorf("the word %q: %w", entry.word, err)
		}

		for _, c := range encoded {
			l.writeByte(c ^ obfuscate)
		}

		l.writeByte(byte(entry.id))
		l.writeByte(byte(entry.kind))
	}

	l.writeByte(0)
	l.pad()

	return nil
}

// vocabularyWord lays one word out as the five bytes the parser compares.
//
// The order of the two steps is the whole of it: encode first, then uppercase,
// and only what is ASCII. Done the other way round an accented letter is
// uppercased into a letter the DAAD character set does not have — there is no
// capital Ó in it, which is why a source writes "CABRó" — and the encoder then
// spells it as a graphic, three bytes where the word wanted one.
//
// Five is all the parser reads, so the word is cut or padded with spaces to
// that. See drb.php:697-709, which does the same in the same order.
func vocabularyWord(word string) ([]byte, error) {
	encoded, err := charset.EncodeBytes(lowerAccents(word))
	if err != nil {
		return nil, err
	}

	out := make([]byte, wordLen)

	for i := range out {
		c := byte(' ')

		if i < len(encoded) {
			c = encoded[i]
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}
		}

		out[i] = c
	}

	return out, nil
}

// accentedCaps are the capitals the DAAD character set does not have, and the
// letters it keeps in their place.
//
// It is the five vowels and only those. The set has both cases of ñ, ç and ü —
// codes 26 and 27, 28 and 29, 30 and 31 — but of an accented vowel it has one,
// and that one is the lower case. A text can get round it by drawing the
// capital from the second character set, but a vocabulary word has five bytes
// and no room for a three-byte graphic, so here the case is given up.
//
// The store normalises a word to upper case as it files it, which is what makes
// this necessary on the way out. The same fold is in the front end, under
// vocSymbol, for the same reason.
var accentedCaps = map[rune]rune{
	'Á': 'á', 'É': 'é', 'Í': 'í', 'Ó': 'ó', 'Ú': 'ú',
}

func lowerAccents(word string) string {
	return strings.Map(func(r rune) rune {
		if lower, accented := accentedCaps[r]; accented {
			return lower
		}

		return r
	}, word)
}

type vocEntry struct {
	word string
	id   ddb.ID
	kind ddb.WordKind
}

// vocabularyOrder unfolds the synonyms and sorts them.
//
// The database groups the spellings that share a value under one word and the
// binary has one entry each, so they are unfolded here. The order is the
// alphabetical one, by spelling and then by kind, because that is the order a
// compiler's tree leaves them in and the one the parser's own lookup expects to
// walk. Insertion order would build a database that reads but whose words are
// found in a different sequence.
func (l *linker) vocabularyOrder() []vocEntry {
	var out []vocEntry

	for _, w := range *l.db.Words() {
		// The null word is in the store so that a lookup for "_" finds
		// something, but no /VOC line declares it and the binary has no row
		// for it.
		if w.Kind == ddb.NoWord {
			continue
		}

		for _, synonym := range w.Synonyms {
			out = append(out, vocEntry{word: synonym, id: w.ID, kind: w.Kind})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].word != out[j].word {
			return out[i].word < out[j].word
		}

		return out[i].kind < out[j].kind
	})

	return out
}

// tokens is the compression table, and there is none yet: a zero byte, and a
// zero in the header where its address would be, which is what the interpreter
// reads as "the texts are in clear".
func (l *linker) tokens() error {
	l.at[ptrTokens] = 0

	l.writeByte(0)
	l.pad()

	return nil
}

// connections are the exits of each location: a word and where it leads, in
// pairs, ending in a terminator. Then the table of where each list went.
func (l *linker) connections() error {
	locations := l.db.Locations()
	offsets := make([]int, 0, locations.Len())

	for i := range locations.Len() {
		l.pad()
		offsets = append(offsets, l.here())

		for _, c := range l.db.Conns().GetByLocation(ddb.ID(i)) {
			l.writeByte(byte(c.Word))
			l.writeByte(byte(c.To))
		}

		l.writeByte(endOfList)
	}

	l.pad()
	l.at[ptrConnections] = l.here()

	for _, offset := range offsets {
		l.writeWord(offset)
	}

	l.pad()

	return nil
}

// processes are the program. Three levels: the blocks of condacts, then a list
// of entries per process pointing at them, then the table of those lists.
//
// They are laid in that order because each level needs the addresses of the one
// before it.
func (l *linker) processes() error {
	all := *l.db.Processes()

	blocks := make([][]int, len(all))

	// A block already written is pointed at again rather than written twice,
	// and so is any tail of one: a block that does what another block's last
	// three condacts do can start in the middle of it. Entries that share
	// their ending are common — a dozen ways of saying a thing that all end in
	// "DONE" — so the saving is worth the bookkeeping.
	written := map[string]int{}

	for i, p := range all {
		blocks[i] = make([]int, len(p.Entries))

		for j, e := range p.Entries {
			block, starts, err := l.block(e.Condacts)
			if err != nil {
				return fmt.Errorf("process %d, entry %d: %w", p.ID, j, err)
			}

			if at, already := written[string(block)]; already {
				blocks[i][j] = at

				continue
			}

			// Each block starts where the machine can address one, which on a
			// padded machine means an even byte. It is only paid when a block
			// is written: one that is reused is already somewhere legal.
			l.pad()

			base := l.here()
			blocks[i][j] = base

			for _, start := range starts {
				// A tail can only be entered where the machine allows an
				// address to be, which on a padded one is an even byte.
				if l.align && (base+start)%2 != 0 {
					continue
				}

				if tail := string(block[start:]); written[tail] == 0 {
					written[tail] = base + start
				}
			}

			l.out = append(l.out, block...)
		}
	}

	l.pad()

	entries := make([]int, len(all))

	for i, p := range all {
		entries[i] = l.here()

		for j, e := range p.Entries {
			l.writeByte(byte(e.Verb))
			l.writeByte(byte(e.Noun))
			l.writeWord(blocks[i][j])
		}

		l.writeByte(endOfProcess)
		l.pad()
	}

	l.at[ptrProcesses] = l.here()

	for _, offset := range entries {
		l.writeWord(offset)
	}

	return nil
}

// PAUSE counts in units of the note the machine takes as whole, and what that
// is worth in milliseconds is not the same on each of them. A pause written
// once has to last the same everywhere, so the number is rescaled on the way
// out: see getBaseLength at drb.php:1529-1568.
//
// 200 is the unit itself, so a machine listed at 200 leaves the operand alone.
const (
	opPause        = 35
	wholeNote      = 200
	maxPauseLength = 255
)

// noteLength is what a whole note is worth on each machine, in milliseconds.
// A machine not named here keeps the unit, which is what the reference does
// with its default.
func noteLength(m arch.Machine, subtarget string) int {
	switch m {
	case arch.ZX81:
		return 195
	case arch.ZX:
		// The models with more to do in a frame count faster.
		switch strings.ToUpper(subtarget) {
		case "NEXT", "UNO", "PLUS3", "128K":
			return 100
		}

		return 195
	case arch.C64:
		return 120
	case arch.CP4:
		return 80
	case arch.PC:
		if strings.EqualFold(subtarget, "VGA256") {
			return 120
		}

		return 200
	case arch.HTML, arch.MSX, arch.MSX2:
		return 230
	case arch.CPC:
		return 300
	}

	return wholeNote
}

// operand is one byte of a condact, which is the value the source wrote unless
// the machine wants it in its own units.
func (l *linker) operand(c ddb.Condact, i int, p ddb.Param) byte {
	// Only the duration, and only when it was written as a number: an
	// indirect operand names a flag, and the flag's content is not ours to
	// rescale.
	if c.Opcode != opPause || i != 0 || p.Indirect {
		return byte(p.Value)
	}

	scaled := (p.Value.Int()*noteLength(l.opts.Machine, l.opts.Subtarget) + wholeNote/2) / wholeNote

	return byte(min(scaled, maxPauseLength))
}

// block builds one entry's condacts, without placing them.
//
// It comes back as bytes rather than going straight out because two entries
// that do the same thing produce the same bytes, and then only one of them
// needs writing. The second return is where each condact begins, which is what
// lets a shorter block be matched against the tail of a longer one: a tail is
// only a block if it starts where a condact does.
//
// A condact that ends the entry by itself makes the terminator unnecessary:
// anything after it would never run, because the interpreter stops there. So
// the block ends at the first one, and only a block without one pays the byte.
func (l *linker) block(condacts []ddb.Condact) (out []byte, starts []int, err error) {
	for _, c := range condacts {
		def, found := ddb.LookupCondact(c.Opcode)
		if !found {
			return nil, nil, fmt.Errorf("opcode %d is not a condact", c.Opcode)
		}

		if len(c.Params) != len(def.Params) {
			return nil, nil, fmt.Errorf("%s takes %d operands and has %d",
				def.Name, len(def.Params), len(c.Params))
		}

		starts = append(starts, len(out))

		opcode := byte(c.Opcode)
		if len(c.Params) > 0 && c.Params[0].Indirect {
			opcode |= indirectBit
		}

		out = append(out, opcode)

		for i, p := range c.Params {
			out = append(out, l.operand(c, i, p))
		}

		if def.Terminal {
			return out, starts, nil
		}
	}

	return append(out, endOfList), starts, nil
}
