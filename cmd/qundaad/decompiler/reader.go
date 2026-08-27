package decompiler

import (
	"errors"
	"fmt"
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
	data        []byte
	bigEndian   bool
	headerSize  int
	placeholder string   // the leading token every interpreter skips
	tokens      []string // the 128 a text can refer to, indexed from zero
	vocabulary  []VocabularyEntry
}

// NewReader works out how to read the database instead of trusting the target.
//
// Endianness and header size are not stated anywhere in the file, and the
// reference tooling gets them from a table indexed by target, which is how it
// ends up reporting a PC database as little-endian when it is not. Both can be
// deduced from the data: the last word of the header holds the length of the
// file, so only one combination makes it add up.
func NewReader(data []byte) (*Reader, error) {
	if len(data) < headerSizeV1 {
		return nil, fmt.Errorf("too short to be a DDB: %d bytes", len(data))
	}

	for _, size := range []int{headerSizeV1, headerSizeV2} {
		for _, big := range []bool{true, false} {
			r := &Reader{data: data, bigEndian: big, headerSize: size}
			if r.lengthAgrees() {
				if err := r.readTokens(); err != nil {
					return nil, err
				}

				if err := r.readVocabulary(); err != nil {
					return nil, err
				}

				return r, nil
			}
		}
	}

	return nil, errors.New("cannot determine endianness and header size: " +
		"no combination makes the declared length match the file")
}

// lengthAgrees checks the invariant the detection rests on. The match is not
// asked to be exact: platforms that align to a word pad the end of the file,
// so the declared length can fall a little short of the real one.
func (r *Reader) lengthAgrees() bool {
	declared := int(r.word(r.headerSize - 2))
	if declared > len(r.data) {
		return false
	}

	const maxPadding = 512

	return len(r.data)-declared < maxPadding
}

func (r *Reader) word(offset int) uint16 {
	hi, lo := r.data[offset], r.data[offset+1]
	if !r.bigEndian {
		hi, lo = lo, hi
	}

	return uint16(hi)<<8 | uint16(lo)
}

// Pointer returns one of the header pointers, by index.
func (r *Reader) Pointer(index int) int {
	return int(r.word(8 + 2*index))
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
func (r *Reader) DeclaredLength() int { return int(r.word(r.headerSize - 2)) }
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
	if p == 0 || p >= len(r.data) {
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
	if index == 0 || index+2*count > len(r.data) {
		return nil, fmt.Errorf("text index out of range: %d for %d entries", index, count)
	}

	texts := make([]string, 0, count)

	for i := range count {
		addr := int(r.word(index + 2*i))
		if addr >= len(r.data) {
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
	if index == 0 || index+2*count > len(r.data) {
		return nil, fmt.Errorf("connection index out of range: %d for %d locations", index, count)
	}

	all := make([][]Movement, 0, count)

	for i := range count {
		p := int(r.word(index + 2*i))
		if p >= len(r.data) {
			return nil, fmt.Errorf("location %d points outside the file: %d", i, p)
		}

		var moves []Movement

		for {
			if p >= len(r.data) {
				return nil, fmt.Errorf("location %d: connections run past the end of the file", i)
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
