package pack

// The Amstrad PCW's illustrations, which are bitmaps and not vectors.
//
// Every other eight-bit edition in the corpus draws: the database carries a
// list of lines and fills and the interpreter walks it. The PCW does not. Its
// disks carry a .DAT of monochrome bitmaps, 111 KiB of them against the 30 the
// Spectrum spends on the same adventure, and the interpreter blits.
//
// The format was read off the 1989 disks rather than out of the interpreter,
// and then checked against the interpreter: a disk was built with Cozumel's
// own PARTE001.DAT and a PICTURE at it, run on a PCW, and the screen taken out
// of the emulator's RAM. 6656 bytes of bitmap, 6656 matched. See 9.9 of
// docs/DAAD_v2_v3_Research/09-graficos.md.
//
// An archive is a six-byte header, 256 slots of ten bytes indexed by the
// number a PICTURE names, and then the pictures:
//
//	00 00 04 00 nn nn   the last word is how many pictures there are
//	....                256 slots: four bytes of offset, six of flags
//	....                the first picture, at 6 + 2560 = 2566
//
// A picture is six bytes of its own — width with bit 15 set, height, and how
// long its body is — and then the body.
//
// The body is where the work is. Five bytes of preamble: how many markers
// there are, and four slots to put them in. A marker is a byte that is never
// written on its own; what follows it is how many times it repeats, and zero
// means 256. Everything else is a literal. The markers are chosen per picture,
// which is what lets this beat a fixed scheme without carrying a table.
//
// And the rows go back and forth — row 0 left to right, row 1 right to left,
// and on down. Nothing in the file says so; it is what the bytes do.

import (
	"errors"
	"fmt"
	"sort"

	"github.com/jorgefuertes/QDAAD/internal/ddbfile"
)

const (
	// The screen a picture has to fit inside, and the four bytes each one
	// carries to say how much of it it takes: width and height, two
	// little-endian words, and then the bitmap.
	//
	// The measurements travel with the picture because they are not all one
	// shape. In La Aventura Original a room is 512 by 112 and the title is 512
	// by 176, and a title forced to the height of a room is a frame with its
	// bottom cut off.
	pcwScreenWidth  = 720
	pcwScreenHeight = 256
	pcwPictureSize  = 4

	// pcwArchiveHeader and pcwSlots are the table in front of the pictures.
	pcwArchiveHeader = 6
	pcwSlots         = 256
	pcwSlotSize      = 10
	pcwFirstPicture  = pcwArchiveHeader + pcwSlots*pcwSlotSize

	// pcwPictureEntry is the six bytes in front of every picture: width,
	// height and how long the body is.
	pcwPictureEntry = 6
	// pcwMaxMarkers is how many bytes a picture may nominate as repeat
	// markers, and the preamble has a slot for each whether it is used or not.
	pcwMaxMarkers = 4
	// pcwWidthFlag is set in the width word of every picture on the 1989
	// disks. What it means is not known; what is known is that they all have
	// it, so ours do too.
	pcwWidthFlag = 0x8000
)

// pcwMeasure reads the four bytes in front of a picture and checks that what
// follows them is the bitmap they describe.
func pcwMeasure(file []byte) (bitmap []byte, width, height int, err error) {
	if len(file) < pcwPictureSize {
		return nil, 0, 0, errors.New("a PCW picture starts with its width and height")
	}

	width = int(file[0]) | int(file[1])<<8
	height = int(file[2]) | int(file[3])<<8
	bitmap = file[pcwPictureSize:]

	switch {
	case width == 0 || width%8 != 0 || width > pcwScreenWidth:
		return nil, 0, 0, fmt.Errorf(
			"%d is not a width a PCW picture can have: a multiple of 8 up to %d",
			width, pcwScreenWidth)
	case height == 0 || height > pcwScreenHeight:
		return nil, 0, 0, fmt.Errorf(
			"%d is not a height a PCW picture can have: up to %d", height, pcwScreenHeight)
	case len(bitmap) != width/8*height:
		return nil, 0, 0, fmt.Errorf("a %dx%d picture is %d bytes and this is %d",
			width, height, width/8*height, len(bitmap))
	}

	return bitmap, width, height, nil
}

