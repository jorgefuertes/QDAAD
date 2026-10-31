// Package pack turns a linked database into something a machine will load.
//
// The linker's job ends at the .DDB, which is the adventure and nothing else:
// no interpreter, no loader, no idea of what a tape is. This is the step after
// — put the interpreter, the database and whatever else the machine needs into
// the container it boots from, at the addresses it expects them.
//
// The reference does this with daadmaker, whose source is DRC's
// work/DRC/src/daadmaker.pas, and what is written here follows it: the same
// addresses, the same three blocks, the same loader. That is what
// script/compare-daadmaker.sh checks, and it is the third of the three
// comparisons this repository rests on — the front end against drf, the linker
// against drb, and the tape against daadmaker.
package pack

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
)

// The 48K Spectrum's memory, once the loader has finished:
//
//	0x6000..        the interpreter
//	       ..0x83FF its workspace, which it writes into and we must not
//	0x8400..        the database
//	       ..       whatever is left
//	       ..       the image index, read downwards from its last byte
//	     ..0xFFFF   the graphics stub, ending on the last byte of RAM
//
// The last two travel as one block, index first, because the index is only ever
// read as the thing immediately below the stub (daadmaker.pas:312-326).
//
// Three of those four addresses are forced. The database's is the linker's
// already — every pointer inside it was written as an absolute address from
// 0x8400 — so loading it anywhere else makes every pointer wrong. The stub's is
// forced from the other end, below. Only the interpreter's is a choice, and it
// is the interpreter's own: it is not relocatable, and 0x6000 is where it was
// assembled to run.
//
// Between the interpreter's last byte and 0x8400 is its workspace, which looks
// like free memory and is not. It is written into from the moment the
// interpreter starts, and anything put there is overwritten before the first
// word of text is printed.
const interpreterOrigin = 0x6000

// The graphics stub is loaded so that its last byte is the last byte of RAM,
// which is not a convention but an address it is pinned to: the interpreter
// reads fixed locations in the last sixteen bytes of memory to find out where
// everything in the stub is, so the stub has to end exactly there or those
// reads find whatever happens to be lying around.
const topOfMemory = 0x10000

// What the stub holds, in the order it holds it: a short header, a font of 256
// glyphs eight bytes each, and a trailer of pointers at the very top.
//
// The name says graphics and the file is called DAAD.SDG, which is why it took
// a while to work out that a build with no pictures in it needs the thing
// anyway — and needs it more than it needs anything optional. Without it the
// interpreter reads its own pointers out of uninitialised memory and runs off
// into nothing before printing a character.
//
// The font is the one that ends up on screen, and it is replaceable: a .CHR
// file is written straight over it, 2048 bytes ending 28 bytes before the end
// of the stub, which is to say just under the trailer (daadmaker.pas:320-325).
// Every ZX build of DAAD READY passes AD8x6.CHR and so does this one, so the
// stub's own font is never what is seen — which is why the two differ in a
// hundred-odd glyphs and it does not matter.
const (
	// charsetSize is a font: 256 glyphs of 8 bytes.
	charsetSize = 2048
	// charsetTrailer is what follows the font inside the stub, and the only
	// way the font's position is expressed. The reference counts 2076 back
	// from the end of the stub and the index together, which is the same
	// number written less plainly.
	charsetTrailer = 2076 - charsetSize
	// charsetHeader is the one a .CHR may carry. Some are 2048 bytes and some
	// are 2048 plus this, and the longer kind is read past rather than
	// refused.
	charsetHeader = 128
)

// Options is what a tape can be told beyond the database itself.
type Options struct {
	// Language picks the interpreter. The Spectrum's carries its own system
	// messages, so there is one per language and they are not interchangeable.
	Language language.Language
	// Name goes in the block headers, where the ROM shows it while loading.
	// Ten characters, upper case, and the tenth is taken for a letter saying
	// which part this is.
	Name string
	// Charset replaces the font the graphics stub carries. Empty takes the
	// built-in AD8x6.CHR, which is what every ZX build of DAAD READY passes;
	// the stub's own font is never what anybody sees.
	Charset []byte
	// Images are the pictures, by the number a PICTURE names them with, each
	// one a whole Spectrum screen of 6912 bytes. What is carried is the top of
	// each, down to its own last row with a pixel in it — see pictureLines —
	// compressed and indexed; what will not fit is left out and said so
	// through Note.
	Images map[int][]byte
	// Note is told anything the person packaging should know but that is not
	// an error — which so far is one thing, and it is the one thing a
	// packager must never do quietly: pictures left out for want of room. A
	// build that silently carries four of forty-one looks like a build that
	// carries them all until somebody plays it.
	Note func(string)
}

