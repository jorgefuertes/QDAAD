package media

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// The DSK container, as CPCEMU defined it and everything since has followed.
// It describes the disk the way the drive saw it: a track at a time, each with
// its own list of sectors, because the machines of the era formatted their
// disks however suited their loader.
//
// Two variants exist. The original gives every track the same size, padding the
// short ones; the extended one carries a table of sizes instead, so a track can
// be shorter or missing altogether.
//
// Only the first eight characters identify the original variant. The rest of
// that opening line is free text the writing tool chooses, and the tools
// disagree: CPCEMU wrote "MV - CPCEMU Disk-File", while the Amstrad PCW disks
// of Los Templos Sagrados say "MV - CPC format Disk Image (DU54)". Everything
// after it is laid out the same, so eight characters is what we ask for.
const (
	dskHeaderSize   = 256
	dskTrackHeader  = 256
	dskSignature    = "MV - CPC"
	dskSignatureExt = "EXTENDED CPC DSK"
)

// Fields of the disk information block.
const (
	dskCreator    = 0x22
	dskNumTracks  = 0x30
	dskNumSides   = 0x31
	dskTrackSize  = 0x32 // the original variant only
	dskSizeTable  = 0x34 // the extended variant only, one byte per track
	dskCreatorLen = 14
)

// Fields of a track information block.
const (
	trackSignature   = "Track-Info"
	trackNumSectors  = 0x15
	trackSectorList  = 0x18
	trackSectorEntry = 8
)

// Fields of one entry in the sector list.
const (
	sectorID     = 2
	sectorSize   = 3
	sectorLength = 6 // the extended variant only
)

// DSK reads an Amstrad CPC, Amstrad PCW or Spectrum +3 disk image.
//
// These disks come two ways and the container says nothing about which. Some
// were formatted for the game's own loader — the Amstrad disk of La Aventura
// Original holds 1 KiB sectors from track 1 onwards, the Spectrum one rotates
// its sector numbering from track to track — and carry no directory at all.
// Others, the PCW ones among them, are ordinary CP/M volumes. So the directory
// is looked for, and a caller that finds no files still has the payload to
// search through.
type DSK struct {
	data     []byte
	extended bool
	tracks   int
	sides    int
}

func NewDSK(data []byte) (*DSK, error) {
	if len(data) < dskHeaderSize {
		return nil, fmt.Errorf("too short to hold a disk information block: %d bytes", len(data))
	}

	d := &DSK{
		data:     data,
		extended: bytes.HasPrefix(data, []byte(dskSignatureExt)),
		tracks:   int(data[dskNumTracks]),
		sides:    int(data[dskNumSides]),
	}

	if d.tracks == 0 || d.sides == 0 {
		return nil, fmt.Errorf("the image declares %d tracks and %d sides", d.tracks, d.sides)
	}

	return d, nil
}

func (d *DSK) Format() string {
	variant := "CPCEMU"
	if d.extended {
		variant = "extended CPCEMU"
	}

	// A tool whose opening line runs past the creator field overwrites it, so
	// what is there is only worth printing if it reads as a name.
	creator := strings.Map(func(r rune) rune {
		if r < ' ' || r > '~' {
			return -1
		}

		return r
	}, string(d.data[dskCreator:dskCreator+dskCreatorLen]))

	creator = strings.TrimSpace(creator)
	if creator == "" || strings.Contains(creator, "Disk-Info") {
		return fmt.Sprintf("%s disk image, %d tracks, %d side(s)", variant, d.tracks, d.sides)
	}

	return fmt.Sprintf("%s disk image, %d tracks, %d side(s), written by %s",
		variant, d.tracks, d.sides, creator)
}

// Files returns what the CP/M directory names, when there is one.
//
// These disks come both ways and the container does not say which: some carry
// an ordinary AMSDOS filesystem and others were formatted for the game's own
// loader and hold nothing but a run of sectors. Reading the directory is the
// test. When it names nothing the error says so, rather than leaving a caller
// to read an empty list as an empty disk.
func (d *DSK) Files() ([]File, error) {
	files := cpmFiles(d.Payload(), d.trackBytes())
	if len(files) > 0 {
		return files, nil
	}

	return nil, fmt.Errorf("%s: formatted for its own loader, with no filesystem to walk", d.Format())
}

// trackBytes is how much one track adds to the payload, which is what turns a
// count of reserved tracks into an offset into it.
func (d *DSK) trackBytes() int {
	offset := dskHeaderSize

	for i := range d.tracks * d.sides {
		size := d.trackSize(i)
		if size == 0 {
			continue
		}

		if offset+dskTrackHeader > len(d.data) {
			break
		}

		return len(d.trackPayload(d.data[offset:min(offset+size, len(d.data))]))
	}

	return 0
}

// Payload returns the sector data of the whole disk, tracks in order and the
// sectors of each in the order of their numbering rather than the order they
// were written in. Unformatted tracks contribute nothing.
//
// The numbering matters. The Spectrum disk starts each track at a different
// sector so the drive need not wait for the disk to come round, and reading the
// sectors in the order the track lists them would interleave the result.
func (d *DSK) Payload() []byte {
	out := make([]byte, 0, len(d.data))
	offset := dskHeaderSize

	for i := range d.tracks * d.sides {
		size := d.trackSize(i)
		if size == 0 {
			// An unformatted track takes up no room in the image at all.
			continue
		}

		if offset+dskTrackHeader > len(d.data) {
			break
		}

		out = append(out, d.trackPayload(d.data[offset:min(offset+size, len(d.data))])...)
		offset += size
	}

	return out
}

// trackSize gives the room a track takes in the image, which is where the two
// variants differ: a fixed size for all of them, or one byte per track.
func (d *DSK) trackSize(track int) int {
	if !d.extended {
		return int(d.data[dskTrackSize]) | int(d.data[dskTrackSize+1])<<8
	}

	at := dskSizeTable + track
	if at >= len(d.data) {
		return 0
	}

	return int(d.data[at]) * dskTrackHeader
}

// trackPayload pulls the sectors out of one track. The data follows the track
// information block in the order the sector list gives, each sector as long as
// its own entry says, so the list has to be walked to know where each starts.
func (d *DSK) trackPayload(track []byte) []byte {
	if !bytes.HasPrefix(track, []byte(trackSignature)) {
		return nil
	}

	count := int(track[trackNumSectors])

	type sector struct {
		id   int
		data []byte
	}

	sectors := make([]sector, 0, count)
	at := dskTrackHeader

	for i := range count {
		entry := trackSectorList + i*trackSectorEntry
		if entry+trackSectorEntry > dskTrackHeader {
			break
		}

		// The original variant has no length field, so the size code stands in:
		// it is the power of two, over 128 bytes, that the sector holds.
		length := 128 << track[entry+sectorSize]
		if d.extended {
			if declared := int(track[entry+sectorLength]) | int(track[entry+sectorLength+1])<<8; declared > 0 {
				length = declared
			}
		}

		if at+length > len(track) {
			break
		}

		sectors = append(sectors, sector{id: int(track[entry+sectorID]), data: track[at : at+length]})
		at += length
	}

	sort.SliceStable(sectors, func(i, j int) bool { return sectors[i].id < sectors[j].id })

	out := make([]byte, 0, at)
	for _, s := range sectors {
		out = append(out, s.data...)
	}

	return out
}
