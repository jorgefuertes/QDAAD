package pack

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// pcwDecode is the reading of the format this file writes, and it is here
// rather than in the package because nothing in a release ever reads one back.
//
// It is also the test's whole point. The encoder was written from 186
// illustrations on the 1989 disks; what says it is right is that the same
// rules read back what they wrote, and that what they read is the bitmap that
// went in.
func pcwDecode(body []byte, want int) []byte {
	if len(body) == want {
		return body // stored raw, which the length is how you tell
	}

	marks := [256]bool{}
	for _, m := range body[1 : 1+int(body[0])] {
		marks[m] = true
	}

	out := make([]byte, 0, want)

	for i := 1 + pcwMaxMarkers; i < len(body); {
		b := body[i]
		i++

		if !marks[b] || i >= len(body) {
			out = append(out, b)

			continue
		}

		run := int(body[i])
		if run == 0 {
			run = 256
		}

		i++

		for range run {
			out = append(out, b)
		}
	}

	return out
}

// unweave puts the rows back the way round they were.
func unweave(stream []byte, stride, height int) []byte {
	out := make([]byte, 0, len(stream))

	for y := range height {
		row := stream[y*stride : (y+1)*stride]
		if y%2 == 0 {
			out = append(out, row...)

			continue
		}

		for i := len(row) - 1; i >= 0; i-- {
			out = append(out, row[i])
		}
	}

	return out
}

// What goes in comes out, for bitmaps of every shape that matters.
//
// Flat ones are what the markers are for and ought to come out tiny; noise has
// nothing to repeat and is stored raw; and the middle one is what a dithered
// drawing actually looks like, which is where the format earns its keep.
func TestAPCWPictureSurvivesBeingEncoded(t *testing.T) {
	const (
		width  = 512
		height = 112
		stride = width / 8
		bytes  = stride * height
	)

	dice := rand.New(rand.NewSource(1989))

	for _, c := range []struct {
		name string
		fill func(i int) byte
	}{
		{"todo blanco", func(int) byte { return 0xFF }},
		{"todo negro", func(int) byte { return 0x00 }},
		{"tramado", func(i int) byte { return []byte{0xCC, 0x33}[(i/64)%2] }},
		{"ruido", func(int) byte { return byte(dice.Intn(256)) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			want := make([]byte, bytes)
			for i := range want {
				want[i] = c.fill(i)
			}

			body, err := pcwPicture(want, width, height)
			require.NoError(t, err)

			got := unweave(pcwDecode(body, bytes), stride, height)
			require.Equal(t, want, got, "the picture came back changed")
		})
	}
}

// A flat picture is nearly free and noise is never bigger than the bitmap.
//
// The second is the one worth asserting: a run-length scheme on noise makes
// things worse, and a packager that quietly wrote 8 KiB where 6.5 would do
// would cost four illustrations on a disk that has room for thirty.
func TestAPCWPictureIsNeverBiggerThanItsBitmap(t *testing.T) {
	const bytes = 512 / 8 * 112

	flat := make([]byte, bytes)

	body, err := pcwPicture(flat, 512, 112)
	require.NoError(t, err)
	require.Less(t, len(body), 100, "a picture of one colour should be a handful of bytes")

	dice := rand.New(rand.NewSource(1989))
	noise := make([]byte, bytes)

	for i := range noise {
		noise[i] = byte(dice.Intn(256))
	}

	body, err = pcwPicture(noise, 512, 112)
	require.NoError(t, err)
	require.LessOrEqual(t, len(body), bytes)
}

// A picture says how big it is, and what it says has to be true.
//
// Worth its own test because the alternative is silent: a file one row short
// of what its header claims is an archive the interpreter reads off the end
// of, and the picture before it on screen is whatever was in memory.
func TestAPCWPictureIsMeasuredBeforeItIsBelieved(t *testing.T) {
	for _, c := range []struct {
		name string
		file []byte
		says string
	}{
		{"nothing at all", nil, "starts with its width"},
		{
			"a width that is not a multiple of eight",
			append([]byte{100, 0, 8, 0}, make([]byte, 100)...), "multiple of 8",
		},
		{
			"a width wider than the screen",
			append([]byte{0x00, 0x04, 8, 0}, make([]byte, 1024)...), "multiple of 8 up to 720",
		},
		{
			"a height taller than the screen",
			append([]byte{8, 0, 0x01, 0x02}, make([]byte, 513)...), "up to 256",
		},
		{
			"a bitmap that is not the size it claims",
			append([]byte{512 & 0xFF, 512 >> 8, 112, 0}, make([]byte, 10)...), "7168 bytes",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, _, _, err := pcwMeasure(c.file)
			require.ErrorContains(t, err, c.says)
		})
	}
}

// The archive is the table the interpreter indexes by picture number.
func TestAPCWArchiveIndexesByPictureNumber(t *testing.T) {
	flat := append([]byte{512 & 0xFF, 512 >> 8, 112, 0}, make([]byte, 512/8*112)...)

	archive, left, err := pcwArchive(map[int][]byte{3: flat, 27: flat}, nil, 0)
	require.NoError(t, err)
	require.Empty(t, left, "nothing was left out of a disk with no limit")

	require.Equal(t, byte(2), archive[4], "the header counts the pictures")

	at := func(n int) int {
		s := archive[pcwArchiveHeader+n*pcwSlotSize:]
		return int(s[0]) | int(s[1])<<8 | int(s[2])<<16 | int(s[3])<<24
	}

	require.Equal(t, pcwFirstPicture, at(3), "the first picture goes right behind the table")
	require.NotZero(t, at(27))
	require.Zero(t, at(4), "a number with no picture has no offset")
}
