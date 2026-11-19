package pack

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/media"
)

// The loader that shows a picture while an Amstrad release goes in.
//
// Without one a CPC release is a single binary and RUN"NAME" enters the
// interpreter, which draws nothing before it starts. Showing something first
// means a program that runs before it, and there is nowhere to put that but in
// front: a page of code that sets the mode and the palette, drops the picture
// on the display, pulls in whatever else the release carries, and jumps into
// the game.
//
// It is machine code and not Locomotive BASIC, which would have been four lines
// and a tokeniser we do not have. What it costs is having to place the pieces by
// hand, and these are the two addresses that takes:
//
//	0x0040  the two kilobytes CAS IN OPEN reads a record into. It is the
//	        firmware's own low memory, it runs to exactly 0x0840, and it is what
//	        the interpreter itself passes — which is the point: a buffer below
//	        the game is a buffer no load can walk over.
//	 top    the loader, in the last page below the interpreter's graphics
//	        block. At the top and not above the game, because what goes above
//	        the game is the adventure's own pictures and they want every byte
//	        of it. See cpcbank.go.
//
// The order matters and is the whole point: the picture goes up first and
// everything else loads underneath it, so it is on the screen for as long as
// the loading takes rather than appearing when it is over.
const (
	// cpcLoaderBuffer is where a record is read while a file loads: the
	// firmware's low memory, which ends exactly where the game begins.
	cpcLoaderBuffer = 0x0040
	// cpcBufferSize is how much of a file goes into it at a time, which is two
	// kilobytes because that is what a record is. It is the same two kilobytes
	// an xmessage file holds, and not by coincidence: see cpcxmessages.go.
	cpcBufferSize = cpcInterpreterOrigin - cpcLoaderBuffer
	// cpcLoaderSuffix and cpcGameSuffix are what the two files are called.
	// The loader takes the plain name because it is the one a player types.
	cpcLoaderSuffix = ".BIN"
	cpcScreenSuffix = ".SCR"
	cpcGameSuffix   = ".GAM"
	// cpcBankSuffix is the block of pictures that travels in memory with the
	// game. See cpcbank.go.
	cpcBankSuffix = ".PIC"
)

// The firmware entries the loader calls.
const (
	scrSetMode    = 0xBC0E
	scrSetInk     = 0xBC32
	scrSetBorder  = 0xBC38
	casNoisy      = 0xBC6B
	casStartMotor = 0xBC6E
	casInOpen     = 0xBC77
	casInClose    = 0xBC7A
	casInDirect   = 0xBC83
	// The three of the kernel's jumpblock that put AMSDOS back. They are one
	// entry along from where a careless reading puts them: 0xBCC8 is KL CHOKE
	// OFF and not KL ROM WALK, and calling that one is what an afternoon went
	// on.
	klRomWalk      = 0xBCCB
	klInitBack     = 0xBCCE
	klFindCommand  = 0xBCD4
	mcStartProgram = 0xBD16
)

// cpcLoaderSize is the room kept for the loader below the interpreter's
// graphics block.
//
// Two pages and not one, which is what it was. One was enough for the loader
// itself and stopped being enough for its tail: every file it has to read costs
// forty bytes of name and address, and a 6128 release with its pictures in all
// four spare banks reads eight files. At three blocks it was already twenty-four
// bytes past the page and nothing said so; at four it was sixty-four and the
// game came back to BASIC. cpcLoaderCode refuses rather than overrun now.
const cpcLoaderSize = 0x200

// cpcLoaderOrigin is where the loader sits: below the interpreter's graphics
// block.
//
// At the top and not above the game, because what goes above the game is the
// pictures, and they want every byte of it. It stays out of the way of all
// three loads: the picture goes to the display, the pictures to the ground
// under it, and the game to 0x0840.
func cpcLoaderOrigin() int { return cpcGraphicsTop - cpcLoaderSize }

// cpcLoaderPalette is what a loader is given: the sixteen pens and the border,
// as the firmware numbers its colours, which is what qundaad writes beside a
// loading screen.
const cpcPaletteFile = 17

// CPCLoaderRelease is a CPC release of three files rather than one: the loader,
// the picture it shows, and the game it then runs.
//
// The game keeps the whole of the binary it would have been on its own — the
// interpreter, the database and the graphics block, glued the way MCRF glues
// them — and only its name changes, because the name a player types has to be
// the loader's.
// cpcLoad is a file the loader pulls in before it enters the game. A config of
// zero means plain memory; anything else is a gate array configuration that has
// to be in force while the file goes in, which is how a picture bank reaches
// the spare RAM of a 6128.
type cpcLoad struct {
	name   string
	at     int
	config int
}

