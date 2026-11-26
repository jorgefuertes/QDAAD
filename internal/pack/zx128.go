package pack

import (
	"fmt"
	"sort"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
)

// The 128K Spectrum's memory, which is the 48K's with four more banks and a
// different answer to where everything that will not fit goes.
//
// Below 0xC000 nothing changes: the interpreter at 0x6000, the database at
// 0x8400, the index and the graphics stub ending on the last byte of RAM. What
// changes is the 16K window at 0xC000, which on this machine is one of eight
// banks and can be swapped for another. DAAD uses it for everything the 48K had
// to leave out:
//
//	bank 5            the screen and the interpreter's own workspace: never free
//	bank 2            the database, and reserved whole even when it is smaller
//	bank 0            what the database spills past 0xC000, then pictures, with
//	                  the index and the stub at the top
//	bank 1            512 bytes of RAMSAVE and a 6912-byte picture buffer, then
//	                  pictures
//	banks 3, 4, 6, 7  the xmessages first, then pictures
//
// That is work/daad-ready/DOC/MemoryLayoutZX128K.txt. The order they are filled
// in is pager128k.php's, and it is not the numeric one: the four empty banks
// first, because an xmessage chunk has to start at the beginning of a bank, then
// bank 1, then whatever bank 0 has left.
//
// This is the whole of what a 128K release adds. The pictures are the same files
// a +3 carries — SCRMAKER's spectrum128 target writes both, so plus3Picture is
// what makes them — the tape is the same four blocks, and the BASIC loader is
// byte for byte the 48K one. What is new is the paging, which is this file.
const (
	// zx128BankSize is a bank.
	zx128BankSize = 0x4000
	// zx128BankWindow is where the paged bank appears. Every offset the index
	// carries is an address in this window rather than an offset into a bank,
	// which is the same number with this added.
	zx128BankWindow = 0xC000
	// zx128Reserved is what bank 1 keeps for itself: 512 bytes of RAMSAVE and
	// then a whole screen's worth of buffer, which is where the interpreter
	// uncompresses a picture before blitting it.
	zx128Reserved = 512 + screenSize
	// zx128XMessageBanks is how many banks an .XMB may spread over, and with
	// 16K each it is also where the 64K ceiling on the XMES offset comes from.
	zx128XMessageBanks = 4
	// zx128StubRoom is what the top of bank 0 is reserved for.
	//
	// It is the reference's number and not the stub's: the graphics stub is
	// 2089 bytes, and pager128k.php:148 reserves 2099. The ten bytes of slack
	// are left in rather than tidied away, because the number decides how much
	// of bank 0 a picture may have and tidying it would put a picture ten bytes
	// further up than the tape daadmaker writes.
	zx128StubRoom = 2099
	// zx128IndexPictures is how many pictures the index has room for, and it is
	// a range rather than a count: pager128k.php:290 looks for 000.128 up to
	// 254.128, so a picture numbered 255 is laid out in a bank and then never
	// written into the index, which leaves it unreachable.
	zx128IndexPictures = 255
)

// zx128Banks is the order the banks are filled in.
var zx128Banks = []int{3, 4, 6, 7, 1, 0}

// zx128Bank is a bank with something in it: its number, where its contents go
// in the window, and the contents.
//
// It becomes one headerless block on the tape. Headerless because the BASIC
// loader does not load these — it loads the four blocks below 0xC000 and jumps —
// and the interpreter reads the rest itself, pulling each one into the bank the
// index named.
type zx128Bank struct {
	number int
	at     int
	body   []byte
}

// zx128Slot is one thing in a bank: a picture or a chunk of the .XMB.
type zx128Slot struct {
	bank   int
	offset int
	body   []byte
	// picture is the number a PICTURE names it with. An xmessage chunk is not a
	// picture and has none, which is what -1 says.
	picture int
}

