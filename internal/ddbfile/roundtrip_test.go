package ddbfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/back"
	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/jorgefuertes/QDAAD/internal/pack"
	"github.com/stretchr/testify/require"
)

// This reads what our own compiler writes, which sounds like the least
// interesting test in the repository and was the one that was missing.
//
// Everything else checks one direction. The three comparison harnesses hold
// what we write against what the reference writes; the decompiler's own tests
// read the adventures of 1989 to 1991. Nothing read a database we had just
// built, and so nothing noticed that this reader could not:
//
//   - The token table opens with a marker that is, by definition, a byte no
//     text contains. It was being decoded as text, and there is no such
//     character.
//   - A table was assumed to hold 128 tokens. The databases of 1989 do; ours
//     do not, because the linker keeps only the tokens that earn their place.
//   - And where that table ends is in no pointer of the header: a text
//     pointer names its index, and the strings sit before it. Reading 128
//     anyway walks into the strings and decodes them as tokens.
//
// All three only happen on a database we wrote. A test that goes round the
// loop is the only thing that could have found them.
func TestWeCanReadWhatWeWrite(t *testing.T) {
	// The template is the smallest real source there is, and the one every
	// other harness starts from.
	source := sourceInUTF8(t, "../../work/DRC/BLANK_ES.DSF")

	for _, target := range []struct {
		name      string
		machine   arch.Machine
		subtarget string
	}{
		{"a 48K Spectrum", arch.ZX, "48K"},
		{"a PC", arch.PC, "VGA"},
		// Big-endian, which is the one arrangement a Spectrum build never
		// exercises and where a reader that guessed would guess wrong.
		{"an Atari ST", arch.ST, ""},
	} {
		t.Run(target.name, func(t *testing.T) {
			opts := legacy.Options{
				Machine: target.machine, Subtarget: target.subtarget,
				Language: language.LanguageSpanish, Version: 3,
			}

			db, err := legacy.Analyze(source, opts)
			require.NoError(t, err)

			image, err := back.Link(db, back.Options{
				Machine: opts.Machine, Subtarget: opts.Subtarget,
				Language: opts.Language, Version: opts.Version,
			})
			require.NoError(t, err)

			r, ok := ddbfile.At(image, 0)
			require.True(t, ok, "the reader will not take a database we just built")

			// Read every section. What matters is less what each says than
			// that none of them fails: a reader that cannot get through a
			// database is no use to the interpreter, whatever it reports.
			require.Equal(t, target.machine == arch.ST, r.BigEndian())

			processes, err := r.Processes()
			require.NoError(t, err)
			require.NotEmpty(t, processes)

			for _, read := range []struct {
				what string
				get  func() ([]string, error)
			}{
				{"system messages", r.SystemMessages},
				{"messages", r.Messages},
				{"object texts", r.ObjectTexts},
				{"location texts", r.LocationTexts},
			} {
				texts, err := read.get()
				require.NoError(t, err, read.what)
				require.NotEmpty(t, texts, read.what)
			}

			_, err = r.Connections()
			require.NoError(t, err)

			objects, err := r.Objects()
			require.NoError(t, err)
			require.NotEmpty(t, objects)

			require.NotEmpty(t, r.Vocabulary())
		})
	}
}

// And the one that says the reading is right rather than merely possible: a
// text we know the words of comes back with those words in it.
//
// It is held to the text and not to the bytes on purpose. A compressed text is
// not the characters an author typed — the tokens have been taken out of it —
// so the only thing worth comparing is what it means.
func TestTheTextComesBackSayingWhatItSaid(t *testing.T) {
	source := sourceInUTF8(t, "../../work/DRC/BLANK_ES.DSF")

	db, err := legacy.Analyze(source, legacy.Options{
		Machine: arch.ZX, Subtarget: "48K",
		Language: language.LanguageSpanish, Version: 3,
	})
	require.NoError(t, err)

	image, err := back.Link(db, back.Options{
		Machine: arch.ZX, Subtarget: "48K",
		Language: language.LanguageSpanish, Version: 3,
	})
	require.NoError(t, err)

	r, ok := ddbfile.At(image, 0)
	require.True(t, ok)

	locations, err := r.LocationTexts()
	require.NoError(t, err)
	require.Contains(t, locations[0], "BASE DE DATOS",
		"the template's first location, read back out of the bytes we wrote")

	// The compression table is where all three of the bugs lived, so it gets
	// looked at rather than merely survived.
	require.NotEmpty(t, r.Tokens())

	for i, token := range r.Tokens() {
		require.NotEmpty(t, token, "token %d", i)
		require.Less(t, len(token), 16, "token %d is a token and not a sentence", i)
	}
}

