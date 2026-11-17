package pack

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
)

// An Amstrad CPC release, which is one file and is made the way MCRF makes it.
//
// This is the machine the Spectrum's two shapes do not cover. A 48K tape is one
// stream because a tape has no names; a +3 disk is four files because the
// machine can load them by name. A CPC has a disk and loads by name too, and
// still wants one file — because the pieces are glued before they ever reach
// the disk, into a binary that AMSDOS loads in one go and runs.
//
// The layout is MCRF's, from work/DRC/src/mcrf.pas:
//
//	0x0840  the interpreter, padded with zeros to where the database starts
//	0x2880  the database
//	  then  the graphics block, with the font written 48 bytes into it
//
// and the whole thing carries one AMSDOS header saying it loads at 0x0840 and
// is entered there.
//
// Two of those numbers are not arbitrary and one is not even ours. 0x2880 is
// where the linker writes the database's pointers for a CPC build — it is
// arch.CPC's base address — so the padding is not decoration: a database one
// byte off from there is a database every pointer of which is wrong.
const (
	// cpcInterpreterOrigin is where the interpreter is loaded and entered.
	// MCRF calls it INTAT.
	cpcInterpreterOrigin = 0x0840
	// cpcCeiling is where the firmware's variables begin. Above them is the
	// video memory at 0xC000, and a build that reaches either is a build that
	// does not run. MCRF checks the same number.
	cpcCeiling = 0xA6FC
	// cpcFontOffset is how far into the graphics block the font sits. MCRF
	// seeks here and writes 2048 bytes over whatever was there.
	cpcFontOffset = 48
	// cpcGraphicsLengthAt is where in the interpreter its own copy of the
	// graphics length goes. MCRF patches the file three bytes past the end of
	// the header, which is the interpreter's own bytes 3 and 4.
	cpcGraphicsLengthAt = 3
)

// CPCRelease gathers a CPC build.
//
// It is one file unless a loading screen was asked for, and then it is three:
// a release that shows a picture while it loads needs something that runs
// before the interpreter does, and that something has to load the other two.
// See cpcloader.go.
func CPCRelease(database []byte, opts Options) (*Release, error) {
	if len(opts.Loading) > 0 {
		return cpcLoaderRelease(database, opts, cpcOnDisc)
	}

	binary, err := cpcBinary(database, opts)
	if err != nil {
		return nil, err
	}

	// RUN"name" is what starts it, so the name is the one the player types.
	// DAAD's own releases add a BASIC loader called DISC that asks which part
	// you want; a release of one part has nothing to ask.
	return NewRelease(File{
		Name: opts.Name + ".BIN",
		Body: binary,
		Note: "interpreter, database and graphics in one file: RUN\"" + opts.Name + "\"",
	}), nil
}

// CPCDisk writes a CPC release onto a disk image.
//
// The binary goes on first and the pictures after it, one file each, which is
// what the interpreter opens them as. They are what runs out of room — a
// picture here is the rows of a screen with no compression anywhere in the
// path, so a tall one is sixteen kilobytes of a disk with a hundred and
// seventy-eight — so they go on in number order until the disk is full and the
// rest are named rather than dropped quietly.
func CPCDisk(database []byte, opts Options) ([]byte, error) {
	game, err := cpcBinary(database, opts)
	if err != nil {
		return nil, err
	}

	bank, err := cpcDiskBank(opts, cpcInterpreterOrigin+len(media.WithoutAMSDOSHeader(game)))
	if err != nil {
		return nil, err
	}

	release, err := cpcRelease(database, opts, cpcOnDisc, bank)
	if err != nil {
		return nil, err
	}

	files := make([]media.File, 0, len(release.Files()))
	for _, f := range release.Files() {
		files = append(files, media.File{Name: f.Name, Data: f.Body})
	}

	// With the pictures in memory there is nothing left for the disk to hold,
	// and the drive is never touched once the adventure is running.
	var pictures []media.File

	if len(bank.carried) == 0 {
		if pictures, err = cpcPictureFiles(opts); err != nil {
			return nil, err
		}
	} else if opts.Note != nil && len(bank.left) == 0 {
		opts.Note(fmt.Sprintf("all %d pictures travel in the spare banks with the game",
			len(bank.carried)))
	}

	disk, left := fitOnDisk(files, pictures)
	if len(left) > 0 && opts.Note != nil {
		opts.Note(fmt.Sprintf(
			"%d of the %d pictures are on the disk; %s would not fit",
			len(pictures)-len(left), len(pictures), strings.Join(left, ", ")))
	}

	return disk, nil
}

// fitOnDisk puts as many of the pictures on as will go, and says which ones did
// not. It asks the disk rather than working the arithmetic out here: the block
// size, the reserved tracks and the size of the directory are the disk's to
// know, and a copy of them here would be a copy to get wrong.
func fitOnDisk(release, pictures []media.File) ([]byte, []string) {
	disk, err := media.DataDisk(release)
	if err != nil {
		return nil, nil
	}

	on := release

	for at, picture := range pictures {
		bigger, err := media.DataDisk(append(slices.Clone(on), picture))
		if err != nil {
			left := make([]string, 0, len(pictures)-at)
			for _, f := range pictures[at:] {
				left = append(left, f.Name)
			}

			return disk, left
		}

		disk, on = bigger, append(on, picture)
	}

	return disk, nil
}

