package pack

import (
	"strings"
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

// A Spectrum this does not write for says which, and the 128K says why.
//
// The two answers are different on purpose. esxDOS, the Next and the ZX-Uno are
// machines nobody has got to yet, and the error reads like it; the 128K is a
// decision — its interpreter does not do version 3 and was withdrawn from DAAD
// READY — and an error that read "not yet" would be promising work nobody
// intends to do. The +3 is the 128K machine this writes for.
func TestASpectrumWeDoNotPackageSaysSo(t *testing.T) {
	for _, subtarget := range []string{"esxdos", "next", "uno"} {
		_, err := ZXRelease(someDatabase(), subtarget, Options{Language: language.LanguageSpanish})
		require.ErrorContains(t, err, "48k or plus3")
	}

	_, err := ZXRelease(someDatabase(), "128k", Options{Language: language.LanguageSpanish})
	require.ErrorContains(t, err, "ZXDAAD128")
	require.ErrorContains(t, err, "plus3", "it should say what to build instead")
}

// The picture a disk shows while it loads, and the moment it gets to be looked
// at afterwards.
//
// A tape got this for nothing: its last block is tens of kilobytes at 1500
// baud, so the picture is up for most of a minute. A disk reads the same bytes
// in an instant, so without the PAUSE the picture would be gone about as fast
// as it arrived — which is why the wait is here and not in ZX48.
func TestADiskShowsItsLoadingPictureAndWaits(t *testing.T) {
	loading := make([]byte, screenSize)
	loading[0] = 0xAA

	release, err := zxPlus3(make([]byte, 2000), Options{
		Language: language.LanguageSpanish,
		Name:     "prueba",
		Loading:  loading,
	})
	require.NoError(t, err)

	// A file of its own, addressed at video memory, and first after the loader
	// so that it is on screen while the rest goes in.
	files := release.Files()
	require.Equal(t, plus3Loader, files[0].Name)
	require.Equal(t, plus3Loading, files[1].Name)

	require.Equal(t, loading, media.WithoutPlus3Header(files[1].Body),
		"the picture itself, under the 128-byte +3DOS header")

	// And the loader: the picture, then the POKE that quiets the ROM, then
	// everything else, then three seconds, then the jump.
	program := media.WithoutPlus3Header(files[0].Body)

	loader := string(program)
	require.Contains(t, loader, plus3Loading, "the loader never asks for it")

	picture := strings.Index(loader, plus3Loading)
	interpreter := strings.Index(loader, plus3Interpreter)
	require.Less(t, picture, interpreter, "the picture loads after the rest of the disk")

	require.Contains(t, string(program), string([]byte{0xF2}), "no PAUSE before the jump")

	// A disk with no picture is the disk it always was.
	plain, err := zxPlus3(make([]byte, 2000), Options{
		Language: language.LanguageSpanish, Name: "prueba",
	})
	require.NoError(t, err)

	for _, f := range plain.Files() {
		require.NotEqual(t, plus3Loading, f.Name)
	}

	bare := media.WithoutPlus3Header(plain.Files()[0].Body)
	require.NotContains(t, string(bare), string([]byte{0xF2}),
		"nothing to look at, so nothing to wait for")
}
