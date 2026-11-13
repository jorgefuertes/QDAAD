package pack

import (
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
)

// An Amstrad CPC release, which is one file and is made the way MCRF makes it.
//
// This is the machine the Spectrum's two shapes do not cover. A 48K tape is one
// stream because a tape has no names; a +3 disk is four files because the
// machine can load them by name. A CPC has a disk and loads by name too, and
// still wants one file — because the pieces are glued before they ever reach
// the disk, into a binary that AMSDOS loads in one go and runs.
//
// The layout is MCRF's, from work/DRC/src/mcrf.pas:
//
//	0x0840  the interpreter, padded with zeros to where the database starts
//	0x2880  the database
//	  then  the graphics block, with the font written 48 bytes into it
//
// and the whole thing carries one AMSDOS header saying it loads at 0x0840 and
// is entered there.
//
// Two of those numbers are not arbitrary and one is not even ours. 0x2880 is
// where the linker writes the database's pointers for a CPC build — it is
// arch.CPC's base address — so the padding is not decoration: a database one
// byte off from there is a database every pointer of which is wrong.
const (
	// cpcInterpreterOrigin is where the interpreter is loaded and entered.
	// MCRF calls it INTAT.
	cpcInterpreterOrigin = 0x0840
	// cpcCeiling is where the firmware's variables begin. Above them is the
	// video memory at 0xC000, and a build that reaches either is a build that
	// does not run. MCRF checks the same number.
	cpcCeiling = 0xA6FC
	// cpcFontOffset is how far into the graphics block the font sits. MCRF
	// seeks here and writes 2048 bytes over whatever was there.
	cpcFontOffset = 48
	// cpcGraphicsLengthAt is where in the interpreter its own copy of the
	// graphics length goes. MCRF patches the file three bytes past the end of
	// the header, which is the interpreter's own bytes 3 and 4.
	cpcGraphicsLengthAt = 3
)

// CPCRelease gathers a CPC build: one binary, with an AMSDOS header.
func CPCRelease(database []byte, opts Options) (*Release, error) {
	binary, err := cpcBinary(database, opts)
	if err != nil {
		return nil, err
	}

	// RUN"name" is what starts it, so the name is the one the player types.
	// DAAD's own releases add a BASIC loader called DISC that asks which part
	// you want; a release of one part has nothing to ask.
	return NewRelease(File{
		Name: opts.Name + ".BIN",
		Body: binary,
		Note: "interpreter, database and graphics in one file: RUN\"" + opts.Name + "\"",
	}), nil
}

// CPCDisk writes a CPC release onto a disk image.
func CPCDisk(database []byte, opts Options) ([]byte, error) {
	release, err := CPCRelease(database, opts)
	if err != nil {
		return nil, err
	}

	files := make([]media.File, 0, len(release.Files()))
	for _, f := range release.Files() {
		files = append(files, media.File{Name: f.Name, Data: f.Body})
	}

	return media.DataDisk(files)
}

func cpcBinary(database []byte, opts Options) ([]byte, error) {
	_, interpreter, err := runtime.Interpreter(runtime.CPC, opts.Language)
	if err != nil {
		return nil, err
	}

	companions, err := runtime.Companions(runtime.CPC)
	if err != nil {
		return nil, err
	}

	graphics, carried := companions["BLANK.GRA"]
	if !carried {
		return nil, fmt.Errorf("the built-in CPC graphics block is missing")
	}

	// The graphics block ships with an AMSDOS header of its own and the
	// interpreter does not, which is why this is asked rather than assumed.
	graphics = media.WithoutAMSDOSHeader(graphics)
	interpreter = media.WithoutAMSDOSHeader(interpreter)

	databaseOrigin := int(arch.CPC.BaseAddress(""))

	room := databaseOrigin - cpcInterpreterOrigin
	if len(interpreter) > room {
		return nil, fmt.Errorf(
			"the interpreter is %d bytes and the database starts %d after it, at 0x%04X",
			len(interpreter), room, databaseOrigin)
	}

	// The interpreter is told how much graphics follows it, in two bytes of
	// its own. It has to be a copy: the original is embedded and shared.
	code := make([]byte, room)
	copy(code, interpreter)
	code[cpcGraphicsLengthAt] = byte(len(graphics))
	code[cpcGraphicsLengthAt+1] = byte(len(graphics) >> 8)

	body := make([]byte, 0, room+len(database)+len(graphics))
	body = append(body, code...)
	body = append(body, database...)

	fontAt := len(body) + cpcFontOffset
	body = append(body, graphics...)

	// The font goes a fixed distance into the graphics block, which is where
	// the interpreter looks for it. MCRF seeks to the same place.
	if len(opts.Charset) > 0 {
		glyphs, err := charsetGlyphs(opts.Charset)
		if err != nil {
			return nil, err
		}

		if fontAt+len(glyphs) > len(body) {
			return nil, fmt.Errorf("the graphics block has no room for a font: %d bytes",
				len(graphics))
		}

		copy(body[fontAt:], glyphs)
	}

	if end := cpcInterpreterOrigin + len(body); end >= cpcCeiling {
		return nil, fmt.Errorf(
			"the build ends at 0x%04X and the firmware's variables start at 0x%04X: "+
				"%d bytes too much between the database and the graphics",
			end, cpcCeiling, end-cpcCeiling+1)
	}

	header, err := media.AMSDOSHeader(opts.Name+".BIN",
		cpcInterpreterOrigin, cpcInterpreterOrigin, len(body))
	if err != nil {
		return nil, err
	}

	return append(header, body...), nil
}