// CPCTape writes a CPC release as a tape.
//
// It carries the game and, where one was asked for, the loader that shows a
// picture while it goes in — in that order, because a tape is read in the order
// it was written.
//
// The adventure's own pictures are another matter and are left off unless
// asked for. The interpreter opens them by name through the firmware's
// cassette vectors, which on a machine with no disk really do go to the tape,
// so finding one means winding forwards to it and never back: the ones ahead
// of the head are drawn and the first one that is not sends the machine to the
// end of the tape looking. Nothing crashes — the interpreter asks whether a
// picture exists before it draws one, and takes no for an answer — but a tape
// is not a thing you can ask in any order.
//
// The AMSDOS headers come off on the way. A disk file carries its load address
// in a header of its own; a tape carries it in the record the firmware reads
// before the bytes, and a file with both would land 128 bytes adrift.
func CPCTape(database []byte, opts Options) ([]byte, error) {
	// How far the game reaches decides where the pictures can live, so it is
	// built first even though it goes on the tape last.
	game, err := cpcBinary(database, opts)
	if err != nil {
		return nil, err
	}

	bank, err := cpcTapeBank(opts, cpcInterpreterOrigin+len(media.WithoutAMSDOSHeader(game)))
	if err != nil {
		return nil, err
	}

	release, err := cpcRelease(database, opts, cpcOnTape, bank)
	if err != nil {
		return nil, err
	}

	files := make([]media.TapeFile, 0, len(release.Files()))

	for _, f := range release.Files() {
		load, entry, ok := media.AMSDOSAddresses(f.Body)
		if !ok {
			return nil, fmt.Errorf("%s has no header saying where it loads", f.Name)
		}

		files = append(files, media.TapeFile{
			Name:  f.Name,
			Data:  media.WithoutAMSDOSHeader(f.Body),
			Load:  load,
			Entry: entry,
		})
	}

	return media.CDT(files)
}

// cpcRelease is the files a medium will carry, with the bank among them and the
// game pointed at it.
//
// The bank's files go in ahead of the game, which is the order the loader asks
// for them in and the only order a tape can be read in.
func cpcRelease(
	database []byte, opts Options, medium cpcMedium, bank cpcBank,
) (*Release, error) {
	var first []cpcLoad

	for _, part := range bank.files() {
		first = append(first, cpcLoad{name: part.Name, at: part.at, config: part.config})
	}

	var (
		release *Release
		err     error
	)

	if len(opts.Loading) > 0 {
		release, err = cpcLoaderRelease(database, opts, medium, first...)
	} else {
		release, err = CPCRelease(database, opts)
	}

	if err != nil {
		return nil, err
	}

	on := release.Files()

	// The game is pointed at the bank, or — on a tape, where there is no other
	// way to reach a picture — told to stop asking for them.
	for i, f := range on {
		if !strings.HasSuffix(f.Name, cpcGameSuffix) && len(on) != 1 {
			continue
		}

		if len(bank.carried) > 0 {
			if on[i].Body, err = cpcPointAtTheBank(f.Body, bank); err != nil {
				return nil, err
			}

			continue
		}

		if medium == cpcOnTape {
			on[i].Body = cpcNoPictures(f.Body)
		}
	}

	for _, part := range bank.files() {
		header, err := media.AMSDOSHeader(part.Name, part.at, part.at, len(part.Body))
		if err != nil {
			return nil, err
		}

		part.Body = append(header, part.Body...)
		on = slices.Insert(on, len(on)-1, part.File)
	}

	if opts.Note != nil && len(bank.left) > 0 {
		opts.Note(fmt.Sprintf("%d of the %d pictures travel in memory with the game; "+
			"%s would not fit",
			len(bank.carried), len(opts.Images), cpcPictureList(bank.left)))
	}

	return NewRelease(on...), nil
}

// cpcTapeBank is the pictures that will travel in memory, or none.
//
// None, and not an error, in the two cases where there is nothing to be done:
// an adventure with no pictures at all, and one packaged without a loading
// screen. The second is not a whim — the bank has to be read into memory by
// something, and that something is the loader, which exists to show the
// picture. A tape asked for pictures and not for a loading screen is told.
func cpcTapeBank(opts Options, gameEnd int) (cpcBank, error) {
	if len(opts.Images) == 0 {
		return cpcBank{}, nil
	}

	if len(opts.Loading) == 0 {
		if opts.Note != nil {
			opts.Note("the tape carries no pictures: they travel in memory and it takes " +
				"the loader to put them there, so a tape with pictures wants --loading too")
		}

		return cpcBank{}, nil
	}

	return cpcPictureBank(opts, gameEnd)
}

