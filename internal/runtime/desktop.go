//go:build desktop

// The desktop players, left out unless asked for with -tags desktop.
//
// They are 3.6 MB against the 566 KB of the fifteen classic machines put
// together, and they are the part of this most likely to be replaced by a
// runtime of our own. Building without the tag leaves them out of the binary
// entirely, which is why the assets are embedded here and not with the rest.
//
// What they are is ADP, a separate project by José Luis Cebrián at
// https://github.com/jlcebrian/ADP. DAAD READY ships built binaries of it, and
// what is embedded here is **BETA0.3**, the newest the kit carries. ADP moves at
// its own pace, so refreshing this is a job somebody has to do by hand.
//
// The file names in the kit do not say which is newest, and reading them the
// obvious way gets it backwards: on macOS the bundle called ADP.app is 0.2d and
// the newer one is ADP-EXPERIMENTAL.app, and on Windows player.exe is 0.2d
// while adp-player.exe is BETA0.3. The versions here were read out of the
// binaries, not off the names.
package runtime

import (
	"embed"
	"io/fs"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
)

const (
	Windows Target = "windows"
	Linux   Target = "linux"
	MacOS   Target = "macos"
)

//go:embed assets/windows assets/linux assets/macos
var desktop embed.FS

func init() {
	sources = append(sources, fs.FS(desktop))

	// One player serves both languages: ADP reads its messages from the
	// database rather than carrying them, as the DOS interpreter does.
	both := func(path string) map[language.Language]string {
		return map[language.Language]string{
			language.LanguageEnglish: path,
			language.LanguageSpanish: path,
		}
	}

	runtimes[Windows] = Runtime{interpreter: both("windows/adp-player.exe")}

	// Stripped of its debug information before being embedded, which took it
	// from 1.8 MB to 399 KB. Nothing else about it was touched.
	runtimes[Linux] = Runtime{interpreter: both("linux/player")}

	// macOS is the one target whose runtime is a tree and not a file: the
	// binary is no use without the bundle around it.
	//
	// And it is **arm64 only**. The universal build in the kit is the older
	// 0.2d, so an Intel Mac is not served by this and would need that one.
	runtimes[MacOS] = Runtime{
		interpreter: both("macos/ADP.app/Contents/MacOS/ADP"),
		companions: []string{
			"macos/ADP.app/Contents/Info.plist",
			"macos/ADP.app/Contents/Resources/ADP.icns",
		},
	}
}
