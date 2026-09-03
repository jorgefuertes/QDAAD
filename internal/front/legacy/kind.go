package legacy

// Kind is the class of a token. It is deliberately coarse: which section or
// which directive is a separate field, because the parser mostly asks "is this
// a section marker?" to close a list and only sometimes "which one?".
//
// There is no kind for a newline or for a comment. The scanner eats both, and
// that is the whole point of working on tokens instead of lines.
type Kind uint8

const (
	EOF       Kind = iota
	Section        // /VOC, /PRO…  which one is in SectionID
	Directive      // #define, #db…  which one is in DirectiveID
	ListEntry      // /123 or /NAME: numbered carries Num, named carries Text
	String         // "…", already stripped of its quotes
	Number         // -?[0-9]+, signed
	Label          // $name
	Ident          // a word, a condact, a symbol, a type: the context decides
	Wildcard       // _ and * are the same token
	Entry          // >
	Indirect       // @
)

func (k Kind) String() string {
	switch k {
	case EOF:
		return "end of file"
	case Section:
		return "section marker"
	case Directive:
		return "directive"
	case ListEntry:
		return "list entry"
	case String:
		return "string"
	case Number:
		return "number"
	case Label:
		return "label"
	case Ident:
		return "identifier"
	case Wildcard:
		return "wildcard"
	case Entry:
		return "entry sign"
	case Indirect:
		return "indirection"
	}

	return "unknown"
}
