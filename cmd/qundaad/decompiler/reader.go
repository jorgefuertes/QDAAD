package decompiler

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Obfuscation applied to every text byte and to every vocabulary word.
const obfuscate = 0xFF

// The header is a fixed set of counters followed by a list of words. Version 1
// databases have twelve of those words; version 2 inserted the pointer to the
// extra object attributes, so they have thirteen.
const (
	headerSizeV1 = 32
	headerSizeV2 = 34
)

// Index of each pointer in the header word list.
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
	ptrObjectAttrs // version 2 onwards only
)

// Reader gives structured access to a DDB held in memory. The whole file is
// kept as a slice on purpose: a database is at most 64 KiB, so random access
// costs nothing and the pointers can be followed directly.
type Reader struct {
	data       []byte
	bigEndian  bool
	headerSize int
	base       int // the address the database was laid at, zero in a file
	offset     int // where it was found, when it was found inside something

	// unreadableProcess is the first process whose pointer is not an address,
	// or -1 when every one of them reads. See Processes.
	unreadableProcess int
	placeholder       string   // the leading token every interpreter skips
	tokens            []string // the 128 a text can refer to, indexed from zero
	vocabulary        []VocabularyEntry
}

// FindDatabases looks for databases lying inside something that is not one.
//
// The eight-bit editions never shipped a database as a file. The Commodore one
// is linked into the program that loads it; the Amstrad and Spectrum ones are
// laid straight onto disks formatted for their own loader, with no filesystem
// at all. There is nothing naming them, so the only way in is to recognise a
// header where it lies and let it prove itself.
//
// They come back in the order they appear, which for a multi-part adventure is
// the order of its parts.
func FindDatabases(data []byte) []*Reader {
	var found []*Reader

	for off := 0; off+headerSizeV1 <= len(data); {
		if !looksLikeHeader(data[off:]) {
			off++

			continue
		}

		r, ok := readerAt(data, off)
		if !ok || !r.readsThrough() {
			off++

			continue
		}

		r.offset = off
		found = append(found, r)

		// No point looking inside one we have already read.
		off += r.DeclaredLength()
	}

	return found
}

// looksLikeHeader is the cheap test that keeps the search down to a handful of
// candidates. Everything it accepts still has to survive readerAt.
func looksLikeHeader(d []byte) bool {
	const lastMachine = 7 // the Amstrad PCW, and there was never an eighth

	version, machine, nullWord := d[0], d[1]>>4, d[2]

	return (version == 1 || version == 2) &&
		machine <= lastMachine &&
		nullWord > ' ' && nullWord < 0x7F &&
		// An adventure with no objects, no locations or no processes is not one.
		d[3] != 0 && d[4] != 0 && d[7] != 0
}

// readsThrough is the further test a candidate found by searching has to pass.
//
// A header that describes itself and a token table that parses are not enough
// when megabytes are being sifted: a run of bytes can manage both by luck. Made
// to read every text table and every connection list as well, it cannot.
//
// The process table is left out on purpose. Decoding it depends on the arity of
// each condact, and where that is still uncertain a real database could fail
// here and be passed over in silence.
func (r *Reader) readsThrough() bool {
	tables := [...]struct {
		pointer int
		count   int
	}{
		{ptrSysMessageText, r.NumSysMessages()},
		{ptrMessageText, r.NumMessages()},
		{ptrObjectText, r.NumObjects()},
		{ptrLocationText, r.NumLocations()},
	}

	characters, escapes := 0, 0

	for _, t := range tables {
		texts, err := r.TextTable(t.pointer, t.count)
		if err != nil {
			return false
		}

		for _, text := range texts {
			characters += len(text)
			escapes += strings.Count(text, "\\")
		}
	}

	if !readsLikeProse(characters, escapes) {
		return false
	}

	if _, err := r.Connections(); err != nil {
		return false
	}

	_, err := r.Objects()

	return err == nil
}

// readsLikeProse is the last thing between a candidate and being decompiled,
// and the only one that looks at what the text says rather than where it is.
//
// The structural checks can all pass on something that is not a database. The
// MSX edition of El Jabato has a header and a vocabulary lying where they can be
// read — its 288 words match the PC edition exactly — but its text is kept
// somewhere else, so the addresses lead to bytes that decode into a stream of
// escapes and token fragments. Every check but this one lets it through, and
// what comes out is nonsense that looks like source.
//
// Real text is prose with the odd escape in it. Across the databases we can
// read, the highest proportion is one escape in every ninety-six characters;
// the nonsense above runs at one in thirteen. One in twenty-five sits clear of
// both.
func readsLikeProse(characters, escapes int) bool {
	const leastCharactersPerEscape = 25

	if characters == 0 {
		return false
	}

	return escapes*leastCharactersPerEscape <= characters
}

