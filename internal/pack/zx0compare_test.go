//go:build compare

// Holds our ZX0 streams against Einar Saukas' own coder, over the blocks a
// Spectrum build really carries.
//
// It lives behind a build tag because it needs a C compiler and the coder's
// source out of work/, neither of which a plain "go test ./..." has. Run it
// with make compare-zx0, which builds the bridge and sets ZX0REF.
//
// It is in internal/pack rather than internal/zx0 because the blocks are the
// point: a 48K build compresses the runs cutScreen takes out of a screen and a
// +3 build the whole screen plus3Crop blanks the bottom of, and neither of
// those is a thing a raw .scr file is. Checking the format against pictures
// nobody ships would prove less.
package pack

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jorgefuertes/QDAAD/internal/zx0"
)

// The screens every Spectrum tree in examples/ carries, which is what a build
// of La Aventura Original actually puts on a tape or a disk.
const (
	screenRoot = "../../examples/AO/ZXSpectrum/"
	screenGlob = screenRoot + "*/part*/gfx/scr/*.scr"
)

// The two formats Saukas' compress() writes, as its invert_mode takes them.
//
// classicMode is the oracle: it is what DCP.c:301 passes, and the only one the
// interpreter's decoder reads. otherMode is here to be refused — see
// TestTheClassicModeIsTheOneOurDecoderReads.
const (
	classicMode = "1"
	otherMode   = "0"
)

// TestTheClassicModeIsTheOneOurDecoderReads pins which ZX0 this is.
//
// Saukas' compress() takes an invert_mode, and the two modes are two formats:
// one of them is what dzx0_standard.asm reads — the 68 bytes of Z80 the
// interpreter runs, and what internal/zx0's Decompress is a translation of —
// and the other walks it off into memory. DCP.c:301 passes TRUE, so TRUE is
// what the reference pipeline writes and what the rest of this file compares
// against.
//
// This asks rather than trusts the call site, because of what the answer costs
// if it ever changes: every picture on every tape and disk written here is a
// stream our coder wrote, so the mode is not a detail of the comparison, it is
// the ground the pictures stand on.
func TestTheClassicModeIsTheOneOurDecoderReads(t *testing.T) {
	reference, _ := setUp(t)

	// A block with a run in it. The question is how an offset is coded, so a
	// block the coder finds no match in would not tell the modes apart.
	block := []byte("the same bytes over and over, the same bytes over and over")

	back, err := zx0.Decompress(reference(t, classicMode, block))
	require.NoError(t, err, "the classic mode is unreadable, which leaves no format to write")
	require.Equal(t, block, back)

	// And the other mode is not, which is what makes the first a choice and
	// not a coincidence.
	back, err = zx0.Decompress(reference(t, otherMode, block))

	require.False(t, err == nil && bytes.Equal(block, back),
		"both modes read back, so this tells the two formats nothing apart")
}

// TestSaukasOwnStreamsReadBack is the check the rest of the ZX0 work rests on.
//
// Every other test of this format is our coder against our decoder, which
// proves they agree and not that either is right: a shared misreading of how
// an offset is coded passes a round trip and walks the interpreter off into
// memory. This one takes streams our code had no hand in writing, hands them
// to our decoder, and requires the bytes back. Passing it means the decoder is
// the format and not our idea of it — and since the coder's output round-trips
// through that same decoder, the coder is too.
func TestSaukasOwnStreamsReadBack(t *testing.T) {
	reference, blocks := setUp(t)

	for _, b := range blocks {
		t.Run(b.name, func(t *testing.T) {
			back, err := zx0.Decompress(reference(t, classicMode, b.body))
			require.NoError(t, err, "our decoder would not read the reference's stream")
			require.Equal(t, b.body, back, "the reference's stream came back wrong")
		})
	}
}

