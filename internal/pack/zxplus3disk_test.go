package pack

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/zx0"
	"github.com/stretchr/testify/require"
)

// The +3 release read back out of the disk it was written to.
//
// Everything else about the +3 is checked on the pieces: zxrelease_test.go on
// the four files, zxplus3cache_test.go on the buffer's index, zxplus3images.go
// on the geometry. What none of them opens is the .dsk, and the .dsk is what
// gets sent — so until this, the only thing that had ever asked whether the
// filesystem gives the files back was somebody loading a disk in ZEsarUX and
// playing it.
//
// It goes through media.NewDSK rather than our own structures on purpose. A
// writer agreeing with itself proves nothing; what the interpreter does is open
// a file by name and read it, and the nearest thing to that here is the reader,
// which was written for the disks of 1989 and knows nothing about the writer.

// buildPlus3 packages a +3 release twice: as the files it is made of, and as
// what a reader finds after they have been through a disk.
//
// The two are not compared byte for byte, and both reasons are the
// filesystem's. +3DOS hides the 128-byte header from whatever reads the file,
// which the reader does too — see WithoutPlus3Header — so a file comes back
// without it. And CP/M stores a length in records of 128 bytes, so a file
// comes back as long as its last record rather than as long as it is. Neither
// is a difference the machine sees: the interpreter reads structures and not
// lengths, which is why a ZX0 stream ends itself and the buffer's index ends
// with a 255.
func buildPlus3(images map[int][]byte, note func(string)) plus3Built {
	opts := plus3Options(images)
	opts.Note = note

	return buildPlus3Options(opts)
}

// buildPlus3Options is the same for a test that needs an option the shorthand
// does not reach.
func buildPlus3Options(opts Options) plus3Built {
	fail := func(err error) plus3Built { return plus3Built{err: err} }

	bundle, err := ZXRelease(someDatabase(), "plus3", opts)
	if err != nil {
		return fail(err)
	}

	built := plus3Built{
		release: make(map[string][]byte, bundle.Total()),
		disk:    map[string][]byte{},
	}

	for _, f := range bundle.Files() {
		built.release[f.Name] = f.Body
	}

	if built.image, err = ZXDisk(someDatabase(), "plus3", opts); err != nil {
		return fail(err)
	}

	volume, err := media.NewDSK(built.image)
	if err != nil {
		return fail(err)
	}

	files, err := volume.Files()
	if err != nil {
		return fail(err)
	}

	for _, f := range files {
		if _, seen := built.disk[f.Name]; seen {
			return fail(fmt.Errorf("two files called %s", f.Name))
		}

		built.disk[f.Name] = f.Data
	}

	return built
}

// opened is the fixture with the checks every one of these tests would make
// first: that it built, and that it fits on a disk.
func opened(t *testing.T, built plus3Built) (release, disk map[string][]byte) {
	t.Helper()

	require.NoError(t, built.err)
	require.LessOrEqual(t, len(built.image), 194816, "more than a 177K disk and its container")

	return built.release, built.disk
}

// A handful of pictures: a disk with everything on it and nothing having to
// leave, so that what is read back is what was meant to be there.
func someScreens() map[int][]byte { return screens(8) }

// Pictures that will not pack, for the cases about running out of room.
//
// A fixed generator and not a formula in the position, because anything
// polynomial in a byte repeats every 256 of them and ZX0 finds that at once:
// the first attempt packed 6912 bytes down to 40 and the disk held all forty
// pictures with room to spare.
func incompressibleScreens(count int) map[int][]byte {
	out := make(map[int][]byte, count)

	for n := range count {
		one := make([]byte, screenSize)

		x := uint32(n)*2654435761 + 1
		for at := range one {
			x = x*1103515245 + 12345
			one[at] = byte(x >> 16)
		}

		out[n] = one
	}

	return out
}

// Spanish, named, and with pictures, which is the build the examples make.
func plus3Options(images map[int][]byte) Options {
	return Options{Language: language.LanguageSpanish, Name: "AO", Images: images}
}

// The two disks these tests read, each built once for the whole package.
//
// Building one is not cheap: ZXDisk asks ZXRelease for the files and the tests
// ask for them again to compare, so every picture is packaged twice, and
// packaging a picture means the optimal parse over a whole screen — see
// internal/zx0. Shared, the full disk is built once instead of five times.
//
// Nothing written here is modified by a test. A map handed to several of them
// would be a trap, and the reason it is not one is that every assertion reads.
var (
	smallDisk = sync.OnceValue(func() plus3Built { return buildPlus3(someScreens(), nil) })

	// More pictures than the directory holds, which is what forces the buffer.
	bufferedDisk = sync.OnceValue(func() plus3Built {
		return buildPlus3(screens(overrunning()), nil)
	})
)

// plus3Built is a +3 release and the same release after a round trip through a
// disk image.
type plus3Built struct {
	release map[string][]byte
	disk    map[string][]byte
	image   []byte
	err     error
}

