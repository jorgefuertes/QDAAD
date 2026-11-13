package pack

import (
	"errors"
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
		image, err := media.Plus3Disk(files)
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
		// A tape has no filesystem, and an xmessage is a text read out of a
		// file by offset. The reference build script stops here too rather
		// than writing a tape that cannot print half its text — and it is why
		// the 48K trees of La Aventura Original are built with
		// -force-normal-messages.
		if len(opts.XMessages) > 0 {
			return nil, errors.New(
				"a 48K tape has no disk to read xmessages from: compile them as " +
					"ordinary messages, or build for plus3")
		}

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

	case "128k":
		// Not a gap. A 128K release is a tape built around ZXDAAD128, and that
		// interpreter was withdrawn from DAAD READY in version B — see
		// docs/DAAD_v2_v3_Research/08-interpretes.md, which ranks it last of
		// the five for version 3 and says plainly that it is not an option for
		// it. The +3 is the 128K machine this packages, and it packages it as
		// a disk because that is what a +3 has.
		return nil, errors.New(
			"a 128K release would need ZXDAAD128, which DAAD READY withdrew in version B " +
				"and which does not do version 3: build for plus3, which is the 128K machine " +
				"this writes for")

	default:
		return nil, fmt.Errorf("%q is not a Spectrum this writes: it is 48k or plus3",
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
	// plus3Loading is the picture shown while the rest of the disk goes in. It
	// is a file like any other and loads straight into video memory, so the
	// loader needs nothing but its name.
	plus3Loading  = "LOADING.SCR"
	plus3Database = "PART1.DDB"
	plus3Graphics = "PART1.SDG"
	// plus3Pictures is page zero, and it is a file only when the adventure has
	// pictures. DAAD's own +3 releases buffer theirs into a DAAD.GRA instead,
	// which is a different scheme for a machine with more memory to put it in;
	// this carries what the 48K build carries, to the same address.
	plus3Pictures = "PART1.PAG"
	// plus3XMessages is the texts read off disk. The name is built into the
	// interpreter as a string — XMBFilename in PLUS3.ASM — so it is the one
	// file here whose name is not even a convention.
	plus3XMessages = "0.XMB"
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

	// Which pictures go into the buffer and which stay files of their own, and
	// it comes first because the answer decides whether the interpreter needs
	// fixing before it is written out. See zxplus3cache.go.
	// Every name that is not a picture, so that the count of pictures that
	// have to leave the directory is the right one. A loading screen is one of
	// them, and forgetting it cost part two its drawing 107.
	others := plus3OtherFiles
	if len(opts.Loading) > 0 {
		others++
	}

	if len(opts.XMessages) > 0 {
		others++
	}

	// A +3 disk and so a +3 directory, which is the only reason this knows the
	// number: chooseCached decides how many pictures have to move and the disk
	// decides how many names there are to move them out of.
	cached, loose, err := chooseCached(opts.Images, media.Plus3Names(), others)
	if err != nil {
		return nil, err
	}

	if len(cached) > 0 {
		if built.interpreter, err = plus3DisableInterrupts(
			built.interpreter, interpreterOrigin, int(arch.ZX.BaseAddress("")),
		); err != nil {
			return nil, fmt.Errorf("buffered pictures: %w", err)
		}
	}

	named := []string{plus3Interpreter, plus3Database, plus3Graphics}
	if len(built.page.body) > 0 {
		named = append(named, plus3Pictures)
	}

	var loader media.BASIC

	loader.Clear(interpreterOrigin - 1)

	// The picture first, and on its own, because the POKE that quiets the ROM
	// has to go between it and everything else: before it there is nothing to
	// write over, and after the rest there is nothing left to quiet.
	if len(opts.Loading) > 0 {
		loader.BlackScreen()
		loader.LoadNamed(plus3Loading)
		loader.HideLoadingMessages()
	}

	loader.LoadNamed(named...)

	// And a moment to look at it. A tape gives you one for nothing — the last
	// block takes seconds to come off it — and a disk does not, so the picture
	// would otherwise be replaced by the first location about as fast as it
	// appeared.
	if len(opts.Loading) > 0 {
		loader.Pause(plus3LoadingFrames)
	}

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

	if len(opts.Loading) > 0 {
		parts = append([]struct {
			name, note string
			at         int
			body       []byte
		}{{plus3Loading, "the picture shown while the rest loads", screenBase, opts.Loading}},
			parts...)
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
	// The split is about the directory and not about speed: an adventure with
	// more pictures than the disk has names has to put some of them somewhere
	// that is not a file. Since the disk started asking for 128 names instead
	// of 64 that is nothing we ship — both parts of La Aventura Original go on
	// as files — so chooseCached comes back empty and this writes nothing. It
	// stays because the limit is a limit and an adventure will reach it. See
	// zxplus3cache.go.
	if buffer, err := plus3Cache(cached); err != nil {
		return nil, err
	} else if len(buffer) > 0 {
		release.Add(plus3CacheName,
			fmt.Sprintf("%d pictures, read into page one at start-up", len(cached)),
			buffer)
	}

	// The texts that are not in the database, under the one name that will do:
	// the interpreter has "0.XMB" built into it as a string and opens it
	// itself, so this is not in the loader's list of things to LOAD.
	//
	// No +3DOS header either, and for the same reason the pictures have none:
	// the interpreter reads from an offset the compiler worked out, and 128
	// bytes in front of the first message would put every one of them 128
	// bytes out. The reference draws the line in the same place — SPECFORM
	// goes over DAAD.BIN and the screens, never over this.
	if len(opts.XMessages) > 0 {
		release.Add(plus3XMessages,
			fmt.Sprintf("%d bytes of texts the processes read from disk", len(opts.XMessages)),
			opts.XMessages)
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

// plus3LoadingFrames is how long the picture stays up once everything is in,
// in fiftieths of a second: three seconds.
//
// A tape needs none of this. Its last block is tens of kilobytes coming off at
// 1500 baud, so the picture is up for the best part of a minute whether anybody
// wanted it or not. A disk reads the same bytes in a moment, and without a
// wait the picture is gone about as fast as it arrived.
const plus3LoadingFrames = 3 * 50
