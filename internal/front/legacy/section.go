package legacy

// SectionID names one of the source's sections.
//
// The order they must appear in is the order of these constants, which is what
// the parser checks against: the compiler of reference fixes it and gives no
// end markers, so a section runs until the next one starts.
type SectionID uint8

const (
	SecCTL SectionID = iota
	SecVOC
	SecSTX
	SecMTX
	SecOTX
	SecLTX
	SecCON
	SecOBJ
	SecPRO
	SecEND

	// SecTOK is not part of version 3. QUnDAAD writes it as a record of the
	// compression table the original database carried, and the front end skips
	// it whole: the table is the back end's to work out from the texts it ends
	// up with. It is recognised only so that we know where it ends.
	SecTOK
)

var sectionNames = map[string]SectionID{
	"/CTL": SecCTL,
	"/VOC": SecVOC,
	"/STX": SecSTX,
	"/MTX": SecMTX,
	"/OTX": SecOTX,
	"/LTX": SecLTX,
	"/CON": SecCON,
	"/OBJ": SecOBJ,
	"/PRO": SecPRO,
	"/END": SecEND,
	"/TOK": SecTOK,
}

var sectionCanonical = [...]string{
	SecCTL: "/CTL", SecVOC: "/VOC", SecSTX: "/STX", SecMTX: "/MTX",
	SecOTX: "/OTX", SecLTX: "/LTX", SecCON: "/CON", SecOBJ: "/OBJ",
	SecPRO: "/PRO", SecEND: "/END", SecTOK: "/TOK",
}

func (s SectionID) String() string {
	if int(s) < len(sectionCanonical) {
		return sectionCanonical[s]
	}

	return "/?"
}

// lookupSection matches a whole marker, never a prefix. The difference is not
// academic: "/CTLX" is a named list entry, not the /CTL section followed by an
// X, because the longest match wins.
func lookupSection(text string) (SectionID, bool) {
	id, found := sectionNames[text]

	return id, found
}
