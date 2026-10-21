package back

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/stretchr/testify/require"
)

// A word of the vocabulary is five bytes and has no room to be clever.
//
// The order of the two steps is what this is really about: encode first, then
// uppercase, and only what is ASCII. The other way round an accented letter
// becomes a capital the DAAD character set does not have, and the encoder
// spells that as a graphic — three bytes where one was wanted, and a word the
// parser will never match.
func TestAVocabularyWordIsFiveBytesOfTheDaadCharacterSet(t *testing.T) {
	for _, tc := range []struct {
		name, word string
		want       []byte
	}{
		{"short words are padded with spaces", "DA", []byte("DA   ")},
		{"long ones are cut to five", "DESCRIBIR", []byte("DESCR")},
		{"lower case is raised", "dar", []byte("DAR  ")},
		{
			// The store files a word upper-cased, so this is the shape the
			// linker actually sees for "CABRó".
			name: "an accented capital falls back to its only code",
			word: "CABRÓ",
			want: []byte{'C', 'A', 'B', 'R', 0x18},
		},
		{
			// The set has both cases of these three, so neither is folded.
			name: "the enye keeps its own capital",
			word: "AÑO",
			want: []byte{'A', 0x1B, 'O', ' ', ' '},
		},
		{"and so does the u with diaeresis", "Ü", []byte{0x1F, ' ', ' ', ' ', ' '}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := vocabularyWord(tc.word)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// A pause is written in units of the note a machine takes as whole, and that
// is not the same number of milliseconds everywhere. So the operand is not the
// one the source wrote: it is rescaled, or a pause written once lasts a
// different time on each machine it is built for.
func TestAPauseIsRescaledToTheMachine(t *testing.T) {
	for _, tc := range []struct {
		name      string
		machine   arch.Machine
		subtarget string
		wrote     ddb.ID
		want      byte
	}{
		{"a 48K Spectrum counts at 195", arch.ZX, "48K", 200, 195},
		{"a 128K one counts at 100", arch.ZX, "128K", 200, 100},
		{"a PC is the unit itself", arch.PC, "VGA", 200, 200},
		{"a CPC counts slower", arch.CPC, "", 100, 150},
		{"and cannot be asked for more than a byte", arch.CPC, "", 200, 255},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := &linker{opts: Options{Machine: tc.machine, Subtarget: tc.subtarget}}

			got := l.operand(
				ddb.Condact{Opcode: opPause},
				0,
				ddb.Param{Value: tc.wrote},
			)

			require.Equal(t, tc.want, got)
		})
	}

	// An indirect operand names a flag, and what the flag holds at run time is
	// not ours to touch.
	t.Run("an indirect pause is left alone", func(t *testing.T) {
		l := &linker{opts: Options{Machine: arch.ZX, Subtarget: "48K"}}

		got := l.operand(
			ddb.Condact{Opcode: opPause},
			0,
			ddb.Param{Value: 200, Indirect: true},
		)

		require.Equal(t, byte(200), got)
	})
}

// The header says which machine and which parser, and the parser is one bit.
func TestTheHeaderNamesTheMachineAndTheParser(t *testing.T) {
	for _, tc := range []struct {
		name    string
		machine arch.Machine
		lang    language.Language
		want    byte
	}{
		{"a Spectrum speaking Spanish", arch.ZX, language.LanguageSpanish, 0x11},
		{"a Spectrum speaking English", arch.ZX, language.LanguageEnglish, 0x10},
		{"a PC speaking Spanish", arch.PC, language.LanguageSpanish, 0x01},
		{"an Amiga speaking English", arch.AMIGA, language.LanguageEnglish, 0x60},
	} {
		t.Run(tc.name, func(t *testing.T) {
			image, err := Link(bare(t), Options{
				Machine: tc.machine, Subtarget: "48K",
				Language: tc.lang, Version: 3,
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, image[1])
		})
	}
}

// Every address in the header has to point inside the database, and the last
// one has to be its end. A pointer left at zero is a section never laid.
func TestEveryPointerLandsInsideTheDatabase(t *testing.T) {
	const base = 0x8400

	image, err := Link(bare(t), Options{
		Machine: arch.ZX, Subtarget: "48K",
		Language: language.LanguageSpanish, Version: 3,
	})
	require.NoError(t, err)

	read := func(slot int) int {
		at := counters + 2*slot

		return int(image[at]) | int(image[at+1])<<8
	}

	require.Equal(t, base+len(image), read(ptrEndAddress),
		"the last word is where the database ends")

	for slot := range ptrEndAddress {
		// The compression table is the one that is allowed to be absent: a
		// database whose texts no token pays for carries none, and says so
		// with a zero where its address would be.
		if slot == ptrTokens && read(slot) == 0 {
			continue
		}

		require.GreaterOrEqual(t, read(slot), base+headerSize, "pointer %d", slot)
		require.Less(t, read(slot), base+len(image), "pointer %d", slot)
	}
}

// A database laid for a big-endian machine is the same database with its words
// the other way round, which is the one thing a Spectrum build never exercises.
//
// The two machines compared are the PC and the Atari ST, because they agree
// about everything else: both load at zero and both want their words aligned,
// so every address comes out the same number and only its bytes are swapped.
func TestTheByteOrderFollowsTheMachine(t *testing.T) {
	little, err := Link(bare(t), Options{
		Machine: arch.PC, Subtarget: "VGA",
		Language: language.LanguageSpanish, Version: 3,
	})
	require.NoError(t, err)

	big, err := Link(bare(t), Options{
		Machine:  arch.ST,
		Language: language.LanguageSpanish, Version: 3,
	})
	require.NoError(t, err)

	require.Len(t, big, len(little), "the same database either way round")

	for slot := range pointers {
		at := counters + 2*slot

		require.Equal(t, little[at], big[at+1], "pointer %d, low byte", slot)
		require.Equal(t, little[at+1], big[at], "pointer %d, high byte", slot)
	}
}

// bare is the smallest database that can be laid: one of everything the header
// counts, so that no section is empty and every pointer has somewhere to go.
func bare(t *testing.T) *ddb.DDB {
	t.Helper()

	db := ddb.New()
	db.NullWord = '_'

	require.NoError(t, db.Words().NewLegacy(1, ddb.Noun, "NORTE"))
	require.NoError(t, db.Words().NewLegacy(20, ddb.Verb, "COGER"))
	require.NoError(t, db.Locations().NewLegacy(0, "Una cueva."))
	require.NoError(t, db.Objects().NewLegacy(ddb.Object{
		ID: 0, Description: "una llave", Name: 30, Adjective: 2, Weight: 3,
	}))
	require.NoError(t, db.Messages().NewLegacy(0, ddb.UserMessage, "No puedes."))
	require.NoError(t, db.Messages().NewLegacy(0, ddb.SystemMessage, "Está oscuro."))

	db.Conns().Add(0, 1, 0)

	require.NoError(t, db.Processes().NewLegacy(ddb.Process{
		ID: 0,
		Entries: []ddb.Entry{{
			Verb: 20, Noun: 30,
			Condacts: []ddb.Condact{{Opcode: 22}}, // DONE
		}},
	}))

	return db
}
