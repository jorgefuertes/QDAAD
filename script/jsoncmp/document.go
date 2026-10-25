package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

// document is the intermediate form as read back in.
//
// It is deliberately loose. The two writers differ in small ways — one emits a
// key the other leaves out, one writes a number where the other writes a
// string — and a strict struct would fail to parse rather than report the
// difference. Condacts in particular have keys built from a parameter's
// position, so that record stays a map.
type document struct {
	Settings     []map[string]any `json:"settings"`
	Symbols      []symbol         `json:"symbols"`
	Externs      []extern         `json:"externs"`
	Vocabulary   []vocWord        `json:"vocabulary"`
	ObjectData   []objectData     `json:"object_data"`
	Connections  []connection     `json:"connections"`
	Messages     []text           `json:"messages"`
	Sysmess      []text           `json:"sysmess"`
	Locations    []text           `json:"locations"`
	Objects      []text           `json:"objects"`
	XMessages    []text           `json:"xmessages"`
	OtherStrings []text           `json:"other_strings"`
	Processes    []process        `json:"processes"`
}

type symbol struct {
	Symbol string `json:"symbol"`
	Value  int    `json:"Value"`
}

type extern struct {
	FilePath string `json:"FilePath"`
}

type vocWord struct {
	VocWord string `json:"VocWord"`
	Value   int    `json:"Value"`
	VocType int    `json:"VocType"`
}

type objectData struct {
	Value      int `json:"Value"`
	Noun       int `json:"Noun"`
	Adjective  int `json:"Adjective"`
	Container  int `json:"Container"`
	Wearable   int `json:"Wearable"`
	Flags      int `json:"Flags"`
	Weight     int `json:"Weight"`
	InitialyAt int `json:"InitialyAt"`
}

func (o objectData) String() string {
	return fmt.Sprintf("noun %d adj %d at %d weight %d flags %04X container %d wearable %d",
		o.Noun, o.Adjective, o.InitialyAt, o.Weight, o.Flags, o.Container, o.Wearable)
}

type connection struct {
	FromLoc   int `json:"FromLoc"`
	ToLoc     int `json:"ToLoc"`
	Direction int `json:"Direction"`
}

type text struct {
	Value int    `json:"Value"`
	Text  string `json:"Text"`
}

type process struct {
	Value   int     `json:"Value"`
	Entries []entry `json:"entries"`
}

type entry struct {
	Entry    string           `json:"Entry"`
	Verb     int              `json:"Verb"`
	Noun     int              `json:"Noun"`
	Condacts []map[string]any `json:"condacts"`
}

// load reads one of the two files.
//
// repair says the file was written by the other front end and needs rescuing
// before a JSON reader will take it. Ours is written by a JSON encoder and is
// valid by construction; putting it through the same repair would corrupt it.
func load(path string, repair bool) (*document, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if repair {
		body = repairEscapes(latin1ToUTF8(body))
	}

	var doc document
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

// latin1ToUTF8 rescues a file that is not the encoding JSON is defined to be.
//
// The other front end writes its messages as \u escapes but its vocabulary
// words as raw bytes, so a file holding "CADÁVER" comes out as ISO-8859-1
// inside something labelled JSON, and a strict reader sees invalid UTF-8. A
// file that already is valid UTF-8 is left alone — which is every file ours
// writes, and every one of theirs from a source with no accented vocabulary.
func latin1ToUTF8(body []byte) []byte {
	if utf8.Valid(body) {
		return body
	}

	out := make([]rune, 0, len(body))
	for _, b := range body {
		out = append(out, rune(b))
	}

	return []byte(string(out))
}

// repairEscapes makes the other front end's output readable.
//
// It does not escape backslashes. A text carrying "\k" is written out as "\k",
// and a JSON reader either rejects the file or — worse — silently decodes a
// stray "\b" as a backspace that was never in the adventure. Both happen in
// the released games.
//
// Only two backslash sequences are deliberate there: the "\uXXXX" it builds
// its character codes from, and the '\"' of FixDoubleQuotes. Every other
// backslash is a character of the text and is doubled so it survives as one.
func repairEscapes(body []byte) []byte {
	var out []byte

	inString := false

	for i := 0; i < len(body); i++ {
		c := body[i]

		if c != '\\' {
			// A quote not preceded by a backslash opens or closes a string;
			// the loop never reaches one that is, because the branch below
			// consumes the pair.
			if c == '"' {
				inString = !inString
			}

			out = append(out, c)

			continue
		}

		if !inString || i+1 >= len(body) {
			out = append(out, c)

			continue
		}

		// The two it means, kept whole so the character after the backslash is
		// never read as if it started something.
		if body[i+1] == 'u' || body[i+1] == '"' {
			out = append(out, c, body[i+1])
			i++

			continue
		}

		out = append(out, '\\', '\\')
	}

	return out
}

// condact is one instruction, flattened out of the map its keys are spread
// across. The name is what is compared and not the opcode: a fake condact is
// numbered differently by the two and still means the same thing.
type condact struct {
	Name   string
	Params []param
}

type param struct {
	Indirect bool
	Value    int
}

func (c condact) String() string {
	if len(c.Params) == 0 {
		return c.Name
	}

	parts := make([]string, 0, len(c.Params))
	for _, p := range c.Params {
		if p.Indirect {
			parts = append(parts, "@"+fmt.Sprint(p.Value))

			continue
		}

		parts = append(parts, fmt.Sprint(p.Value))
	}

	return c.Name + " " + strings.Join(parts, " ")
}

// readCondact pulls the numbered keys back out.
//
// NumParams says how many there are, and Indirection<n> and Param<n> carry
// each one. A record that leaves NumParams out is taken as having none, which
// is what a condact with no operands looks like.
func readCondact(raw map[string]any) condact {
	c := condact{Name: str(raw["Condact"])}

	n := num(raw["NumParams"])
	for i := 1; i <= n; i++ {
		c.Params = append(c.Params, param{
			Indirect: num(raw[fmt.Sprintf("Indirection%d", i)]) != 0,
			Value:    num(raw[fmt.Sprintf("Param%d", i)]),
		})
	}

	return c
}

func num(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		var parsed int
		_, _ = fmt.Sscanf(n, "%d", &parsed)

		return parsed
	}

	return 0
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	return ""
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
