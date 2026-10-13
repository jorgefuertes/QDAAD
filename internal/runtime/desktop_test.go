//go:build desktop

package runtime_test

import (
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
	"github.com/stretchr/testify/require"
)

// macOS is the one target whose runtime is a tree: the names come back with the
// bundle path on them, because that is where the files have to be written.
func TestMacOSComesBackAsABundle(t *testing.T) {
	name, body, err := runtime.Interpreter(runtime.MacOS, language.LanguageEnglish)
	require.NoError(t, err)
	require.Equal(t, "ADP.app/Contents/MacOS/ADP", name)
	require.NotEmpty(t, body)

	companions, err := runtime.Companions(runtime.MacOS)
	require.NoError(t, err)

	// Without these the bundle is not one, and macOS will not run it.
	require.Contains(t, companions, "ADP.app/Contents/Info.plist")
	require.Contains(t, companions, "ADP.app/Contents/Resources/ADP.icns")
}

// Everywhere else the target folder comes off and a bare name is left, which is
// the same rule applied to a path one level deep.
func TestTheOtherDesktopPlayersAreOneFile(t *testing.T) {
	for target, want := range map[runtime.Target]string{
		runtime.Windows: "adp-player.exe",
		runtime.Linux:   "player",
	} {
		name, body, err := runtime.Interpreter(target, language.LanguageSpanish)
		require.NoError(t, err, "%s", target)
		require.Equal(t, want, name)
		require.NotEmpty(t, body)
	}
}
