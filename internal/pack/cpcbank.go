package pack

import (
	"fmt"
	"slices"
	"sort"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/zx0"
)

// The adventure's pictures kept in memory instead of on the medium.
//
// The interpreter reads a picture out of a file every time one is shown, which
// is what a disk is for and what a tape cannot do: a tape is read forwards and
// a PICTURE asks in whatever order the player walks. So the pictures travel
// with the game and stay in memory, and the interpreter is pointed at them.
//
// What makes them fit is compression. A picture is the rows of a mode 1 screen
// the adventure does not write over, band by band, and those come down to about
// 1900 bytes each with ZX0 — the 28 of the first part of La Aventura Original
// are 53K against the 245K they take as they are.
//
// Band by band costs about a quarter against one stream for the whole screen,
// which is what this did first and what made every drawing paint the text area
// black on its way past. See cpcBankEntry.
//
// Where they go depends on the machine. On a 64K one it is whatever is left
// between the end of the game and the loader, which is around fifteen and so
// about a dozen pictures. On a 128K one it is the four spare banks, paged into
// the window at 0x4000 one at a time, and then the whole adventure fits with
// room over.
//
// Paging is ours to do because the interpreter never does it: the only ports it
// writes to are 0xF4 and 0xF6, which are the PPI, and it leaves the gate
// array's memory configuration alone from beginning to end.
//
// Two patches point the interpreter here, and both are three bytes:
//
//	095F  call 0x0CCA  ->  call <ours>   is this picture here? carry says
//	09C0  call 0x0CCA  ->  call <ours>   put it on the screen
//	09C3  ld a,0xff    ->  jp 0x0A36     and skip the whole file reader
//
// The first is the question the interpreter already asks before it draws
// anything, and it already takes no for an answer — which is what makes a
// picture that did not fit come out as no picture rather than as a crash.
const (
	// cpcProbeAt and cpcDrawAt are where the interpreter calls the routine
	// that opens a picture file. Both become calls to ours.
	cpcProbeAt = 0x095F
	cpcDrawAt  = 0x09C0
	// cpcDrawTail is where the file reader ends and the interpreter picks up
	// again: past the close, which there is nothing to close.
	cpcDrawTail = 0x0A36
	// cpcOpenPicture is the call being replaced, and it is checked rather
	// than assumed.
	cpcOpenPicture = 0x0CCA

	// cpcShownPicture is where the interpreter keeps the number of the picture
	// it is about to draw.
	cpcShownPicture = 0x2301
	// cpcInkBuffer is the sixteen bytes it reads a palette out of, and
	// cpcSetPalette the routine that hands them to the firmware.
	cpcInkBuffer  = 0x0D02
	cpcSetPalette = 0x0CB7

	// cpcGraphicsTop is where the interpreter moves its graphics block to
	// before it starts, and so the ceiling of what we may use. It takes the
	// 2096 bytes of BLANK.GRA down from 0xA600.
	cpcGraphicsTop = 0xA600 - 2096

	// cpcBankHeader is what each picture carries in front of its stream: the
	// number, the whole entry's length, and the sixteen inks.
	cpcBankHeader = 1 + 2 + cpcPaletteSize
	// cpcIndexRecord is what the index holds for each picture: the number,
	// which configuration of memory it is in, and where in it.
	cpcIndexRecord = 4
	// cpcIndexEnd closes the index, in place of a picture number.
	cpcIndexEnd = 0xFF

	// The gate array's memory configurations. 0xC0 is the plain 64K a machine
	// starts in; 0xC4 to 0xC7 each put one of a 6128's spare banks in the
	// window at 0x4000.
	cpcMainMemory = 0xC0
	cpcFirstBank  = 0xC4
	cpcBanks      = 4
	cpcBankWindow = 0x4000
	cpcBankSize   = 0x4000

	// cpcResidentSize is the room kept under the loader for the code that
	// reads the bank and for the index.
	cpcResidentSize = 0x200
)

// cpcBlock is one lump of the bank as it travels: where it loads and which
// configuration of memory has to be in force while it does.
type cpcBlock struct {
	config int
	at     int
	body   []byte
}