// The texts that live outside the database go on the disk under the one name
// the interpreter will look for.
//
// It is the only file here whose name is not a convention: PLUS3.ASM carries
// "0.XMB" as a string and opens that, so a release that called it anything
// else would be a release whose every XMES printed nothing. And it goes on
// without a +3DOS header, like the pictures and unlike everything the loader
// LOADs — the offsets in the processes are into the file, so 128 bytes in
// front of the first message would put every one of them 128 bytes out.
func TestThePlus3DiskCarriesTheXMessagesUnderTheirOwnName(t *testing.T) {
	// Not real xmessages: what is on the disk is whatever the linker handed
	// over, and this is about the name and the headerlessness.
	texts := []byte("\xff\xfe\xfd\xfc the texts, as the linker laid them")

	opts := plus3Options(someScreens())
	opts.XMessages = texts

	built := buildPlus3Options(opts)
	_, disk := opened(t, built)

	body, ok := disk[plus3XMessages]
	require.True(t, ok, "the interpreter opens 0.XMB and the disk has no such file")

	require.Equal(t, texts, body[:len(texts)],
		"the file came off the disk changed, so no offset in it means what it meant")
	require.Equal(t, body, media.WithoutPlus3Header(body),
		"it has a +3DOS header, which would put every message 128 bytes out")
}

// A 48K tape has nowhere to read them from, and refuses rather than shipping an
// adventure that cannot print half its text.
func TestA48KTapeRefusesXMessages(t *testing.T) {
	opts := plus3Options(nil)
	opts.XMessages = []byte("the texts")

	_, err := ZXRelease(someDatabase(), "48k", opts)
	require.ErrorContains(t, err, "no disk to read xmessages from")
}

// Every file in the release is on the disk, under the name it was given, with
// the bytes it was given.
//
// The names are the loader's and not ours — it LOADs them one by one — so a
// name the reader cannot find is a name the machine cannot find, and the +3
// stops on "File not found" with the border still the colour the BASIC left it.
func TestThePlus3DiskGivesBackEveryFileItWasGiven(t *testing.T) {
	release, disk := opened(t, smallDisk())

	require.Len(t, disk, len(release), "the disk holds a different number of files")

	for name, body := range release {
		t.Run(name, func(t *testing.T) {
			got, ok := disk[name]
			require.True(t, ok, "the loader opens %s and the disk has no such file", name)

			want := media.WithoutPlus3Header(body)

			require.GreaterOrEqual(t, len(got), len(want), "%s came back short", name)
			require.Equal(t, want, got[:len(want)], "%s came off the disk changed", name)
			require.Less(t, len(got)-len(want), media.Plus3HeaderSize,
				"%s came back more than one record longer than it is", name)
		})
	}
}

// The four the loader opens carry a +3DOS header and the pictures do not.
//
// This is not a tidiness: the interpreter opens a picture and reads five bytes
// of geometry off the front of it, so 128 bytes of header there is a drawing
// the width of whatever byte 0 turns out to be. The reference pipeline draws
// the same line — SPECFORM puts a header on DAAD.BIN and the loading screen,
// and the .128 files go on the disk as CPCDiskXP finds them.
func TestOnlyTheFilesTheLoaderOpensCarryAHeader(t *testing.T) {
	release, _ := opened(t, smallDisk())

	for name, body := range release {
		headed := len(body) != len(media.WithoutPlus3Header(body))

		if strings.HasSuffix(name, plus3PictureSuffix) {
			require.False(t, headed,
				"%s has a +3DOS header, so its first five bytes are not the geometry", name)

			continue
		}

		require.True(t, headed, "%s has no +3DOS header, or one that does not add up", name)
	}
}

// Every picture on the disk comes back out as the screen it was made from.
//
// This is the one that turns loading a disk in an emulator into something go
// test does. A picture is five bytes of geometry and a ZX0 stream, and the
// interpreter reads exactly that much before handing the rest to the
// decompressor: a file the filesystem gave back a record short, or a block
// written at the wrong offset, is not a drawing that looks wrong, it is a
// screen of flashing attributes or a machine that resets.
func TestEveryPictureOnThePlus3DiskComesBackOut(t *testing.T) {
	pictures := someScreens()

	_, disk := opened(t, smallDisk())

	for n, screen := range pictures {
		name := plus3PictureName(n)

		t.Run(name, func(t *testing.T) {
			body, ok := disk[name]
			require.True(t, ok, "picture %d is not on the disk under the name a PICTURE asks for", n)
			require.Greater(t, len(body), plus3PictureHeader)

			lines := pictureLines(screen)

			require.Equal(t,
				[]byte{canvasWidth / 8, byte(lines / 8), canvasWidth / 6, 0, plus3TopOfScreen},
				body[:plus3PictureHeader],
				"not the geometry PLUS3.ASM reads into its five variables")

			out, err := zx0.Decompress(body[plus3PictureHeader:])
			require.NoError(t, err, "the stream on the disk will not decompress")
			require.Equal(t, plus3Crop(screen, lines), out,
				"the picture off the disk is not the screen it was made from")
		})
	}
}