// sourceInUTF8 copies a source the kit ships in ISO-8859-1 into a readable
// temporary one. The front end takes UTF-8 and says so rather than guessing.
func sourceInUTF8(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("%s is not in the checkout: %v", path, err)
	}

	runes := make([]rune, 0, len(raw))
	for _, b := range raw {
		runes = append(runes, rune(b))
	}

	out := filepath.Join(t.TempDir(), filepath.Base(path))
	require.NoError(t, os.WriteFile(out, []byte(string(runes)), 0o600))

	return out
}

// And the same thing through qrun's own container, which is the one that is
// supposed to need no working out at all.
//
// The three machines are chosen for what they disagree about: a Spectrum is
// laid at 0x8400 and packs its sections tight, a PC at zero and aligns them to
// even addresses, and an Atari ST does that with its words the other way
// round. Every one of those is a thing the old reader has to deduce and this
// one is told.
func TestOurOwnContainerNeedsNoGuessing(t *testing.T) {
	source := sourceInUTF8(t, "../../work/DRC/BLANK_ES.DSF")

	for _, target := range []struct {
		name      string
		machine   arch.Machine
		subtarget string
	}{
		{"a 48K Spectrum, laid at 0x8400", arch.ZX, "48K"},
		{"a PC, laid at zero and word-aligned", arch.PC, "VGA"},
		{"an Atari ST, big-endian", arch.ST, ""},
	} {
		t.Run(target.name, func(t *testing.T) {
			opts := legacy.Options{
				Machine: target.machine, Subtarget: target.subtarget,
				Language: language.LanguageSpanish, Version: 3,
			}

			db, err := legacy.Analyze(source, opts)
			require.NoError(t, err)

			image, layout, err := back.LinkWithLayout(db, back.Options{
				Machine: opts.Machine, Subtarget: opts.Subtarget,
				Language: opts.Language, Version: opts.Version,
			})
			require.NoError(t, err)

			wrapped, err := pack.QRun(image,
				int(opts.Machine.BaseAddress(opts.Subtarget)), opts.Machine.IsBigEndian(), layout)
			require.NoError(t, err)

			r, err := ddbfile.OpenQRun(wrapped)
			require.NoError(t, err)

			require.Equal(t, opts.Machine.IsBigEndian(), r.BigEndian())
			require.Equal(t, int(opts.Machine.BaseAddress(opts.Subtarget)), r.Base())

			// The adventure reads the same whatever it was laid out for,
			// which is the whole point of saying so rather than deducing it.
			locations, err := r.LocationTexts()
			require.NoError(t, err)
			require.Contains(t, locations[0], "BASE DE DATOS")

			for i, token := range r.Tokens() {
				require.NotEmpty(t, token, "token %d", i)
				require.NotContains(t, token, "\x00", "token %d is padding, not a token", i)
			}
		})
	}
}

// A file that is not ours says so, and says which part of it was wrong. The
// old reader answers "no" and leaves the rest to guesswork.
func TestTheContainerRefusesPlainly(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		data       []byte
	}{
		{"empty", "too short", nil},
		{"not ours", "not a qrun database", []byte("PK\x03\x04and then some more")},
		{"a version we do not read", "version", append([]byte("QDB1\x7f\x00\x00\x00\x00\x00"), make([]byte, 64)...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ddbfile.OpenQRun(tc.data)
			require.ErrorContains(t, err, tc.want)
		})
	}
}