// cpcBank is everything the pictures need: the code that reads them, the index
// that finds them, and the blocks they travel in.
type cpcBank struct {
	// name is the adventure's, which the files it travels in are called after.
	name string
	// at is where the code and the index live, which is under the loader and
	// out of the window a bank is paged into.
	at int
	// drawAt is how far into that the routine that puts a picture up begins,
	// and entriesAt where the index starts.
	drawAt    int
	entriesAt int
	resident  []byte
	blocks    []cpcBlock
	// carried and left are which pictures are in memory and which are not.
	carried []int
	left    []int
	// standIn is the picture the ones in left are pointed at, or zero when
	// there is none and they are simply not there.
	standIn int
	// onDisc are the ones that did not fit in memory and went on the disk
	// instead, compressed, to be read into a buffer when they are asked for;
	// read is where that buffer is. A tape has neither.
	onDisc []File
	read   int
}

// cpcDiscPicture is the memory configuration an index entry carries when the
// picture is not in memory at all but a file on the disk.
//
// It is not a configuration the gate array has — those are 0xC0 and 0xC4 to
// 0xC7 — which is what makes it safe to say this with the same byte.
const cpcDiscPicture = 0xFF

// cpcResidentAt is where the bank's own code and index sit: the pages below
// the loader, which is out of the window a bank is paged into.
func cpcResidentAt() int { return cpcLoaderOrigin() - cpcResidentSize }

