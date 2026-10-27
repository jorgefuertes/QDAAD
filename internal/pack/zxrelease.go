package pack

import (
	"fmt"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
)

// What a Spectrum release is made of, which depends on the Spectrum.
//
// A 48K is a tape machine and its release is one stream: the loader, the
// interpreter, the database and the graphics, glued together so that the BASIC
// at the front can move RAMTOP out of the way before anything lands above it.
// That is ZX48 and it is one file.
//
// A +3 has a disk, so its release is the files loose: a BASIC loader, the
// interpreter, the database and the graphics block, each one its own file and
// each named what the loader looks for. The machine does the gluing at load
// time instead of the packager doing it beforehand.
//
// The two are the same adventure and a different release, which is why the
// subtarget is not a detail here.

// ZXRelease gathers what a Spectrum build needs, for the Spectrum it is built
// for.
//
// subtarget is the one the compiler was given: empty or 48k for a tape machine,
// plus3 for the one with a disk.
func ZXRelease(database []byte, subtarget string, opts Options) (*Release, error) {
	switch strings.ToLower(subtarget) {
	case "", "48k":
		tape, err := ZX48(database, opts)
		if err != nil {
			return nil, err
		}

		// One file, because a tape is one file. Asking for this release as a
		// directory puts the tape in a directory, which is what somebody
		// wanting to keep a build beside its notes is after.
		return NewRelease(File{
			Name: opts.Name + ".tap",
			Body: tape,
			Note: "48K tape: BASIC loader, interpreter, database and graphics in one stream",
		}), nil

	case "plus3":
		return zxPlus3(database, opts)

	default:
		return nil, fmt.Errorf("%q is not a Spectrum this packages yet: it is 48k or plus3",
			subtarget)
	}
}

// The names the +3 release uses.
//
// DISK is the one that is not a choice: a +3 boots a disk from its own menu by
// loading a BASIC program called DISK, with no extension, so a loader by any
// other name is a loader nobody runs. The rest are ours, and they are the names
// DAAD's own releases use.
const (
	plus3Loader      = "DISK"
	plus3Interpreter = "DAAD.BIN"
	plus3Database    = "PART1.DDB"
	plus3Graphics    = "PART1.SDG"
)

// zxPlus3 gathers a +3 release: a loader, an interpreter, a database and the
// graphics block, each its own file.
//
// The three blocks are the same three the tape carries and they go to the same
// three addresses — the interpreter at 0x6000, the database where the linker
// wrote its pointers for, the index and stub ending on the last byte of RAM.
// What changes is who does the gluing: on a tape the packager does it
// beforehand, and here the machine does it at load time.
//
// The loader is written rather than borrowed. DAAD ships one, and it asks which
// part you want and shows a loading screen; this one loads a game and starts
// it, which is all a release of one part needs and is three lines:
//
//	10 CLEAR VAL "24575"
//	20 LOAD "DAAD.BIN"CODE: LOAD "PART1.DDB"CODE: LOAD "PART1.SDG"CODE
//	30 RANDOMIZE USR VAL "24576"
//
// CLEAR has to come first and has to be right: the loader itself lives under
// RAMTOP, and a block landing in the BASIC area takes the loader with it
// halfway through loading. Each LOAD names no address, so where a file goes is
// the file's own business — read from the +3DOS header it carries — and the
// loader cannot contradict the linker.
func zxPlus3(database []byte, opts Options) (*Release, error) {
	// The +3's interpreter is its own and not the 48K one, but it is pinned to
	// the same 0x6000: both start with a JP into a 0x6000-based image, and both
	// hold a JP 0x6032 eight bytes in.
	interpreter, top, topOrigin, err := zxBlocks(runtime.ZXPlus3, database, opts)
	if err != nil {
		return nil, err
	}

	var loader media.BASIC

	loader.Clear(interpreterOrigin - 1)
	loader.LoadNamed(plus3Interpreter, plus3Database, plus3Graphics)
	loader.RandomizeUSR(interpreterOrigin)

	boot, err := media.Plus3Program(media.FirstLine, loader.Bytes())
	if err != nil {
		return nil, err
	}

	release := NewRelease(File{
		Name: plus3Loader, Body: boot,
		Note: "BASIC loader: the +3 boot menu runs this",
	})

	for _, part := range []struct {
		name, note string
		at         int
		body       []byte
	}{
		{plus3Interpreter, "interpreter", interpreterOrigin, interpreter},
		{plus3Database, "the adventure", int(arch.ZX.BaseAddress("")), database},
		{plus3Graphics, "image index, graphics stub and font", topOrigin, top},
	} {
		body, err := media.Plus3Code(part.at, part.body)
		if err != nil {
			return nil, err
		}

		release.Add(part.name, fmt.Sprintf("%s, loads at 0x%04X", part.note, part.at), body)
	}

	return release, nil
}