// ZX48 builds a .TAP for a 48K Spectrum.
//
// The tape is a BASIC loader and three CODE blocks: the interpreter, the
// database, and the image index and graphics stub together. The loader takes
// the memory above the interpreter out of BASIC's reach before loading anything
// into it, which is the whole reason it cannot simply be three LOADs.
// zxBlocks is the three blocks a Spectrum build is made of, wherever they are
// going to be put.
//
// A tape and a +3 disk carry the same things to the same addresses and differ
// only in who assembles them, so this is what they share: the interpreter for
// the target asked for, page zero with the pictures in it, the index and
// graphics stub as one block, and the checks that say whether any of it fits.
func zxBlocks(target runtime.Target, database []byte, opts Options) (zxParts, error) {
	_, interpreter, err := runtime.Interpreter(target, opts.Language)
	if err != nil {
		return zxParts{}, err
	}

	// The stub and the font come from the 48K set whatever the target: they are
	// data rather than code, and every ZX interpreter reads the same two.
	companions, err := runtime.Companions(runtime.ZX48)
	if err != nil {
		return zxParts{}, err
	}

	stub, carried := companions["DAAD.SDG"]
	if !carried {
		return zxParts{}, fmt.Errorf("the built-in graphics stub is missing")
	}

	// Read from the machine rather than written here, so that the address the
	// loader puts it at and the addresses the linker wrote inside it cannot
	// drift apart. A Spectrum's base does not depend on its subtarget.
	databaseOrigin := int(arch.ZX.BaseAddress(""))

	page, dropped, err := zxPictures(opts.Images, databaseOrigin+len(database), len(stub))
	if err != nil {
		return zxParts{}, err
	}

	// The index and the stub are one block, index first, ending on the last
	// byte of RAM.
	top := append(imageIndex(page), stub...)

	charset := opts.Charset
	if len(charset) == 0 {
		charset = companions["AD8x6.CHR"]
	}

	if err := replaceCharset(top, charset); err != nil {
		return zxParts{}, err
	}

	if len(dropped) > 0 && opts.Note != nil {
		opts.Note(fmt.Sprintf("%d of the %d pictures did not fit and are not on this build: %s",
			len(dropped), len(opts.Images), numberList(dropped)))
	}

	out := zxParts{
		interpreter: interpreter,
		page:        page,
		top:         top,
		topOrigin:   topOfMemory - len(top),
		dropped:     dropped,
	}

	if end := interpreterOrigin + len(interpreter); end > databaseOrigin {
		return zxParts{}, fmt.Errorf(
			"the interpreter ends at 0x%04X and the database starts at 0x%04X", end, databaseOrigin)
	}

	// The one limit a real adventure will meet. There is no paging on a 48K
	// and nothing to move out of the way: the database has to fit between the
	// interpreter's workspace and the stub, and if it does not, the build is
	// for a bigger machine.
	if end := databaseOrigin + len(database); end > out.topOrigin {
		return zxParts{}, fmt.Errorf(
			"the database is %d bytes and %d is all there is between 0x%04X and the graphics stub at 0x%04X",
			len(database), out.topOrigin-databaseOrigin, databaseOrigin, out.topOrigin)
	}

	return out, nil
}

// zxParts is what a Spectrum build is made of, wherever it is going to be put.
type zxParts struct {
	interpreter []byte
	// page is page zero: the pictures, at the address it says.
	page imagePage
	// top is the image index and the graphics stub, as one block ending on the
	// last byte of RAM.
	top       []byte
	topOrigin int
	// dropped is the pictures there was no room for, in number order.
	dropped []int
}

// zxPictures lays the pictures out between the database and the stub.
//
// Where they go is not a choice either: 0xC000, or straight after the database
// when the database has already run past that, which is what pager48k.php
// works out and what the interpreter assumes. What is left over for them is
// everything up to the stub.
func zxPictures(images map[int][]byte, databaseEnd, stub int) (imagePage, []int, error) {
	const pictureData = 0xC000

	at := pictureData
	if databaseEnd >= pictureData {
		at = databaseEnd + 1
	}

	if len(images) == 0 {
		return imagePage{at: at, starts: map[int]int{}}, nil, nil
	}

	// The index grows downwards from the stub, so the room for pictures is
	// what is left once the stub and the index's own fixed part are out of it.
	const indexFixed = 6

	room := topOfMemory - at - stub - indexFixed

	return buildImagePage(images, at, room)
}