// cpcPictureBank builds it for a game that ends where it says, in a machine
// with the memory the options say.
//
// onDisc says whether what does not fit in memory can go on the medium as a
// file of its own. A disk can be asked for a picture by name at any moment and
// a tape cannot, which is the whole difference between the two here.
func cpcPictureBank(opts Options, gameEnd int, onDisc bool) (cpcBank, error) {
	bank := cpcBank{at: cpcResidentAt(), name: opts.Name}

	// The buffer a picture read from the disk lands in, which is the memory
	// above the game — the same ground the pictures themselves use on a 64K
	// machine, and empty on a 128K one because there they are in the banks.
	bank.read = (gameEnd + 0xFF) &^ 0xFF

	// Where a picture may go, in the order they are filled. A 64K machine has
	// one such place and a 128K one has four, each a bank of its own.
	type ground struct {
		config, at, room int
	}

	var where []ground

	if opts.Memory >= 128 {
		for i := range cpcBanks {
			where = append(where, ground{
				config: cpcFirstBank + i, at: cpcBankWindow, room: cpcBankSize,
			})
		}
	} else {
		at := (gameEnd + 0xFF) &^ 0xFF
		where = append(where, ground{config: cpcMainMemory, at: at, room: bank.at - at})
	}

	// Wherever the pictures go, the code that reads them is under the loader,
	// and a game that reaches that far has nowhere to put either.
	if gameEnd > bank.at || where[0].room <= 0 {
		return cpcBank{}, fmt.Errorf(
			"the game ends at 0x%04X and there is no room under 0x%04X for the pictures",
			gameEnd, bank.at)
	}

	numbers := make([]int, 0, len(opts.Images))
	for number := range opts.Images {
		numbers = append(numbers, number)
	}

	sort.Ints(numbers)

	blocks := make([][]byte, len(where))

	var index []byte

	entries, err := compressPictures(numbers, opts.Progress, func(number int) ([]byte, error) {
		return cpcBankEntry(number, opts, cpcPictureRows(opts.Images[number], opts.Lines))
	})
	if err != nil {
		return cpcBank{}, err
	}

	// Where each picture ended up, so that the index can be written in number
	// order afterwards however the laying out went.
	type placed struct{ config, at int }

	landed := make(map[int]placed, len(numbers))

	// How much of the buffer the biggest picture read from the disk needs.
	widest := 0

	// Laying them out is a pass of its own, after all of them are compressed,
	// because where one goes depends on how much room the ones before it took.
	//
	// The stand-in goes first when there is one. It is the picture every other
	// one falls back on, so it is the one picture that must not be the one that
	// does not fit.
	for _, i := range cpcStandInFirst(numbers, opts.Generic) {
		number, entry := numbers[i], entries[i]

		// The first place it fits, which fills the blocks rather than evening
		// them out and is what gets the most pictures in.
		put := -1

		for place, g := range where {
			if len(blocks[place])+len(entry) <= g.room {
				put = place

				break
			}
		}

		if put < 0 {
			// No room in memory. On a disk that is not the end of it: the
			// picture goes on as a file of its own, compressed exactly as it
			// would have travelled, and is read into a buffer when it is asked
			// for. The drive is touched for those and for no others.
			//
			// It is what the +3 does with all of its pictures and what this
			// machine could not do by itself: its interpreter reads a picture
			// file straight into video memory and unpacks nothing. The
			// decompressor that makes the banks possible is the same one that
			// makes this possible, and it is already resident.
			if onDisc {
				file, err := cpcPictureFile(number, entry, bank.read)
				if err != nil {
					return cpcBank{}, err
				}

				landed[number] = placed{config: cpcDiscPicture}
				bank.onDisc = append(bank.onDisc, file)
				widest = max(widest, len(entry))
			} else {
				bank.left = append(bank.left, number)
			}

			continue
		}

		at := where[put].at + len(blocks[put])

		landed[number] = placed{config: where[put].config, at: at}
		blocks[put] = append(blocks[put], entry...)
		bank.carried = append(bank.carried, number)
	}

	sort.Ints(bank.left)
	sort.Ints(bank.carried)

	// And the index, in number order.
	//
	// A picture that did not fit is named all the same and pointed at the
	// stand-in, because the alternative is worse than a wrong drawing: PICTURE
	// is a condition in version 3 and a picture that is not there fails it,
	// which abandons the rest of the entry. At the grate of La Aventura
	// Original that entry goes on to say what is on the grate, and in the
	// adventure's own loop it goes on to put the text window back where the
	// text goes — so a picture left off a tape took a paragraph and the layout
	// with it. Pointed here, every PICTURE succeeds and all that is lost is the
	// drawing.
	// Zero is no stand-in and not picture zero, which is what the flag's own
	// default says and has to be tested for: an adventure with a picture
	// numbered zero would otherwise have every miss fall on it quietly.
	stand, hasStand := landed[opts.Generic]
	hasStand = hasStand && opts.Generic != 0

	switch {
	case hasStand:
		bank.standIn = opts.Generic

	// Named one the adventure does not have, which is what asking for the
	// first part's number while packaging the second looks like. Quietly
	// ignoring it is how a release goes out with the flag passed and nothing
	// standing in.
	case opts.Generic != 0 && len(opts.Images) > 0:
		opts.Note.say(NoticeLeftOut,
			"--generic names picture %03d and this adventure has no such picture, "+
				"so the ones that do not fit stand in for nothing",
			opts.Generic)
	}

	for _, number := range numbers {
		where, ok := landed[number]
		if !ok {
			if !hasStand {
				continue
			}

			where = stand
		}

		index = append(index,
			byte(number), byte(where.config), byte(where.at), byte(where.at>>8))
	}

	index = append(index, cpcIndexEnd)

	// The buffer the pictures read from the disk land in is between the game
	// and the bank's own code, and a game that reaches far enough leaves no
	// room for it. Saying so beats a picture unpacking over the index that
	// found it.
	if widest > 0 && bank.read+widest > bank.at {
		return cpcBank{}, fmt.Errorf(
			"the biggest picture read from the disk is %d bytes and there is room for %d "+
				"between the game at 0x%04X and the picture code at 0x%04X",
			widest, bank.at-bank.read, bank.read, bank.at)
	}

	code, drawAt, err := cpcBankCode(bank.at, bank.read)
	if err != nil {
		return cpcBank{}, err
	}

	if len(code)+len(index) > cpcResidentSize {
		return cpcBank{}, fmt.Errorf(
			"the bank's code and index come to %d bytes and there is room for %d",
			len(code)+len(index), cpcResidentSize)
	}

	bank.drawAt, bank.entriesAt = drawAt, len(code)
	bank.resident = append(code, index...)

	for i, body := range blocks {
		if len(body) == 0 {
			continue
		}

		bank.blocks = append(bank.blocks, cpcBlock{
			config: where[i].config, at: where[i].at, body: body,
		})
	}

	return bank, nil
}