// ZX128 builds a .TAP for a 128K Spectrum.
//
// The tape is the 48K's and then some: the same BASIC loader, the same four
// blocks below 0xC000 — the loading screen, the interpreter at 0x6000, the
// database at 0x8400, the index and the graphics stub ending on the last byte of
// RAM — and after them one headerless block per bank with anything in it.
//
// The loader is byte for byte the 48K one, four LOADs and a jump, because the
// bank blocks are not its business: it loads what lives below the window and
// starts the interpreter, and the interpreter reads the rest off the tape itself,
// putting each block in the bank the index named. That is what the headers are
// dropped for, and it is why a 128K tape has twelve blocks where its loader
// mentions four.
func ZX128(database []byte, opts Options) ([]byte, error) {
	interpreter, stub, charset, err := zxStart(runtime.ZX128, opts)
	if err != nil {
		return nil, err
	}

	databaseOrigin := int(arch.ZX.BaseAddress(""))
	databaseEnd := databaseOrigin + len(database)

	banks, index, _, err := zx128Pages(
		opts.Images, opts.XMessages, databaseEnd, opts.Note, opts.Progress)
	if err != nil {
		return nil, err
	}

	// The index and the stub are one block, index first, ending on the last
	// byte of RAM — which on this machine is the last byte of bank 0.
	top := append(index, stub...)

	if err := replaceCharset(top, charset); err != nil {
		return nil, err
	}

	topOrigin := topOfMemory - len(top)

	if end := interpreterOrigin + len(interpreter); end > databaseOrigin {
		return nil, fmt.Errorf(
			"the interpreter ends at 0x%04X and the database starts at 0x%04X", end, databaseOrigin)
	}

	// The database runs from 0x8400 through bank 2 and on into bank 0, and what
	// stops it is the stub at the top of bank 0. Unlike the 48K there is
	// somewhere else for the texts to go, so this is the one error worth
	// pointing at a cure: XMESSAGE moves a message into the banks the pictures
	// and the xmessages share.
	if databaseEnd > topOrigin {
		return nil, fmt.Errorf(
			"the database is %d bytes and %d is all there is between 0x%04X and the graphics "+
				"stub at 0x%04X: move some messages to XMESSAGE, which puts their text in the "+
				"upper banks",
			len(database), topOrigin-databaseOrigin, databaseOrigin, topOrigin)
	}

	game := []tapeBlock{
		{'I', interpreterOrigin, interpreter},
		{'D', databaseOrigin, database},
		{'G', topOrigin, top},
	}

	var loader media.BASIC

	loader.Clear(interpreterOrigin - 1)

	parts := game

	if len(opts.Loading) > 0 {
		loader.BlackScreen()
		loader.LoadCode(1)
		loader.HideLoadingMessages()

		parts = append([]tapeBlock{{'S', screenBase, opts.Loading}}, game...)
	}

	loader.LoadCode(len(game))
	loader.RandomizeUSR(interpreterOrigin)

	var tape media.TAP

	tape.Program(blockName(opts.Name, 0), loader.Bytes(), media.FirstLine)

	for _, part := range parts {
		if err := tape.Code(blockName(opts.Name, part.letter), part.at, part.body); err != nil {
			return nil, err
		}
	}

	// The banks last, in the order the index lists them, which is the order the
	// interpreter will ask for them in.
	for _, bank := range banks {
		tape.Headerless(bank.body)
	}

	return tape.Bytes(), nil
}

// zx128Pages lays the xmessages and the pictures out across the banks and
// writes the index that says where each of them landed.
//
// databaseEnd is one past the database's last byte, which is what decides how
// much of bank 0 the database has taken. offered is every picture the build has,
// whether or not it ends up fitting: the room at the top of bank 0 depends on
// how many entries the index will have, and the reference counts the .128 files
// on disk rather than the ones it managed to place.
func zx128Pages(
	pictures map[int][]byte, xmessages []byte, databaseEnd int, note Notes, report Progress,
) ([]zx128Bank, []byte, []int, error) {
	// What the database spilled past 0xC000, and so where bank 0's own
	// contents start.
	//
	// The extra byte is the reference's (pager128k.php:140) and not an
	// off-by-one corrected on the way through: a database ending exactly at
	// 0xC000 leaves bank 0 whole, and one byte past it loses two. Matching the
	// tape matters more than the byte.
	spare := 0
	if databaseEnd >= zx128BankWindow {
		spare = databaseEnd - zx128BankWindow + 1
	}

	built, numbers, err := zx128Pictures(pictures, note, report)
	if err != nil {
		return nil, nil, nil, err
	}

	// The top of bank 0 goes to the stub, the index and the list of banks, and
	// what is left is all a picture may have there.
	room := make([]int, len(zx128Banks))
	for i := range zx128Banks {
		room[i] = zx128BankSize
	}

	room[len(room)-1] = zx128BankSize - zx128StubRoom -
		zx128IndexEntry*len(pictures) - zx128IndexFixed(len(zx128Banks))

	// Where the next thing in each bank goes. The four empty banks start at
	// nothing, bank 1 after its buffers, bank 0 after the database.
	offset := make([]int, len(zx128Banks))
	offset[len(offset)-2] = zx128Reserved
	offset[len(offset)-1] = spare

	var slots []zx128Slot

	// The xmessages first, and each chunk at the very start of its bank. That
	// is not tidiness: the offset an XMES condact carries is a position in all
	// of them laid end to end, so the interpreter finds the bank with
	// offset/16384 and the byte with offset%16384. A chunk starting anywhere
	// but zero moves every text the linker has already pointed at.
	for i, chunk := range zx128Chunks(xmessages) {
		slots = append(slots, zx128Slot{bank: zx128Banks[i], offset: 0, body: chunk, picture: -1})
		offset[i] = len(chunk)
	}

	// Then the pictures, biggest first, each into the first bank with room for
	// it. Biggest first because the gaps left over are what a later picture has
	// to fit in, and a big one has nowhere else to go.
	order := make([]int, len(numbers))
	copy(order, numbers)
	sort.SliceStable(order, func(i, j int) bool {
		return len(built[order[i]]) > len(built[order[j]])
	})

	var dropped []int

	for _, n := range order {
		body := built[n]

		placed := false

		for i := range zx128Banks {
			if offset[i]+len(body) > room[i] {
				continue
			}

			slots = append(slots, zx128Slot{
				bank: zx128Banks[i], offset: offset[i], body: body, picture: n,
			})
			offset[i] += len(body)
			placed = true

			break
		}

		if !placed {
			dropped = append(dropped, n)
		}
	}

	sort.Ints(dropped)

	if len(dropped) > 0 {
		note.say(NoticeLeftOut,
			"%d of the %d pictures did not fit in the banks and are not on this build: %s",
			len(dropped), len(pictures), numberList(dropped))
	}

	banks := zx128Assemble(slots)

	return banks, zx128Index(slots, spare), dropped, nil
}

