// Package runtime carries the interpreters a compiled adventure needs to run,
// built into the binary so that emitting a tape or a disk takes nothing but
// this program.
//
// These are the version 3 interpreters DAAD READY ships, not the ones of the
// 1991 release. The two generations are not interchangeable — DS48IS.P3F is of
// 1991 and DS48IS3.BIN is version 3, and the 3 in the name is the difference —
// and only the later ones read a database this compiler writes. The originals
// stay in work/ as reference material; it is one of them the vector engine was
// read out of.
//
// DAAD was placed in the public domain by Andrés Samudio, who founded Aventuras
// AD. The interpreters are distributed as binaries: their sources were never
// released.
package runtime

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strings"

	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
)

// The classic machines are listed one by one rather than embedding the whole
// tree, so that the desktop players can be left out by a build tag and actually
// save their bytes. An //go:embed of the directory would carry them regardless.
//
//go:embed assets/zx48 assets/zx128 assets/zxplus3 assets/zxnext assets/zxuno
//go:embed assets/zxesxdos assets/cpc assets/c64 assets/cp4 assets/msx1
//go:embed assets/msx2 assets/pcw assets/st assets/amiga assets/pc
var classic embed.FS

// sources is where read looks, in order. The desktop build adds its own.
var sources = []fs.FS{classic}

// Target names one machine to emit for.
//
// It is not arch.Machine, and it cannot be: six of these are the same machine.
// A 48K Spectrum, a 128K, a +3, a Next, a ZX-Uno and an esxDOS all write ZX in
// byte 1 of the database, and each one needs a different interpreter. What
// tells them apart is the target, which is what the build asks for.
type Target string

const (
	ZX48     Target = "zx48"
	ZX128    Target = "zx128"
	ZXPlus3  Target = "zxplus3"
	ZXNext   Target = "zxnext"
	ZXUno    Target = "zxuno"
	ZXesxDOS Target = "zxesxdos"
	CPC      Target = "cpc"
	C64      Target = "c64"
	CP4      Target = "cp4"
	MSX1     Target = "msx1"
	MSX2     Target = "msx2"
	PCW      Target = "pcw"
	ST       Target = "st"
	Amiga    Target = "amiga"
	PC       Target = "pc"
)

// Runtime is everything a target needs to have a game built for it.
type Runtime struct {
	// Interpreter is the file the machine runs. One per language, because the
	// system messages the interpreter carries are written into it.
	interpreter map[language.Language]string
	// Companions travel beside the interpreter: loaders, fonts, the graphics
	// stub. They do not depend on the language.
	companions []string
	// ADP is the newer player, where one exists. It is a separate program and
	// not a drop-in replacement, so a caller has to ask for it.
	adp string
}

// The table is written out rather than worked out from the names. There is no
// pattern to deduce: DS48IS3.BIN, DCPCISF3.BIN, dc64isf3.prg, DMSXISF3.BIN and
// DPCWISF3.BIN all name the Spanish interpreter of their machine, and no rule
// generates the five.
var runtimes = map[Target]Runtime{
	ZX48: {interpreter: map[language.Language]string{
		language.LanguageEnglish: "zx48/DS48IE3.BIN",
		language.LanguageSpanish: "zx48/DS48IS3.BIN",
	}},
	ZX128: {interpreter: map[language.Language]string{
		language.LanguageEnglish: "zx128/DS128KE3.BIN",
		language.LanguageSpanish: "zx128/DS128KS3.BIN",
	}},
	ZXPlus3: {interpreter: map[language.Language]string{
		language.LanguageEnglish: "zxplus3/DSP3E3.BIN",
		language.LanguageSpanish: "zxplus3/DSP3S3.BIN",
	}},
	ZXNext: {interpreter: map[language.Language]string{
		language.LanguageEnglish: "zxnext/DSNEXTE3.BIN",
		language.LanguageSpanish: "zxnext/DSNEXTS3.BIN",
	}},
	ZXUno: {interpreter: map[language.Language]string{
		language.LanguageEnglish: "zxuno/DSZXUNOE3.BIN",
		language.LanguageSpanish: "zxuno/DSZXUNOS3.BIN",
	}},
	ZXesxDOS: {interpreter: map[language.Language]string{
		language.LanguageEnglish: "zxesxdos/DSESXDOSE3.BIN",
		language.LanguageSpanish: "zxesxdos/DSESXDOSS3.BIN",
	}},
	CPC: {
		interpreter: map[language.Language]string{
			language.LanguageEnglish: "cpc/DCPCIEF3.BIN",
			language.LanguageSpanish: "cpc/DCPCISF3.BIN",
		},
		companions: []string{"cpc/BLANK.GRA"},
	},
	C64: {
		interpreter: map[language.Language]string{
			language.LanguageEnglish: "c64/dc64ief3.prg",
			language.LanguageSpanish: "c64/dc64isf3.prg",
		},
		companions: []string{"c64/apart1.prg"},
	},
	CP4: {
		interpreter: map[language.Language]string{
			language.LanguageEnglish: "cp4/dcp4ief3.prg",
			language.LanguageSpanish: "cp4/dcp4isf3.prg",
		},
		companions: []string{"cp4/apart1.prg"},
	},
	MSX1: {
		interpreter: map[language.Language]string{
			language.LanguageEnglish: "msx1/DMSXIEF3.BIN",
			language.LanguageSpanish: "msx1/DMSXISF3.BIN",
		},
		companions: []string{"msx1/DAAD.MDG", "msx1/msxdaad.com"},
	},
	MSX2: {
		interpreter: map[language.Language]string{
			language.LanguageEnglish: "msx2/msx2daad_3.0.1_EN_SC8.com",
			language.LanguageSpanish: "msx2/msx2daad_3.0.1_ES_SC8.com",
		},
		companions: []string{"msx2/GLUE.IM8"},
	},
	PCW: {interpreter: map[language.Language]string{
		language.LanguageEnglish: "pcw/DPCWIEF3.BIN",
		language.LanguageSpanish: "pcw/DPCWISF3.BIN",
	}},
	ST: {
		interpreter: map[language.Language]string{
			language.LanguageEnglish: "st/DSTEI3.PRG",
			language.LanguageSpanish: "st/DSTSI3.PRG",
		},
		adp: "st/adp.prg",
	},
	Amiga: {
		interpreter: map[language.Language]string{
			language.LanguageEnglish: "amiga/DAAMIGAEI3",
			language.LanguageSpanish: "amiga/DAAMIGASI3",
		},
		adp: "amiga/ADP.EXE",
	},
	// The DOS interpreter is the same whatever the language: it reads its
	// messages from the database rather than carrying them.
	PC: {interpreter: map[language.Language]string{
		language.LanguageEnglish: "pc/PCDAAD.EXE",
		language.LanguageSpanish: "pc/PCDAAD.EXE",
	}},
}