// cpcBankFile is one of the files the bank travels in, and where it goes.
type cpcBankFile struct {
	File

	at     int
	config int
}

// files is the bank as files to be read before the game: the code and its index
// first, and then a block for every ground the pictures were laid in.
//
// Names are the adventure's with an extension of their own, because AMSDOS has
// nothing else to tell files apart by: .PIC for the code, .PI0 and up for the
// blocks, in the order the loader has to read them.
func (b cpcBank) files() []cpcBankFile {
	if len(b.carried) == 0 {
		return nil
	}

	files := []cpcBankFile{{
		File: File{
			Name: b.name + cpcBankSuffix,
			Body: b.resident,
			Note: fmt.Sprintf("where %d pictures are looked up and unpacked", len(b.carried)),
		},
		at: b.at,
	}}

	for i, block := range b.blocks {
		name := fmt.Sprintf("%s%s%d", b.name, cpcBankSuffix[:3], i)

		note := "pictures, compressed"
		if block.config != cpcMainMemory {
			note += fmt.Sprintf(", for the bank at 0x%02X", block.config)
		}

		files = append(files, cpcBankFile{
			File:   File{Name: name, Body: block.body, Note: note},
			at:     block.at,
			config: block.config,
		})
	}

	return files
}

// cpcBankEntry is one picture: what it is called, how long the whole of it is,
// the inks it is read through, and the rows it draws on, compressed.
//
// One stream per band and not one for the whole screen, each with its own
// length in front of it. The whole screen was simpler and was wrong: it carried
// the rows below the picture as blanks, and a stream that unpacks straight onto
// the display writes those blanks too. Every drawing was painting the text area
// black on its way past.
//
// That is invisible where a picture is drawn before anything is written under
// it, which is what the loop does for a location that has one. It is not
// invisible on the generic picture, which the port draws after the turn has
// already printed — see the flag 40 in v3-loop.sce — and there the description
// went black. Eight streams and eight destinations leave the rows below the
// picture alone, which is what the file path has always done.
func cpcBankEntry(number int, opts Options, rows int) ([]byte, error) {
	screen := opts.Images[number]
	if len(screen) != cpcPackScreen {
		return nil, fmt.Errorf("picture %03d is %d bytes, not a screen", number, len(screen))
	}

	palette := opts.Palettes[number]
	if len(palette) == 0 {
		return nil, fmt.Errorf("picture %03d has no inks beside it", number)
	}

	entry := make([]byte, 0, cpcBankHeader+cpcPackScreen/2)
	entry = append(entry, byte(number), 0, 0)
	entry = append(entry, palette[:min(len(palette), cpcPaletteSize)]...)

	for len(entry) < cpcBankHeader {
		entry = append(entry, 0)
	}

	for _, band := range cpcBands(screen, rows) {
		stream := zx0.Compress(band)

		entry = append(entry, byte(len(stream)), byte(len(stream)>>8))
		entry = append(entry, stream...)
	}

	entry[1], entry[2] = byte(len(entry)), byte(len(entry)>>8)

	return entry, nil
}

// cpcPictureFile is a picture the disk carries because memory had no room for
// it: the same entry the bank would have held, under the name the interpreter
// builds for itself.
//
// It carries an AMSDOS header like every other binary on the disk. The address
// in it is the buffer, which is where the loader would put the file if anybody
// asked it to — nobody does, because what reads this is CAS IN DIRECT with the
// address in HL, but a header has to say something and saying the truth costs
// nothing.
func cpcPictureFile(number int, entry []byte, at int) (File, error) {
	name := fmt.Sprintf("%03d%s", number, cpcPictureSuffix)

	header, err := media.AMSDOSHeader(name, at, at, len(entry))
	if err != nil {
		return File{}, err
	}

	return File{
		Name: name,
		Body: append(header, entry...),
		Note: fmt.Sprintf("picture %03d, compressed, read in when it is asked for", number),
	}, nil
}