// cpcMedium is what the release will be read from, which the loader has to know
// because putting AMSDOS back is right on a disk and wrong on a tape.
type cpcMedium int

const (
	cpcOnDisc cpcMedium = iota
	cpcOnTape
)

func cpcLoaderRelease(
	database []byte, opts Options, medium cpcMedium, first ...cpcLoad,
) (*Release, error) {
	game, err := cpcBinary(database, opts)
	if err != nil {
		return nil, err
	}

	screen, palette, err := cpcLoadingScreen(opts)
	if err != nil {
		return nil, err
	}

	body := media.WithoutAMSDOSHeader(game)

	origin := cpcLoaderOrigin()

	loader, err := cpcLoaderCode(opts.Name, palette, origin, medium, first)
	if err != nil {
		return nil, err
	}

	header, err := media.AMSDOSHeader(opts.Name+cpcLoaderSuffix, origin, origin, len(loader))
	if err != nil {
		return nil, err
	}

	screenHeader, err := media.AMSDOSHeader(opts.Name+cpcScreenSuffix,
		cpcScreenBase, cpcScreenBase, len(screen))
	if err != nil {
		return nil, err
	}

	gameHeader, err := media.AMSDOSHeader(opts.Name+cpcGameSuffix,
		cpcInterpreterOrigin, cpcInterpreterOrigin, len(body))
	if err != nil {
		return nil, err
	}

	return NewRelease(
		File{
			Name: opts.Name + cpcLoaderSuffix,
			Body: append(header, loader...),
			Note: `sets the screen up and pulls the rest in: RUN"` + opts.Name + `"`,
		},
		File{
			Name: opts.Name + cpcScreenSuffix,
			Body: append(screenHeader, screen...),
			Note: "the picture shown while the game loads",
		},
		File{
			Name: opts.Name + cpcGameSuffix,
			Body: append(gameHeader, body...),
			Note: "interpreter, database and graphics in one file",
		},
	), nil
}

// cpcLoadingScreen checks what the picture has to be: a whole display and the
// colours to read it through.
func cpcLoadingScreen(opts Options) (screen, palette []byte, err error) {
	if len(opts.Loading) != cpcPackScreen {
		return nil, nil, fmt.Errorf(
			"a CPC loading screen is the whole of video memory, %d bytes, and this is %d",
			cpcPackScreen, len(opts.Loading))
	}

	if len(opts.LoadingPalette) < cpcPaletteFile {
		return nil, nil, fmt.Errorf(
			"a CPC loading screen needs its %d colours beside it: sixteen pens and the border",
			cpcPaletteFile)
	}

	return opts.Loading, opts.LoadingPalette[:cpcPaletteFile], nil
}

