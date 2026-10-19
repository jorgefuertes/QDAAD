package decompiler

import "strings"

// The character set itself lives in internal/charset, which the compiler reads
// with as well: one table, walked in both directions, so that what is written
// here is what gets read back. What stays in this file is the one piece of
// layout the /TOK section needs and nothing else does.

// tokenSource renders a token for the /TOK section, where the sources of the
// time wrote spaces as the null word character.
//
// Careful: PCDAAD and jDAAD expand that character back to a space at run time,
// but msx2daad and NextDAAD do not. A token whose spaces matter is therefore
// not portable across every interpreter.
func tokenSource(token string, nullWord byte) string {
	return strings.ReplaceAll(token, " ", string(rune(nullWord)))
}