// cpcStandInFirst is the order the pictures are laid out in: the stand-in, if
// there is one and it is among them, and then the rest in the order they came.
//
// It hands back positions and not numbers because what the caller has beside
// each number is the compressed entry, and the two are kept in step by index.
func cpcStandInFirst(numbers []int, standIn int) []int {
	order := make([]int, 0, len(numbers))

	first := -1
	for i, number := range numbers {
		if number == standIn {
			first = i

			break
		}
	}

	if first >= 0 {
		order = append(order, first)
	}

	for i := range numbers {
		if i != first {
			order = append(order, i)
		}
	}

	return order
}

// cpcBands is the rows of a picture the adventure does not write over, band by
// band, which is how video memory holds them: eight of them 0x800 apart, each
// as many bytes wide as the picture is tall in character rows.
func cpcBands(screen []byte, rows int) [][]byte {
	out := make([][]byte, 0, cpcPackBands)

	for band := range cpcPackBands {
		at := band * cpcPackBandSize
		out = append(out, screen[at:at+rows*cpcPackBytesWide])
	}

	return out
}

// cpcBankCode is what reads the bank: three short routines and Einar Saukas'
// decompressor, assembled at the address the bank will live at.
//
// read is where a picture that lives on the disk rather than in memory is read
// into. Everything after that is the same for both: the two differ in where the
// bytes come from and in nothing else, which is what makes carrying some of the
// pictures on the disk twenty-odd bytes of code rather than a second routine.
func cpcBankCode(at, read int) (code []byte, drawAt int, err error) {
	a := &cpcAssembly{}

	// probe(A) — is this picture here? The interpreter asks before it draws,
	// and wants carry set and zero clear for yes.
	a.here("probe")
	a.bytes(0x57) // ld d,a
	a.callAt("find")
	a.bytes(0xD0)       // ret nc
	a.bytes(0x3E, 0x01) // ld a,1
	a.bytes(0xB7)       // or a      (zero clear, carry clear)
	a.bytes(0x37)       // scf
	a.bytes(0xC9)       // ret

	// draw — put the picture the interpreter named onto the screen.
	a.here("draw")
	a.word(0x3A, cpcShownPicture) // ld a,(the picture it wants)
	a.bytes(0x57)                 // ld d,a
	a.callAt("find")
	a.bytes(0xD0) // ret nc

	// A picture the index says is on the disk is fetched first, and after that
	// there is no difference between the two: what the rest of this wants is
	// HL at the start of an entry, and a buffer full of one is an entry.
	//
	// The file is opened by the interpreter's own routine, which builds the
	// name out of the number and quiets the firmware's loading messages on the
	// way. Doing it with the interrupts off is not a liberty either: it is what
	// the interpreter does at 0x09BC, three instructions before the call this
	// one replaces.
	a.bytes(0x78) // ld a,b
	a.bytes(0x3C) // inc a         0xFF, the disk, becomes zero
	a.near(0x20, "draw.paged")

	// The number again out of the interpreter's own byte, and not out of D:
	// find ends by building the address with "ld d,(hl)", so by the time it
	// comes back the number it was handed is gone. Asking for picture zero is
	// a file that is not there, a clear carry and a drawing that never appears,
	// which is what this did until a disk with pictures on it met an Amstrad.
	a.word(0x3A, cpcShownPicture) // ld a,(the picture it wants)
	a.call(cpcOpenPicture)        // build the name and open it
	a.bytes(0xD0)                 // ret nc
	a.word(0x21, read)            // ld hl,<the buffer>
	a.bytes(0xE5)                 // push hl
	a.call(casInDirect)           // the whole of it, in one go
	a.call(casInClose)            //
	a.bytes(0xE1)                 // pop hl
	a.bytes(0x06, cpcMainMemory)  // ld b,0xc0     and no paging to do

	// Page the right memory in. On a 64K machine the index names the plain
	// configuration and this changes nothing; on a 128K one it swaps a bank
	// into the window at 0x4000. Nothing that matters is in that window —
	// the code is up here, the record buffer is at 0x0040, and the interpreter
	// has the interrupts off before it calls any of this.
	a.here("draw.paged")
	a.bytes(0x48)       // ld c,b
	a.bytes(0x06, 0x7F) // ld b,0x7f
	a.bytes(0xED, 0x49) // out (c),c

	// The inks first, through the interpreter's own routine, so that the
	// screen is already right when the pixels land on it.
	a.bytes(0xE5)                // push hl
	a.bytes(0x23, 0x23, 0x23)    // inc hl x3   (past the number and the length)
	a.word(0x11, cpcInkBuffer)   // ld de,<the interpreter's inks>
	a.word(0x01, cpcPaletteSize) // ld bc,16
	a.bytes(0xED, 0xB0)          // ldir
	a.word(0x21, cpcInkBuffer)   // ld hl,<them again>
	a.call(cpcSetPalette)        //
	a.bytes(0xE1)                // pop hl
	a.word(0x01, cpcBankHeader)  // ld bc,<the header>
	a.bytes(0x09)                // add hl,bc
	a.word(0x11, cpcScreenBase)  // ld de,0xC000

	// One band at a time, each stream onto the eighth of the display it belongs
	// to. Only the rows the picture covers are written, so whatever is already
	// printed below it stays where it is.
	a.bytes(0x06, cpcPackBands) // ld b,8
	a.here("draw.band")
	a.bytes(0xC5)                     // push bc
	a.bytes(0x4E)                     // ld c,(hl)     the stream's length
	a.bytes(0x23)                     // inc hl
	a.bytes(0x46)                     // ld b,(hl)
	a.bytes(0x23)                     // inc hl
	a.bytes(0xE5)                     // push hl       where this stream starts
	a.bytes(0x09)                     // add hl,bc     and where the next one does
	a.bytes(0xE3)                     // ex (sp),hl    keep that, take this
	a.bytes(0xD5)                     // push de
	a.callAt("dzx0")                  //
	a.bytes(0xD1)                     // pop de
	a.bytes(0x7A)                     // ld a,d
	a.bytes(0xC6, cpcPackBandSize>>8) // add a,8
	a.bytes(0x57)                     // ld d,a        the next band, 0x800 on
	a.bytes(0xE1)                     // pop hl        the next stream
	a.bytes(0xC1)                     // pop bc
	a.near(0x10, "draw.band")         // djnz

	// And put the memory back the way the interpreter believes it is.
	a.word(0x01, 0x7F00|cpcMainMemory) // ld bc,0x7fc0
	a.bytes(0xED, 0x49)                // out (c),c
	a.bytes(0xC9)                      // ret

	// find(D) — look the number up in the index. B comes back as the memory
	// configuration and HL as where the picture lies in it.
	a.here("find")
	a.at(0x21, "entries")
	a.here("find.next")
	a.bytes(0x7E)              // ld a,(hl)      the number
	a.bytes(0x3C)              // inc a
	a.near(0x28, "find.none")  // jr z,<the end of the index>
	a.bytes(0x3D)              // dec a
	a.bytes(0xBA)              // cp d
	a.near(0x28, "find.found") // jr z,<this one>
	a.bytes(0x23, 0x23, 0x23, 0x23)
	a.near(0x18, "find.next")
	a.here("find.none")
	a.bytes(0xB7) // or a      (carry clear: we have not got it)
	a.bytes(0xC9) // ret
	a.here("find.found")
	a.bytes(0x23) // inc hl
	a.bytes(0x46) // ld b,(hl)     the configuration
	a.bytes(0x23) // inc hl
	a.bytes(0x5E) // ld e,(hl)
	a.bytes(0x23) // inc hl
	a.bytes(0x56) // ld d,(hl)
	a.bytes(0xEB) // ex de,hl      and where it lies
	a.bytes(0x37) // scf
	a.bytes(0xC9) // ret

	a.here("dzx0")
	cpcDZX0(a)

	a.here("entries")

	body, err := a.resolve(at)
	if err != nil {
		return nil, 0, err
	}

	return body, a.labels["draw"], nil
}

