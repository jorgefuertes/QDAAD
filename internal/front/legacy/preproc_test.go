package legacy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// preprocess scans a snippet and runs it through the preprocessor.
func preprocess(t *testing.T, source string) ([]Token, Symbols, error) {
	t.Helper()

	var lines []Line
	for i, text := range strings.Split(source, "\n") {
		lines = append(lines, Line{File: "t.sce", Num: i + 1, Text: text})
	}

	tokens, err := Scan(lines)
	require.NoError(t, err)

	return Preprocess(tokens)
}

// texts names what survived, so that a test can say what it expects to be left.
func texts(tokens []Token) []string {
	out := make([]string, 0, len(tokens))

	for _, t := range tokens {
		if t.Kind != EOF {
			out = append(out, t.Describe())
		}
	}

	return out
}

func TestPreprocessorResolvesDefines(t *testing.T) {
	_, symbols, err := preprocess(t, `
#define fDark 0
#define fScore 30
#define fNext "fScore + 1"
`)
	require.NoError(t, err)

	require.Equal(t, Symbols{"fDark": 0, "fScore": 30, "fNext": 31}, symbols)
}

func TestPreprocessorRefusesADoubleDefine(t *testing.T) {
	_, _, err := preprocess(t, "#define A 1\n#define A 2\n")
	require.ErrorContains(t, err, "defined twice")
}

// The branch not taken leaves nothing behind, and the directives themselves
// never reach the parser.
func TestPreprocessorDropsTheDeadBranch(t *testing.T) {
	got, _, err := preprocess(t, `
#define PC 1
#ifdef "PC"
YES
#else
NO
#endif
`)
	require.NoError(t, err)
	require.Equal(t, []string{"YES"}, texts(got))
}

func TestPreprocessorTakesTheElseWhenTheSymbolIsMissing(t *testing.T) {
	got, _, err := preprocess(t, `
#ifdef "NOTHERE"
YES
#else
NO
#endif
`)
	require.NoError(t, err)
	require.Equal(t, []string{"NO"}, texts(got))
}

func TestPreprocessorNegatesWithIfndef(t *testing.T) {
	got, _, err := preprocess(t, "#ifndef \"NOTHERE\"\nYES\n#endif\n")
	require.NoError(t, err)
	require.Equal(t, []string{"YES"}, texts(got))
}

// A conditional inside a branch that is being skipped must not end the skip,
// neither with its #endif nor with its #else.
func TestPreprocessorCountsNestedConditionals(t *testing.T) {
	got, _, err := preprocess(t, `
#ifdef "NOTHERE"
  #ifdef "ALSONOT"
  DEEP
  #else
  DEEPELSE
  #endif
  SHALLOW
#else
KEEP
#endif
AFTER
`)
	require.NoError(t, err)
	require.Equal(t, []string{"KEEP", "AFTER"}, texts(got))
}

func TestPreprocessorReportsUnbalancedConditionals(t *testing.T) {
	_, _, err := preprocess(t, "#define A 1\n#ifdef \"A\"\nYES\n")
	require.ErrorContains(t, err, "#endif missing")

	_, _, err = preprocess(t, "#endif\n")
	require.ErrorContains(t, err, "#endif without #ifdef")

	_, _, err = preprocess(t, "#else\n")
	require.ErrorContains(t, err, "#else without #ifdef")
}

// The operand has to be quoted. It is a rule of the language: the quotes are
// what mark an expression rather than a piece of text.
func TestPreprocessorWantsTheConditionalOperandQuoted(t *testing.T) {
	_, _, err := preprocess(t, "#ifdef PC\n#endif\n")
	require.ErrorContains(t, err, "between quotes")
}

// This is the split that is expensive to get wrong: five directives are grammar
// rather than preprocessing, because they emit bytes into a condact block, and
// they have to come out untouched.
func TestPreprocessorPassesTheGrammarDirectivesThrough(t *testing.T) {
	got, _, err := preprocess(t, `
#db 65
#dw 513
#hex "AABB"
#incbin "x.bin"
#userptr 3
`)
	require.NoError(t, err)

	require.Equal(t, []string{
		"#db", "65", "#dw", "513", "#hex", `"AABB"`,
		"#incbin", `"x.bin"`, "#userptr", "3",
	}, texts(got))
}

func TestExpressions(t *testing.T) {
	symbols := Symbols{"TEN": 10, "TWO": 2}

	for _, tc := range []struct {
		text string
		want int
	}{
		{"1 + 2 * 3", 7},
		{"(1 + 2) * 3", 9},
		{"TEN / TWO", 5},
		{"TEN % 3", 1},
		{"-TEN + 1", -9},
		{"TEN * (TWO - 4)", -20},
	} {
		got, err := evaluate(Token{Kind: String, Text: tc.text}, symbols)
		require.NoError(t, err, tc.text)
		require.Equal(t, tc.want, got, tc.text)
	}
}

func TestExpressionsRejectWhatTheyCannotWorkOut(t *testing.T) {
	for _, tc := range []struct{ text, says string }{
		{"NOPE + 1", "is not defined"},
		{"1 / 0", "division by zero"},
		{"(1 + 2", "never closed"},
		{"1 2", "is left over"},
		{"1 +", "value is missing"},
	} {
		_, err := evaluate(Token{Kind: String, Text: tc.text}, Symbols{})
		require.ErrorContains(t, err, tc.says, "evaluating %q", tc.text)
	}
}
