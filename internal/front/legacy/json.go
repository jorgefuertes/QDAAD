package legacy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// This file writes the intermediate form the two halves of DAAD READY talk
// through: drf reads a source and emits it, drb reads it and emits a database.
// It is documented in docs/DAAD_v2_v3_Research/12-formato-dsf.md §9.
//
// Emitting it is worth doing for its own sake — it is what lets either half be
// replaced — but the reason it comes before the database emitter is narrower.
// Reading all sixty-six decompiled sources without an error only proves they
// were read, not that they were understood. Two front ends over the same source
// have to agree on every opcode and every parameter, and this is the form that
// agreement can be checked in.
//
// The field names are the ones drf writes, capitals and inconsistencies and
// all — "InitialyAt" is spelled that way there — because a consumer of the real
// thing has to be able to read ours.

// Document is a whole source in the intermediate form.
type Document struct {
	Settings     []DocSettings   `json:"settings"`
	Symbols      []DocSymbol     `json:"symbols"`
	Externs      []DocExtern     `json:"externs"`
	Vocabulary   []DocWord       `json:"vocabulary"`
	ObjectData   []DocObject     `json:"object_data"`
	Connections  []DocConnection `json:"connections"`
	Messages     []DocText       `json:"messages"`
	Sysmess      []DocText       `json:"sysmess"`
	Locations    []DocText       `json:"locations"`
	Objects      []DocText       `json:"objects"`
	XMessages    []DocText       `json:"xmessages"`
	OtherStrings []DocText       `json:"other_strings"`
	Processes    []DocProcess    `json:"processes"`
}

// DocSettings are the switches, written as numbers because that is how they
// are written there: a JSON boolean never appears in this format.
type DocSettings struct {
	ClassicMode int `json:"classic_mode"`
	DebugMode   int `json:"debug_mode"`
	V3Code      int `json:"v3code"`
	MaluvaUsed  int `json:"maluva_used"`
}

type DocSymbol struct {
	Symbol string `json:"symbol"`
	Value  int    `json:"Value"`
}

type DocExtern struct {
	FilePath string `json:"FilePath"`
}

type DocWord struct {
	VocWord string `json:"VocWord"`
	Value   int    `json:"Value"`
	VocType int    `json:"VocType"`
}

type DocObject struct {
	Value      int `json:"Value"`
	Noun       int `json:"Noun"`
	Adjective  int `json:"Adjective"`
	Container  int `json:"Container"`
	Wearable   int `json:"Wearable"`
	Flags      int `json:"Flags"`
	Weight     int `json:"Weight"`
	InitialyAt int `json:"InitialyAt"`
}

type DocConnection struct {
	FromLoc   int `json:"FromLoc"`
	ToLoc     int `json:"ToLoc"`
	Direction int `json:"Direction"`
}

type DocText struct {
	Value int    `json:"Value"`
	Text  string `json:"Text"`
}

type DocProcess struct {
	Value   int        `json:"Value"`
	Entries []DocEntry `json:"entries"`
}

type DocEntry struct {
	// Entry is the verb and the noun spelled out, which is there to be read by
	// a person rather than by a program: the numbers beside it are the truth.
	Entry    string       `json:"Entry"`
	Verb     int          `json:"Verb"`
	Noun     int          `json:"Noun"`
	Condacts []DocCondact `json:"condacts"`
}

// DocCondact is the one record whose keys depend on its contents: a condact
// with two parameters has Indirection1, Indirection2, Param1 and Param2, and
// one with none has neither. See MarshalJSON.
type DocCondact struct {
	Opcode  int
	Condact string
	Params  []DocParam
}

type DocParam struct {
	Indirect bool
	Value    int
}

// MarshalJSON writes the numbered keys by hand.
//
// It cannot be a struct, and a map would not do either: the keys are built from
// the parameter's position, and a map would sort them so that Param10 lands
// between Param1 and Param2. Written out in order, the record reads the way the
// condact does.
func (c DocCondact) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer

	b.WriteByte('{')

	write := func(key string, value any) error {
		if b.Len() > 1 {
			b.WriteByte(',')
		}

		name, err := json.Marshal(key)
		if err != nil {
			return err
		}

		b.Write(name)
		b.WriteByte(':')

		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}

		b.Write(encoded)

		return nil
	}

	if err := write("Opcode", c.Opcode); err != nil {
		return nil, err
	}

	if err := write("Condact", c.Condact); err != nil {
		return nil, err
	}

	// Every indirection flag first, then every value, which is the order they
	// are written in and not one this code is free to choose.
	for i, p := range c.Params {
		if err := write(fmt.Sprintf("Indirection%d", i+1), boolToInt(p.Indirect)); err != nil {
			return nil, err
		}
	}

	for i, p := range c.Params {
		if err := write(fmt.Sprintf("Param%d", i+1), p.Value); err != nil {
			return nil, err
		}
	}

	if err := write("NumParams", len(c.Params)); err != nil {
		return nil, err
	}

	b.WriteByte('}')

	return b.Bytes(), nil
}