// cpcDZX0 is the ZX0 decoder by Einar Saukas and Urusergi, the standard
// version, assembled from his own source at work/ZXDAAD128/src/asm.
//
// Sixty-eight bytes, HL the stream and DE where it goes. It is written out
// instruction by instruction rather than dropped in as a blob for the same
// reason the loader is: a blob is a thing nobody can correct.
func cpcDZX0(a *cpcAssembly) {
	a.word(0x01, 0xFFFF) // ld bc,0xffff   the offset starts at one
	a.bytes(0xC5)        // push bc
	a.bytes(0x03)        // inc bc
	a.bytes(0x3E, 0x80)  // ld a,0x80

	a.here("zx0.literals")
	a.callAt("zx0.elias") // how many
	a.bytes(0xED, 0xB0)   // ldir           copy them
	a.bytes(0x87)         // add a,a        from the last offset or a new one?
	a.near(0x38, "zx0.new")
	a.callAt("zx0.elias")

	a.here("zx0.copy")
	a.bytes(0xE3)       // ex (sp),hl
	a.bytes(0xE5)       // push hl
	a.bytes(0x19)       // add hl,de
	a.bytes(0xED, 0xB0) // ldir
	a.bytes(0xE1)       // pop hl
	a.bytes(0xE3)       // ex (sp),hl
	a.bytes(0x87)       // add a,a
	a.near(0x30, "zx0.literals")

	a.here("zx0.new")
	a.bytes(0xC1)       // pop bc
	a.bytes(0x0E, 0xFE) // ld c,0xfe
	a.callAt("zx0.elias.loop")
	a.bytes(0x0C)        // inc c
	a.bytes(0xC8)        // ret z          the end marker
	a.bytes(0x41)        // ld b,c
	a.bytes(0x4E)        // ld c,(hl)
	a.bytes(0x23)        // inc hl
	a.bytes(0xCB, 0x18)  // rr b
	a.bytes(0xCB, 0x19)  // rr c
	a.bytes(0xC5)        // push bc
	a.word(0x01, 0x0001) // ld bc,1
	a.at(0xD4, "zx0.elias.backtrack")
	a.bytes(0x03) // inc bc
	a.near(0x18, "zx0.copy")

	a.here("zx0.elias")
	a.bytes(0x0C) // inc c

	a.here("zx0.elias.loop")
	a.bytes(0x87) // add a,a
	a.near(0x20, "zx0.elias.skip")
	a.bytes(0x7E) // ld a,(hl)
	a.bytes(0x23) // inc hl
	a.bytes(0x17) // rla

	a.here("zx0.elias.skip")
	a.bytes(0xD8) // ret c

	a.here("zx0.elias.backtrack")
	a.bytes(0x87)       // add a,a
	a.bytes(0xCB, 0x11) // rl c
	a.bytes(0xCB, 0x10) // rl b
	a.near(0x18, "zx0.elias.loop")
}