// cpcDiskBank is the same question for a disk, where the answer is usually no.
//
// A disk can open a picture whenever the adventure asks for one, which is what
// the interpreter was written to do and what keeps memory free. The bank is
// only worth it on a 128K machine, where the four spare banks hold every
// picture an adventure has and draw them with no drive at all — and there, as
// on a tape, it takes the loader to put them in memory.
func cpcDiskBank(opts Options, gameEnd int) (cpcBank, error) {
	if len(opts.Images) == 0 || opts.Memory < 128 || len(opts.Loading) == 0 {
		return cpcBank{}, nil
	}

	return cpcPictureBank(opts, gameEnd)
}

// cpcPictureList names a handful of pictures for a person to read.
func cpcPictureList(numbers []int) string {
	names := make([]string, 0, len(numbers))
	for _, n := range numbers {
		names = append(names, fmt.Sprintf("%03d", n))
	}

	return strings.Join(names, ", ")
}

// cpcNoPictures stops a build from asking for its pictures at all.
//
// The interpreter never draws one without first asking whether it is there:
// the routine at 0x095E opens the file, and a clear carry on the way back
// means no, which the caller takes without complaint. On a disk the asking is
// free. On a tape it is not — the firmware has no way to say "not on this
// tape", so it stops the machine and asks for PLAY in the middle of the
// adventure.
//
// So the asking is taken out. The call to the routine that builds the name and
// opens the file becomes `or a` and two nothings, which is the clear carry the
// caller is already written to handle, and no cassette is ever touched.
//
//	095E: f5        push af
//	095F: cd ca 0c  call 0x0cca   ->  b7 00 00   or a / nop / nop
//	0962: 30 0e     jr nc,0x0972  ->  taken: the not-found way out
//
// Both interpreters we ship have those bytes in that place, and a build whose
// bytes are not those is left exactly as it came.
func cpcNoPictures(binary []byte) []byte {
	const (
		at   = 0x095F - cpcInterpreterOrigin + media.AMSDOSHeaderSize
		call = 3
	)

	if len(binary) < at+call {
		return binary
	}

	if binary[at] != 0xCD || binary[at+1] != 0xCA || binary[at+2] != 0x0C {
		return binary
	}

	out := slices.Clone(binary)
	out[at], out[at+1], out[at+2] = 0xB7, 0x00, 0x00

	return out
}

func cpcBinary(database []byte, opts Options) ([]byte, error) {
	_, interpreter, err := runtime.Interpreter(runtime.CPC, opts.Language)
	if err != nil {
		return nil, err
	}

	companions, err := runtime.Companions(runtime.CPC)
	if err != nil {
		return nil, err
	}

	graphics, carried := companions["BLANK.GRA"]
	if !carried {
		return nil, fmt.Errorf("the built-in CPC graphics block is missing")
	}

	// The graphics block ships with an AMSDOS header of its own and the
	// interpreter does not, which is why this is asked rather than assumed.
	graphics = media.WithoutAMSDOSHeader(graphics)
	interpreter = media.WithoutAMSDOSHeader(interpreter)

	databaseOrigin := int(arch.CPC.BaseAddress(""))

	room := databaseOrigin - cpcInterpreterOrigin
	if len(interpreter) > room {
		return nil, fmt.Errorf(
			"the interpreter is %d bytes and the database starts %d after it, at 0x%04X",
			len(interpreter), room, databaseOrigin)
	}

	// The interpreter is told how much graphics follows it, in two bytes of
	// its own. It has to be a copy: the original is embedded and shared.
	code := make([]byte, room)
	copy(code, interpreter)
	code[cpcGraphicsLengthAt] = byte(len(graphics))
	code[cpcGraphicsLengthAt+1] = byte(len(graphics) >> 8)

	body := make([]byte, 0, room+len(database)+len(graphics))
	body = append(body, code...)
	body = append(body, database...)

	fontAt := len(body) + cpcFontOffset
	body = append(body, graphics...)

	// The font goes a fixed distance into the graphics block, which is where
	// the interpreter looks for it. MCRF seeks to the same place.
	if len(opts.Charset) > 0 {
		glyphs, err := charsetGlyphs(opts.Charset)
		if err != nil {
			return nil, err
		}

		if fontAt+len(glyphs) > len(body) {
			return nil, fmt.Errorf("the graphics block has no room for a font: %d bytes",
				len(graphics))
		}

		copy(body[fontAt:], glyphs)
	}

	if end := cpcInterpreterOrigin + len(body); end >= cpcCeiling {
		return nil, fmt.Errorf(
			"the build ends at 0x%04X and the firmware's variables start at 0x%04X: "+
				"%d bytes too much between the database and the graphics",
			end, cpcCeiling, end-cpcCeiling+1)
	}

	header, err := media.AMSDOSHeader(opts.Name+".BIN",
		cpcInterpreterOrigin, cpcInterpreterOrigin, len(body))
	if err != nil {
		return nil, err
	}

	return append(header, body...), nil
}
