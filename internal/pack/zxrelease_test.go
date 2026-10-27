package pack

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

// A database big enough to be real and small enough to fit, which is all these
// need of one: nothing here reads it.
func someDatabase() []byte { return make([]byte, 10000) }

// A 48K release is one file, and that file is the tape.
//
// This is the case the whole Release type exists for. A tape already holds the
// interpreter, the database and the graphics glued into one stream, so there is
// nothing to put in a directory but the tape — and asking for a directory has
// to work anyway, because the caller asked for a directory.
func TestA48KReleaseIsTheTapeAndNothingElse(t *testing.T) {
	r, err := ZXRelease(someDatabase(), "48k", Options{
		Language: language.LanguageSpanish, Name: "AO",
	})
	require.NoError(t, err)

	one, err := r.Single()
	require.NoError(t, err, "a 48K release should be one file")
	require.Equal(t, "AO.tap", one.Name)

	// An empty subtarget is a 48K: it is what the compiler is given when
	// nobody says which Spectrum.
	plain, err := ZXRelease(someDatabase(), "", Options{
		Language: language.LanguageSpanish, Name: "AO",
	})
	require.NoError(t, err)
	require.Equal(t, r.Total(), plain.Total())
}

// A +3 release is four files, because a disk machine does the gluing at load
// time instead of having it done beforehand.
func TestAPlus3ReleaseIsFourFilesTheLoaderCanFind(t *testing.T) {
	r, err := ZXRelease(someDatabase(), "plus3", Options{
		Language: language.LanguageSpanish, Name: "AO",
	})
	require.NoError(t, err)

	require.Equal(t,
		[]string{plus3Loader, plus3Interpreter, plus3Database, plus3Graphics},
		[]string{r.Files()[0].Name, r.Files()[1].Name, r.Files()[2].Name, r.Files()[3].Name},
		"the names are the loader's and not ours to choose")

	// DISK with no extension is the one that is not a choice: it is what the
	// +3's own boot menu loads, so a loader by any other name is one nobody
	// runs.
	require.Equal(t, "DISK", r.Files()[0].Name)

	for _, f := range r.Files() {
		require.Equal(t, "PLUS3DOS", string(f.Body[:8]),
			"%s has no +3DOS header, so the machine will not read it as a file", f.Name)
	}

	// The two blocks with a fixed home go where the linker wrote its pointers
	// for, which the header of each file is what states.
	for i, want := range map[int]int{1: 0x6000, 2: 0x8400} {
		body := r.Files()[i].Body
		require.Equal(t, want, int(body[18])|int(body[19])<<8,
			"%s loads at the wrong address", r.Files()[i].Name)
	}

	// The graphics block ends on the last byte of RAM, so where it begins
	// depends on how big it is — and whatever that is, it is above the
	// database and below the top.
	gfx := r.Files()[3].Body
	at := int(gfx[18]) | int(gfx[19])<<8
	require.Greater(t, at, 0x8400+len(someDatabase()))
	require.Equal(t, 0x10000, at+len(gfx)-media.Plus3HeaderSize,
		"the graphics block does not end on the last byte of RAM")
}

// And the loader is a program the +3 will start, not a file that happens to
// hold BASIC.
func TestThePlus3LoaderStartsAtItsFirstLine(t *testing.T) {
	r, err := ZXRelease(someDatabase(), "plus3", Options{
		Language: language.LanguageSpanish, Name: "AO",
	})
	require.NoError(t, err)

	disk := r.Files()[0].Body

	require.Equal(t, byte(0), disk[15], "DISK is not saved as a BASIC program")
	require.Equal(t, media.FirstLine, int(disk[18])|int(disk[19])<<8,
		"saved to start anywhere but the first line, the loader sits at a LOAD it never set up")

	// CLEAR is the first thing it does. The loader lives under RAMTOP, so a
	// block landing in the BASIC area takes the loader with it halfway through.
	const tokCLEAR = 0xFD
	require.Equal(t, byte(tokCLEAR), disk[media.Plus3HeaderSize+4],
		"the first statement is not CLEAR")
}

func TestASpectrumWeDoNotPackageSaysSo(t *testing.T) {
	_, err := ZXRelease(someDatabase(), "128k", Options{Language: language.LanguageSpanish})
	require.ErrorContains(t, err, "48k or plus3")
}
