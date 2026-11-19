package pack

import (
	"fmt"
	"sort"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

// cpcBankOptions is an adventure whose pictures are noise, so that they
// compress like real ones rather than down to nothing.
func cpcBankOptions(pictures int) Options {
	images := map[int][]byte{}
	palettes := map[int][]byte{}

	seed := 1

	for n := range pictures {
		screen := make([]byte, cpcPackScreen)
		for band := range cpcPackBands {
			for i := range 12 * cpcPackBytesWide {
				seed = (seed*1103515245 + 12345) & 0x7FFFFFFF
				screen[band*cpcPackBandSize+i] = byte(seed >> 16)
			}
		}

		images[n] = screen
		palettes[n] = []byte{0, 11, 25, 9}
	}

	return Options{
		Language: language.LanguageSpanish, Name: "AO",
		Images: images, Palettes: palettes, Lines: 96,
		Note: func(Notice, string) {},
	}
}

// Which picture it is on, said as it goes. Compressing one is about half a
// second, so a bank of forty is half a minute of a tool that would otherwise
// look hung — and what makes the wait readable is the counting, not the phrase.
//
// The counting is all this asks for. The pictures are compressed several at a
// time, so which of them is named on which line is whichever finished when, and
// a test that pinned that down would be testing the scheduler. What has to hold
// is that the count goes up by one to the total and that every picture is said
// once: a number that repeats or a picture that goes by unmentioned is the bug
// this is here for.
func TestTheBankSaysWhichPictureItIsOn(t *testing.T) {
	opts := cpcBankOptions(3)
	for n := range opts.Images {
		opts.Images[n] = cpcScreenOf(12)
	}

	var (
		counts []string
		said   []string
	)

	opts.Progress = func(done, total int, what string) {
		counts = append(counts, fmt.Sprintf("%d/%d", done, total))
		said = append(said, what)
	}

	_, err := cpcPictureBank(opts, 0x6000, false)
	require.NoError(t, err)

	require.Equal(t, []string{"1/3", "2/3", "3/3"}, counts)

	sort.Strings(said)
	require.Equal(t, []string{
		"compressing picture 000",
		"compressing picture 001",
		"compressing picture 002",
	}, said)
}

// What goes into memory: the code and the index under the loader, and the
// pictures themselves in a block of their own.
func TestWhatIsInACPCPictureBank(t *testing.T) {
	// Flat pictures, which compress to nothing much, so that all three fit and
	// the shape of the index is what is being read.
	opts := cpcBankOptions(3)
	for n := range opts.Images {
		opts.Images[n] = cpcScreenOf(12)
	}

	bank, err := cpcPictureBank(opts, 0x6000, false)
	require.NoError(t, err)

	require.Equal(t, cpcResidentAt(), bank.at, "under the loader, out of the window")
	require.Equal(t, []int{0, 1, 2}, bank.carried)
	require.Empty(t, bank.left)

	require.Len(t, bank.blocks, 1, "a 64K machine has one place to put them")
	require.Equal(t, cpcMainMemory, bank.blocks[0].config)
	require.Equal(t, 0x6000, bank.blocks[0].at, "the page after the game")

	// Walking the index the way the Z80 does lands on the terminator, and every
	// entry it names is where the block says the picture is.
	index := bank.resident[bank.entriesAt:]

	walk, n := 0, 0

	for ; index[walk] != cpcIndexEnd; walk, n = walk+cpcIndexRecord, n+1 {
		require.Equal(t, byte(n), index[walk], "entries are in number order")
		require.Equal(t, byte(cpcMainMemory), index[walk+1])

		at := int(index[walk+2]) | int(index[walk+3])<<8
		into := at - bank.blocks[0].at

		require.Equal(t, byte(n), bank.blocks[0].body[into], "and point at their picture")
		require.Equal(t, []byte{0, 11, 25, 9}, bank.blocks[0].body[into+3:into+7])
	}

	require.Equal(t, 3, n, "three pictures and then the end")
	require.Len(t, index, walk+1, "and nothing after the end")
}

// On a 128K machine the pictures go in the spare banks, which is room enough
// for an adventure's worth of them.
func TestACPCPictureBankUsesTheSpareBanks(t *testing.T) {
	// Noise, which does not compress, so these are several times the size of a
	// real picture: whatever fits here fits in life with room over.
	opts := cpcBankOptions(8)
	opts.Memory = 128

	// A game that reaches 0x9000 leaves nothing above it, and it no longer
	// matters: none of this is in the game's own memory.
	bank, err := cpcPictureBank(opts, 0x9000, false)
	require.NoError(t, err)

	require.Len(t, bank.carried, 8)
	require.Empty(t, bank.left)
	require.Len(t, bank.blocks, cpcBanks, "and they fill the banks in turn")

	// Each block is one bank, filled before the next is started, and none of
	// them reaches past the window.
	for i, block := range bank.blocks {
		require.Equal(t, cpcFirstBank+i, block.config)
		require.Equal(t, cpcBankWindow, block.at)
		require.LessOrEqual(t, len(block.body), cpcBankSize)
	}

	// And they travel as files the loader reads with that bank in the window.
	files := bank.files()
	require.Equal(t, "AO.PIC", files[0].Name)
	require.Equal(t, cpcResidentAt(), files[0].at)
	require.Zero(t, files[0].config, "the code is in plain memory")

	for i, f := range files[1:] {
		require.Equal(t, fmt.Sprintf("AO.PI%d", i), f.Name)
		require.Equal(t, cpcFirstBank+i, f.config)
	}
}

// A 64K bank only holds what fits between the end of the game and the code
// under the loader. The rest are named rather than dropped quietly.
func TestACPCBankSaysWhatWouldNotFit(t *testing.T) {
	bank, err := cpcPictureBank(cpcBankOptions(12), 0x6000, false)
	require.NoError(t, err)

	require.NotEmpty(t, bank.carried)
	require.NotEmpty(t, bank.left)
	require.Len(t, append(bank.carried, bank.left...), 12)

	// And it stops below the code, which is below the graphics block the
	// interpreter moves to the top of memory and is not ours to write on.
	block := bank.blocks[0]
	require.LessOrEqual(t, block.at+len(block.body), bank.at)
	require.LessOrEqual(t, bank.at+len(bank.resident), cpcGraphicsTop)
}

// The two calls that open a picture file become calls to the bank, and the
// file reader is jumped over. A binary that does not hold the expected bytes
// is handed back untouched.
func TestPointingTheInterpreterAtTheBank(t *testing.T) {
	opts := cpcBankOptions(2)

	binary, err := cpcBinary(make([]byte, 5000), opts)
	require.NoError(t, err)

	bank, err := cpcPictureBank(opts, 0x6000, false)
	require.NoError(t, err)

	patched, err := cpcPointAtTheBank(binary, bank)
	require.NoError(t, err)

	at := func(address int) []byte {
		from := address - cpcInterpreterOrigin + media.AMSDOSHeaderSize

		return patched[from : from+3]
	}

	probe := bank.at
	draw := bank.at + bank.drawAt

	require.Equal(t, []byte{0xCD, byte(probe), byte(probe >> 8)}, at(cpcProbeAt))
	require.Equal(t, []byte{0xCD, byte(draw), byte(draw >> 8)}, at(cpcDrawAt))
	require.Equal(t, []byte{0xC3, byte(cpcDrawTail & 0xFF), byte(cpcDrawTail >> 8)}, at(cpcDrawAt+3))

	// Somebody else's bytes are left alone.
	odd := make([]byte, len(binary))
	copy(odd, binary)
	odd[cpcProbeAt-cpcInterpreterOrigin+media.AMSDOSHeaderSize] = 0x00

	same, err := cpcPointAtTheBank(odd, bank)
	require.NoError(t, err)
	require.Equal(t, odd, same)
}

// The decompressor is Einar Saukas' own, and his source says how long it is.
func TestTheBankCarriesSaukasDecoder(t *testing.T) {
	a := &cpcAssembly{}
	a.here("dzx0")
	cpcDZX0(a)

	_, err := a.resolve(0x6000)
	require.NoError(t, err)
	require.Len(t, a.code, 68, "the standard version is 68 bytes only")
}
