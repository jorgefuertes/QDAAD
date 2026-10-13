package main

import (
	"fmt"
	"strings"
)

// The two front ends disagree about when text stops being text, and this is
// where the disagreement is bridged rather than reported.
//
// DAAD READY's resolves the character set in the front end: by the time its
// JSON is written, "á" has become byte 0x15 and "#n" has become a carriage
// return. Ours keeps the text as the author wrote it, in UTF-8, and leaves the
// mapping to the emitter — the same argument ddb.NormalizeWord makes for
// accents, that the low charset is a property of the machine being built for
// and not of the adventure.
//
// Neither is wrong and both produce the same game. So to compare them, ours is
// put through the same conversion before the two texts are held against each
// other. The conversion below is UJSONExport.pas:157-265 followed line for
// line, including the order the replacements happen in, which matters: the
// per-character pass writes "#g" and "#t" markers that the escape pass then
// turns into control codes, so an author who wrote "#g" by hand is treated the
// same way.

// daadChars is the Spanish half of the low character set: the codes DAAD keeps
// its accented letters at, where an ASCII machine keeps control codes.
var daadChars = map[rune]rune{
	'ª': 0x10, '¡': 0x11, '¿': 0x12, '«': 0x13, '»': 0x14,
	'á': 0x15, 'é': 0x16, 'í': 0x17, 'ó': 0x18, 'ú': 0x19,
	'ñ': 0x1A, 'Ñ': 0x1B, 'ç': 0x1C, 'Ç': 0x1D,
	'ü': 0x1E, 'Ü': 0x1F,
	// ß is the one that lands in the low charset without a graphic escape.
	'ß': 0x7F,
}

// graphicChars are the letters that do not fit in the low charset and travel
// as a graphic: "#g", the slot in the second character set, and "#t" to switch
// back. Every accented letter no Spanish adventure needed is here.
var graphicChars = map[rune]rune{
	// Uppercase vowels with an acute accent.
	'Á': 0x7B, 'É': 0x7C, 'Í': 0x7D, 'Ó': 0x7E, 'Ú': 0x7F,
	// Lowercase vowels with the other accents.
	'à': 0x10, 'ã': 0x11, 'ä': 0x12, 'â': 0x13,
	'è': 0x14, 'ë': 0x15, 'ê': 0x16,
	'ì': 0x17, 'ï': 0x18, 'î': 0x19,
	'ò': 0x1A, 'õ': 0x1B, 'ö': 0x1C, 'ô': 0x1D,
	'ù': 0x1E, 'û': 0x1F,
	// The same, uppercase.
	'À': 0x20, 'Ã': 0x21, 'Ä': 0x22, 'Â': 0x23,
	'È': 0x24, 'Ë': 0x25, 'Ê': 0x26,
	'Ì': 0x27, 'Ï': 0x28, 'Î': 0x29,
	'Ò': 0x2A, 'Õ': 0x2B, 'Ö': 0x2C, 'Ô': 0x2D,
	'Ù': 0x2E, 'Û': 0x2F,
	// The letters the northern languages need.
	'ý': 0x3A, 'Ý': 0x3B, 'þ': 0x3C, 'Þ': 0x3D, 'å': 0x3E, 'Å': 0x3F,
	'ð': 0x5B, 'Ð': 0x5C, 'ø': 0x5D, 'Ø': 0x5E,
}

// escapes are the sequences an author writes with a hash. They are applied
// after the characters, over the whole string, which is what lets the graphic
// markers the pass above wrote be resolved by the same rule.
var escapes = []struct{ from, to string }{
	{"#e", "#g\x60#t"}, // the euro sign, which is a graphic like the rest
	{"#g", "\x0E"},
	{"#t", "\x0F"},
	{"#b", "\x0B"},
	{"#s", " "},
	{"#f", "\x7F"},
	{"#k", "\x0C"},
	{"#n", "\x0D"},
	{"#r", "\x0D"},
	{`\n`, "\x0D"},
	{`\r`, "\x0D"},
}

// daadEncode puts a text through the conversion DAAD READY's front end does,
// so that ours can be compared against it.
func daadEncode(s string) string {
	var b strings.Builder

	for _, r := range s {
		if low, spanish := daadChars[r]; spanish {
			b.WriteRune(low)

			continue
		}

		if slot, graphic := graphicChars[r]; graphic {
			fmt.Fprintf(&b, "#g%c#t", slot)

			continue
		}

		b.WriteRune(r)
	}

	out := b.String()

	for _, e := range escapes {
		out = strings.ReplaceAll(out, e.from, e.to)
	}

	// #A to #P are the sixteen codes above the escapes, written as letters so
	// that a source can carry them through a text editor.
	for c := 'A'; c <= 'P'; c++ {
		out = strings.ReplaceAll(out, "#"+string(c), string(rune(c-'A'+0x10)))
	}

	return out
}