// pcwPicture encodes one bitmap as a picture body.
//
// The bitmap is rows of width/8 bytes, the high bit of each byte the leftmost
// pixel and a set bit white, which is the way the screen is read out of the
// machine and the way the 1989 pictures are stored.
func pcwPicture(bitmap []byte, width, height int) ([]byte, error) {
	stream := pcwBoustrophedon(bitmap, width/8, height)
	markers := pcwMarkers(stream)

	body := make([]byte, 0, len(stream))
	body = append(body, byte(len(markers)))
	body = append(body, markers...)
	body = append(body, make([]byte, pcwMaxMarkers-len(markers))...)

	isMarker := [256]bool{}
	for _, m := range markers {
		isMarker[m] = true
	}

	for i := 0; i < len(stream); {
		b := stream[i]
		if !isMarker[b] {
			body = append(body, b)
			i++

			continue
		}

		run := 1
		for run < 256 && i+run < len(stream) && stream[i+run] == b {
			run++
		}

		// A run of 256 is written as zero, which is the only reading that
		// makes the 1989 files decode to the length they declare.
		body = append(body, b, byte(run))
		i += run
	}

	// Compressing something incompressible is worse than not: a picture whose
	// declared length is exactly its bitmap is stored raw, and the 1989 disks
	// have five of those.
	if len(body) >= len(stream) {
		return stream, nil
	}

	return body, nil
}

