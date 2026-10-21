package pack

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
	"github.com/stretchr/testify/require"
)

// The tape is a loader and three blocks, and every one of the three has to land
// where the interpreter expects it. None of the addresses is ours to pick: the
// database's was written into its own pointers by the linker, the stub's is
// pinned to the top of memory, and the interpreter is not relocatable.
//
// What the whole of this is held against is script/compare-daadmaker.sh, which
// packs the same database with the reference and compares the bytes. These
// check the shape so that a failure there has somewhere to start.
func TestTheTapeLandsEveryPieceWhereItBelongs(t *testing.T) {
	tape, err := ZX48(make([]byte, 2000), Options{
		Language: language.LanguageSpanish, Name: "prueba",
	})
	require.NoError(t, err)

	var at, length []int

	var names []string

	for _, block := range split(t, tape) {
		if block[0] != 0x00 || block[1] != 3 {
			continue // not a CODE header
		}

		names = append(names, string(block[2:12]))
		length = append(length, int(block[12])|int(block[13])<<8)
		at = append(at, int(block[14])|int(block[15])<<8)
	}

	require.Len(t, at, 3, "the interpreter, the database, and the index with the stub")

	require.Equal(t, 0x6000, at[0], "the interpreter is not relocatable")
	require.Equal(t, 0x8400, at[1], "and the database's pointers were written from here")
	require.Equal(t, 2000, length[1])

	// The last byte of the block is the last byte of RAM. The interpreter
	// reads fixed addresses inside those final bytes, so this is not a
	// convention that could be changed.
	require.Equal(t, 0x10000, at[2]+length[2])

	// The name is nine characters and a letter saying which part this is.
	require.Equal(t, []string{"PRUEBA   I", "PRUEBA   D", "PRUEBA   G"}, names)
}

// A database too big to fit is the one failure a real adventure will meet, and
// it has to be said rather than written out as a tape that loads over its own
// index and dies without explaining itself.
func TestADatabaseThatDoesNotFitIsRefused(t *testing.T) {
	_, err := ZX48(make([]byte, 0x8000), Options{Language: language.LanguageSpanish})

	require.ErrorContains(t, err, "is all there is")
}

// The index is read downwards from its last byte, which is why it is written
// backwards. A build with no pictures still carries one: without it the
// interpreter reads uninitialised memory the first time a PICTURE runs, and the
// official template runs one before printing anything.
func TestTheEmptyIndexStillSaysThereAreNoPictures(t *testing.T) {
	index := imageIndex(0x8400 + 2000)

	require.Equal(t, []byte{
		0xFF, // no more pictures
		0xFF, // no more pages
		0x00, 0x00,
		0x00, 0xC0,
	}, index, "written backwards, because it is read backwards")

	// Once the database runs past 0xC000 the pictures go after it instead, so
	// that the two never overlap.
	require.Equal(t, []byte{0x01, 0xD0}, imageIndex(0xD000)[4:],
		"0xD001, still backwards")
}

// The font that ends up on screen is the one written over the stub, not the one
// the stub was shipped with. Every ZX build of DAAD READY passes AD8x6.CHR and
// so does this, which is why the two fonts differ in a hundred-odd glyphs and
// it has never mattered.
func TestTheFontIsWrittenOverTheOneTheStubCarries(t *testing.T) {
	companions, err := runtime.Companions(runtime.ZX48)
	require.NoError(t, err)

	font := companions["AD8x6.CHR"]
	require.Len(t, font, charsetSize)

	tape, err := ZX48(make([]byte, 2000), Options{Language: language.LanguageSpanish})
	require.NoError(t, err)

	blocks := split(t, tape)
	stub := blocks[len(blocks)-1][1 : len(blocks[len(blocks)-1])-1]

	at := len(stub) - charsetSize - charsetTrailer
	require.Equal(t, font, stub[at:at+charsetSize])

	// And the trailer after it is untouched, which is what the interpreter
	// reads to find the font in the first place. The block ends on the last
	// byte of RAM, so the word at 0xFFF7 is nine bytes from its end.
	pointer := stub[len(stub)-9:]
	require.Equal(t, topOfMemory-len(stub)+at,
		int(pointer[0])|int(pointer[1])<<8,
		"the pointer at 0xFFF7 still names where the font starts")
}

// A charset may come with a 128-byte header, and the longer kind is read past
// rather than refused. Anything else is neither.
func TestACharsetIsTwoKilobytesWithOrWithoutItsHeader(t *testing.T) {
	stub := make([]byte, charsetSize+charsetTrailer)

	require.NoError(t, replaceCharset(stub, nil), "none at all leaves the stub alone")

	plain := make([]byte, charsetSize)
	plain[0] = 0xAA
	require.NoError(t, replaceCharset(stub, plain))
	require.Equal(t, byte(0xAA), stub[0])

	headed := make([]byte, charsetHeader+charsetSize)
	headed[charsetHeader] = 0xBB
	require.NoError(t, replaceCharset(stub, headed))
	require.Equal(t, byte(0xBB), stub[0], "the header is skipped, not written")

	require.ErrorContains(t, replaceCharset(stub, make([]byte, 100)), "and this one is 100")
}

func split(t *testing.T, tape []byte) [][]byte {
	t.Helper()

	var blocks [][]byte

	for at := 0; at < len(tape); {
		require.Less(t, at+2, len(tape))

		length := int(tape[at]) | int(tape[at+1])<<8
		at += 2

		require.LessOrEqual(t, at+length, len(tape))

		blocks = append(blocks, tape[at:at+length])
		at += length
	}

	return blocks
}
