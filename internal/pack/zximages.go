package pack

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/zx0"
)

// The pictures of a Spectrum build, which are not carried as pictures.
//
// A screen is 6912 bytes and an adventure has dozens of them, so what goes on
// the tape is the part of each one that shows — the top rows, above where the
// text window starts — cut into the runs that are contiguous in video memory,
// each run compressed on its own. The interpreter loads a run straight to the
// address it came from, which is why the address travels with it and why the
// cutting follows the hardware's layout rather than the picture's.
//
// This is ZXsplitter.php and pager48k.php, in one place because they are one
// job: the first decides where the cuts fall and the second writes the table
// that says where everything landed.
const (
	// screenBase is where video memory starts, and every offset in the table
	// is an address there rather than an index into the file.
	screenBase = 0x4000
	// attributeBase is where the colours begin, 6144 bytes in.
	attributeBase = screenBase + 6144
	// screenSize is a whole screen: pixels and attributes.
	screenSize = 6912
	// bytesWide is a row of character cells.
	bytesWide = 32
	// imageLines is how much of a screen a picture occupies. It is DAAD's
	// IMGLINES, whose default is 96 and which every build in the kit leaves
	// alone; below it the adventure prints.
	imageLines = 96
)

// imageBlock is one run of a screen: where it goes and what goes there.
type imageBlock struct {
	at   int
	body []byte
}

// screenBlocks works out which runs of video memory the top lines of a screen
// occupy, merged where they touch.
//
// They do not form one run, and that is the whole difficulty: the display is
// three banks of 64 lines, and inside a bank the pixel row of a character
// comes before the character row, so the first 96 lines are 32 bytes here and
// 32 bytes 256 further on, over and over. Merging what touches turns a few
// hundred scraps into a handful of runs, which is what makes compressing them
// worth anything.
func screenBlocks(lines int) ([]imageBlock, error) {
	if lines < 8 || lines > 192 || lines%8 != 0 {
		return nil, fmt.Errorf("a picture is 8 to 192 lines and a multiple of 8, not %d", lines)
	}

	type span struct{ at, length int }

	spans := make([]span, 0, lines+1)

	for y := range lines {
		bank, pixelRow, charRow := y>>6, y&7, (y>>3)&7
		spans = append(spans, span{
			at:     screenBase + bank*2048 + pixelRow*256 + charRow*bytesWide,
			length: bytesWide,
		})
	}

	// The attributes are the easy half: one run, as many character rows as the
	// picture is tall.
	spans = append(spans, span{at: attributeBase, length: lines / 8 * bytesWide})

	sort.Slice(spans, func(i, j int) bool { return spans[i].at < spans[j].at })

	out := make([]imageBlock, 0, 16)
	start, end := spans[0].at, spans[0].at+spans[0].length

	for _, s := range spans[1:] {
		if s.at <= end {
			end = max(end, s.at+s.length)

			continue
		}

		out = append(out, imageBlock{at: start, body: make([]byte, end-start)})
		start, end = s.at, s.at+s.length
	}

	return append(out, imageBlock{at: start, body: make([]byte, end-start)}), nil
}

// cutScreen takes the runs out of a screen.
func cutScreen(screen []byte, lines int) ([]imageBlock, error) {
	if len(screen) != screenSize {
		return nil, fmt.Errorf("a Spectrum screen is %d bytes, not %d", screenSize, len(screen))
	}

	blocks, err := screenBlocks(lines)
	if err != nil {
		return nil, err
	}

	for i := range blocks {
		from := blocks[i].at - screenBase
		copy(blocks[i].body, screen[from:from+len(blocks[i].body)])
	}

	return blocks, nil
}

// imagePage is page zero: every picture's blocks, compressed, one after
// another, and where each picture's run of them begins.
type imagePage struct {
	body []byte
	// at is where in memory the page is loaded, and starts is where each
	// picture begins, by picture number, as an address inside it.
	at     int
	starts map[int]int
}

// buildImagePage lays the pictures out at the address given.
//
// A picture is written as its blocks — each one a destination address, a
// compressed length, and the bytes — and a zero word to say there are no
// more. The order is by picture number, which is not required by anything but
// makes two builds of the same adventure the same file.
//
// A picture that does not fit is left out rather than cut short, and the
// caller is told which, because a 48K Spectrum running out of room for the
// last few pictures is an ordinary outcome and not a failure of the build.
func buildImagePage(images map[int][]byte, lines, at, room int) (imagePage, []int, error) {
	page := imagePage{at: at, starts: map[int]int{}}

	numbers := make([]int, 0, len(images))
	for n := range images {
		numbers = append(numbers, n)
	}

	sort.Ints(numbers)

	var dropped []int

	for _, n := range numbers {
		blocks, err := cutScreen(images[n], lines)
		if err != nil {
			return imagePage{}, nil, fmt.Errorf("picture %d: %w", n, err)
		}

		var one []byte

		for _, b := range blocks {
			packed := zx0.Compress(b.body)

			one = append(one, byte(b.at), byte(b.at>>8))
			one = append(one, byte(len(packed)), byte(len(packed)>>8))
			one = append(one, packed...)
		}

		// The end of the blocks, which the interpreter reads as a destination
		// of zero.
		one = append(one, 0, 0)

		// Three bytes for the entry this picture will take in the index, on
		// top of the data itself.
		const indexEntry = 3

		if len(page.body)+len(one)+indexEntry > room {
			dropped = append(dropped, n)

			continue
		}

		page.starts[n] = at + len(page.body)
		page.body = append(page.body, one...)
	}

	return page, dropped, nil
}

// numberList writes a list of picture numbers the way a person reads one,
// with the long middle of it left out.
func numberList(numbers []int) string {
	const show = 8

	out := make([]string, 0, show+1)
	for _, n := range numbers[:min(show, len(numbers))] {
		out = append(out, strconv.Itoa(n))
	}

	if len(numbers) > show {
		out = append(out, fmt.Sprintf("and %d more", len(numbers)-show))
	}

	return strings.Join(out, ", ")
}
