package pack

// Making a PCW release start on its own.
//
// A PCW disk of 1989 does not boot. The player puts a CP/M system disk in,
// waits for the A> prompt, swaps disks and types DAAD — which is what DAAD
// READY's PCW.BAT asks of its emulator with a second -A, and what the Cozumel
// and Los Templos Sagrados disks of 1989 expected, both of them DATA format
// with no reserved track to boot from.
//
// Boot asks for the other thing: the system on the disk and the adventure
// coming up by itself. Three pieces make that happen.
//
//   - The reserved track, Amstrad's loader, copied whole off the CP/M disk
//     that DAAD READY ships. It reads a name out of the directory.
//   - That name, J14SCPM3.EMS, which is CP/M Plus itself, as an ordinary file.
//   - Something to run DAAD.COM once CP/M is up.
//
// The third is the only one with any choice in it. CP/M Plus looks for
// PROFILE.SUB on the system drive at cold boot and runs SUBMIT on it, so the
// usual answer is to ship PROFILE.SUB and Digital Research's SUBMIT.COM. The
// CP/M disk here has no SUBMIT.COM on it and there is no Spanish one to be
// found, so this writes its own: sixteen bytes that ignore what they were
// called with and chain to DAAD.COM. It is the same hook, with a program of
// ours in it instead of somebody else's binary.

import (
	"errors"
	"fmt"

	"github.com/jorgefuertes/QDAAD/internal/media"
	"github.com/jorgefuertes/QDAAD/internal/runtime"
)

const (
	// pcwSystem is the file Amstrad's loader opens, and the name is the
	// loader's and not ours: it is written into the reserved track.
	pcwSystem = "J14SCPM3.EMS"
	// pcwProfile is what the CCP looks for at cold boot, and pcwSubmit is what
	// it runs on it.
	pcwProfile = "PROFILE.SUB"
	pcwSubmit  = "SUBMIT.COM"
)

// pcwChain is our SUBMIT.COM: copy a command line to the default DMA buffer and
// ask the BDOS to chain to it.
//
// Function 47 takes the line from 0x0080 as a null-terminated string and
// returns to nobody — the program it names replaces this one — so there is no
// stack to keep and no error to report. What CP/M would normally do here is
// read PROFILE.SUB and feed its lines to the CCP one at a time; this does not
// read it at all, because a release has exactly one line to run and it is known
// at build time.
//
//	ld hl,cmd / ld de,0x0080 / ld bc,6 / ldir / ld c,47 / jp 5 / "DAAD",0
var pcwChain = []byte{
	0x21, 0x10, 0x01, // ld hl,0x0110
	0x11, 0x80, 0x00, // ld de,0x0080
	0x01, 0x06, 0x00, // ld bc,6
	0xED, 0xB0, // ldir
	0x0E, 0x2F, // ld c,47
	0xC3, 0x05, 0x00, // jp 5
	'D', 'A', 'A', 'D', 0x00, 0x00, // at 0x0110, and the count above says so
}

// pcwBootFiles is what a self-starting disk carries on top of the release.
func pcwBootFiles() ([]File, []byte, error) {
	beside, err := runtime.Companions(runtime.PCW)
	if err != nil {
		return nil, nil, err
	}

	boot, ok := beside["CPMBOOT.BIN"]
	if !ok {
		return nil, nil, errors.New("the PCW boot track is missing")
	}

	system, ok := beside[pcwSystem]
	if !ok {
		return nil, nil, fmt.Errorf("%s, which is CP/M itself, is missing", pcwSystem)
	}

	// A record of a .SUB is a counted string in 128 bytes. One line is all
	// this needs and the CCP reads the last record first, so order is moot.
	profile := make([]byte, 128)
	profile[0] = byte(len("DAAD"))
	copy(profile[1:], "DAAD")

	return []File{{
		Name: pcwSystem,
		Body: system,
		Note: "CP/M Plus, which the loader in track 0 opens by this name",
	}, {
		Name: pcwSubmit,
		Body: pcwChain,
		Note: "ours, and all it does is chain to DAAD.COM",
	}, {
		Name: pcwProfile,
		Body: profile,
		Note: "what CP/M runs at cold boot, which is how the adventure comes up",
	}}, boot, nil
}

// pcwBootDisk writes a PCW disk that starts the adventure on its own.
func pcwBootDisk(release *Release, opts Options) ([]byte, error) {
	extra, boot, err := pcwBootFiles()
	if err != nil {
		return nil, err
	}

	files := make([]media.File, 0, len(release.Files())+len(extra))
	for _, f := range release.Files() {
		files = append(files, media.File{Name: f.Name, Data: f.Body})
	}

	room := 0

	for _, f := range extra {
		files = append(files, media.File{Name: f.Name, Data: f.Body})
		room += len(f.Body)
	}

	opts.Note.say(NoticeSaid,
		"the disk starts on its own: CP/M is on it and PROFILE.SUB runs DAAD, "+
			"which costs %d KiB and hands out Amstrad's CP/M with the adventure",
		room/1024)

	return media.BootDisk(boot, files)
}

// pcwRoom is what is left of the disk for illustrations once everything that
// has to be on it is.
//
// A disk that boots is a track shorter and forty kilobytes heavier, and both
// come off the drawings, because the drawings are the only thing here that can
// be left behind: an adventure without its database is not an adventure.
func pcwRoom(files []File, opts Options) (int, error) {
	room := media.PCWRoom()

	if opts.Boot {
		extra, _, err := pcwBootFiles()
		if err != nil {
			return 0, err
		}

		for _, f := range extra {
			room -= media.DataCost(len(f.Body))
		}
	}

	for _, f := range files {
		room -= media.DataCost(len(f.Body))
	}

	// And the archive's own table, which is there whether anything is in it.
	return room - pcwFirstPicture, nil
}
