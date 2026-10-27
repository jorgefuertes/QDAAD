package pack

import (
	"fmt"
	"strings"
)

// What a Spectrum release is made of, which depends on the Spectrum.
//
// A 48K is a tape machine and its release is one stream: the loader, the
// interpreter, the database and the graphics, glued together so that the BASIC
// at the front can move RAMTOP out of the way before anything lands above it.
// That is ZX48 and it is one file.
//
// A +3 has a disk, so its release is the files loose: a BASIC loader, the
// interpreter, the database and the graphics block, each one its own file and
// each named what the loader looks for. The machine does the gluing at load
// time instead of the packager doing it beforehand.
//
// The two are the same adventure and a different release, which is why the
// subtarget is not a detail here.

// ZXRelease gathers what a Spectrum build needs, for the Spectrum it is built
// for.
//
// subtarget is the one the compiler was given: empty or 48k for a tape machine,
// plus3 for the one with a disk.
func ZXRelease(database []byte, subtarget string, opts Options) (*Release, error) {
	switch strings.ToLower(subtarget) {
	case "", "48k":
		tape, err := ZX48(database, opts)
		if err != nil {
			return nil, err
		}

		// One file, because a tape is one file. Asking for this release as a
		// directory puts the tape in a directory, which is what somebody
		// wanting to keep a build beside its notes is after.
		return NewRelease(File{
			Name: opts.Name + ".tap",
			Body: tape,
			Note: "48K tape: BASIC loader, interpreter, database and graphics in one stream",
		}), nil

	case "plus3":
		return nil, fmt.Errorf("the +3 release needs the DISK loader, " +
			"which is not among the interpreters we carry")

	default:
		return nil, fmt.Errorf("%q is not a Spectrum this packages yet: it is 48k or plus3",
			subtarget)
	}
}
