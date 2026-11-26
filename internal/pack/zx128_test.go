package pack

import (
	"os"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
	"github.com/stretchr/testify/require"
)

// The 128K release is the 48K one with the banks above 0xC000 put to use, so
// what these check is the paging and the table that describes it. Everything
// below the window — the loader, the three addresses, the stub on the last byte
// of RAM — is the 48K's and is checked there.
//
// The oracle for the lot is a real DAAD READY tape; see
// TestTheRealTapeAgrees at the bottom, which is the one test here that can
// actually be wrong about the format rather than about this code.

// A tape with nothing in the banks still has to be a tape. Most of the corpus
// has neither pictures nor xmessages, and the index has to be written anyway:
// the interpreter reads it on every PICTURE, and the official template runs one
// before it prints a word.
func TestABuildWithEmptyBanksIsStillATape(t *testing.T) {
	tape, err := ZX128(make([]byte, 2000), Options{
		Language: language.LanguageSpanish, Name: "prueba",
	})
	require.NoError(t, err)

	blocks := split(t, tape)

	// A loader and three CODE blocks, each with its header, and nothing after
	// them: eight blocks and no headerless one.
	require.Len(t, blocks, 8)

	var at, length []int

	for _, block := range blocks {
		if block[0] != 0x00 || block[1] != 3 {
			continue
		}

		length = append(length, int(block[12])|int(block[13])<<8)
		at = append(at, int(block[14])|int(block[15])<<8)
	}

	require.Equal(t, []int{0x6000, 0x8400}, at[:2])
	require.Equal(t, 2000, length[1])
	require.Equal(t, 0x10000, at[2]+length[2], "the stub ends on the last byte of RAM")
}

// The interpreter is the 128K's own and not the 48K's, and it is the one DAAD
// READY ships for version 3.
func TestTheInterpreterIsThe128KOne(t *testing.T) {
	name, _, err := runtime.Interpreter(runtime.ZX128, language.LanguageEnglish)
	require.NoError(t, err)
	require.Equal(t, "DS128KE3.BIN", name)
}

// The index is read downwards from the byte below the stub, so it is written
// backwards. With nothing in the banks it is the two words of bank zero, the
// marker that closes an empty list of banks, and the marker that closes an empty
// list of pictures.
func TestAnEmptyIndexIsTheTwoWordsAndTwoMarkers(t *testing.T) {
	index := zx128Index(nil, 0)

	require.Equal(t, []byte{0xFF, 0xFF, 0x00, 0xC0, 0x00, 0x00}, index)
}

// An xmessage chunk has to begin at the start of its bank, and that is not
// tidiness. The offset an XMES carries is a position in every chunk laid end to
// end: the interpreter divides by the bank size to find the bank and takes the
// remainder as the byte. A chunk one byte into its bank moves every text the
// linker already pointed at.
func TestXMessageChunksFillWholeBanksFromTheirStart(t *testing.T) {
	// Two banks' worth and a bit, so that the last chunk is a short one.
	xmb := make([]byte, 2*zx128BankSize+100)
	for i := range xmb {
		xmb[i] = byte(i)
	}

	banks, index, dropped, err := zx128Pages(nil, xmb, 0x8400, nil, nil)
	require.NoError(t, err)
	require.Empty(t, dropped)

	require.Len(t, banks, 3)
	require.Equal(t, []int{3, 4, 6}, []int{banks[0].number, banks[1].number, banks[2].number})

	for _, bank := range banks {
		require.Equal(t, zx128BankWindow, bank.at, "every chunk starts at the top of its bank")
	}

	require.Equal(t, zx128BankSize, len(banks[0].body))
	require.Equal(t, zx128BankSize, len(banks[1].body))
	require.Equal(t, 100, len(banks[2].body), "the last chunk is not padded out")

	// The three banks in the index, in the order they are filled, and no
	// pictures.
	require.Equal(t, []byte{0xFF, 0xFF, 0x06, 0x04, 0x03, 0x00, 0xC0, 0x00, 0x00}, index)
}

// An .XMB past four banks is past the 64K an XMES offset can name, so the
// linker refused it long before here. What this pins is that nothing is written
// into a fifth bank, where there is none.
func TestXMessagesStopAtFourBanks(t *testing.T) {
	require.Len(t, zx128Chunks(make([]byte, 5*zx128BankSize)), zx128XMessageBanks)
}

