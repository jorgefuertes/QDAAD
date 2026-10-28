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
	"github.com/jorgefuertes/QDAAD/internal/ddb"
)

// Analyze reads a source and returns the database it describes.
//
// The options are not a convenience. A source asks what it is being built for
// and branches on the answer, so the same file compiled for a Spectrum and for
// an Amiga is meant to give two different databases.
func Analyze(filename string, opts Options) (*ddb.DDB, error) {
	db, _, err := Compile(filename, opts)

	return db, err
}

// Compile is Analyze with what the directives left behind kept as well.
//
// Three of the things a build needs are not in a database and never will be:
// the symbols, the externals to link and the flags the source set. A caller
// that only wants the world can use Analyze and ignore them; one that has to
// write the intermediate form, or hand the whole thing to a back end, needs
// both halves.
func Compile(filename string, opts Options) (*ddb.DDB, *Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, nil, err
	}

	res, err := Read(filename, opts)
	if err != nil {
		return nil, nil, err
	}

	// What the header comments say, which the scanner never sees because they
	// are comments. The directives have had their turn by now, so this fills
	// in beside them rather than over them. See settings.go.
	header, err := readHeaderSettings(filename)
	if err != nil {
		return nil, nil, err
	}

	if header.Charset != "" {
		res.Settings.Charset = header.Charset
	}

	db, err := Parse(res, opts)
	if err != nil {
		return nil, nil, err
	}

	return db, res, nil
}

// Read runs the passes that come before the parser: it resolves the includes,
// scans, and preprocesses. What comes back is the token stream the parser is
// meant to walk, along with the symbols, the externals and the flags the
// directives left behind.
//
// It is exported because those three passes are worth using — and testing — on
// their own, without a database to fill in.
func Read(filename string, opts Options) (*Result, error) {
	lines, err := Resolve(filename)
	if err != nil {
		return nil, err
	}

	tokens, err := Scan(lines)
	if err != nil {
		return nil, err
	}

	return Preprocess(tokens, opts)
}
