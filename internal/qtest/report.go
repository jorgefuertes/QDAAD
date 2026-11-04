package qtest

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/jorgefuertes/QDAAD/internal/console"
)

// What a run says while it happens.
//
// The step is the unit. A caret announces one as it opens — so a run that
// hangs or crashes has already said which step it was in, which it could not
// when the name waited for the verdict — and the verdict lands at the end of
// that same line when its END is reached. Everything in between, a DEBUG or
// the adventure's own words under --show, breaks the line; the badge then
// stands on its own at the left margin, because a verdict floating at the end
// of a blank stretch reads as belonging to nothing.
//
// A failure is written where it happens, with the line it was written on, what
// was wanted, and what the adventure said instead — in two parts, because the
// two are read for different reasons. First the answer to the last order the
// script gave, named, which is where the phrase for an EXPECT comes from when
// somebody is writing one. Then everything else still unmatched, which is what
// a step whose EXPECTs are in the wrong order looks like.

// How much of what the adventure said goes in a failure: enough to recognise
// where the game got to, not so much that the failure scrolls away.
//
// The answer to the last order gets the most room, because it is the one
// somebody writing a script is reading. What came before it is context and
// gets less.
const (
	shown  = 12
	before = 6
)

// badge is how wide OK and FAIL are drawn. The same width for both, because
// they are read as one column and a column of two sizes is a column that
// wobbles.
const badge = 6

// Where a step's title is padded to, so that the badges line up however long
// the names are, and the ceiling that stops a long one pushing them off the
// screen.
const (
	titleGap  = 2
	titleWide = 64
)

// The report's own styles, built on the palette in internal/console so that a
// run of tests looks like the rest of the tools.
//
// The badges are the one place where a fixed colour is right rather than an
// adaptive one: the background is green or red whatever the terminal is, so
// the ink on it has to be chosen against that background and not against the
// terminal's.
var (
	stepCaret = console.Style(console.Blue)
	stepName  = console.Style(console.White)

	okBadge = lipgloss.NewStyle().
		Background(lipgloss.Color(console.Hex(console.Green, false))).
		Foreground(lipgloss.Color(console.Hex(console.Black, false))).
		Bold(true).Width(badge).Align(lipgloss.Center)

	failBadge = lipgloss.NewStyle().
			Background(lipgloss.Color(console.Hex(console.Red, false))).
			Foreground(lipgloss.Color(console.Hex(console.White, true))).
			Bold(true).Width(badge).Align(lipgloss.Center)

	frame = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(console.Adaptive(console.Cyan)).
		Padding(0, 1)
)

// Report collects what a run says. It writes as it goes, because a script that
// hangs or crashes has still told you something by the step it got to.
type Report struct {
	out io.Writer

	passed int
	failed int

	// wide is what step titles are padded to. Nothing is lined up until
	// somebody says how wide the names are; see Align.
	wide int

	// standing is true while a step's title has been written and its newline
	// is being held back for the badge that closes it.
	standing bool

	// midLine is true when the last thing written did not end a line, which is
	// what --show leaves behind: the adventure stops mid-sentence at its own
	// prompt and the next thing the report writes would be stuck to it. See
	// margin.
	midLine bool
}

// NewReport writes to w. Giving it io.Discard is how a test that only cares
// about the verdict keeps quiet.
func NewReport(w io.Writer) *Report { return &Report{out: w} }

// Align says how wide the longest step title is, so that every badge lands in
// the same column. Without it the titles are not padded and each badge follows
// its own name.
func (r *Report) Align(width int) { r.wide = min(width, titleWide) }

// Title names the run, in a frame, because it is the one line that is about
// the whole of what follows rather than about a step of it.
func (r *Report) Title(name string) {
	if name == "" {
		return
	}

	r.sayf("%s\n\n", frame.Render(console.TitleStyle.Render(name)))
}

// Step announces one as it opens. The newline is held back, because the badge
// that closes the step goes at the end of this line.
func (r *Report) Step(name string) {
	r.breakLine()
	r.margin()

	r.sayf("%s %s", stepCaret.Render(console.RightCaret), stepName.Render(name))

	// What the badge follows is the last line of the name, which is the whole
	// of it unless somebody has put a break in one.
	if pad := r.wide - lastWidth(name); pad > 0 {
		r.sayf("%s", strings.Repeat(" ", pad))
	}

	r.standing = true
}

// Pass is an END reached, which is a step that held all the way through.
func (r *Report) Pass() {
	r.passed++

	r.verdict(okBadge.Render("OK"))
}

// Heard is what the adventure had said when a step failed, which is most of
// the answer to why it did.
type Heard struct {
	// Order is the last thing the script gave it, as the script wrote it, and
	// After is what it said back.
	//
	// That is the pair somebody writing a script from scratch wants: not
	// everything still unmatched, which is the leftovers of however many
	// commands ago, but the answer to the command they just typed — the one
	// holding the phrase to lift out and put in an EXPECT.
	Order string
	After string

	// Unmatched is everything said that no EXPECT has matched, which is what a
	// failure of order needs instead.
	Unmatched string
}