// readerAt tries to read a database beginning at off, working out how without
// being told: neither the endianness, nor the size of the header, nor the
// address the database was laid at is stated anywhere in it.
//
// Working them out is the point. The reference tooling takes endianness and
// header size from a table indexed by the target it was built for, which is how
// it comes to report a PC database as little-endian when it is not. Here every
// combination is tried and the one that reads is kept. What makes that safe is
// that the header has to describe itself consistently — a length that fits and
// eleven or twelve pointers that all land inside it — and then the token table
// and the vocabulary have to parse. Rubbish does not survive it.
//
// A database in a file of its own is tried first, and told apart by the length
// it declares accounting for the file. Its addresses are already offsets, so
// there is nothing to work out about where it was laid.
//
// Only when that fails is it taken for one found inside something bigger, and
// the address deduced. The order matters: a database with padding between its
// header and its vocabulary — Los Templos Sagrados has twenty-six bytes of it —
// would otherwise be taken for one laid at an address, and every pointer in it
// shifted by the width of the padding.
func readerAt(data []byte, off int) (*Reader, bool) {
	if off < 0 || off+headerSizeV1 > len(data) {
		return nil, false
	}

	for _, whole := range []bool{true, false} {
		for _, size := range []int{headerSizeV1, headerSizeV2} {
			for _, big := range []bool{true, false} {
				r := &Reader{data: data[off:], bigEndian: big, headerSize: size, unreadableProcess: -1}

				if !whole {
					// The vocabulary follows the header, near enough, so its
					// pointer gives away the address it was all laid at.
					r.base = int(r.word(8+2*ptrVocabulary)) - size
				}

				if !r.headerAgrees() {
					continue
				}

				if whole && !r.lengthAccountsForFile() {
					continue
				}

				if err := r.readTokens(); err != nil {
					continue
				}

				if err := r.readVocabulary(); err != nil {
					continue
				}

				return r, true
			}
		}
	}

	return nil, false
}

// lengthAccountsForFile says whether the declared length covers what is there,
// which is what marks a database out as a file of its own rather than something
// sitting inside a larger one.
//
// The match is not asked to be exact: the machines that align to a word pad the
// end of the file, so the declared length can fall a little short of it.
func (r *Reader) lengthAccountsForFile() bool {
	const maxPadding = 512

	return len(r.data)-r.DeclaredLength() < maxPadding
}

// headerAgrees checks that the header describes itself: a length that fits in
// what is there, and every pointer landing between the end of the header and
// that length.
//
// The length is not asked to match exactly. A database in a file of its own can
// fall short of it, because the machines that align to a word pad the end; one
// found inside a program or on a disk is followed by whatever came next.
func (r *Reader) headerAgrees() bool {
	if r.base < 0 {
		return false
	}

	declared := r.DeclaredLength()
	if declared <= r.headerSize || declared > len(r.data) {
		return false
	}

	// Words from eight to the end of the header are pointers, bar the last,
	// which is the length. A pointer of zero names a section that is not there
	// and has nothing to be in range of.
	for i := range (r.headerSize-8)/2 - 1 {
		p := r.Pointer(i)
		if p == 0 {
			continue
		}

		if p < r.headerSize || p >= declared {
			return false
		}
	}

	return true
}

func (r *Reader) word(offset int) uint16 {
	hi, lo := r.data[offset], r.data[offset+1]
	if !r.bigEndian {
		hi, lo = lo, hi
	}

	return uint16(hi)<<8 | uint16(lo)
}

// address reads a word that holds an address and turns it into an offset from
// the start of the database. The tables point at one another this way, so
// anything read as a destination has to come through here.
func (r *Reader) address(offset int) int {
	return int(r.word(offset)) - r.base
}

// inRange says whether an address read out of the database lands inside it,
// with room for need bytes.
//
// The lower bound matters as much as the upper one. A database found inside a
// program or on a disk holds addresses rather than offsets, so a wrong guess at
// where it was laid comes out as a negative address: a crash waiting to happen
// if it is only ever checked against the end.
func (r *Reader) inRange(addr, need int) bool {
	return addr >= r.headerSize && need >= 0 && addr+need <= len(r.data)
}

