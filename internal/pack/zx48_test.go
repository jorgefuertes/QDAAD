package pack

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
	"github.com/jorgefuertes/QDAAD/internal/zx0"
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
	page, _, err := zxPictures(nil, 0x8400+2000, 2099)
	require.NoError(t, err)

	require.Equal(t, []byte{
		0xFF, // no more pictures
		0xFF, // no more pages
		0x00, 0x00,
		0x00, 0xC0,
	}, imageIndex(page), "written backwards, because it is read backwards")

	// Once the database runs past 0xC000 the pictures go after it instead, so
	// that the two never overlap.
	page, _, err = zxPictures(nil, 0xD000, 2099)
	require.NoError(t, err)
	require.Equal(t, []byte{0x01, 0xD0}, imageIndex(page)[4:], "0xD001, still backwards")
}

// And a build with pictures says where each one is, in the same backwards
// order: three bytes a picture, number first and then the address.
func TestTheIndexPointsAtEachPicture(t *testing.T) {
	page, dropped, err := zxPictures(map[int][]byte{
		3: make([]byte, screenSize),
		7: make([]byte, screenSize),
	}, 0x8400+2000, 2099)
	require.NoError(t, err)
	require.Empty(t, dropped, "two blank screens fit in a 48K with room to spare")

	index := imageIndex(page)

	// Backwards: end marker, then picture 7, then picture 3, then the page.
	require.EqualValues(t, 0xFF, index[0], "no more pictures")
	require.EqualValues(t, 7, index[3], "the second picture's number")
	require.EqualValues(t, 3, index[6], "the first picture's number")

	// Picture 3 starts where the page does, and 7 after it.
	require.Equal(t, 0xC000, page.starts[3])
	require.Greater(t, page.starts[7], page.starts[3])

	at := int(index[5])<<8 | int(index[4])
	require.Equal(t, page.starts[3], at, "the address of picture 3, backwards")
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

// A picture, from a screen to a tape and back out of it the way the
// interpreter reads one.
//
// This is the whole chain in one test, because every link of it is a place to
// get an address wrong and none of them fails loudly: the screen is cut into
// runs of video memory, each run is compressed, the runs are laid out in page
// zero, and the index says where each picture's runs begin. A tape that gets
// any of that wrong is a tape that loads and draws rubbish.
func TestAPictureSurvivesBeingPackagedAndReadBack(t *testing.T) {
	// A screen with something recognisable in it: a diagonal of pixels and an
	// attribute per cell, so that a layout mistake moves something.
	screen := make([]byte, screenSize)
	for y := range imageLines {
		at := (y>>6)*2048 + (y&7)*256 + ((y>>3)&7)*bytesWide
		screen[at+y/8] = 0x80 >> (y % 8)
	}

	for cell := range imageLines / 8 * bytesWide {
		screen[6144+cell] = byte(cell)
	}

	tape, err := ZX48(make([]byte, 2000), Options{
		Language: language.LanguageSpanish,
		Images:   map[int][]byte{9: screen},
	})
	require.NoError(t, err)

	ram := loadTape(t, tape)

	// The index ends on the byte below the graphics stub and is read down
	// from there: page zero's address and size, a marker, then three bytes a
	// picture, then a marker.
	at := stubOrigin - 1
	next := func() byte { b := ram[at]; at--; return b }

	pageAt := int(next())<<8 | int(next())
	pageSize := int(next())<<8 | int(next())
	require.EqualValues(t, 0xFF, next(), "the no-more-pages marker")
	require.Positive(t, pageSize)

	require.EqualValues(t, 9, next(), "the picture's number")
	where := int(next())<<8 | int(next())
	require.EqualValues(t, 0xFF, next(), "the no-more-pictures marker")
	require.GreaterOrEqual(t, where, pageAt, "the picture is not inside page zero")

	// And now what the interpreter does with it: every block to the address
	// it names, until the one that names zero.
	var drawn [screenSize]byte

	for p := where; ; {
		dest := int(ram[p]) | int(ram[p+1])<<8
		if dest == 0 {
			break
		}

		size := int(ram[p+2]) | int(ram[p+3])<<8

		body, err := zx0.Decompress(ram[p+4 : p+4+size])
		require.NoError(t, err)

		copy(drawn[dest-screenBase:], body)
		p += 4 + size
	}

	// Everything the picture covers came back; what it does not cover was
	// never carried and is left alone, which is why this compares the blocks
	// rather than the whole screen.
	blocks, err := screenBlocks(imageLines)
	require.NoError(t, err)

	for _, b := range blocks {
		from := b.at - screenBase
		require.Equal(t, screen[from:from+len(b.body)], drawn[from:from+len(b.body)],
			"the run at 0x%04X did not come back", b.at)
	}
}

// stubOrigin is where the graphics stub is pinned, which is what the index is
// found by: it ends on the byte below it.
const stubOrigin = 0xF7D7

// loadTape plays a .TAP into 64K of memory the way the ROM would: a header
// says where the block after it goes.
func loadTape(t *testing.T, tape []byte) []byte {
	t.Helper()

	ram := make([]byte, 0x10000)
	load := 0

	for at := 0; at+2 <= len(tape); {
		length := int(tape[at]) | int(tape[at+1])<<8
		at += 2

		require.LessOrEqual(t, at+length, len(tape), "the tape ends mid-block")

		body := tape[at : at+length]
		at += length

		if body[0] == 0x00 {
			load = int(body[14]) | int(body[15])<<8

			continue
		}

		copy(ram[load:], body[1:len(body)-1])
	}

	return ram
}
