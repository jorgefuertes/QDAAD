package back

import "strings"

// Text compression, which is a dictionary of 128 short strings and nothing
// cleverer: a byte with bit 7 set is not a character, it is "the token numbered
// by the other seven bits, expanded here".
//
// The table is not worked out from the texts. The reference carries one per
// language and keeps the ones that earn their place in this particular
// adventure, which is a different thing and worth saying plainly: a token that
// no text uses costs a byte and saves none, and one used once costs more than
// it saves. So the table is pruned, and only then are the texts rewritten.
//
// Two things decide whether this is byte-for-byte the reference's: which
// tokens survive the pruning, and the number each survivor ends up with. The
// second is why the texts cannot be rewritten during the pruning — a token's
// number is its place in the *final* table, which is not known until the
// pruning is over.

const (
	// firstToken is the byte that stands for the first entry of the table.
	//
	// It is 127 and not 128 because entry zero is a placeholder: every table
	// the reference carries opens with a byte no text contains, so nothing
	// ever refers to it and the real tokens start at 128. Our own decompiler
	// reads it the same way round — a placeholder, then 128 tokens.
	firstToken = 127
	// maxTokens is the placeholder and the 128 a byte can name after it.
	maxTokens = 129
)

// reference is the byte that stands for a token. It is one byte and never
// more, which is the point of the whole exercise, so it cannot be built by
// converting a number to a string: above 127 that would be two bytes of UTF-8.
func reference(number int) string {
	return string([]byte{byte(firstToken + number)})
}

// compress picks a table for a language and prunes it against these texts.
//
// The tables that get compressed are the locations, the user messages and the
// system messages, in that order and not the object texts: see
// getCompressableTables at drb.php:310-321, where "advanced" names those three
// and the extended messages we do not write yet.
//
// The order matters and is not an accident of how this is written. The pruning
// measures each token against texts the tokens before it have already been
// substituted into, so a token is judged on what is left after the earlier ones
// have taken what they wanted.
// The tables are rewritten in place, so what a caller holds afterwards is the
// compressed text.
func compress(language string, tables ...[]string) (tokens []string, savings int) {
	table, known := compressionTokens[language]
	if !known {
		table = compressionTokens["EN"]
	}

	tokens, savings = worthKeeping(table, tables...)

	// Now that every token has its final number, the texts can be written.
	for i, token := range tokens {
		for _, bodies := range tables {
			for j, body := range bodies {
				bodies[j] = strings.ReplaceAll(body, token, reference(i))
			}
		}
	}

	return tokens, savings
}

// worthKeeping is the pruning, and it follows the reference's arithmetic
// rather than a better one: a token's first use is counted as costing a byte
// and every use after it as saving its length less the one byte the reference
// takes. See drb.php:160-196.
//
// The first token is never dropped whatever it is worth. In every table the
// reference carries it is a single byte that no text contains, so it earns
// nothing and is kept so that the numbering starts where the interpreter
// expects.
func worthKeeping(table []string, tables ...[]string) ([]string, int) {
	// A copy, because the measuring substitutes as it goes and the real texts
	// are not rewritten until every token has its final number.
	var texts []string

	for _, bodies := range tables {
		texts = append(texts, bodies...)
	}

	kept := make([]string, 0, maxTokens)
	total := 0

	for i, token := range table {
		if len(kept) == maxTokens {
			break
		}

		if i == 0 {
			kept = append(kept, token)

			continue
		}

		// What this token is worth, measured against the texts as the tokens
		// before it have left them, and taking what it finds as it goes.
		earned := 0
		seen := false

		for j, text := range texts {
			uses := strings.Count(text, token)
			if uses == 0 {
				continue
			}

			for range uses {
				if seen {
					earned += len(token) - 1
				} else {
					earned, seen = -1, true
				}
			}

			texts[j] = strings.ReplaceAll(text, token, reference(i))
		}

		if earned > 0 {
			kept = append(kept, token)
			total += earned
		}
	}

	return kept, total
}

// write lays the table out: every token's bytes in turn, with the top bit set
// on the last of each, which is how the interpreter knows where one ends.
func (l *linker) writeTokens(tokens []string) {
	for _, token := range tokens {
		for i := range len(token) {
			b := token[i]
			if i == len(token)-1 {
				b |= 0x80
			}

			l.writeByte(b)
		}
	}
}