// Pointer returns one of the header pointers, by index.
//
// A pointer of zero means the section is not in this database: the compression
// table is optional, and the CGA build of El Jabato was compiled without one.
// It stays zero rather than being relocated into an address below the base.
func (r *Reader) Pointer(index int) int {
	if r.word(8+2*index) == 0 {
		return 0
	}

	return r.address(8 + 2*index)
}

func (r *Reader) BigEndian() bool     { return r.bigEndian }
func (r *Reader) HeaderSize() int     { return r.headerSize }
func (r *Reader) Version() byte       { return r.data[0] }
func (r *Reader) Machine() byte       { return r.data[1] >> 4 }
func (r *Reader) Language() byte      { return r.data[1] & 0x0F }
func (r *Reader) NullWord() byte      { return r.data[2] }
func (r *Reader) NumObjects() int     { return int(r.data[3]) }
func (r *Reader) NumLocations() int   { return int(r.data[4]) }
func (r *Reader) NumMessages() int    { return int(r.data[5]) }
func (r *Reader) NumSysMessages() int { return int(r.data[6]) }
func (r *Reader) NumProcesses() int   { return int(r.data[7]) }
func (r *Reader) DeclaredLength() int { return r.address(r.headerSize - 2) }

// Base is the address the database was laid at, and Offset where it was found
// in whatever held it. Both are zero for a database in a file of its own, and
// both are deduced rather than read.
func (r *Reader) Base() int { return r.base }

// UnreadableProcess returns the first process that could not be read, or -1 if
// they all could.
func (r *Reader) UnreadableProcess() int { return r.unreadableProcess }

func (r *Reader) Offset() int { return r.offset }

func (r *Reader) Tokens() []string    { return r.tokens }
func (r *Reader) Placeholder() string { return r.placeholder }

// readTokens loads the compression table. It has no counter and no terminator:
// tokens follow one another and the last byte of each has bit 7 set.
//
// The first one is a placeholder that no text ever refers to. Every interpreter
// skips it — PCDAAD hardcodes "tokenPos + 1", msx2daad walks to the first
// terminator — so the table really holds 129 entries: the placeholder and the
// 128 that texts can name.
func (r *Reader) readTokens() error {
	const numTokens = 128

	p := r.Pointer(ptrTokens)
	if p == 0 {
		// Compiled without a compression table. Every text then stands on its
		// own, and none may refer to a token, because there are none.
		return nil
	}

	if !r.inRange(p, 1) {
		return fmt.Errorf("token pointer out of range: %d", p)
	}

	next := func() (string, error) {
		var raw []byte

		for {
			if p >= len(r.data) {
				return "", errors.New("token table runs past the end of the file")
			}

			b := r.data[p]
			p++

			raw = append(raw, b&0x7F)

			if b&0x80 != 0 {
				return tokenText(raw), nil
			}
		}
	}

	placeholder, err := next()
	if err != nil {
		return err
	}

	r.placeholder = placeholder
	r.tokens = make([]string, 0, numTokens)

	for range numTokens {
		token, err := next()
		if err != nil {
			return err
		}

		r.tokens = append(r.tokens, token)
	}

	return nil
}

// Text decodes a string at an address, undoing the obfuscation and expanding
// compression tokens. A decoded byte with bit 7 set is a token reference; the
// string ends at 0x0A.
func (r *Reader) Text(addr int) (string, error) {
	var sb strings.Builder

	if !r.inRange(addr, 1) {
		return "", fmt.Errorf("text address outside the database: %d", addr)
	}

	for addr < len(r.data) {
		code := r.data[addr] ^ obfuscate
		addr++

		if code == codeEndOfText {
			return sb.String(), nil
		}

		if code&0x80 != 0 {
			index := int(code & 0x7F)
			if index >= len(r.tokens) {
				return sb.String(), fmt.Errorf("text refers to token %d, out of %d", index, len(r.tokens))
			}

			sb.WriteString(r.tokens[index])

			continue
		}

		sb.WriteString(decodeChar(code))
	}

	return sb.String(), errors.New("unterminated text: reached the end of the file")
}

