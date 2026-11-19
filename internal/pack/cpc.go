package pack

import (
	"errors"
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
	release, err := cpcGame(database, opts, cpcOnDisc)
	if err != nil {
		return nil, err
	}

	texts, err := cpcXMessageFiles(opts)
	if err != nil {
		return nil, err
	}

	return NewRelease(append(release.Files(), texts...)...), nil
}

// cpcGame is the release before the adventure's own files are laid beside it:
// the interpreter, the database and the graphics in one binary, or the three a
// loading screen takes.
func cpcGame(database []byte, opts Options, medium cpcMedium, first ...cpcLoad) (*Release, error) {
	if len(opts.Loading) > 0 {
		return cpcLoaderRelease(database, opts, medium, first...)
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

// CPCDisk writes a CPC release onto a disk image, for a 6128 and no other
// machine.
//
// The pictures go in the four spare banks, compressed, and the drive is not
// touched again once the adventure is running. That is the only disk there is
// here now, and the one it replaced is worth saying out loud because the choice
// was real: a 64K disk carried the pictures as files of their own and the
// interpreter opened them by name, which cost nothing to write and did not
// work. Nothing compresses a picture on that path — the interpreter reads one
// straight into video memory — so a drawing of fourteen character rows is
// sixteen kilobytes of a disk with a hundred and seventy-eight, and nine of the
// twenty-eight of La Aventura Original fit. The other nineteen were not merely
// missing: PICTURE is a condition in version 3, so each one failed it and
// abandoned the rest of the entry it was in, taking the location's own text and
// the window that text is printed in. A disk that carries a third of the
// drawings and breaks the layout at the rest is not a release, and the machine
// that needs it is the one machine a 1989 adventure ported in 2026 need not be
// held back by.
func CPCDisk(database []byte, opts Options) ([]byte, error) {
	if opts.Memory < 128 {
		return nil, fmt.Errorf(
			"a CPC disk carries its pictures in the spare banks of a 6128, so it wants " +
				"--ram 128\n\ta 464 or a 664 loads from tape: --format cdt")
	}

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

	if len(bank.carried) > 0 && len(bank.left) == 0 && len(bank.onDisc) == 0 {
		opts.Note.say(NoticeSaid, "all %d pictures travel in the spare banks with the game",
			len(bank.carried))
	}

	files := make([]media.File, 0, len(release.Files()))
	for _, f := range release.Files() {
		files = append(files, media.File{Name: f.Name, Data: f.Body})
	}

	return media.DataDisk(files)
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
	// An xmessage is a text read from a position inside a file, and the
	// interpreter reaches it by opening that file through the firmware's
	// cassette vectors: on a machine with no disk they really do go to the
	// tape, which is a thing that can be read forwards and not asked. Stopping
	// here is what the 48K tape does for the same reason — see ZXRelease.
	if len(opts.XMessages) > 0 {
		return nil, errors.New(
			"a tape has no disk to read xmessages from: compile them as ordinary " +
				"messages, or package this as a disk")
	}

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

	release, err = cpcGame(database, opts, medium, first...)
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

	if len(bank.left) > 0 {
		// Which they show instead matters as much as which did not fit: with a
		// stand-in they are a wrong drawing, and without one they are a PICTURE
		// that fails and takes the rest of its entry with it.
		instead := "and show nothing, which also costs whatever the rest of " +
			"their entry would have said: --generic names a picture to put up in their place"
		if bank.standIn > 0 {
			instead = fmt.Sprintf("and show picture %03d instead", bank.standIn)
		}

		opts.Note.say(NoticeLeftOut, "%d of the %d pictures travel in memory with the game; "+
			"%s would not fit %s",
			len(bank.carried), len(opts.Images), cpcPictureList(bank.left), instead)
	}

	// The texts the processes read off the disk, which go on ahead of the
	// pictures because a picture that does not fit is a picture nobody sees and
	// a text that does not fit is an adventure that cannot be played.
	texts, err := cpcXMessageFiles(opts)
	if err != nil {
		return nil, err
	}

	on = append(on, texts...)

	// And the pictures memory had no room for, which the disk has room for and
	// the code up there knows how to go and get.
	if len(bank.onDisc) > 0 {
		opts.Note.say(NoticeSaid,
			"%d pictures are read off the disk as they are asked for; the other %d are "+
				"in the banks and want no drive at all",
			len(bank.onDisc), len(bank.carried))

		on = append(on, bank.onDisc...)
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
		opts.Note.say(NoticeLeftOut,
			"the tape carries no pictures: they travel in memory and it takes "+
				"the loader to put them there, so a tape with pictures wants --loading too")

		return cpcBank{}, nil
	}

	return cpcPictureBank(opts, gameEnd, false)
}

// cpcDiskBank is the same question for a disk, where the answer is usually no.
//
// A disk can open a picture whenever the adventure asks for one, which is what
// the interpreter was written to do and what keeps memory free. The bank is
// only worth it on a 128K machine, where the four spare banks hold every
// picture an adventure has and draw them with no drive at all — and there, as
// on a tape, it takes the loader to put them in memory.
func cpcDiskBank(opts Options, gameEnd int) (cpcBank, error) {
	if len(opts.Images) == 0 || opts.Memory < 128 {
		return cpcBank{}, nil
	}

	// Asked for the banks and given nothing to read them with. Saying so
	// matters more here than on a tape: this one still writes a disk, and a
	// disk with nine of twenty-eight pictures on it looks like a disk that ran
	// out of room rather than one that was built the wrong way.
	if len(opts.Loading) == 0 {
		opts.Note.say(NoticeLeftOut,
			"the pictures are not in the spare banks: it takes the loader to put "+
				"them there, so --ram 128 wants --loading too")

		return cpcBank{}, nil
	}

	return cpcPictureBank(opts, gameEnd, true)
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
