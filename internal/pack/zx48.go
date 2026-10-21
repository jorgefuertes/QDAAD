// Package pack turns a linked database into something a machine will load.
//
// The linker's job ends at the .DDB, which is the adventure and nothing else:
// no interpreter, no loader, no idea of what a tape is. This is the step after
// — put the interpreter, the database and whatever else the machine needs into
// the container it boots from, at the addresses it expects them.
//
// The reference does this with daadmaker.exe, a Windows program whose source
// was never released. So unlike the front end and the linker, which are held
// byte-for-byte against drf and drb, nothing here can be compared against
// anything. What it can be held against is a machine: the layout below is the
// one that boots, and it was arrived at by watching it do so.
package pack

import (
	"fmt"

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
// The font is why no charset file is passed here. The reference's daadmaker
// takes one, and the font inside the stub is recognisably AD8x6.CHR with
// changes: slot 0x7F blanked, which is the forced space and has to print as
// nothing, and the whole upper half replaced with the graphics characters. A
// raw .CHR is not that, so writing one over the stub's font would be a guess,
// and a guess that prints ß wherever a forced space was asked for.

// ZX48 builds a .TAP for a 48K Spectrum.
//
// The tape is a BASIC loader and three CODE blocks. The loader takes the memory
// above the interpreter out of BASIC's reach before loading anything into it,
// which is the whole reason it cannot simply be three LOADs.
func ZX48(database []byte, lang language.Language, name string) ([]byte, error) {
	_, interpreter, err := runtime.Interpreter(runtime.ZX48, lang)
	if err != nil {
		return nil, err
	}

	companions, err := runtime.Companions(runtime.ZX48)
	if err != nil {
		return nil, err
	}

	stub, carried := companions["DAAD.SDG"]
	if !carried {
		return nil, fmt.Errorf("the built-in graphics stub is missing")
	}

	// Read from the machine rather than written here, so that the address the
	// loader puts it at and the addresses the linker wrote inside it cannot
	// drift apart. A Spectrum's base does not depend on its subtarget.
	databaseOrigin := int(arch.ZX.BaseAddress(""))
	stubOrigin := topOfMemory - len(stub)

	index := imageIndex(databaseOrigin + len(database))
	indexOrigin := stubOrigin - len(index)

	if end := interpreterOrigin + len(interpreter); end > databaseOrigin {
		return nil, fmt.Errorf(
			"the interpreter ends at 0x%04X and the database starts at 0x%04X", end, databaseOrigin)
	}

	// The one limit a real adventure will meet. There is no paging on a 48K
	// and nothing to move out of the way: the database has to fit between the
	// interpreter's workspace and the index, and if it does not, the build is
	// for a bigger machine.
	if end := databaseOrigin + len(database); end > indexOrigin {
		return nil, fmt.Errorf(
			"the database is %d bytes and %d is all there is between 0x%04X and the image index at 0x%04X",
			len(database), indexOrigin-databaseOrigin, databaseOrigin, indexOrigin)
	}

	parts := []struct {
		at   int
		body []byte
	}{
		{interpreterOrigin, interpreter},
		{databaseOrigin, database},
		{indexOrigin, index},
		{stubOrigin, stub},
	}

	var loader media.BASIC

	// RAMTOP goes to the byte below the interpreter, so that everything from
	// there up belongs to us. CLEAR also empties the variables and resets the
	// stack, which is why nothing can usefully come before it.
	loader.Clear(interpreterOrigin - 1)

	for range parts {
		loader.LoadCode()
	}

	loader.RandomizeUSR(interpreterOrigin)

	var tape media.TAP

	tape.Program(name, loader.Bytes(), media.FirstLine)

	for _, part := range parts {
		if err := tape.Code(name, part.at, part.body); err != nil {
			return nil, err
		}
	}

	return tape.Bytes(), nil
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
// Pictures are not written yet, so page zero is empty and the table is the two
// end markers and nothing between them. The address is still worked out the way
// the reference works it out, so that it is already right when there is
// something to point at: picture data goes at 0xC000, or straight after the
// database when the database has already run past that.
func imageIndex(databaseEnd int) []byte {
	const pictureData = 0xC000

	at := pictureData
	if databaseEnd >= pictureData {
		at = databaseEnd + 1
	}

	const (
		noMorePages    = 0xFF
		noMorePictures = 0xFF
	)

	forwards := []byte{
		byte(at >> 8), byte(at),
		0, 0, // the size of page zero, which holds no pictures
		noMorePages,
		noMorePictures,
	}

	backwards := make([]byte, len(forwards))
	for i, b := range forwards {
		backwards[len(forwards)-1-i] = b
	}

	return backwards
}
