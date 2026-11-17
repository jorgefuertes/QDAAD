package media

import (
	"fmt"
)

// The Amstrad's own tape, written rather than read.
//
// A CDT is a TZX — same signature, same blocks, another extension — so what is
// different is not the container but what goes in it. A Spectrum tape is one
// block to a file and the ROM does the rest; an Amstrad tape is a firmware
// format of its own, and the firmware is what has to find its way through it:
//
//   - A file is cut into blocks of two kilobytes.
//   - Every block is written twice over: a header record saying what it is,
//     then a data record with the bytes.
//   - A record is a sync byte, then the bytes in segments of 256 with two of
//     checksum after each, and four bytes of 0xFF to close it.
//
// All of that was read off La Aventura Original's own Amstrad tape, which is
// where the numbers below come from as well. The checksum was the part worth
// checking rather than assuming: CRC-16/CCITT over the segment, started at
// 0xFFFF, complemented, and written high byte first — which reproduces the
// 0xF1D3 that tape carries on its first record.
const (
	// The sync byte that opens a record, and says which of the two it is.
	cdtHeaderSync = 0x2C
	cdtDataSync   = 0x16

	cdtBlockSize   = 2048 // a file is cut into these
	cdtSegmentSize = 256  // and a record written in these
	cdtHeaderSize  = 64   // of which a header record uses this many
	cdtNameSize    = 16
	// cdtTrailer is the four bytes of 0xFF that close a record.
	cdtTrailer = 4

	// cdtBinary is the file type for a binary: loaded where its header says
	// and entered there, which is what RUN"name" does with one.
	cdtBinary = 0x02
)

// The pulses, in T-states, as La Aventura Original's Amstrad tape has them.
// They are a measurement of a real tape and not a specification — the rips of
// Cozumel and Los Templos Sagrados give the same shape with a few T-states
// either way — so they are kept together and named rather than scattered.
const (
	cdtPilotPulse  = 2270
	cdtSync1Pulse  = 1111
	cdtSync2Pulse  = 1111
	cdtZeroPulse   = 1144
	cdtOnePulse    = 2288 // twice the zero, which is how the Amstrad writes a bit
	cdtPilotPulses = 4096
	cdtLastBits    = 8
	// The quiet after a record, in milliseconds.
	//
	// Fifteen between a header and the data it describes, which is what the
	// original leaves. Between one block and the next it has to be longer: the
	// firmware stops the motor when a record ends and starts it again for the
	// next, and fifteen milliseconds is not enough to get going and find the
	// pilot tone. Too short and the machine sails past a block and asks to
	// rewind the tape, which is exactly what it did.
	cdtRecordGap = 15
	cdtBlockGap  = 1500
	cdtFileGap   = 2000
)

// TapeFile is one file as a tape carries it: its name, its bytes, and the two
// addresses the firmware needs to put it back where it belongs.
type TapeFile struct {
	Name  string
	Data  []byte
	Load  int
	Entry int
}

// CDT writes the files given as an Amstrad tape, in the order given, which is
// the order the machine finds them in.
func CDT(files []TapeFile) ([]byte, error) {
	out := []byte(tapeSignature)
	out = append(out, 1, 20) // the version of the format, 1.20

	for _, f := range files {
		body, err := cdtFile(f)
		if err != nil {
			return nil, err
		}

		out = append(out, body...)
	}

	return out, nil
}

