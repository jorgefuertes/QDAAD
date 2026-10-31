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

// ZXDisk writes a Spectrum release onto a +3 disk image.
//
// It is the +3 release and nothing else: the four files, in the order the
// bundle lists them, so that a CAT shows the loader first. A 48K build has no
// disk to go on — the subtarget decides the release and a tape release is one
// file — so asking for a disk of one says so rather than writing a disk with an
// interpreter on it and no game.
func ZXDisk(database []byte, subtarget string, opts Options) ([]byte, error) {
	release, err := ZXRelease(database, subtarget, opts)
	if err != nil {
		return nil, err
	}

	if len(release.Files()) == 1 {
		return nil, fmt.Errorf(
			"a %s release is one file and belongs on a tape: build it as --format tap",
			subtarget)
	}

	files := make([]media.File, 0, len(release.Files()))
	for _, f := range release.Files() {
		files = append(files, media.File{Name: f.Name, Data: f.Body})
	}

	// A disk that will not hold every picture is an ordinary outcome: they
	// are the only part that can be left out and still leave an adventure
	// anybody can play. The last come off until the rest fit, and what came
	// off is said out loud.
	var dropped []string

	for {
		image, err := media.DataDisk(files)
		if err == nil {
			if len(dropped) > 0 && opts.Note != nil {
				opts.Note(fmt.Sprintf("%d pictures did not fit on the disk and are not on it: %s",
					len(dropped), strings.Join(dropped, ", ")))
			}

			return image, nil
		}

		last := len(files) - 1
		if last < 0 || !strings.HasSuffix(files[last].Name, plus3PictureSuffix) {
			return nil, err
		}

		dropped = append(dropped, files[last].Name)
		files = files[:last]
	}
}

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
	// plus3Pictures is page zero, and it is a file only when the adventure has
	// pictures. DAAD's own +3 releases buffer theirs into a DAAD.GRA instead,
	// which is a different scheme for a machine with more memory to put it in;
	// this carries what the 48K build carries, to the same address.
	plus3Pictures = "PART1.PAG"
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
	// Built with no pictures in them: a 48K carries its pictures in memory
	// and the index says where each landed, but here they are files on the
	// disk, so the index stays empty and the interpreter goes and opens one.
	inMemory := opts
	inMemory.Images = nil

	built, err := zxBlocks(runtime.ZXPlus3, database, inMemory)
	if err != nil {
		return nil, err
	}

	named := []string{plus3Interpreter, plus3Database, plus3Graphics}
	if len(built.page.body) > 0 {
		named = append(named, plus3Pictures)
	}

	var loader media.BASIC

	loader.Clear(interpreterOrigin - 1)
	loader.LoadNamed(named...)
	loader.RandomizeUSR(interpreterOrigin)

	boot, err := media.Plus3Program(media.FirstLine, loader.Bytes())
	if err != nil {
		return nil, err
	}

	release := NewRelease(File{
		Name: plus3Loader, Body: boot,
		Note: "BASIC loader: the +3 boot menu runs this",
	})

	parts := []struct {
		name, note string
		at         int
		body       []byte
	}{
		{plus3Interpreter, "interpreter", interpreterOrigin, built.interpreter},
		{plus3Database, "the adventure", int(arch.ZX.BaseAddress("")), database},
		{plus3Graphics, "image index, graphics stub and font", built.topOrigin, built.top},
	}

	if len(built.page.body) > 0 {
		parts = append(parts, struct {
			name, note string
			at         int
			body       []byte
		}{plus3Pictures, "the pictures", built.page.at, built.page.body})
	}

	for _, part := range parts {
		body, err := media.Plus3Code(part.at, part.body)
		if err != nil {
			return nil, err
		}

		release.Add(part.name, fmt.Sprintf("%s, loads at 0x%04X", part.note, part.at), body)
	}

	// The pictures, in two places. As many as page one holds go into DAAD.GRA,
	// which the interpreter reads once at start-up and keeps; the rest are
	// files of their own, opened by name when a PICTURE names them.
	//
	// The split is not about speed, although the buffered ones are faster for
	// nothing. It is about the directory: a +3DOS one has 64 entries, and an
	// adventure with 69 pictures and four other files has more names than
	// that however empty the disk is. See zxplus3cache.go.
	cached, loose, err := chooseCached(opts.Images)
	if err != nil {
		return nil, err
	}

	if buffer, err := plus3Cache(cached); err != nil {
		return nil, err
	} else if len(buffer) > 0 {
		release.Add(plus3CacheName,
			fmt.Sprintf("%d pictures, read into page one at start-up", len(cached)),
			buffer)
	}

	for _, number := range loose {
		picture, err := plus3Picture(opts.Images[number])
		if err != nil {
			return nil, fmt.Errorf("picture %d: %w", number, err)
		}

		release.Add(plus3PictureName(number), fmt.Sprintf("picture %d", number), picture)
	}

	return release, nil
}