// NewDocument turns a compiled source into the intermediate form.
//
// It takes the result of the passes before the parser as well as the database,
// because three of the thirteen sections are not in a database at all: the
// symbols, the externals and the settings are what the directives left behind.
func NewDocument(db *ddb.DDB, res *Result, opts Options) *Document {
	return &Document{
		Settings:     []DocSettings{docSettings(res, opts)},
		Symbols:      docSymbols(res.Symbols),
		Externs:      docExterns(res.Linked),
		Vocabulary:   docVocabulary(db),
		ObjectData:   docObjectData(db),
		Connections:  docConnections(db),
		Messages:     docMessages(db, ddb.UserMessage),
		Sysmess:      docMessages(db, ddb.SystemMessage),
		Locations:    docLocations(db),
		Objects:      docObjectTexts(db),
		XMessages:    []DocText{},
		OtherStrings: []DocText{},
		Processes:    docProcesses(db),
	}
}

// WriteJSON writes the document where a caller asks for it.
func (d *Document) WriteJSON(to *bytes.Buffer) error {
	encoder := json.NewEncoder(to)
	encoder.SetIndent("", "    ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(d)
}

func docSettings(res *Result, opts Options) DocSettings {
	return DocSettings{
		ClassicMode: boolToInt(res.Settings.Classic),
		DebugMode:   boolToInt(res.Settings.Debug),
		V3Code:      boolToInt(opts.Version >= 3),
		// Maluva is switched on by a fake condact, and none are read yet.
		MaluvaUsed: 0,
	}
}

// docSymbols writes the table out sorted by name.
//
// The compiler of reference writes it in the reverse of the order the symbols
// were defined, which is an artefact of how it walks its list and not something
// worth copying. Sorted, the same source gives the same file twice.
func docSymbols(symbols Symbols) []DocSymbol {
	out := make([]DocSymbol, 0, len(symbols))
	for name, value := range symbols {
		out = append(out, DocSymbol{Symbol: name, Value: value})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })

	return out
}

func docExterns(linked Externals) []DocExtern {
	out := make([]DocExtern, 0, len(linked.Extern))
	for _, path := range linked.Extern {
		out = append(out, DocExtern{FilePath: path})
	}

	return out
}

