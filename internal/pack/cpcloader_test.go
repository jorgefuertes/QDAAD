package pack

import (
	"slices"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/stretchr/testify/require"
)

// cpcLoadingOptions is a build with a picture to show while it loads.
func cpcLoadingOptions() Options {
	palette := make([]byte, cpcPaletteFile)
	for i := range palette {
		palette[i] = byte(i + 1)
	}

	return Options{
		Language: language.LanguageSpanish, Name: "AO",
		Loading: make([]byte, cpcPackScreen), LoadingPalette: palette,
	}
}

// A release with a loading screen is three files and not one, in the order the
// loader asks for them: itself first, because it is what RUN starts.
func TestTheCPCLoaderReleaseIsThreeFiles(t *testing.T) {
	r, err := CPCRelease(make([]byte, 5000), cpcLoadingOptions())
	require.NoError(t, err)

	names := make([]string, 0, 3)
	for _, f := range r.Files() {
		names = append(names, f.Name)
	}

	require.Equal(t, []string{"AO.BIN", "AO.SCR", "AO.GAM"}, names)

	// The loader is a page and no more: it sits in the last one below the
	// interpreter's graphics block, and everything under that is for the
	// pictures.
	loader := media.WithoutAMSDOSHeader(r.Files()[0].Body)
	require.LessOrEqual(t, len(loader), 0x100)

	// The picture is the whole of a display.
	require.Len(t, media.WithoutAMSDOSHeader(r.Files()[1].Body), cpcPackScreen)

	// The loader is what a player types, so it takes the plain name; the game
	// keeps the whole binary it would have been on its own.
	plain, err := CPCRelease(make([]byte, 5000),
		Options{Language: language.LanguageSpanish, Name: "AO"})
	require.NoError(t, err)

	one, err := plain.Single()
	require.NoError(t, err)

	require.Equal(t, media.WithoutAMSDOSHeader(one.Body),
		media.WithoutAMSDOSHeader(r.Files()[2].Body))
}

// The loader puts AMSDOS back before it asks for anything.
//
// This is the whole of why it works, and it is four steps rather than the one
// that looks like it should do: RUN enters a binary through MC START PROGRAM,
// which resets the firmware and takes AMSDOS's patches with it, so a loader
// that opens a file without this gets "Press PLAY then any key" with the disk
// in the drive. Worth a test because nothing about it is guessable and the
// symptom is a release that looks built and will not load.
func TestTheCPCLoaderPutsAMSDOSBack(t *testing.T) {
	r, err := CPCRelease(make([]byte, 5000), cpcLoadingOptions())
	require.NoError(t, err)

	loader := media.WithoutAMSDOSHeader(r.Files()[0].Body)
	calls := cpcFirmwareCalls(loader)

	for _, want := range []int{klRomWalk, klFindCommand, mcStartProgram, klInitBack} {
		require.Contains(t, calls, want)
	}

	// And in that order: walking the ROMs and finding the disc one happen
	// before the restart, and initialising it by number after.
	require.Less(t, indexOf(calls, klRomWalk), indexOf(calls, mcStartProgram))
	require.Less(t, indexOf(calls, klFindCommand), indexOf(calls, mcStartProgram))
	require.Greater(t, indexOf(calls, klInitBack), indexOf(calls, mcStartProgram))

	// DISC is the command it looks for, and finding it is finding AMSDOS.
	require.Contains(t, string(loader), "DIS")

	// And it turns the cassette messages off, or the firmware prints over the
	// picture it has just put up.
	require.Contains(t, calls, casNoisy)
}

// And a tape loader does not, which is the same fact read the other way round.
//
// The walk alone brings AMSDOS back on any machine with a drive, and AMSDOS
// hooks the cassette vectors to the disk: from there the loader asks a disk
// that is not there for its files and says "Drive A: disc missing" over the
// picture it has just put up. Worth a test for the same reason as the last one
// — the symptom is a release that looks built and will not load.
func TestTheCPCTapeLoaderLeavesAMSDOSAlone(t *testing.T) {
	opts := cpcLoadingOptions()
	opts.Images = nil

	tape, err := CPCTape(make([]byte, 5000), opts)
	require.NoError(t, err)

	loader, err := cpcLoaderCode(opts.Name, opts.LoadingPalette[:cpcPaletteFile],
		cpcLoaderOrigin(), cpcOnTape, nil)
	require.NoError(t, err)

	calls := cpcFirmwareCalls(loader)

	for _, not := range []int{klRomWalk, klFindCommand, mcStartProgram, klInitBack} {
		require.NotContains(t, calls, not)
	}

	// It still reads the tape and still keeps the firmware quiet.
	require.Contains(t, calls, casNoisy)
	require.Contains(t, calls, casInOpen)
	require.Subset(t, tape, loader, "and that is the loader the tape carries")
}

// cpcFirmwareCalls is every firmware entry a piece of code calls or jumps to.
func cpcFirmwareCalls(code []byte) []int {
	calls := []int{}

	for at := 0; at+2 < len(code); at++ {
		if code[at] != 0xCD && code[at] != 0xC3 && code[at] != 0xC4 {
			continue
		}

		if to := int(code[at+1]) | int(code[at+2])<<8; to >= 0xBB00 {
			calls = append(calls, to)
		}
	}

	return calls
}

func indexOf(in []int, want int) int {
	for i, v := range in {
		if v == want {
			return i
		}
	}

	return -1
}

// A picture that is not a whole display, or that comes without its colours, is
// refused rather than packed into something that shows noise.
func TestACPCLoadingScreenIsCheckedBeforeItIsPacked(t *testing.T) {
	opts := cpcLoadingOptions()
	opts.Loading = make([]byte, 100)

	_, err := CPCRelease(make([]byte, 5000), opts)
	require.ErrorContains(t, err, "whole of video memory")

	opts = cpcLoadingOptions()
	opts.LoadingPalette = nil

	_, err = CPCRelease(make([]byte, 5000), opts)
	require.ErrorContains(t, err, "colours")
}

// A tape with no pictures on it says so in the interpreter, rather than
// letting the adventure find out at the first PICTURE and stop to ask for
// PLAY. Three bytes, and only on a tape: a disk can be asked in any order and
// answers for free.
func TestATapeWithoutPicturesStopsAsking(t *testing.T) {
	opts := Options{Language: language.LanguageSpanish, Name: "AO"}

	binary, err := cpcBinary(make([]byte, 5000), opts)
	require.NoError(t, err)

	const at = 0x095F - cpcInterpreterOrigin + media.AMSDOSHeaderSize

	// What the interpreter ships with: the call that builds the name and opens
	// the file.
	require.Equal(t, []byte{0xCD, 0xCA, 0x0C}, binary[at:at+3])

	// And what goes on a tape instead: a clear carry, which is the answer the
	// caller is already written to handle.
	quiet := cpcNoPictures(binary)
	require.Equal(t, []byte{0xB7, 0x00, 0x00}, quiet[at:at+3])

	// The original is left alone rather than patched in place, because it is
	// the same bytes a disk release is built from.
	require.Equal(t, []byte{0xCD, 0xCA, 0x0C}, binary[at:at+3])

	// A build whose bytes are not those is handed back as it came.
	odd := slices.Clone(binary)
	odd[at] = 0x00
	require.Equal(t, odd, cpcNoPictures(odd))
}