// Bank 1 is not empty when the build starts: the interpreter keeps 512 bytes of
// RAMSAVE there and a whole screen's worth of buffer to uncompress a picture
// into. A picture written over either of them is a picture that corrupts a save
// or itself.
func TestBankOneStartsAfterItsBuffers(t *testing.T) {
	// Fill the four free banks so that the next picture has to go to bank 1.
	xmb := make([]byte, zx128XMessageBanks*zx128BankSize)

	banks, _, dropped, err := zx128Pages(
		map[int][]byte{7: blankScreen()}, xmb, 0x8400, nil, nil)
	require.NoError(t, err)
	require.Empty(t, dropped)

	require.Len(t, banks, 5)
	require.Equal(t, 1, banks[4].number)
	require.Equal(t, zx128BankWindow+zx128Reserved, banks[4].at)
}

// Bank 0 is where the database spills to, so what is left of it for pictures
// starts after the database and stops before the stub at the top.
func TestBankZeroStartsAfterTheDatabaseSpill(t *testing.T) {
	// The four free banks full of xmessages, and two pictures that will not
	// both fit in what is left of bank 1: the second has nowhere but bank 0.
	xmb := make([]byte, zx128XMessageBanks*zx128BankSize)

	// A database running 0x400 bytes past the window.
	const spill = 0x400

	banks, index, dropped, err := zx128Pages(
		map[int][]byte{1: noisyScreen(1), 2: noisyScreen(2)}, xmb,
		zx128BankWindow+spill-1, nil, nil)
	require.NoError(t, err)
	require.Empty(t, dropped)

	last := banks[len(banks)-1]
	require.Equal(t, 0, last.number)
	require.Equal(t, zx128BankWindow+spill, last.at,
		"the pictures in bank 0 begin where the database stopped")

	// And the index says so, in the word it keeps for bank 0 alone.
	offset := int(index[len(index)-3])<<8 | int(index[len(index)-4])
	require.Equal(t, zx128BankWindow+spill, offset)

	size := int(index[len(index)-1])<<8 | int(index[len(index)-2])
	require.Equal(t, len(last.body), size)
}

// Biggest first, because the gaps a big picture could have used are the ones a
// small one fits in anyway. Filling in number order wastes the only resource
// this machine is short of.
func TestPicturesGoInBiggestFirst(t *testing.T) {
	// Two pictures of different sizes, and a bank with room for the larger
	// alone: the larger has to be the one that goes in.
	small, err := plus3Picture(blankScreen())
	require.NoError(t, err)

	big, err := plus3Picture(noisyScreen(1))
	require.NoError(t, err)

	require.Greater(t, len(big), len(small), "the noisy one has to compress worse")

	// Three banks of xmessages, and bank 7 left with room for exactly the big
	// picture. Both pictures fit either way round — what differs is where:
	// biggest first puts the big one in the gap it exactly fills, and number
	// order puts the small one there and sends the big one on to bank 1.
	//
	// So bank 7's size is the whole test.
	xmb := make([]byte, 3*zx128BankSize+zx128BankSize-len(big))

	banks, _, dropped, err := zx128Pages(
		map[int][]byte{1: blankScreen(), 2: noisyScreen(1)}, xmb, 0x8400, nil, nil)
	require.NoError(t, err)
	require.Empty(t, dropped)

	require.Equal(t, 7, banks[3].number)
	require.Equal(t, zx128BankSize, len(banks[3].body),
		"bank 7 is full, so the big picture is the one that went into it")
}

// A picture that fits nowhere is left out and said out loud. The reference stops
// the build instead, and that is the one place this deliberately differs: a
// build that carries forty pictures of forty-one is an adventure somebody can
// play, and a build that carries none is not.
func TestAPictureThatFitsNowhereIsDroppedAndSaidOutLoud(t *testing.T) {
	// The four free banks full of xmessages and a database filling bank 0 up to
	// the stub, which leaves bank 1 and nothing else. Four screens that will
	// not compress do not fit in bank 1 between them.
	xmb := make([]byte, zx128XMessageBanks*zx128BankSize)

	pictures := map[int][]byte{}
	for n := 1; n <= 4; n++ {
		pictures[n] = noisyScreen(uint32(n))
	}

	var said []string

	note := func(level Notice, text string) {
		if level == NoticeLeftOut {
			said = append(said, text)
		}
	}

	_, _, dropped, err := zx128Pages(
		pictures, xmb, topOfMemory-zx128StubRoom, Notes(note), nil)
	require.NoError(t, err)

	require.NotEmpty(t, dropped, "bank 1 holds one of them at most")
	require.Len(t, said, 1)
	require.Contains(t, said[0], "did not fit")
}

// The index names pictures 0 to 254, so one numbered 255 would be laid out in a
// bank and then be unreachable. It is dropped where it can still be explained.
func TestAPictureTheIndexCannotNameIsDropped(t *testing.T) {
	var said []string

	note := func(_ Notice, text string) { said = append(said, text) }

	_, _, dropped, err := zx128Pages(
		map[int][]byte{255: blankScreen()}, nil, 0x8400, Notes(note), nil)
	require.NoError(t, err)

	require.Empty(t, dropped, "it never got as far as being laid out")
	require.Len(t, said, 1)
	require.Contains(t, said[0], "cannot be reached")
}