// cpcPointAtTheBank rewrites the two calls that open a picture file so that
// they reach the bank instead, and takes the file reader out of the way.
//
// A binary whose bytes are not the ones expected is handed back untouched: it
// is a patch on somebody else's program and it says so rather than trusting a
// number.
func cpcPointAtTheBank(binary []byte, bank cpcBank) ([]byte, error) {
	probe, draw := bank.at, bank.at+bank.drawAt

	out := slices.Clone(binary)

	for _, patch := range []struct {
		at   int
		with []byte
	}{
		{cpcProbeAt, []byte{0xCD, byte(probe), byte(probe >> 8)}},
		{cpcDrawAt, []byte{0xCD, byte(draw), byte(draw >> 8)}},
		{cpcDrawAt + 3, []byte{0xC3, byte(cpcDrawTail & 0xFF), byte(cpcDrawTail >> 8)}},
	} {
		at := patch.at - cpcInterpreterOrigin + media.AMSDOSHeaderSize
		if at+len(patch.with) > len(out) {
			return binary, nil
		}

		copy(out[at:], patch.with)
	}

	// The two calls being replaced are the same one, and it is checked before
	// anything is written over it.
	for _, at := range []int{cpcProbeAt, cpcDrawAt} {
		was := at - cpcInterpreterOrigin + media.AMSDOSHeaderSize
		if binary[was] != 0xCD ||
			binary[was+1] != byte(cpcOpenPicture&0xFF) ||
			binary[was+2] != byte(cpcOpenPicture>>8) {
			return binary, nil
		}
	}

	return out, nil
}