func ZX48(database []byte, opts Options) ([]byte, error) {
	built, err := zxBlocks(runtime.ZX48, database, opts)
	if err != nil {
		return nil, err
	}

	databaseOrigin := int(arch.ZX.BaseAddress(""))

	// The letter is the tenth character of the name, which is why the name has
	// nine to itself. I for the interpreter, D for the database, G for the
	// stub: daadmaker.pas:305-317.
	parts := []struct {
		letter byte
		at     int
		body   []byte
	}{
		{'I', interpreterOrigin, built.interpreter},
		{'D', databaseOrigin, database},
		{'G', built.topOrigin, built.top},
	}

	// P for the pictures, which are a block only when there are some. It
	// loads after the database and before the stub, which is where it lives.
	if len(built.page.body) > 0 {
		parts = append(parts, struct {
			letter byte
			at     int
			body   []byte
		}{'P', built.page.at, built.page.body})
	}

	var loader media.BASIC

	// RAMTOP goes to the byte below the interpreter, so that everything from
	// there up belongs to us. CLEAR also empties the variables and resets the
	// stack, which is why nothing can usefully come before it.
	loader.Clear(interpreterOrigin - 1)
	loader.LoadCode(len(parts))
	loader.RandomizeUSR(interpreterOrigin)

	var tape media.TAP

	tape.Program(blockName(opts.Name, 0), loader.Bytes(), media.FirstLine)

	for _, part := range parts {
		if err := tape.Code(blockName(opts.Name, part.letter), part.at, part.body); err != nil {
			return nil, err
		}
	}

	return tape.Bytes(), nil
}

// blockName pads a name to the ten characters a tape header has room for and
// puts the part's letter in the last of them. The program block keeps the plain
// name, trailing spaces and all.
func blockName(name string, letter byte) string {
	padded := []byte(strings.ToUpper(name) + "          ")[:10]

	if letter != 0 {
		padded[9] = letter
	}

	return string(padded)
}

// replaceCharset writes a font over the one the graphics stub carries.
//
// The font sits 2076 bytes from the end of the block, which is the position the
// reference computes and the one the stub's own trailer points at. A .CHR is
// 2048 bytes, or that plus a 128-byte header which is read past.
// charsetGlyphs is the 2048 bytes of a font, with the header some .CHR files
// carry read past.
//
// Which end it goes at is the caller's business and differs by machine: a
// Spectrum's font sits a fixed distance from the end of the graphics block and
// a CPC's a fixed distance from the start of it.
func charsetGlyphs(charset []byte) ([]byte, error) {
	switch len(charset) {
	case charsetSize:
		return charset, nil
	case charsetSize + charsetHeader:
		return charset[charsetHeader:], nil
	default:
		return nil, fmt.Errorf("a charset is %d bytes, or %d with a header, and this one is %d",
			charsetSize, charsetSize+charsetHeader, len(charset))
	}
}

func replaceCharset(top, charset []byte) error {
	if len(charset) == 0 {
		return nil
	}

	charset, err := charsetGlyphs(charset)
	if err != nil {
		return err
	}

	at := len(top) - charsetSize - charsetTrailer
	if at < 0 {
		return fmt.Errorf("the graphics stub has no room for a font: %d bytes", len(top))
	}

	copy(top[at:], charset)

	return nil
}

// imageIndex writes the table that says where the pictures are.
//
// A build with no pictures still needs one, and that is the whole reason this
// function exists rather than being left out as an optimisation. The
// interpreter looks the table up whenever a PICTURE runs, and the official
// template runs one before it prints anything; with no table it reads whatever
// the machine happened to have in that memory and goes wherever that points.
//
// It is read *downwards* from its last byte, which is why it is assembled
// forwards and then reversed — the same thing pager48k.php does, and for the
// same reason. Laid out in the order the interpreter reads it:
//
//	a word: where page zero, the picture data, was loaded
//	a word: how many bytes of it there are
//	0xFF:   no more pages
//	then three bytes per picture — its number and where it starts —
//	0xFF:   no more pictures
//
// Both words are read most significant byte first, which is the other way round
// from everything else on this machine and follows from the reversal.
//
// A build with no pictures writes the two end markers and nothing between
// them, which is the case that has to keep working: most of the corpus has no
// bitmaps at all.
func imageIndex(page imagePage) []byte {
	const (
		noMorePages    = 0xFF
		noMorePictures = 0xFF
	)

	forwards := []byte{
		byte(page.at >> 8), byte(page.at),
		byte(len(page.body) >> 8), byte(len(page.body)),
		noMorePages,
	}

	// Three bytes a picture, in number order so that two builds of the same
	// adventure come out the same file.
	numbers := make([]int, 0, len(page.starts))
	for n := range page.starts {
		numbers = append(numbers, n)
	}

	sort.Ints(numbers)

	for _, n := range numbers {
		at := page.starts[n]
		forwards = append(forwards, byte(n), byte(at>>8), byte(at))
	}

	forwards = append(forwards, noMorePictures)

	backwards := make([]byte, len(forwards))
	for i, b := range forwards {
		backwards[len(forwards)-1-i] = b
	}

	return backwards
}
