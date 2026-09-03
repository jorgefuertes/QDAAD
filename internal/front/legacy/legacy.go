// Package legacy reads a DSF source — the format DAAD READY version 3 uses —
// and turns it into the structures of internal/ddb.
//
// It runs in four passes, and only the first knows what a line is:
//
//  1. Resolve pulls in the #include files, keeping where every line came from.
//  2. Scan turns those lines into tokens.
//  3. Preprocess resolves #define and the conditionals, and hands back the
//     tokens that survive.
//  4. Parse reads those tokens by sections and fills in the database.
//
// Working on tokens rather than on lines is what makes the whole of the
// language readable: a line is not a unit of it. The template DAAD READY ships
// has entries whose verb, noun and first condact are all on one line, and a
// condact whose parameters are on the next one is just as legal.
package legacy

import (
	"errors"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// ErrNotImplemented marks the passes that are not written yet.
var ErrNotImplemented = errors.New("not implemented yet")

// Analyze reads a source and returns the database it describes.
func Analyze(filename string) (*ddb.DDB, error) {
	tokens, _, err := Read(filename)
	if err != nil {
		return nil, err
	}

	_ = tokens

	return nil, ErrNotImplemented
}

// Read runs the passes that come before the parser: it resolves the includes,
// scans, and preprocesses. What comes back is the token stream the parser is
// meant to walk, and the symbols #define left behind.
//
// It is exported because those three passes are worth using — and testing — on
// their own, without a database to fill in.
func Read(filename string) ([]Token, Symbols, error) {
	lines, err := Resolve(filename)
	if err != nil {
		return nil, nil, err
	}

	tokens, err := Scan(lines)
	if err != nil {
		return nil, nil, err
	}

	return Preprocess(tokens)
}
