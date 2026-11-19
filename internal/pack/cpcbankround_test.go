package pack

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/zx0"
	"github.com/stretchr/testify/require"
)

// Every picture comes back out of the bank exactly as it went in.
//
// The bank is the one piece of the Amstrad build nobody can read by looking at
// it: the pictures go in compressed and the machine is what unpacks them. So
// this does what the Z80 does — walks the index, finds each entry, and runs its
// eight streams back through ZX0, one per band of video memory — and checks
// what comes out against the screen on disk.
//
// Band by band and not the whole screen, because that is the whole of the
// point: a stream that covered all 16384 bytes would carry the rows below the
// picture as blanks and write them, and what is below the picture is the text.
// What this asks is that nothing past the picture's rows is ever named.
//
// It is also what says a frame 176 rows tall arrives with its bottom edge on.
func TestEveryPictureComesBackOutOfTheBank(t *testing.T) {
	const from = "../../examples/AO/Amstrad/CPC/part1/gfx/scr"

	names, err := filepath.Glob(filepath.Join(from, "*.scr"))
	require.NoError(t, err)
	require.NotEmpty(t, names, "the example's pictures are not where they were")

	opts := Options{
		Language: language.LanguageSpanish, Name: "AO1",
		Images: map[int][]byte{}, Palettes: map[int][]byte{},
		Memory: 128,
		Note:   func(Notice, string) {},
	}

	for _, name := range names {
		number, err := strconv.Atoi(filepath.Base(name)[:3])
		require.NoError(t, err)

		screen, err := os.ReadFile(name)
		require.NoError(t, err)

		palette, err := os.ReadFile(name[:len(name)-4] + ".pal")
		require.NoError(t, err)

		opts.Images[number], opts.Palettes[number] = screen, palette
	}

	bank, err := cpcPictureBank(opts, 0x9000, false)
	require.NoError(t, err)
	require.Len(t, bank.carried, len(opts.Images), "every picture travels")
	require.Empty(t, bank.left)

	// Where each block lands, so that an index entry can be followed to the
	// bytes it names the way the Z80 follows it.
	blocks := map[int]cpcBlock{}
	for _, block := range bank.blocks {
		blocks[block.config] = block
	}

	index := bank.resident[bank.entriesAt:]

	seen := []int{}

	for at := 0; index[at] != cpcIndexEnd; at += cpcIndexRecord {
		number := int(index[at])
		config := int(index[at+1])
		where := int(index[at+2]) | int(index[at+3])<<8

		block, ok := blocks[config]
		require.True(t, ok, "picture %03d names a block that is not there", number)

		entry := block.body[where-block.at:]
		require.Equal(t, byte(number), entry[0], "the entry says which picture it is")

		length := int(entry[1]) | int(entry[2])<<8
		require.Equal(t, opts.Palettes[number][:cpcPaletteSize], entry[3:cpcBankHeader],
			"picture %03d carries its own inks", number)

		// The display as the Z80 leaves it, started as something no picture can
		// write so that what a picture does not touch is visible afterwards.
		const untouched = 0xAA

		display := make([]byte, cpcPackScreen)
		for i := range display {
			display[i] = untouched
		}

		rows := cpcPictureRows(opts.Images[number], opts.Lines)
		at := cpcBankHeader

		for band := range cpcPackBands {
			size := int(entry[at]) | int(entry[at+1])<<8
			at += 2

			strip, err := zx0.Decompress(entry[at : at+size])
			require.NoError(t, err, "picture %03d, band %d does not unpack", number, band)
			require.Len(t, strip, rows*cpcPackBytesWide,
				"picture %03d, band %d is not the rows it draws on", number, band)

			copy(display[band*cpcPackBandSize:], strip)
			at += size
		}

		require.Equal(t, length, at, "picture %03d says a length its streams do not come to", number)

		for band := range cpcPackBands {
			from := band*cpcPackBandSize + rows*cpcPackBytesWide

			for i := from; i < (band+1)*cpcPackBandSize; i++ {
				require.Equal(t, byte(untouched), display[i],
					"picture %03d writes over the text area at %d", number, i)
			}

			want := opts.Images[number][band*cpcPackBandSize : from]
			require.Equal(t, want, display[band*cpcPackBandSize:from],
				"picture %03d comes out changed in band %d", number, band)
		}

		seen = append(seen, number)
	}

	want := make([]int, 0, len(opts.Images))
	for number := range opts.Images {
		want = append(want, number)
	}

	sort.Ints(want)
	require.Equal(t, want, seen, "the index names every picture, in order")

	// And the one this was written for: the frame the title is printed inside
	// is 22 character rows tall, where the other drawings are 14, so any crop
	// at all takes its bottom edge off.
	require.Contains(t, seen, 44)
	require.Equal(t, 22, cpcPictureRows(opts.Images[44], opts.Lines),
		"the title's frame is carried whole")
}