// docVocabulary unfolds the synonyms.
//
// A Word here is a value and a kind with every spelling that reaches it, which
// is how the database stores it; the intermediate form has one record per
// spelling. Both say the same thing and this is the shape that can be compared.
//
// The kind travels as a number, and the numbering is the same on both sides:
// verb 0, adverb 1, noun 2, adjective 3, preposition 4, conjunction 5,
// pronoun 6. See UVocabularyTree.pas:9.
func docVocabulary(db *ddb.DDB) []DocWord {
	var out []DocWord

	for _, w := range *db.Words() {
		// The null word is an entry of the store so that a lookup for "_" has
		// something to find, but it is not a word of the vocabulary and no
		// /VOC line declares it. It has no place in the table.
		if w.Kind == ddb.NoWord {
			continue
		}

		for _, synonym := range w.Synonyms {
			out = append(out, DocWord{
				VocWord: synonym,
				Value:   w.ID.Int(),
				VocType: int(w.Kind),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].VocWord != out[j].VocWord {
			return out[i].VocWord < out[j].VocWord
		}

		return out[i].VocType < out[j].VocType
	})

	return out
}

func docObjectData(db *ddb.DDB) []DocObject {
	objects := db.Objects()
	out := make([]DocObject, 0, objects.Len())

	for i := range objects.Len() {
		obj, err := objects.Get(ddb.ID(i))
		if err != nil {
			continue
		}

		out = append(out, DocObject{
			Value:      obj.ID.Int(),
			Noun:       obj.Name.Int(),
			Adjective:  obj.Adjective.Int(),
			Container:  boolToInt(obj.Container),
			Wearable:   boolToInt(obj.Wearable),
			Flags:      packAttributes(obj.Flags),
			Weight:     int(obj.Weight),
			InitialyAt: obj.InitLocation.Int(),
		})
	}

	return out
}

// packAttributes folds the sixteen columns of an /OBJ line into the word they
// are stored as. The leftmost column is the highest bit, which is what makes
// the line read like the number written out in binary.
func packAttributes(flags [16]bool) int {
	packed := 0

	for i, on := range flags {
		if on {
			packed |= 1 << (len(flags) - 1 - i)
		}
	}

	return packed
}

func docConnections(db *ddb.DDB) []DocConnection {
	conns := *db.Conns()
	out := make([]DocConnection, 0, len(conns))

	for _, c := range conns {
		out = append(out, DocConnection{
			FromLoc:   c.From.Int(),
			ToLoc:     c.To.Int(),
			Direction: c.Word.Int(),
		})
	}

	return out
}

func docMessages(db *ddb.DDB, kind ddb.MessageKind) []DocText {
	messages, _ := db.Messages().GetAllMessages(kind)

	out := make([]DocText, 0, len(messages))
	for _, m := range messages {
		out = append(out, DocText{Value: m.ID.Int(), Text: m.Content})
	}

	return out
}

// docLocations and docObjectTexts read the two tables whose text lives on the
// thing it describes rather than in the message store.
func docLocations(db *ddb.DDB) []DocText {
	locations := db.Locations()
	out := make([]DocText, 0, locations.Len())

	for i := range locations.Len() {
		loc, err := locations.Get(ddb.ID(i))
		if err != nil {
			continue
		}

		out = append(out, DocText{Value: i, Text: loc.Description})
	}

	return out
}

func docObjectTexts(db *ddb.DDB) []DocText {
	objects := db.Objects()
	out := make([]DocText, 0, objects.Len())

	for i := range objects.Len() {
		obj, err := objects.Get(ddb.ID(i))
		if err != nil {
			continue
		}

		out = append(out, DocText{Value: i, Text: obj.Description})
	}

	return out
}

func docProcesses(db *ddb.DDB) []DocProcess {
	processes := *db.Processes()
	out := make([]DocProcess, 0, len(processes))

	for _, p := range processes {
		out = append(out, DocProcess{
			Value:   p.ID.Int(),
			Entries: docEntries(db, p.Entries),
		})
	}

	return out
}

func docEntries(db *ddb.DDB, entries []ddb.Entry) []DocEntry {
	out := make([]DocEntry, 0, len(entries))

	for _, e := range entries {
		out = append(out, DocEntry{
			Entry:    entryName(db, e),
			Verb:     e.Verb.Int(),
			Noun:     e.Noun.Int(),
			Condacts: docCondacts(e.Condacts),
		})
	}

	return out
}

// entryName spells the entry out the way the source wrote it.
//
// The wildcard is an underscore and a word this vocabulary does not have is a
// question mark, both of which are what the compiler of reference writes. The
// verb slot falls back to a noun, because a noun low enough doubles as a verb
// and that is how "NORTE" works as a sentence on its own.
func entryName(db *ddb.DDB, e ddb.Entry) string {
	verb := wordName(db, e.Verb, ddb.Verb)
	if verb == "?" && e.Verb <= ddb.MAX_CONVERTIBLE_NAME {
		verb = wordName(db, e.Verb, ddb.Noun)
	}

	return verb + " " + wordName(db, e.Noun, ddb.Noun)
}

func wordName(db *ddb.DDB, id ddb.ID, kind ddb.WordKind) string {
	if id == ddb.NO_WORD_ID {
		return "_"
	}

	w, found := db.Words().Get(id, kind)
	if !found || len(w.Synonyms) == 0 {
		return "?"
	}

	return w.Synonyms[0]
}

func docCondacts(condacts []ddb.Condact) []DocCondact {
	out := make([]DocCondact, 0, len(condacts))

	for _, c := range condacts {
		name := "?"
		if def, found := ddb.LookupCondact(c.Opcode); found {
			name = def.Name
		}

		params := make([]DocParam, 0, len(c.Params))
		for _, p := range c.Params {
			params = append(params, DocParam{Indirect: p.Indirect, Value: p.Value.Int()})
		}

		out = append(out, DocCondact{
			Opcode:  int(c.Opcode),
			Condact: name,
			Params:  params,
		})
	}

	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}
