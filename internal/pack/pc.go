package pack

import (
	"fmt"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/runtime"
)

// A PC release, which is the one machine here whose release is not a container.
//
// Every other target ends in a thing with a shape: a tape is a stream of blocks
// at fixed addresses, a disk is a format with a directory and a specification.
// A PC has a filesystem and megabytes of it, so DAAD's answer is a directory of
// loose files with the names the interpreter opens, and the whole of the
// packaging is putting the right bytes in the right names. There is no loader,
// no memory map, no paging and no compression.
//
// The names are first-hand, which is the difference from every other target
// here: the interpreter is in the checkout as Pascal, so what it opens is read
// rather than measured.
//
//	DAAD.DDB  the database, and the default when none is named  pcdaad.pas:232
//	DAAD.FNT  the font, tried first                             pcdaad.pas:235
//	DAAD.CHR  the font, if there is no .FNT                      pcdaad.pas:236
//	DAAD.MSD  the picture shown before anything else, which is
//	          picture 65535 under another name                   pic.pas:103
//	NNN.MSD   a picture, three digits                            pic.pas:105-106
//	0.XMB     the texts the processes read off disk
//
// What is not here is DOSBox. DAAD READY's PC release carries dosbox.exe, two
// SDL DLLs and a .BAT that runs them, which is a Windows program for running a
// DOS program and no use to anybody on anything else. So this writes the game
// directory and nothing around it: mount it in whatever DOS you have — dosbox,
// dosbox-x, a real machine — and run PCDAAD.
const (
	pcDatabase   = "DAAD.DDB"
	pcFont       = "DAAD.FNT"
	pcLegacyFont = "DAAD.CHR"
	pcLoading    = pcLoadingName + pcPictureSuffix
	pcXMessages  = "0.XMB"
)

// pcInterpreter is what the interpreter is called in the release.
//
// It keeps the name DAAD READY gives it, because a .BAT that says PCDAAD is
// what every note and every forum post about this interpreter says to type.
const pcInterpreter = "PCDAAD.EXE"

// PCRelease gathers what a PC build needs.
//
// subtarget is the one the compiler was given. VGA256 is the only one DAAD
// READY builds — every PC target in the kit calls DRF with "pc vga256" — and
// it is the only one here: the others change the screen the interpreter draws
// on and so the size and the depth of every picture, which is a different
// conversion and not a flag.
func PCRelease(database []byte, subtarget string, opts Options) (*Release, error) {
	switch strings.ToLower(subtarget) {
	case "", "vga256":
	default:
		return nil, fmt.Errorf(
			"%q is not a PC this writes: it is vga256, which is the one DAAD READY builds",
			subtarget)
	}

	_, interpreter, err := runtime.Interpreter(runtime.PC, opts.Language)
	if err != nil {
		return nil, err
	}

	companions, err := runtime.Companions(runtime.PC)
	if err != nil {
		return nil, err
	}

	release := NewRelease(
		File{Name: pcInterpreter, Body: interpreter, Note: "the interpreter: run this"},
		File{Name: pcDatabase, Body: database, Note: "the adventure"},
	)

	font, name, err := pcFontFile(opts.Charset, companions)
	if err != nil {
		return nil, err
	}

	release.Add(name, pcFontNote(name), font)

	if len(opts.Loading) > 0 {
		// The loading screen is a picture like any other and goes in whole: it
		// is the one that may use the full height, because there is no text
		// under it yet.
		screen, err := pcPicture(opts.Loading, 0)
		if err != nil {
			return nil, fmt.Errorf("the loading screen: %w", err)
		}

		release.Add(pcLoading, "shown before the adventure starts", screen)
	}

	// Each drawing as tall as it is, unless the build says one height for all
	// of them. See pcPicture, and Options.Lines.
	built, numbers, err := pcPictures(opts.Images, opts.Lines, opts.Progress)
	if err != nil {
		return nil, err
	}

	for _, n := range numbers {
		release.Add(pcPictureName(n), fmt.Sprintf("picture %d", n), built[n])
	}

	// One file and not several: the block size for this target is the whole 64
	// KB an XMES offset can name, so there is never a second one. The machines
	// that split theirs are the ones whose operating system cannot seek.
	if len(opts.XMessages) > 0 {
		release.Add(pcXMessages,
			fmt.Sprintf("%d bytes of text the processes read off disk", len(opts.XMessages)),
			opts.XMessages)
	}

	release.Add(pcStarter(opts.Name), "starts it", pcBatch())

	return release, nil
}

// pcFontFile picks the font and the name that decides how it is read.
//
// The interpreter tries DAAD.FNT and then DAAD.CHR, and the two are different
// formats rather than two places to look. A .FNT is SINTAC's: a header, and in
// its last 2304 bytes a table of 256 widths followed by the glyphs, which is
// what makes PC text proportional — the only target here where it is. A .CHR is
// the plain 2048 bytes every eight-bit machine uses, and the interpreter gives
// every glyph a width of six (charset.pas:20-52).
//
// So a source that declares an eight-bit character set gets it, at six pixels a
// glyph, and a source that declares nothing gets the built-in proportional one.
// Handing a 2048-byte charset over as DAAD.FNT would have it read from the
// wrong end and print nothing legible.
func pcFontFile(charset []byte, companions map[string][]byte) (body []byte, name string, err error) {
	if len(charset) == 0 {
		built, carried := companions["PC.FNT"]
		if !carried {
			return nil, "", fmt.Errorf("the built-in PC font is missing")
		}

		return built, pcFont, nil
	}

	// The widths and the glyphs, which is the least a .FNT can be.
	const sintacTail = 256 + charsetSize

	if len(charset) >= sintacTail {
		return charset, pcFont, nil
	}

	if _, err := charsetGlyphs(charset); err != nil {
		return nil, "", fmt.Errorf(
			"a PC font is either SINTAC's, at least %d bytes, or a plain charset: %w",
			sintacTail, err)
	}

	return charset, pcLegacyFont, nil
}

func pcFontNote(name string) string {
	if name == pcLegacyFont {
		return "the character set, every glyph six pixels wide"
	}

	return "the character set, with a width for each glyph"
}

// pcStarter is the name of the batch file that runs the game.
//
// DAAD READY names it after the adventure, which is what somebody typing at a
// DOS prompt wants: the directory lists one obvious thing to type. Eight
// characters and no more, because this is a filesystem that gives you eight.
func pcStarter(name string) string {
	const dosNameLength = 8

	bare := make([]byte, 0, dosNameLength)

	for _, c := range strings.ToUpper(name) {
		if c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			bare = append(bare, byte(c))
		}

		if len(bare) == dosNameLength {
			break
		}
	}

	if len(bare) == 0 {
		return "GAME.BAT"
	}

	return string(bare) + ".BAT"
}

// pcBatch is the batch file itself.
//
// One line, and no parameters: -S is the switch for the 640x400 screen and
// this writes the 320x200 one. CRLF because this is a DOS file and a DOS
// command interpreter reading a lone newline is a DOS command interpreter
// reading the next line as part of this one.
func pcBatch() []byte {
	return []byte("@" + strings.TrimSuffix(pcInterpreter, ".EXE") + "\r\n")
}