// A picture of a 128K build is a picture of a +3 build. SCRMAKER writes both
// with one procedure — five bytes of geometry and the cropped screen as one ZX0
// stream, SCRMAKER.PAS:1083-1093 — so there is one format and one function for
// it, and this is what says so.
func TestAPictureIsThePlus3Picture(t *testing.T) {
	screen := noisyScreen(1)

	want, err := plus3Picture(screen)
	require.NoError(t, err)

	built, numbers, err := zx128Pictures(map[int][]byte{3: screen}, nil, nil)
	require.NoError(t, err)

	require.Equal(t, []int{3}, numbers)
	require.Equal(t, want, built[3])
}

// Two copies of one drawing are twice the room on the machine that has least of
// it. It is said and not fixed: which of the two the adventure should stop
// naming is the author's call.
func TestTheSameDrawingTwiceIsSaid(t *testing.T) {
	var said []string

	note := func(_ Notice, text string) { said = append(said, text) }

	_, _, _, err := zx128Pages(
		map[int][]byte{1: blankScreen(), 2: blankScreen()}, nil, 0x8400, Notes(note), nil)
	require.NoError(t, err)

	require.Len(t, said, 1)
	require.Contains(t, said[0], "the same drawing")
}

// A database that reaches the stub has nowhere left to go, and on this machine
// there is a cure worth naming: XMESSAGE moves a message's text into the banks.
func TestADatabaseThatReachesTheStubIsRefusedWithTheCure(t *testing.T) {
	_, err := ZX128(make([]byte, 0xA000), Options{Language: language.LanguageSpanish})

	require.Error(t, err)
	require.Contains(t, err.Error(), "XMESSAGE")
}

// The real thing, and the only test here that can be wrong about the format.
//
// work/ is not in the checkout, so this skips where the tape is not there —
// which is the same bargain the compare-* harnesses make and for the same
// reason.
//
// The tape is The Secret of Barron Wood as DAAD READY built it: a loading
// screen, the interpreter, a 14447-byte database, the index and stub, and two
// banks of xmessages with no pictures at all. Everything but the BASIC loader is
// held against it byte for byte; the loader is not, because with a loading
// screen this repository writes its own — it blacks the screen first and pokes
// the print channel through CURCHL rather than at a fixed 23739. See
// media.BlackScreen and media.HideLoadingMessages.
func TestTheRealTapeAgrees(t *testing.T) {
	const real = "../../work/aventuras/The_Secret_Of_Barron_Wood/Barron_ZX128K.tap"

	raw, err := os.ReadFile(real)
	if err != nil {
		t.Skipf("%s is not in the checkout: nothing to compare against", real)
	}

	theirs := split(t, raw)
	require.Len(t, theirs, 12)

	body := func(block []byte) []byte { return block[1 : len(block)-1] }

	ours, err := ZX128(body(theirs[7]), Options{
		Language:  language.LanguageEnglish,
		Name:      "TEST",
		Loading:   body(theirs[3]),
		XMessages: append(append([]byte{}, body(theirs[10])...), body(theirs[11])...),
	})
	require.NoError(t, err)

	mine := split(t, ours)
	require.Len(t, mine, len(theirs), "the same blocks in the same order")

	// Block 0 is the program's header and carries the loader's length, and
	// block 1 is the loader. Neither is comparable while the loader is ours.
	for i := 2; i < len(theirs); i++ {
		require.Equal(t, theirs[i], mine[i], "block %d", i)
	}

	// Said plainly because it is the point of the test: the index and the stub
	// are block 9 and the banks are 10 and 11, so a pass means the paging, the
	// index and its reversal are the reference's.
	require.Equal(t, 2097, len(body(theirs[9])))
	require.Equal(t, []byte{0xFF, 0xFF, 0x04, 0x03, 0x00, 0xC0, 0x00, 0x00},
		body(mine[9])[:8])
}

// blankScreen is a screen with nothing on it, which is the picture that
// compresses best and so the one that fits anywhere.
func blankScreen() []byte { return make([]byte, screenSize) }

// noisyScreen is a screen that compresses badly, for the tests that need a
// picture too big to go anywhere. The seed makes two of them that differ, since
// two identical ones would be reported as one drawing used twice.
func noisyScreen(seed uint32) []byte {
	out := make([]byte, screenSize)

	// Not random: a test that is a different size on each run is a test that
	// fails on somebody else's machine. This is a cheap generator with no
	// pattern short enough for ZX0 to exploit.
	state := 12345 + seed

	for i := range out {
		state = state*1664525 + 1013904223
		out[i] = byte(state >> 24)
	}

	return out
}
