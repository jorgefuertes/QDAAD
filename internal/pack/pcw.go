package pack

// An Amstrad PCW release, which is four files on a CP/M disk.
//
// It is the plainest machine this packages, and the reason is that nothing has
// to be glued: a PCW runs CP/M, CP/M has files, and the interpreter opens the
// ones it wants by name. There is no loader to write, no block to lay at an
// address, no bank to page. The disk is not even bootable — the player boots
// CP/M from their own system disk, puts this one in and types DAAD — which is
// what DAAD READY's own PCW.BAT does and what its emulator line says:
//
//	PCW -G 1 -A CPMv1_4.DSK -O ON -W 1000 -A GAME.DSK -G 0 -T "DAAD#0A"
//
// The four files, and they are PCW.BAT's names rather than ours:
//
//	DAAD.COM      the interpreter, which is DPCWISF3.BIN under another name
//	PARTE001.DDB  the database
//	PARTE001.CHR  the character set, behind an AMSDOS header
//	PARTE001.DAT  the illustrations, empty where there are none
//
// and 0.XMB beside them when the adventure has texts outside its database,
// which is the same name the +3 uses and the same one the interpreter carries
// as an FCB at 0x05EA.
//
// The disk is the PCW's own format: sector IDs from 1, one reserved track, and
// a specification in the first sector of it saying so.
//
// The 1989 editions are not. Cozumel and Los Templos Sagrados came on the
// CPC's DATA format — IDs from 0xC1, no reserved track — and so does the
// PCW.DSK every DAAD READY release starts from, because Aventuras AD shipped
// both Amstrads on the same disks. That is five kilobytes more room and it is
// what this wrote for a long time.
//
// What changed it is that the drive JOYCE emulates will not read one. It reads
// the directory, which is track 0, and then answers "error desconocido" for
// the first file — the same way it did for a reference release of the period.
// A disk nobody can load is worth less than five kilobytes.
//
// The illustrations are monochrome bitmaps rather than the vectors every other
// eight-bit edition carries, and the build has to hand them over already
// shaped: 512 by 104 pixels, one bit each. See pcwpictures.go for the format
// and script/pcwpic for where a tree's come from.

import (
	"errors"
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
)

const (
	// pcwInterpreter is what the interpreter is called on the disk. The player
	// types it, so it is the name and not a name.
	pcwInterpreter = "DAAD.COM"
	// pcwPart is the stem the interpreter builds its filenames from. The three
	// digits are the part, and 001 is the only one DAAD READY has ever
	// written: its PCW.BAT moves the database to PARTE001.DDB whatever part is
	// being built.
	pcwPart = "PARTE001"
	// The extensions, which the interpreter carries in a table at 0x267C along
	// with PIC and AGP that nothing here writes yet.
	pcwDatabase = ".DDB"
	pcwCharset  = ".CHR"
	pcwPictures = ".DAT"
	// pcwCover is the presentation screen, and it is not PARTE001: the 1989
	// disks of both Cozumel and Los Templos Sagrados carry one PARTE000.PIC
	// whichever part they are.
	pcwCover = "PARTE000.PIC"
	// pcwCharsetLoad is what the character set's AMSDOS header declares it
	// loads at, which is nowhere: the interpreter reads it as bytes and the
	// address is never used. Zero is what SPECFORM wrote in the one on
	// Cozumel's own disk, and matching it costs nothing.
	pcwCharsetLoad = 0x0000
)

// PCWRelease gathers a PCW build as the files that go on a disk.
func PCWRelease(database []byte, opts Options) (*Release, error) {
	if len(database) == 0 {
		return nil, errors.New("there is no database to put on a PCW disk")
	}

	name, interpreter, err := runtime.Interpreter(runtime.PCW, opts.Language)
	if err != nil {
		return nil, err
	}

	files := []File{{
		Name: pcwInterpreter,
		Body: interpreter,
		Note: fmt.Sprintf("the interpreter, which is %s: boot CP/M and type DAAD", name),
	}, {
		Name: pcwPart + pcwDatabase,
		Body: database,
		Note: "the adventure",
	}}

	// The character set, which is not optional. "Falta el juego de caracters"
	// is a string in the interpreter and what it says when there is none, typo
	// and all — so a release without one is a release that does not start.
	beside, err := runtime.Companions(runtime.PCW)
	if err != nil {
		return nil, err
	}

	charset := opts.Charset
	if len(charset) == 0 {
		if charset = beside["AD8x6.CHR"]; len(charset) == 0 {
			return nil, errors.New("the built-in character set is missing")
		}
	}

	glyphs, err := charsetGlyphs(charset)
	if err != nil {
		return nil, err
	}

	header, err := media.AMSDOSHeader(pcwPart+pcwCharset, pcwCharsetLoad, pcwCharsetLoad, len(glyphs))
	if err != nil {
		return nil, err
	}

	files = append(files, File{
		Name: pcwPart + pcwCharset,
		Body: append(header, glyphs...),
		Note: "the character set, behind the header SPECFORM puts on it",
	})

	// The cover, which the interpreter puts up by itself before the adventure
	// starts: it opens PARTE000.PIC without being asked and without a condact
	// naming it. The name is the part number and 000 is the only one there is.
	if len(opts.Loading) > 0 {
		files = append(files, File{
			Name: pcwCover,
			Body: opts.Loading,
			Note: "the cover, which the interpreter shows before anything else",
		})
	}

	if len(opts.XMessages) > 0 {
		files = append(files, File{
			Name: pcwXMessages,
			Body: opts.XMessages,
			Note: fmt.Sprintf("%d bytes of texts the processes read from disk",
				len(opts.XMessages)),
		})
	}

	// And the illustrations, which get what is left of the disk after
	// everything else is on it. The others are not negotiable — an adventure
	// without its database is not an adventure — so the drawings are what
	// gives, and the packer says which ones did.
	room, err := pcwRoom(files, opts)
	if err != nil {
		return nil, err
	}

	pictures, note, err := pcwPictureFile(beside, database, opts, room)
	if err != nil {
		return nil, err
	}

	files = append(files, File{
		Name: pcwPart + pcwPictures,
		Body: pictures,
		Note: note,
	})

	return NewRelease(files...), nil
}

// pcwXMessages is what the interpreter opens the texts outside the database as.
// It carries the name as a CP/M file control block at 0x05EA — "0" padded to
// eight and "XMB" — which is the same name the +3 uses.
const pcwXMessages = "0.XMB"

// PCWDisk writes a PCW release onto a disk image.
func PCWDisk(database []byte, opts Options) ([]byte, error) {
	release, err := PCWRelease(database, opts)
	if err != nil {
		return nil, err
	}

	if opts.Boot {
		return pcwBootDisk(release, opts)
	}

	files := make([]media.File, 0, len(release.Files()))
	for _, f := range release.Files() {
		files = append(files, media.File{Name: f.Name, Data: f.Body})
	}

	return media.PCWDisk(files)
}