// TestOurStreamsReadBack is the round trip again, over the same blocks: a
// safety net under the one above, so that a failure says which of the two
// halves moved.
func TestOurStreamsReadBack(t *testing.T) {
	_, blocks := setUp(t)

	for _, b := range blocks {
		t.Run(b.name, func(t *testing.T) {
			back, err := zx0.Decompress(zx0.Compress(b.body))
			require.NoError(t, err)
			require.Equal(t, b.body, back)
		})
	}
}

// TestWhatOurCoderCostsAgainstSaukas reports and does not judge.
//
// Our match finder is greedy and his runs a shortest path over the same
// format, so his streams are smaller and ours are not wrong. The number is
// here to be read: it is what decides whether writing the shortest path is
// worth it, and on a 48K tape — where what fits is counted in pictures — a few
// per cent is a picture or two.
func TestWhatOurCoderCostsAgainstSaukas(t *testing.T) {
	reference, blocks := setUp(t)

	var raw, theirs, ours int

	worst, worstAt := 0, ""

	for _, b := range blocks {
		them := len(reference(t, classicMode, b.body))
		us := len(zx0.Compress(b.body))

		raw += len(b.body)
		theirs += them
		ours += us

		if over := (us - them) * 100 / them; over > worst {
			worst, worstAt = over, b.name
		}
	}

	t.Logf("%d blocks, %d bytes raw", len(blocks), raw)
	t.Logf("reference %7d bytes, %d%% of raw", theirs, theirs*100/raw)
	t.Logf("ours      %7d bytes, %d%% of raw", ours, ours*100/raw)

	over := (ours - theirs) * 100 / theirs
	if worst == 0 {
		t.Logf("ours is %d%% over the reference, and no block is bigger than his", over)

		return
	}

	t.Logf("ours is %d%% over the reference, worst %d%% at %s", over, worst, worstAt)
}

// block is one of the things that reaches zx0.Compress in a real build.
type block struct {
	name string
	body []byte
}

// setUp finds the bridge and cuts the screens up the way the packagers do.
func setUp(t *testing.T) (func(*testing.T, string, []byte) []byte, []block) {
	t.Helper()

	bridge := os.Getenv("ZX0REF")
	if bridge == "" {
		t.Skip("ZX0REF is not set: run make compare-zx0, which builds the reference coder")
	}

	names, err := filepath.Glob(screenGlob)
	require.NoError(t, err)

	if len(names) == 0 {
		t.Skipf("no screens in %s", screenGlob)
	}

	sort.Strings(names)

	dir := t.TempDir()

	// The bridge takes two paths rather than a pipe, because that is what
	// Saukas' coder does: it sizes the input file to allocate for it.
	reference := func(t *testing.T, mode string, body []byte) []byte {
		t.Helper()

		in := filepath.Join(dir, "in.bin")
		out := filepath.Join(dir, "out.zx0")

		require.NoError(t, os.WriteFile(in, body, 0o600))

		output, err := exec.Command(bridge, mode, in, out).CombinedOutput()
		require.NoError(t, err, "the reference coder failed: %s", output)

		stream, err := os.ReadFile(out)
		require.NoError(t, err)
		require.NoError(t, os.Remove(out))

		return stream
	}

	var blocks []block

	for _, name := range names {
		screen, err := os.ReadFile(name)
		require.NoError(t, err)
		require.Len(t, screen, screenSize, "%s is not a Spectrum screen", name)

		// The tree, the part and the number, which is enough to find it again.
		short := strings.ReplaceAll(strings.TrimSuffix(
			strings.TrimPrefix(name, screenRoot), ".scr"), "/gfx/scr", "")

		lines := pictureLines(screen)

		// What a +3 build compresses: the whole screen, blanked below the
		// picture.
		blocks = append(blocks, block{
			name: short + "/plus3",
			body: plus3Crop(screen, lines),
		})

		// What a 48K build compresses: the runs of video memory, one stream
		// each.
		cut, err := cutScreen(screen, lines)
		require.NoError(t, err)

		for i, b := range cut {
			blocks = append(blocks, block{
				name: fmt.Sprintf("%s/48k-%02d", short, i),
				body: b.body,
			})
		}
	}

	return reference, blocks
}
