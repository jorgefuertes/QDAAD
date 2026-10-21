package runtime_test

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
	"github.com/stretchr/testify/require"
)

// Every target must answer in both languages, and answer with something. A
// table that names a file which is not embedded would otherwise only be found
// the day someone tries to build for that machine.
func TestEveryTargetHasItsInterpreter(t *testing.T) {
	targets := runtime.Targets()

	// Fifteen classic machines, and three desktop players when the build asked
	// for them. Both counts are named so that a target lost from the table is
	// noticed here rather than the day someone builds for it.
	want := 15
	if desktopBuild {
		want = 18
	}

	require.Len(t, targets, want)

	for _, target := range targets {
		for _, lang := range []language.Language{
			language.LanguageEnglish, language.LanguageSpanish,
		} {
			name, body, err := runtime.Interpreter(target, lang)
			require.NoError(t, err, "%s in %s", target, lang)
			require.NotEmpty(t, name)
			require.NotEmpty(t, body, "%s in %s is empty", target, lang)
		}
	}
}

// The companions have to be readable too, and the name they come back with is
// the one the machine expects rather than the path inside the package.
func TestCompanions(t *testing.T) {
	got, err := runtime.Companions(runtime.C64)
	require.NoError(t, err)
	require.Contains(t, got, "apart1.prg")
	require.NotEmpty(t, got["apart1.prg"])

	got, err = runtime.Companions(runtime.MSX1)
	require.NoError(t, err)
	require.Len(t, got, 2, "the MSX needs its graphics database and its loader")

	// The Spectrum's graphics stub is needed whether or not the adventure has
	// any graphics: the interpreter reads its own pointers out of the last
	// bytes of it. A build without it does not start.
	got, err = runtime.Companions(runtime.ZX48)
	require.NoError(t, err)
	require.Contains(t, got, "DAAD.SDG")
	require.NotEmpty(t, got["DAAD.SDG"])
}

// Only the two 68000 machines have a player of the newer kind.
func TestADP(t *testing.T) {
	for _, target := range []runtime.Target{runtime.ST, runtime.Amiga} {
		_, body, err := runtime.ADP(target)
		require.NoError(t, err, "%s", target)
		require.NotEmpty(t, body)
	}

	_, _, err := runtime.ADP(runtime.ZX48)
	require.ErrorContains(t, err, "no ADP player")
}

func TestAnUnknownTargetSaysSo(t *testing.T) {
	_, _, err := runtime.Interpreter("bbc-micro", language.LanguageEnglish)
	require.ErrorContains(t, err, "no runtime is built in")
}

// Nothing embedded should be unreachable through the table: a file nobody names
// is dead weight in the binary and probably a copy someone forgot to wire up.
func TestNothingIsEmbeddedWithoutBeingNamed(t *testing.T) {
	all, err := runtime.All()
	require.NoError(t, err)

	named := map[string]bool{}

	for _, target := range runtime.Targets() {
		for _, lang := range []language.Language{
			language.LanguageEnglish, language.LanguageSpanish,
		} {
			if _, _, err := runtime.Interpreter(target, lang); err == nil {
				named[nameOf(t, target, lang)] = true
			}
		}
	}

	require.NotEmpty(t, all)
	require.GreaterOrEqual(t, len(all), len(named))
}

func nameOf(t *testing.T, target runtime.Target, lang language.Language) string {
	t.Helper()

	name, _, err := runtime.Interpreter(target, lang)
	require.NoError(t, err)

	return name
}