// cpcLoaderCode assembles the loader.
//
// It is written out as the instructions it is rather than as a blob, because a
// blob is a thing nobody can correct. Every call is a firmware entry, and the
// addresses that point into the loader's own tail are left as labels and
// written over once everything has a place — so that adding an instruction
// cannot quietly move them.
func cpcLoaderCode(
	name string, palette []byte, origin int, medium cpcMedium, first []cpcLoad,
) ([]byte, error) {
	// The tables the code refers to, written after it.
	type tail struct {
		name string
		body []byte
	}

	tails := []tail{{"palette", palette}}

	if medium == cpcOnDisc {
		tails = append(tails,
			// The RSX names KL FIND COMMAND reads: upper case, the last letter
			// of each with its top bit set, and a zero to close the list.
			// AMSDOS's first command is DISC, and finding it is finding AMSDOS.
			tail{"disc", []byte{'D', 'I', 'S', 'C' | 0x80, 0}},
			// Where the number of the ROM it was found in is kept across the
			// restart, which is why it is a byte of memory and not a register.
			tail{"slot", []byte{0}})
	}

	a := &cpcAssembly{}

	// Putting AMSDOS back is the one thing nothing on paper suggests, and it
	// is only ever right on a disk.
	//
	// RUN"NAME" on a binary goes through MC START PROGRAM, which resets the
	// firmware before entering it — and a reset firmware is one whose cassette
	// vectors point at the cassette again, AMSDOS's patches gone. A loader
	// started that way asks the tape for its files with the disk in the drive,
	// and the machine answers "Press PLAY then any key".
	//
	// What puts it back is four steps, and they are the interpreter's own, at
	// 0x2458: walk the background ROMs, ask which one answers to DISC, restart
	// the firmware with every ROM enabled, and then initialise that one by
	// number. The restart in the middle is not optional — walking the ROMs
	// without it leaves the machine back at the BASIC prompt.
	//
	// A tape gets none of it. The walk alone is enough to bring AMSDOS back on
	// any machine that has a drive at all, patches and all, and from then on
	// the files are asked of a disk that is not there: "Drive A: disc missing".
	// Left as MC START PROGRAM handed it over, the firmware is already reading
	// the tape the RUN" came from, which is the whole of what a tape needs.
	if medium == cpcOnDisc {
		a.word(0x11, origin)       // ld de,<the loader: the lowest we need kept>
		a.word(0x21, cpcCeiling-1) // ld hl,<the highest>
		a.call(klRomWalk)
		a.at(0x21, "disc") // ld hl,<the names>
		a.call(klFindCommand)
		a.bytes(0x30, 0x04) // jr nc,+4   (not found: the slot stays zero)
		a.bytes(0x79)       // ld a,c
		a.at(0x32, "slot")  // ld (slot),a
		a.bytes(0x0E, 0xFF) // ld c,0xff  (every ROM on)
		a.at(0x21, "restarted")
		a.jump(mcStartProgram)

		a.here("restarted")
		a.bytes(0xC5, 0xE1) // push bc / pop hl — the free memory MC START PROGRAM hands back
		a.at(0x3A, "slot")  // ld a,(slot)
		a.bytes(0x4F)       // ld c,a
		a.bytes(0xB7)       // or a
		a.callIf(klInitBack)
	}

	// And then tell the cassette manager to be quiet.
	//
	// It is what turns a tape from a firmware demonstration into a release.
	// Left noisy, the machine prints "Loading AO1.SCR block 3" over the picture
	// it has just drawn, and stops between files to ask for a key. Quiet, it
	// prints nothing and does not stop: the motor runs and the files go in
	// under the picture, which is what a loading screen is for.
	//
	// It costs the prompt as well, which is the right trade: the one that
	// matters comes from the RUN" that started the loader, before any of this,
	// and the player has already answered it.
	//
	// On a disk AMSDOS says nothing either way, so this is simply ignored.
	a.bytes(0x3E, 0x01) // ld a,1   (anything but zero is quiet)
	a.call(casNoisy)

	// The mode next: it clears the screen, so anything done before it is done
	// twice.
	a.bytes(0x3E, byte(cpcLoaderMode(palette)))
	a.call(scrSetMode)

	// The border before the pens, so the frame is right from the first moment.
	a.bytes(0x06, palette[16], 0x48) // ld b,border / ld c,b
	a.call(scrSetBorder)

	// Then the sixteen pens: A counts them and HL walks the table.
	a.at(0x21, "palette")
	a.bytes(0xAF) // xor a

	loop := len(a.code)

	a.bytes(0x46, 0x48) // ld b,(hl) / ld c,b
	a.bytes(0xF5, 0xE5) // push af / push hl
	a.call(scrSetInk)
	a.bytes(0xE1, 0xF1) // pop hl / pop af
	a.bytes(0x23, 0x3C) // inc hl / inc a
	a.bytes(0xFE, 0x10) // cp 16
	a.bytes(0x20, byte(int8(loop-(len(a.code)+2))))

	// The picture first, straight onto the display; then whatever travels with
	// the game — the adventure's own pictures, where there are any — and then
	// the game, all of it loading underneath what is already on the screen.
	loads := make([]cpcLoad, 0, len(first)+2)
	loads = append(loads, cpcLoad{name: name + cpcScreenSuffix, at: cpcScreenBase})
	loads = append(loads, first...)
	loads = append(loads, cpcLoad{name: name + cpcGameSuffix, at: cpcInterpreterOrigin})

	for i, load := range loads {
		label := fmt.Sprintf("load%d", i)
		tails = append(tails, tail{label, []byte(load.name)})

		// The motor, by hand. With the messages turned off the firmware does
		// not ask for PLAY, and asking for PLAY is where it would have started
		// the motor. On a disk this does nothing at all.
		a.call(casStartMotor)
		a.bytes(0x06, byte(len(load.name))) // ld b,<length of the name>
		a.at(0x21, label)                   // ld hl,<the name>
		a.word(0x11, cpcLoaderBuffer)       // ld de,<the record buffer>
		a.call(casInOpen)

		// A file bound for a spare bank is read with that bank in the window.
		// Nothing the firmware touches is in it: the code is in ROM, the
		// workspace is above 0xB100 and the record buffer is at 0x0040.
		if load.config != 0 {
			a.word(0x01, 0x7F00|load.config) // ld bc,0x7fcn
			a.bytes(0xED, 0x49)              // out (c),c
		}

		a.word(0x21, load.at) // ld hl,<where it goes>
		a.call(casInDirect)

		if load.config != 0 {
			a.word(0x01, 0x7F00|cpcMainMemory) // ld bc,0x7fc0
			a.bytes(0xED, 0x49)                // out (c),c
		}

		a.call(casInClose)
	}

	// And into the interpreter, which never comes back.
	a.jump(cpcInterpreterOrigin)

	for _, t := range tails {
		a.here(t.name)
		a.bytes(t.body...)
	}

	code, err := a.resolve(origin)
	if err != nil {
		return nil, err
	}

	// What is above the loader is the interpreter's graphics block, and running
	// into it is not a thing that shows up as a wrong pixel: the game comes back
	// to BASIC and there is nothing on screen to say why. See cpcLoaderSize.
	if len(code) > cpcLoaderSize {
		return nil, fmt.Errorf(
			"the loader and the %d files it reads come to %d bytes, and there is room for %d "+
				"below the interpreter's graphics block",
			len(first)+len(tails), len(code), cpcLoaderSize)
	}

	return code, nil
}