// cdtFile writes one file: its blocks, each as a header record and a data
// record.
func cdtFile(f TapeFile) ([]byte, error) {
	name, err := cdtName(f.Name)
	if err != nil {
		return nil, err
	}

	if len(f.Data) == 0 {
		return nil, fmt.Errorf("%s is empty and a tape has nothing to carry", f.Name)
	}

	blocks := (len(f.Data) + cdtBlockSize - 1) / cdtBlockSize

	var out []byte

	for block := range blocks {
		at := block * cdtBlockSize
		body := f.Data[at:min(at+cdtBlockSize, len(f.Data))]

		header := make([]byte, cdtHeaderSize)
		copy(header, name)
		header[16] = byte(block + 1)

		if block == blocks-1 {
			header[17] = 0xFF
		}

		header[18] = cdtBinary
		header[19], header[20] = byte(len(body)), byte(len(body)>>8)
		header[21], header[22] = byte(f.Load+at), byte((f.Load+at)>>8)

		if block == 0 {
			header[23] = 0xFF
		}

		header[24], header[25] = byte(len(f.Data)), byte(len(f.Data)>>8)
		header[26], header[27] = byte(f.Entry), byte(f.Entry>>8)

		gap := cdtBlockGap
		if block == blocks-1 {
			gap = cdtFileGap
		}

		out = append(out, cdtBlock(cdtRecord(cdtHeaderSync, header), cdtRecordGap)...)
		out = append(out, cdtBlock(cdtRecord(cdtDataSync, body), gap)...)
	}

	return out, nil
}

// cdtName is the sixteen characters a tape header carries, padded with zeros.
// The extension goes in it: a tape has no directory and the name is the whole
// of what there is to match against.
func cdtName(name string) ([]byte, error) {
	if len(name) == 0 || len(name) > cdtNameSize {
		return nil, fmt.Errorf("%q is not a name for a tape: one to %d characters",
			name, cdtNameSize)
	}

	out := make([]byte, cdtNameSize)
	copy(out, name)

	return out, nil
}

// cdtRecord lays bytes out as the firmware reads them: the sync, then segments
// of 256 with their checksum, then the four bytes that close it.
//
// A short last segment is padded rather than shortened. The firmware reads
// whole segments and takes the length from the header, so what is in the
// padding never reaches anybody.
func cdtRecord(sync byte, body []byte) []byte {
	segments := (len(body) + cdtSegmentSize - 1) / cdtSegmentSize

	out := make([]byte, 0, 1+segments*(cdtSegmentSize+2)+cdtTrailer)
	out = append(out, sync)

	for segment := range segments {
		at := segment * cdtSegmentSize

		whole := make([]byte, cdtSegmentSize)
		copy(whole, body[at:min(at+cdtSegmentSize, len(body))])

		sum := cdtChecksum(whole)

		out = append(out, whole...)
		out = append(out, byte(sum>>8), byte(sum))
	}

	for range cdtTrailer {
		out = append(out, 0xFF)
	}

	return out
}

// cdtChecksum is CRC-16/CCITT over a segment, complemented. The complement is
// what the firmware compares against, and leaving it out gives a tape that
// looks right and will not load.
func cdtChecksum(segment []byte) int {
	const polynomial = 0x1021

	crc := 0xFFFF

	for _, b := range segment {
		crc ^= int(b) << 8

		for range 8 {
			if crc&0x8000 != 0 {
				crc = (crc<<1 ^ polynomial) & 0xFFFF
			} else {
				crc = crc << 1 & 0xFFFF
			}
		}
	}

	return crc ^ 0xFFFF
}

// cdtBlock wraps a record as a turbo speed data block, which is the only kind
// of block an Amstrad tape is made of: the Spectrum's standard block has the
// ROM's own timings built into it and the Amstrad's are not those.
func cdtBlock(record []byte, pause int) []byte {
	out := []byte{0x11}

	// The order the block describes them in: the pilot pulse, the two sync
	// pulses, the two bit lengths, and how many pilot pulses there are.
	for _, word := range []int{
		cdtPilotPulse, cdtSync1Pulse, cdtSync2Pulse, cdtZeroPulse, cdtOnePulse, cdtPilotPulses,
	} {
		out = append(out, byte(word), byte(word>>8))
	}

	out = append(out, cdtLastBits, byte(pause), byte(pause>>8))
	out = append(out, byte(len(record)), byte(len(record)>>8), byte(len(record)>>16))

	return append(out, record...)
}