// TextTable reads one of the four text tables.
//
// The header pointer does not lead to the texts but to an index of one word
// per message, and the count comes from the header counters. The texts
// themselves sit before the index and are never walked in order: each one is
// reached through its own entry.
func (r *Reader) TextTable(pointerIndex, count int) ([]string, error) {
	index := r.Pointer(pointerIndex)
	if !r.inRange(index, 2*count) {
		return nil, fmt.Errorf("text index out of range: %d for %d entries", index, count)
	}

	texts := make([]string, 0, count)

	for i := range count {
		addr := r.address(index + 2*i)
		if !r.inRange(addr, 1) {
			return nil, fmt.Errorf("text %d points outside the file: %d", i, addr)
		}

		text, err := r.Text(addr)
		if err != nil {
			return nil, fmt.Errorf("text %d: %w", i, err)
		}

		texts = append(texts, text)
	}

	return texts, nil
}

// Condact is one instruction of a process, as the database stores it.
type Condact struct {
	Opcode byte
	// Indirect is bit 7 of the opcode byte: the first parameter is then a flag
	// number and the interpreter uses the flag's content instead.
	Indirect bool
	Params   []byte
}

// Entry is one line of a process table: the verb and noun it answers to, and
// what to do about it. NoWord in either is the wildcard that matches anything.
type Entry struct {
	Verb     byte
	Noun     byte
	Condacts []Condact
}

// Processes reads the whole process section: three levels deep, a word per
// process pointing at a table of 4-byte entries, each of which points at a
// block of condacts.
func (r *Reader) Processes() ([][]Entry, error) {
	const (
		entrySize    = 4
		endOfProcess = 0x00 // in the place of a verb
	)

	count := r.NumProcesses()

	table := r.Pointer(ptrProcesses)
	if !r.inRange(table, 2*count) {
		return nil, fmt.Errorf("process table out of range: %d for %d processes", table, count)
	}

	all := make([][]Entry, 0, count)

	for i := range count {
		p := r.address(table + 2*i)

		// A pointer that is not an address at all ends the table early. The
		// Spectrum edition of Cozumel has one: seventy of its seventy-one
		// processes read, and the last word of the database, where the
		// seventy-first should point, holds 0x0091. Everything read so far is
		// worth keeping, so it is handed back and the gap reported.
		if !r.inRange(p, entrySize) {
			r.unreadableProcess = i

			return all, nil
		}

		var entries []Entry

		for {
			if !r.inRange(p, entrySize) {
				return nil, fmt.Errorf("process %d: entries run outside the database", i)
			}

			if r.data[p] == endOfProcess {
				break
			}

			condacts, err := r.condactsAt(r.address(p + 2))
			if err != nil {
				return nil, fmt.Errorf("process %d, entry %d: %w", i, len(entries), err)
			}

			entries = append(entries, Entry{Verb: r.data[p], Noun: r.data[p+1], Condacts: condacts})
			p += entrySize
		}

		all = append(all, entries)
	}

	return all, nil
}

// condactsAt decodes one block.
//
// The arity is not in the binary, so every step depends on the condact table
// being right: read one parameter too few and everything after it shifts. A
// block ends at 0xFF, or at a condact that ends it by itself, in which case the
// compiler saves the byte.
func (r *Reader) condactsAt(p int) ([]Condact, error) {
	const endOfCondacts = 0xFF

	var condacts []Condact

	for {
		if !r.inRange(p, 1) {
			return nil, errors.New("condacts run outside the database")
		}

		b := r.data[p]
		p++

		if b == endOfCondacts {
			return condacts, nil
		}

		opcode := b &^ 0x80

		def, found := r.Condact(opcode)
		if !found {
			return nil, fmt.Errorf("opcode %d is not a condact", opcode)
		}

		n := def.NumParams()
		if p+n > len(r.data) {
			return nil, fmt.Errorf("%s: parameters run past the end of the file", def.Name)
		}

		condacts = append(condacts, Condact{
			Opcode:   opcode,
			Indirect: b&0x80 != 0,
			Params:   slices.Clone(r.data[p : p+n]),
		})

		p += n

		if def.Terminal {
			return condacts, nil
		}
	}
}

// Object gathers what the database says about one object. There is no object
// record in a DDB: the data lives in parallel arrays with a pointer each, so
// this is assembled by indexing every one of them with the same number.
type Object struct {
	InitialLocation byte
	Weight          byte // 0-63
	Container       bool
	Wearable        bool
	Noun            byte // NoWord when it has none
	Adjective       byte
}

// NoWord marks an absent word, in an object's name as well as in a process
// entry, where it reads as a wildcard.
const NoWord = 0xFF