// cpcAssembly is enough of an assembler to write the loader without counting
// bytes by hand: instructions go in, and the addresses that point at labels are
// written over once everything has a place.
type cpcAssembly struct {
	code   []byte
	labels map[string]int
	fixups []cpcFixup
}

// cpcFixup is an address in the code that is waiting for its label. A relative
// one is a jr, which carries a displacement rather than an address.
type cpcFixup struct {
	at       int
	label    string
	relative bool
}

func (a *cpcAssembly) bytes(b ...byte) { a.code = append(a.code, b...) }

// word is an instruction with an address in it, such as ld de,nnnn.
func (a *cpcAssembly) word(opcode byte, value int) {
	a.bytes(opcode, byte(value), byte(value>>8))
}

func (a *cpcAssembly) call(to int)   { a.word(0xCD, to) }
func (a *cpcAssembly) callIf(to int) { a.word(0xC4, to) } // call nz
func (a *cpcAssembly) jump(to int)   { a.word(0xC3, to) }

// callAt is the same, to a label.
func (a *cpcAssembly) callAt(label string) { a.at(0xCD, label) }

// near is a relative jump to a label: jr, or jr on a condition.
func (a *cpcAssembly) near(opcode byte, label string) {
	a.bytes(opcode)
	a.fixups = append(a.fixups, cpcFixup{at: len(a.code), label: label, relative: true})
	a.bytes(0)
}

// at is an instruction whose address is a label, filled in later.
func (a *cpcAssembly) at(opcode byte, label string) {
	a.bytes(opcode)
	a.fixups = append(a.fixups, cpcFixup{at: len(a.code), label: label})
	a.bytes(0, 0)
}

// here says that a label is at this point.
func (a *cpcAssembly) here(name string) {
	if a.labels == nil {
		a.labels = map[string]int{}
	}

	a.labels[name] = len(a.code)
}

// resolve writes the addresses in, now that the code is laid out and the origin
// is known.
func (a *cpcAssembly) resolve(origin int) ([]byte, error) {
	for _, f := range a.fixups {
		at, known := a.labels[f.label]
		if !known {
			return nil, fmt.Errorf("the loader refers to %q and never says where it is", f.label)
		}

		if f.relative {
			step := at - (f.at + 1)
			if step < -128 || step > 127 {
				return nil, fmt.Errorf("%q is %d bytes away, which is too far for a jr",
					f.label, step)
			}

			a.code[f.at] = byte(int8(step))

			continue
		}

		address := origin + at
		a.code[f.at], a.code[f.at+1] = byte(address), byte(address>>8)
	}

	return a.code, nil
}

// cpcLoaderMode is which mode the picture wants, read off its palette the same
// way the decompiler reads it: mode 1 has four pens, and a picture that asks for
// more than four colours is not one.
func cpcLoaderMode(palette []byte) int {
	distinct := map[byte]bool{}
	for _, ink := range palette[:16] {
		distinct[ink] = true
	}

	if len(distinct) <= 4 {
		return 1
	}

	return 0
}
