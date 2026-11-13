package legacy

import "fmt"

// Token is one lexical unit of the source.
//
// It carries where it came from — the file the author wrote and the line within
// it, not a position in some resolved stream — so that an error points at the
// place a person can go and fix. That is why the origin travels in the token
// rather than in a table on the side.
type Token struct {
	Kind      Kind
	Section   SectionID   // meaningful when Kind is Section
	Directive DirectiveID // meaningful when Kind is Directive
	// Text is already cooked: a string without its quotes, a named list entry
	// without its slash, a label without its dollar.
	Text string
	// Num is the value of a number, or of a numbered list entry.
	Num       int
	File      string
	Line, Col int
}

// Where names the position for an error message.
func (t Token) Where() string {
	return fmt.Sprintf("%s:%d:%d", t.File, t.Line, t.Col)
}

// Describe names the token the way an error message wants it.
func (t Token) Describe() string {
	switch t.Kind {
	case EOF:
		return "end of file"
	case Section:
		return t.Section.String()
	case Directive:
		return t.Directive.String()
	case Number:
		return fmt.Sprintf("%d", t.Num)
	case String:
		return fmt.Sprintf("%q", t.Text)
	case ListEntry:
		if t.Text == "" {
			return fmt.Sprintf("/%d", t.Num)
		}

		return "/" + t.Text
	case Label:
		return "$" + t.Text
	}

	return t.Text
}