// Objects reads the object arrays and joins them.
func (r *Reader) Objects() ([]Object, error) {
	count := r.NumObjects()

	locations := r.Pointer(ptrObjectLocation)
	names := r.Pointer(ptrObjectName)
	weights := r.Pointer(ptrObjectWeight)

	for name, table := range map[string]struct{ at, size int }{
		"initial locations": {locations, count},
		"names":             {names, 2 * count},
		"weights":           {weights, count},
	} {
		if !r.inRange(table.at, table.size) {
			return nil, fmt.Errorf("object %s run outside the database", name)
		}
	}

	objects := make([]Object, 0, count)

	for i := range count {
		// One byte holds the weight and the two flags: the low 6 bits are the
		// weight, bit 6 marks a container and bit 7 something wearable.
		packed := r.data[weights+i]

		objects = append(objects, Object{
			InitialLocation: r.data[locations+i],
			Weight:          packed & 0x3F,
			Container:       packed&0x40 != 0,
			Wearable:        packed&0x80 != 0,
			Noun:            r.data[names+2*i],
			Adjective:       r.data[names+2*i+1],
		})
	}

	return objects, nil
}

// HasObjectAttributes reports whether the database carries the extra attribute
// bits. Version 1 has no pointer for them, so there is nothing to read.
func (r *Reader) HasObjectAttributes() bool {
	return r.headerSize >= headerSizeV2 && r.Pointer(ptrObjectAttrs) != 0
}

// Movement is one exit of a location: a word and where it leads.
type Movement struct {
	Word byte
	To   byte
}

// Connections reads the exits of every location.
//
// Same two-level shape as the texts: the header points at an index of one word
// per location, and each entry is a list of one-byte pairs ending in 0xFF. A
// location with no exits still has a list, holding just the terminator.
func (r *Reader) Connections() ([][]Movement, error) {
	const endOfList = 0xFF

	count := r.NumLocations()

	index := r.Pointer(ptrConnections)
	if !r.inRange(index, 2*count) {
		return nil, fmt.Errorf("connection index out of range: %d for %d locations", index, count)
	}

	all := make([][]Movement, 0, count)

	for i := range count {
		p := r.address(index + 2*i)
		if !r.inRange(p, 1) {
			return nil, fmt.Errorf("location %d points outside the file: %d", i, p)
		}

		var moves []Movement

		for {
			if !r.inRange(p, 1) {
				return nil, fmt.Errorf("location %d: connections run outside the database", i)
			}

			if r.data[p] == endOfList {
				break
			}

			if p+1 >= len(r.data) {
				return nil, fmt.Errorf("location %d: truncated movement", i)
			}

			moves = append(moves, Movement{Word: r.data[p], To: r.data[p+1]})
			p += 2
		}

		all = append(all, moves)
	}

	return all, nil
}

// WordFor names a vocabulary value.
//
// A value can be reused across types, so the type is part of the key, and the
// caller may accept more than one: a movement is "a Verb (or Noun < 20)", and
// this adventure declares its directions as nouns. Several words can share
// value and type — they are synonyms — and the first in the table wins, which
// is also the one the reference decompiler picks.
func (r *Reader) WordFor(value byte, kinds ...byte) (string, bool) {
	for _, kind := range kinds {
		for _, e := range r.vocabulary {
			if e.Value == value && e.Kind == kind {
				return e.Word, true
			}
		}
	}

	return "", false
}

// VocabularyEntry is one word of the vocabulary as the database stores it.
type VocabularyEntry struct {
	Word  string
	Value byte
	Kind  byte
}

// Vocabulary returns the table as the database stores it, in file order.
func (r *Reader) Vocabulary() []VocabularyEntry { return r.vocabulary }

// readVocabulary loads the table. Entries are seven bytes and it ends with a
// single zero where a word would start.
func (r *Reader) readVocabulary() error {
	const entrySize = 7

	p := r.Pointer(ptrVocabulary)
	if p == 0 || p >= len(r.data) {
		return fmt.Errorf("vocabulary pointer out of range: %d", p)
	}

	var entries []VocabularyEntry

	for {
		if p+entrySize > len(r.data) {
			return errors.New("vocabulary runs past the end of the file")
		}

		if r.data[p] == 0 {
			r.vocabulary = entries

			return nil
		}

		var word strings.Builder

		for _, b := range r.data[p : p+5] {
			word.WriteString(decodeChar(b ^ obfuscate))
		}

		entries = append(entries, VocabularyEntry{
			Word:  strings.TrimRight(word.String(), " "),
			Value: r.data[p+5],
			Kind:  r.data[p+6],
		})

		p += entrySize
	}
}
