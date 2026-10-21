package back

// The compression tables, one per language, as DAAD READY carries them.
//
// They are not derived from the texts being compiled and never were: the
// reference has these built in and picks by language, then throws away the
// ones that would not earn their place in this particular adventure. So a
// token here is a guess about a language that somebody made once by reading
// the adventures of the time: "\x20que\x20" is in the Spanish table because
// Spanish prose is full of it.
//
// A token may hold a code that is not ASCII. The last of the Spanish ones
// ends in 0x15, the a-acute of the DAAD character set, which is why
// compression happens after a text is encoded and before it is obfuscated:
// by then a text is written in the same bytes a token is.
//
// Taken from drb.php:172-176, where they are hex strings.
var compressionTokens = map[string][]string{
	// Spanish.
	"ES": {
		"\x00", " que ", "a de ", "o de ", " una ", " del ", "s de ", " de l", " con ", "ente ", " por ",
		" est\x15", "tiene", "s un ", "ante ", " para", " las ", "entra", "n el ", "e de ", "a la ",
		"erior", "ci\x18n ", "ando ", "iente", " el ", " la ", " de ", " con", " en ", "los ", "ado ",
		" se ", "esta", " un ", "las ", "enta", " des", " al ", "ada ", "as ", "es ", "os ", " y ",
		"ado", "te ", "ada", "la ", "ent", "res", "que", "an ", "o p", "rec", "ido", "s, ", "ant", "ina",
		"ida", "lar", "ero", "mpl", "a ", "o ", "er", "es", "or", "ar", "al", "en", "as", "os", "e ",
		"an", "el", "on", "in", "ci", "un", ". ", "co", "re", "di", ", ", "ur", "tr", "de", "su", "ab",
		"ol", "am", "st", "cu", "s ", "ac", "il", "gr", "ad", "te", "y ", "im", "to", "ue", "pi", "gu",
		"ch", "ca", "la", "n ", "ro", "ri", "lo", "mi", "l ", "ti", "ob", "me", "si", "pe", " n", "tu",
		"at", "fi", "do", "em", "ay", "\".", "ll",
	},
	// English.
	"EN": {
		"\x00", " the ", " you ", " are ", "ing ", " to ", " and", " is ", "You ", "and ", "The ",
		"n't ", " of ", " you", "ing", "ed ", " a ", " op", "ith", "out", "ent", " to", " in", "all",
		" th", " it", "ter", "ave", " be", "ver", "her", "and", "ear", "You", " on", "en ", "ose", "no",
		"ic", "ap", " b", "gh", "  ", "ad", "is", " c", "ir", "ay", "ur", "un", "oo", " d", "lo", "ro",
		"ac", "se", "ri", "li", "ti", "om", "bl", "ck", "I ", "ed", "ee", " f", "ha", "pe", "e ", "t ",
		"in", "s ", "th", ", ", "er", "d ", "on", "to", "an", "ar", "en", "ou", "or", "st", ". ", "ow",
		"le", "at", "al", "re", "y ", "ch", "am", "el", " w", "as", "es", "it", " s", "ll", "do", "op",
		"sh", "me", "he", "bo", "hi", "ca", "pl", "il", "cl", " a", "of", " h", "tt", "mo", "ke", "ve",
		"so", "e.", "d.", "t.", "vi", "ly", "id", "sc", " p", "em", "r ",
	},
	// Portuguese.
	"PT": {
		"\x00", "stava ", " para ", "\x0e\x11\x0fo ", " uma ", "do ", "da ", " de ", "ia ", " no ",
		" a ", " o ", " que ", " muito ", "as ", " lugar", "inha ", "ao ", "ante ", "est", "ar ",
		"ra um ", " e ", "na ", ". Ao ", " co", "precis", "par", "os ", "o. ", "enho ", "...\x0d",
		"       ", "sso ", "se ", "em ", "ess", "fazer ", "porta ", "o.\x0d", "ment", "rto ", "ra ",
		"idade", "gar", "carre", "io ", "ender ", "de ", "res", "... ", "local ", "ava ", "nte", "o, ",
		"ro ", "a.\x0d", "ntra", "uma ", "con", "er ", "algu", "um ", ". O ", "inh", " sempre", " por",
		"mais", "me ", "sobre", "era", "are", "sua ", "ica", " tempo ", " al", "des", "ei ", "s. ",
		"orma", "omo ", "Esta", "_.\x0d", "tra", "os.", "\x0b- : qu", "ferr", "depois", "o e", "pal",
		"and", "que ", "uar", "nheci", "com", "orre", "pass", "tin", "e, ", ". A", "iss", "\x0b- ", "a ",
		"o ", "ou", "el", "er", "e ", "al", "oc\x0e\x16\x0f ", "it", "or", "an", "ar", "en", "in", "os",
		"ci", "am ", ".\x0d", ". ", "is", ", ", "es", "on", "ca", "ir", "av", "qu",
	},
	// German.
	"DE": {
		"\x00", " eine ", " der ", " die ", " das ", " ist ", " von ", " ein ", " und", "und ", "Der ",
		"Die ", "Das ", " ung", "lein", "ert ", " du ", " zu ", " ein", " ver", "st ", "mit", "von",
		"hen", "aus", "ent", " zu", "all", " de", "en ", "abe", "ist", " be", "ver", " du", "Du ", "her",
		"und", "ing", "sch", "Du", "ch", "ie", "no", "ic", "ab", " b", "ei", "  ", "ll", " c", "ir",
		"er", "ur", "un", " d", "lo", "ro", "ar", "se", "ri", "li", "ti", "om", "ss", "ck", "I ", "ed",
		"eh", " f", "ha", "pe", "e ", "t ", "in", "s ", "de", ", ", "er", "d ", "zu", "an", "ar", "en",
		"au", "od", "st", ". ", "ho", "le", "at", "al", "re", "y ", "ch", "am", "el", " w", "as", "es",
		"it", " s", "ll", "ma", "op", "me", "he", "bo", "hi", "ka", "ue", "oe", "ae", " a", " h", "tt",
		"mo", "ke", "vo", "so", "e.", "d.", "t.", "vi", "mm", "sc", " p", "em",
	},
	// French.
	"FR": {
		"\x7f", "Je ne ", "porte ", " de ", "Je ", "x pas ", "faire ", "te ", " dans ", " _.\x0d",
		"J'ai ", "ne ", " pour ", "mainte", "er ", "enlev", "peu", "er.\x0d", "a rien", "cela.",
		"\x0dQue ", " plein", "puyez ", " pas", "d\x16j\x0e\x10\x0f", "e.\x0d", "nant ?", "le ",
		"touche", "sur ", "r\x16essa", ".\x0d", "ensui", "lez ", "ou ", "er.", "m\x0e\x14\x0f", "n'ai r",
		"\x0dEt ", " trop ", "lieu. ", "Il n'y", "ontinu", "est", "re", "isque", "moi :\x0d", "rou",
		"asset", "ossi", "ien", "s ", "fichi", "er", " _.", "ur", "om", "cela", "voi", "suis", "port",
		"nant", "\x0e\x16\x0ft", "ai", "Le ", "@ ", "ect", "ez ", "E D", "se", "au ", "\x0e\x14\x0f",
		"on", " - ", "eu", "ou", "mp", "et", "ap", "pa", "n'", "al", "oi", "us", ". ", "pr", "..", "or",
		" d", " c", "ci", ": ", "me", " V", "d.", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f",
		"\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f",
		"\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f", "\x7f",
		"\x7f", "\x7f", "\x7f",
	},
}