// zx128Pictures turns every screen into the file the interpreter reads.
//
// It is plus3Picture and nothing else: SCRMAKER writes the .128 of a 128K build
// and of a +3 build with the same procedure — five bytes of geometry and the
// whole cropped screen as one ZX0 stream, SCRMAKER.PAS:1083-1093 — so there is
// one picture format for the two machines and this is where that is said.
//
// Pictures the index cannot name are dropped here rather than laid out and lost
// later. See zx128IndexPictures.
func zx128Pictures(
	pictures map[int][]byte, note Notes, report Progress,
) (map[int][]byte, []int, error) {
	numbers := make([]int, 0, len(pictures))
	unreachable := make([]int, 0)

	for n := range pictures {
		if n >= zx128IndexPictures {
			unreachable = append(unreachable, n)

			continue
		}

		numbers = append(numbers, n)
	}

	sort.Ints(numbers)
	sort.Ints(unreachable)

	if len(unreachable) > 0 {
		note.say(NoticeLeftOut,
			"the index names pictures 0 to %d, so %s cannot be reached and are not on this build",
			zx128IndexPictures-1, numberList(unreachable))
	}

	bodies, err := compressPictures(numbers, report, func(n int) ([]byte, error) {
		one, err := plus3Picture(pictures[n])
		if err != nil {
			return nil, fmt.Errorf("picture %d: %w", n, err)
		}

		return one, nil
	})
	if err != nil {
		return nil, nil, err
	}

	built := make(map[int][]byte, len(numbers))
	for i, n := range numbers {
		built[n] = bodies[i]
	}

	// Two pictures with the same bytes are two copies of one drawing taking
	// twice the room, and on this machine room is the thing that runs out. It
	// is said rather than fixed, because which of the two the adventure should
	// stop naming is not ours to decide.
	seen := make(map[string]int, len(numbers))

	for _, n := range numbers {
		key := string(built[n])

		if first, repeat := seen[key]; repeat {
			note.say(NoticeSaid,
				"pictures %d and %d are the same drawing: using one of them twice would save %d bytes",
				first, n, len(built[n]))

			continue
		}

		seen[key] = n
	}

	return built, numbers, nil
}

// zx128Chunks cuts the .XMB into one piece per bank.
//
// Cut rather than padded: the interpreter reads a fixed-size block from wherever
// the offset lands and lets the printing routine stop at the 0xF5 that ends the
// message, so the tail of the last chunk is never read as text. The reference
// takes 16K slices and writes however much the last one comes to
// (pager128k.php:80-84), which is why the second block of a real 128K tape is
// not a round 16K.
//
// Four chunks is all there is room for, and it is the same 64K the 16-bit offset
// in an XMES allows, so an .XMB past it was already refused by the linker.
func zx128Chunks(xmessages []byte) [][]byte {
	var out [][]byte

	for i := range zx128XMessageBanks {
		at := i * zx128BankSize
		if at >= len(xmessages) {
			break
		}

		out = append(out, xmessages[at:min(at+zx128BankSize, len(xmessages))])
	}

	return out
}