// Fail is a step that did not hold.
func (r *Report) Fail(at Place, source, why string, heard Heard) {
	r.failed++

	r.verdict(failBadge.Render("FAIL"))

	r.sayf("  %s %s\n", console.PathStyle.Render(at.String()), source)
	r.sayf("  %s\n", why)

	if heard.Order != "" {
		r.heard(fmt.Sprintf("what it said after %s:", heard.Order), heard.After, shown)
	}

	// The whole of what is unmatched only when it holds something the answer
	// above does not, which is what an EXPECT in the wrong order looks like.
	if older := strings.TrimSuffix(heard.Unmatched, heard.After); strings.TrimSpace(older) != "" {
		r.heard("and before that, still unmatched:", older, before)
	}
}

// heard prints a stretch of what the adventure said, with its runs of blank
// lines squeezed.
//
// An adventure pads with newlines wherever its author felt like it, and twelve
// lines of a failure spent on four of those is a failure that says nothing.
// What goes in the transcript is never touched; this is the report deciding
// how to show it.
func (r *Report) heard(what, said string, most int) {
	tail := lastLines(squeeze(said), most)
	if tail == "" {
		return
	}

	r.sayf("  %s\n", console.MutedStyle.Render(what))

	for line := range strings.SplitSeq(tail, "\n") {
		r.sayf("    %s\n", console.MutedStyle.Render(line))
	}
}

// Broke is the machine itself giving up, which is a failure of the run and not
// of any one step: an unimplemented condact, or a loop the spin guard caught.
func (r *Report) Broke(err error) {
	r.failed++

	r.verdict(failBadge.Render("FAIL"))
	r.sayf("  the adventure stopped running\n")
	r.sayf("  %s\n", err)
}

// verdict closes a step with its badge: at the end of the line the step
// opened, or on one of its own if anything has been said since.
func (r *Report) verdict(text string) {
	if r.standing {
		r.standing = false

		r.sayf("%s%s\n", strings.Repeat(" ", titleGap), text)

		return
	}

	r.margin()
	r.sayf("%s\n", text)
}

// Note is something worth saying that is nobody's failure.
func (r *Report) Note(at Place, text string) {
	r.breakLine()
	r.margin()

	r.sayf("  %s %s\n", console.MutedStyle.Render(at.String()), console.MutedStyle.Render(text))
}

// Debug is what a DEBUG order asked for. It goes out plainly, with nothing in
// front of it, so that FLAG_1=23 is a line something else can read.
func (r *Report) Debug(text string) {
	r.breakLine()
	r.margin()

	r.sayf("%s\n", text)
}

// Said echoes what the adventure said, for DEBUG SAID and for --show.
func (r *Report) Said(text string) {
	r.breakLine()

	r.sayf("%s", text)
}

// Summary closes the run: what passed, and what did not if anything did.
func (r *Report) Summary() {
	r.breakLine()
	r.margin()

	r.sayf("\n")

	if r.failed == 0 && r.passed == 0 {
		r.sayf("%s no steps\n", console.WarnStyle.Render(console.RightCaret))

		return
	}

	r.sayf("%s %s passed\n", console.OkStyle.Render(console.RightCaret), steps(r.passed))

	if r.failed > 0 {
		r.sayf("%s %s failed\n", console.ErrStyle.Render(console.RightCaret), steps(r.failed))
	}
}

// Err is the verdict, which is what an exit code is made of.
func (r *Report) Err() error {
	if r.failed == 0 {
		return nil
	}

	return fmt.Errorf("%s failed of %s", steps(r.failed), steps(r.passed+r.failed))
}

// breakLine ends a step's line when something other than its badge has to be
// written.
func (r *Report) breakLine() {
	if !r.standing {
		return
	}

	r.standing = false

	r.sayf("\n")
}

func (r *Report) sayf(format string, args ...any) {
	if r.out == nil {
		return
	}

	text := fmt.Sprintf(format, args...)
	r.midLine = !strings.HasSuffix(text, "\n")

	// Nothing useful can be done about a report that cannot be written: the
	// place it would be said is the place that is failing.
	_, _ = io.WriteString(r.out, text)
}

// margin makes sure what comes next starts at column one.
//
// It is for the lines that are the report's own and have to line up. What the
// adventure said is never touched: a newline put between two of its own Prints
// would be the report rewriting the game's layout.
func (r *Report) margin() {
	if r.midLine {
		r.sayf("\n")
	}
}

// lastWidth is how wide the last line of a name is, which is where whatever
// follows it on the screen begins.
func lastWidth(name string) int {
	if at := strings.LastIndex(name, "\n"); at >= 0 {
		name = name[at+1:]
	}

	return utf8.RuneCountInString(name)
}

func steps(n int) string {
	if n == 1 {
		return "1 step"
	}

	return fmt.Sprintf("%d steps", n)
}

// squeeze turns a run of blank lines into one.
func squeeze(text string) string {
	var out strings.Builder

	blank := false

	for line := range strings.SplitSeq(text, "\n") {
		if strings.TrimSpace(line) == "" {
			if blank {
				continue
			}

			blank = true
		} else {
			blank = false
		}

		if out.Len() > 0 {
			out.WriteByte('\n')
		}

		out.WriteString(strings.TrimRight(line, " \t"))
	}

	return out.String()
}

// lastLines keeps the end of a transcript, which is the part that says where
// the game got to.
func lastLines(text string, most int) string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	if len(lines) > most {
		lines = lines[len(lines)-most:]
	}

	return strings.Join(lines, "\n")
}