// pcwBoustrophedon turns a bitmap into the order the file stores it in, which
// is every other row reversed.
func pcwBoustrophedon(bitmap []byte, stride, height int) []byte {
	out := make([]byte, 0, len(bitmap))

	for y := range height {
		row := bitmap[y*stride : (y+1)*stride]
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

// pcwMarkers picks the bytes worth making markers of.
//
// Naming a byte a marker is not free: every one of its runs becomes two bytes,
// so a byte that mostly appears alone costs one byte each time it does. What
// is worth having is the four with the most to gain, counted run by run.
func pcwMarkers(stream []byte) []byte {
	gain := [256]int{}

	for i := 0; i < len(stream); {
		run := 1
		for i+run < len(stream) && stream[i+run] == stream[i] {
			run++
		}

		// A run of more than 256 is written as several, each costing two.
		for left := run; left > 0; left -= min(left, 256) {
			gain[stream[i]] += min(left, 256) - 2
		}

		i += run
	}

	out := make([]byte, 0, pcwMaxMarkers)

	for range pcwMaxMarkers {
		best, by := -1, 0
		for b := range 256 {
			if gain[b] > by {
				best, by = b, gain[b]
			}
		}

		if best < 0 {
			break
		}

		out = append(out, byte(best))
		gain[best] = 0
	}

	return out
}

// pcwArchive builds the .DAT: the table and as many pictures behind it as room
// allows. What was left out comes back, by number.
//
// Pictures are written in the order of their numbers rather than the order
// they were handed over, so that the same build twice is the same file twice —
// and so that what a full disk drops is the end of the adventure rather than
// whichever one the map happened to yield last.
func pcwArchive(pictures map[int][]byte, named map[int]bool, room int) (
	archive []byte, left []int, err error,
) {
	out := make([]byte, pcwFirstPicture)
	out[2] = 4 // what the 1989 archives carry here

	count := 0

	// The ones the adventure names go on first and in their own pass, because
	// what is at stake for them is not a drawing: see pcwNamed. Only after
	// that does the rest of the disk get filled in number order.
	for _, pass := range []bool{true, false} {
		for number := range pcwSlots {
			file, ok := pictures[number]
			if !ok || named[number] != pass {
				continue
			}

			bitmap, width, height, err := pcwMeasure(file)
			if err != nil {
				return nil, nil, fmt.Errorf("picture %d: %w", number, err)
			}

			body, err := pcwPicture(bitmap, width, height)
			if err != nil {
				return nil, nil, fmt.Errorf("picture %d: %w", number, err)
			}

			if room > 0 && len(out)-pcwFirstPicture+pcwPictureEntry+len(body) > room {
				left = append(left, number)

				continue
			}

			at := len(out)

			slot := out[pcwArchiveHeader+number*pcwSlotSize:]
			slot[0] = byte(at)
			slot[1] = byte(at >> 8)
			slot[2] = byte(at >> 16)
			slot[3] = byte(at >> 24)
			slot[4] = 1 // as the full-size pictures of 1989 have it

			flagged := width | pcwWidthFlag

			out = append(out,
				byte(flagged), byte(flagged>>8),
				byte(height), byte(height>>8),
				byte(len(body)), byte(len(body)>>8))
			out = append(out, body...)

			count++
		}
	}

	sort.Ints(left)

	if count == 0 {
		return nil, nil, errors.New("there are no pictures to put in a PCW archive")
	}

	out[4] = byte(count)
	out[5] = byte(count >> 8)

	return out, left, nil
}

// pcwPictureFile is the .DAT a release carries, and the line to say about it.
//
// An adventure with no drawings still gets one: PICTURE is a condition in
// version 3 and the interpreter opens the file to answer it, so a release
// without the file is a release where every PICTURE fails and takes the rest
// of its entry with it. The empty archive beside the interpreter is what DAAD
// READY's own PCW.DSK carries for exactly that.
func pcwPictureFile(beside map[string][]byte, database []byte, opts Options, room int) (
	body []byte, note string, err error,
) {
	if len(opts.Images) == 0 {
		empty, ok := beside["BLANK.DAT"]
		if !ok {
			return nil, "", errors.New("the PCW's empty archive of illustrations is missing")
		}

		return empty, "no illustrations: an empty archive, which the interpreter still opens", nil
	}

	opts.Progress.at(0, len(opts.Images), "illustrations")

	archive, left, err := pcwArchive(opts.Images, pcwNamed(database), room)
	if err != nil {
		return nil, "", err
	}

	opts.Progress.at(len(opts.Images), len(opts.Images), "illustrations")

	// What did not fit is said out loud. A release that quietly carried
	// nineteen drawings of fifty-six looks exactly like one that carried them
	// all, until somebody walks into the twentieth room.
	if len(left) > 0 {
		opts.Note.say(NoticeLeftOut,
			"%d of the %d illustrations did not fit and are not on the disk: %v. "+
				"None of them is one the adventure asks for by number, so none of them "+
				"breaks it — what is missing is a drawing",
			len(left), len(opts.Images), left)
	}

	return archive, fmt.Sprintf("%d illustrations of %d, %d KiB of bitmaps",
		len(opts.Images)-len(left), len(opts.Images), len(archive)/1024), nil
}

// pcwNamed is the set of pictures the adventure asks for by number.
//
// They are not optional, and a release that drops one is not a release with a
// drawing missing: it is a broken game. PICTURE is a condition in version 3,
// so one that is not there abandons the rest of its entry — and those entries
// do more than draw. In part two of La Aventura Original the entry that puts
// up picture 106 goes on to GOTO 3, which is how the player leaves the opening
// room, so a disk without 106 is a disk that reads the access code and then
// sits there for ever.
//
// A picture reached through a flag — PICTURE @38, the drawing of wherever the
// player is — is the other kind, and that one can go. It is the only condact
// in its entry, the generic stands in for it, and the adventure carries on.
//
// Which is which comes out of the database rather than being declared: the
// opcode is 84 and bit 7 of it says the parameter is a flag number.
func pcwNamed(database []byte) map[int]bool {
	const picture = 84

	out := map[int]bool{}

	db, ok := ddbfile.At(database, 0)
	if !ok {
		return out
	}

	processes, err := db.Processes()
	if err != nil {
		return out
	}

	for _, entries := range processes {
		for _, entry := range entries {
			for _, c := range entry.Condacts {
				if c.Opcode == picture && !c.Indirect && len(c.Params) > 0 {
					out[int(c.Params[0])] = true
				}
			}
		}
	}

	return out
}