// And the buffer comes back out too, walked the way the Z80 walks it.
//
// A disk with more pictures than the directory holds moves the small ones into
// DAAD.GRA, which the interpreter reads whole into page one at start-up and
// then indexes. The offsets in that index are absolute addresses in the page
// rather than offsets into the file, so a file that arrives a byte shifted does
// not fail to load, it draws from the wrong place.
func TestTheBufferOnThePlus3DiskComesBackOut(t *testing.T) {
	pictures := screens(overrunning())

	_, disk := opened(t, bufferedDisk())

	require.Contains(t, disk, plus3CacheName,
		"that many pictures do not fit in the directory, so there should be a buffer")

	// The two bytes in front are the load address the interpreter reads the
	// file to; the page is what follows. See plus3Cache.
	page := disk[plus3CacheName][2:]
	seen := 0

	for at := 0; page[at] != plus3CacheEnd; at += plus3CacheEntry {
		number := int(page[at])
		address := int(binary.LittleEndian.Uint16(page[at+1:]))
		size := int(binary.LittleEndian.Uint16(page[at+3:]))

		require.GreaterOrEqual(t, address, plus3CacheAt, "picture %d points below the page", number)
		require.LessOrEqual(t, address+size, plus3CacheTop, "picture %d runs off the page", number)

		from := address - plus3CacheAt + plus3PictureHeader

		out, err := zx0.Decompress(page[from : from+size])
		require.NoError(t, err, "buffered picture %d", number)
		require.Equal(t, plus3Crop(pictures[number], pictureLines(pictures[number])), out,
			"buffered picture %d is not the screen it was made from", number)

		seen++
	}

	require.NotZero(t, seen)
}

// A disk that cannot hold every picture says which ones it left off.
//
// This is the packager's one place to fail quietly, and the one it must not: a
// build that carries eight of forty drawings looks exactly like a build that
// carries all forty until somebody plays it and reaches the ninth room. The
// pictures are the only part that can be left out and still leave an adventure
// anybody can play, which is what makes dropping them right and dropping them
// silently wrong.
//
// Incompressible screens on purpose. The buffer cannot take them — it has
// under 9 KB for index and pictures together — so they stay files of their
// own, the names fit, and what runs out is the 177K.
func TestAPlus3DiskTooFullSaysWhatCameOff(t *testing.T) {
	pictures := incompressibleScreens(40)

	var note strings.Builder

	_, disk := opened(t, buildPlus3(pictures, func(s string) { note.WriteString(s) }))

	require.Contains(t, note.String(), "did not fit",
		"pictures left off the disk without a word")

	// The ones that stayed are whole, and they are the low numbers: the last
	// come off first, so what is left is a run from the front and not a disk
	// with holes in it.
	on := 0

	for n := range pictures {
		if _, ok := disk[plus3PictureName(n)]; ok {
			on++
		}
	}

	require.NotZero(t, on, "nothing fitted, which is not a disk anybody can play")
	require.Less(t, on, len(pictures), "they all fitted, so this tests nothing")

	for n := range on {
		name := plus3PictureName(n)

		body, ok := disk[name]
		require.True(t, ok, "%s came off and a higher number stayed", name)

		out, err := zx0.Decompress(body[plus3PictureHeader:])
		require.NoError(t, err, "%s", name)
		require.Equal(t, plus3Crop(pictures[n], pictureLines(pictures[n])), out)
	}

	// And the four the loader needs are still there, because a disk of
	// pictures and no adventure is not a release.
	for _, name := range []string{plus3Loader, plus3Interpreter, plus3Database, plus3Graphics} {
		require.Contains(t, disk, name)
	}
}

// The interpreter that reaches the disk is the quieted one.
//
// A build with a buffer gets a DI put in front of the decompressor, appended as
// a stub at the end of the image — see plus3DisableInterrupts, and the comment
// above it for what the 50 Hz interrupt does to a picture otherwise. The patch
// has its own test; what this asks is whether the patched image is the one that
// went on the disk, because an interpreter patched and then not written is a
// bug no test of the patching can see.
//
// The stub is found from the length of the file and not of what came off the
// disk, because the tail of that is the last record's padding.
func TestThePlus3DiskCarriesTheQuietedInterpreter(t *testing.T) {
	release, disk := opened(t, bufferedDisk())

	image := disk[plus3Interpreter]
	ends := len(media.WithoutPlus3Header(release[plus3Interpreter]))

	stub := image[ends-plus3StubSize : ends]

	require.Equal(t, byte(0xF3), stub[0], "the image on the disk does not end in a DI")
	require.Equal(t, byte(0xCD), stub[1], "the stub does not call the decompressor")
	require.Equal(t, byte(0xC9), stub[4], "the stub does not return")
}