// zx128Assemble gathers each bank's slots into the block that loads into it.
//
// The slots of a bank are contiguous and in the order they were placed, so a
// bank's block is simply its slots joined — the one exception being bank 0,
// whose block starts after the database's spill because the database got there
// first.
func zx128Assemble(slots []zx128Slot) []zx128Bank {
	out := make([]zx128Bank, 0, len(zx128Banks))

	for _, number := range zx128Banks {
		var (
			body []byte
			at   = -1
		)

		for _, s := range slots {
			if s.bank != number {
				continue
			}

			if at < 0 {
				at = s.offset
			}

			body = append(body, s.body...)
		}

		if len(body) == 0 {
			continue
		}

		out = append(out, zx128Bank{number: number, at: zx128BankWindow + at, body: body})
	}

	return out
}

// What the index is made of, beyond the pictures themselves.
const (
	// zx128IndexEntry is one picture: its number, where it is, how long it is,
	// and which bank it is in.
	zx128IndexEntry = 6
	// zx128IndexHead is the word of bank 0's size and the word of its offset,
	// which the pictures there are measured from.
	zx128IndexHead = 4
	// zx128IndexMarkers is the 0xFF after the list of banks and the 0xFF after
	// the list of pictures.
	zx128IndexMarkers = 2
)

// zx128IndexFixed is everything in the index that is not a picture, for the
// worst case of every bank being used. The reference reserves for that case
// rather than for the one in hand (pager128k.php:152), and so does this: a
// build that reserved only for the banks it filled would move its pictures.
func zx128IndexFixed(banks int) int {
	return zx128IndexHead + banks + zx128IndexMarkers
}

// zx128Index writes the table that says where everything landed.
//
// Like the 48K's, it is assembled forwards and written out reversed, because the
// interpreter reads it *downwards* from the byte just below the graphics stub.
// Laid out in the order the interpreter reads it:
//
//	a word: how many bytes of bank 0 hold pictures
//	a word: where in the window those start
//	the number of each bank that has something in it
//	0xFF:   no more banks
//	then six bytes per picture — its number, where it is, how long it is, and
//	its bank —
//	0xFF:   no more pictures
//
// Six where the 48K has three, and that is the whole of the /48 flag daadmaker
// takes: the two indexes are read by the same routine and differ in the stride.
//
// The pictures go in number order and not in the order they were placed, so that
// two builds of one adventure come out the same tape.
//
// A build with nothing in the banks still writes an index, for the reason the
// 48K one does: the interpreter looks the table up on every PICTURE, and the
// official template runs one before printing anything.
func zx128Index(slots []zx128Slot, spare int) []byte {
	const (
		noMoreBanks    = 0xFF
		noMorePictures = 0xFF
	)

	// How much of bank 0 the pictures there come to, measured from the window
	// and then taken back to the database's spill.
	//
	// With a database that spilled and no picture in bank 0 this goes negative,
	// and the two bytes written are the low half of that. It is the reference's
	// arithmetic (pager128k.php:263-271) and the interpreter never reads them in
	// that case — bank 0 is not in the list of banks, so there is nothing there
	// to look for.
	end := 0

	for _, s := range slots {
		if s.bank == 0 && s.offset+len(s.body) > end {
			end = s.offset + len(s.body)
		}
	}

	size := end - spare

	forwards := []byte{
		byte(size >> 8), byte(size),
		byte((zx128BankWindow + spare) >> 8), byte(zx128BankWindow + spare),
	}

	for _, number := range zx128Banks {
		for _, s := range slots {
			if s.bank == number {
				forwards = append(forwards, byte(number))

				break
			}
		}
	}

	forwards = append(forwards, noMoreBanks)

	pictures := make([]zx128Slot, 0, len(slots))

	for _, s := range slots {
		if s.picture >= 0 {
			pictures = append(pictures, s)
		}
	}

	sort.Slice(pictures, func(i, j int) bool { return pictures[i].picture < pictures[j].picture })

	for _, s := range pictures {
		at := zx128BankWindow + s.offset

		forwards = append(forwards,
			byte(s.picture),
			byte(at), byte(at>>8),
			byte(len(s.body)), byte(len(s.body)>>8),
			byte(s.bank))
	}

	forwards = append(forwards, noMorePictures)

	backwards := make([]byte, len(forwards))
	for i, b := range forwards {
		backwards[len(forwards)-1-i] = b
	}

	return backwards
}
