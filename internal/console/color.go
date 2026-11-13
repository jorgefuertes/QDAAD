// Package console draws the terminal output of the tools, in the colours of the
// machine the adventures were written for.
package console

import (
	"github.com/charmbracelet/lipgloss"
)

// The ZX Spectrum palette, in the order the hardware numbers it — the same
// order INK and PAPER use, and the one the drawing engine of the eight-bit
// adventures writes into the low bits of a command byte.
//
// Every component is off or full: 0xD7 at normal brightness and 0xFF with
// BRIGHT on, which is the pair emulators settled on. Black is black either way,
// so the palette really offers fifteen colours and not sixteen.
const (
	Black Colour = iota
	Blue
	Red
	Magenta
	Green
	Cyan
	Yellow
	White
)

// Colour is an index into the palette, 0 to 7.
type Colour uint8

// The two brightness levels, as the hardware produced them.
var (
	normal = [8]string{
		Black:   "#000000",
		Blue:    "#0000D7",
		Red:     "#D70000",
		Magenta: "#D700D7",
		Green:   "#00D700",
		Cyan:    "#00D7D7",
		Yellow:  "#D7D700",
		White:   "#c6c6c6",
	}

	bright = [8]string{
		Black:   "#000000",
		Blue:    "#0000FF",
		Red:     "#FF0000",
		Magenta: "#FF00FF",
		Green:   "#00FF00",
		Cyan:    "#00FFFF",
		Yellow:  "#FFFF00",
		White:   "#FFFFFF",
	}
)

// Hex gives the palette entry as it was on the screen, which is what drawing a
// picture wants. Terminal output wants Adaptive instead.
func Hex(c Colour, isBright bool) string {
	if c > White {
		c = White
	}

	if isBright {
		return bright[c]
	}

	return normal[c]
}

// Adaptive turns a palette entry into a colour a terminal can use whatever its
// background.
//
// A Spectrum was bright ink on black paper, so a dark terminal takes the bright
// values and a light one the plain ones, which are dark enough to read on white.
// Four of the eight cannot be had that way and are named here rather than
// hidden in a branch:
//
//   - Black would vanish on a dark background, so it becomes the palette's own
//     white there: asking for ink 0 gives something legible rather than nothing.
//   - White would vanish on a light background, so it becomes near-black there,
//     for the same reason read the other way round.
//   - Yellow at #D7D700 is barely legible on white, and goes darker still.
//   - Blue at #0000FF is barely legible on black, and goes lighter.
//
// The palette itself is untouched: Hex still returns what the hardware showed.
func Adaptive(c Colour) lipgloss.AdaptiveColor {
	if c > White {
		c = White
	}

	switch c {
	case Black:
		return lipgloss.AdaptiveColor{Light: normal[Black], Dark: normal[White]}
	case White:
		return lipgloss.AdaptiveColor{Light: "#1C1C1C", Dark: bright[White]}
	case Yellow:
		return lipgloss.AdaptiveColor{Light: "#8A8A00", Dark: bright[Yellow]}
	case Blue:
		// #0000FF against black measures about 2.4 to 1, too dim to read
		// comfortably. Lightening a pure blue means raising red and green
		// together, which is what turns it mauve, so the red is left at zero —
		// as it is in the palette — and only the green comes up. The result is
		// xterm's colour 33, about 5.8 to 1, and still plainly blue.
		return lipgloss.AdaptiveColor{Light: normal[Blue], Dark: "#0087FF"}
	}

	return lipgloss.AdaptiveColor{Light: normal[c], Dark: bright[c]}
}

// Style is a plain foreground in one of the palette's colours.
func Style(c Colour) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Adaptive(c))
}

// Colours the tools need and the palette does not have. A Spectrum offers eight
// inks and no more, so anything outside them is named here rather than smuggled
// into the arrays above, which promise what the hardware showed.
//
// Both pairs are exact xterm entries, so a terminal of 256 colours lands on them
// instead of approximating.
var (
	// Orange, for a warning: near enough red to read as one, far enough not to
	// be mistaken for an error. Light is xterm 130 and dark is xterm 208; a
	// bright orange on white measures about 2.4 to 1, hence the darker one.
	Orange = lipgloss.AdaptiveColor{Light: "#AF5F00", Dark: "#FF8700"}

	// Grey, for what should be there but not read first. The palette goes black,
	// then #c6c6c6, then white, with no middle to borrow. Light is xterm 242 and
	// dark is xterm 244.
	Grey = lipgloss.AdaptiveColor{Light: "#6C6C6C", Dark: "#808080"}
)

// The styles the tools write with. They are built on the palette so that the
// output of a program about these adventures looks like the machines they ran
// on, and every one of them reads on a light terminal and on a dark one.
var (
	// Title is what a section of output is headed with.
	TitleStyle = Style(Cyan).Bold(true)
	// Subtitle is a heading below a title.
	SubtitleStyle = Style(Cyan)
	// Ok reports something that worked.
	OkStyle = Style(Green)
	// Warn reports something worth knowing that did not stop the work. Orange
	// rather than the palette's yellow, which Path already takes.
	WarnStyle = lipgloss.NewStyle().Foreground(Orange)
	// Err reports something that stopped it.
	DebugStyle = Style(Magenta)
	ErrStyle   = Style(Red).Bold(true)
	// Path names a file, a directory or a section of a source.
	PathStyle = Style(Yellow).Underline(true)
	// Value names a number or a piece of data taken from a database.
	ValueStyle = Style(Blue)
	// Muted is for what should be there but not read first: units, counts,
	// anything that explains the line beside it.
	MutedStyle = lipgloss.NewStyle().Foreground(Grey)
	BoldStyle  = lipgloss.NewStyle().Bold(true)
)