// Interpreter returns the interpreter for a target and a language.
func Interpreter(t Target, lang language.Language) (name string, body []byte, err error) {
	r, known := runtimes[t]
	if !known {
		return "", nil, fmt.Errorf("no runtime is built in for %q", t)
	}

	path, ok := r.interpreter[lang]
	if !ok {
		return "", nil, fmt.Errorf("%s has no interpreter in %s", t, lang)
	}

	return read(path)
}

// Companions returns the files that travel beside the interpreter: loaders,
// fonts, the graphics stub. A target with none returns nothing and no error.
func Companions(t Target) (map[string][]byte, error) {
	r, known := runtimes[t]
	if !known {
		return nil, fmt.Errorf("no runtime is built in for %q", t)
	}

	out := make(map[string][]byte, len(r.companions))

	for _, path := range r.companions {
		name, body, err := read(path)
		if err != nil {
			return nil, err
		}

		out[name] = body
	}

	return out, nil
}

// ADP returns the newer player, for the targets that have one. It is a program
// of its own rather than a drop-in interpreter, so asking for it is deliberate.
func ADP(t Target) (name string, body []byte, err error) {
	r, known := runtimes[t]
	if !known {
		return "", nil, fmt.Errorf("no runtime is built in for %q", t)
	}

	if r.adp == "" {
		return "", nil, fmt.Errorf("%s has no ADP player", t)
	}

	return read(r.adp)
}

// Targets lists what can be emitted for, in a fixed order so that output does
// not shuffle between runs.
func Targets() []Target {
	out := make([]Target, 0, len(runtimes))
	for t := range runtimes {
		out = append(out, t)
	}

	slices.Sort(out)

	return out
}

// read pulls one file out and hands back the name it should be written under.
//
// That name is the path with the target's own folder taken off, which for
// almost everything leaves the bare file name — and for macOS leaves the whole
// bundle path, ADP.app/Contents/MacOS/ADP, which is what has to be written. One
// rule rather than a special case: a target may hand back a tree.
func read(path string) (string, []byte, error) {
	for _, src := range sources {
		body, err := fs.ReadFile(src, "assets/"+path)
		if err != nil {
			continue
		}

		return withoutTarget(path), body, nil
	}

	return "", nil, fmt.Errorf("the built-in %s is missing", path)
}

func withoutTarget(path string) string {
	if at := strings.IndexByte(path, '/'); at >= 0 {
		return path[at+1:]
	}

	return path
}

// All walks everything built in, which is what a test uses to check that the
// table and the files agree.
func All() ([]string, error) {
	var found []string

	for _, src := range sources {
		err := fs.WalkDir(src, "assets", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				found = append(found, path[len("assets/"):])
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(found)

	return found, nil
}
