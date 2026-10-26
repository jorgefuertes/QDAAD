package pack

import (
	"encoding/binary"
	"fmt"
)

// A container for qrun, which exists so that nothing has to be guessed.
//
// Reading a database of 1989 means working four things out that nobody wrote
// down: the address it was laid at, which way round its words go, whether the
// file is the database or merely contains one, and where the compression table
// stops. The reader deduces all four, and has to — a tape from 1989 is not
// going to start telling us.
//
// None of that applies to a database we write ten seconds earlier. So this
// says the three that are properties of the layout, and says them in front:
//
//	"QDB1"   four bytes, so that a file can be recognised rather than tried
//	version  one byte, this format's own and not the adventure's
//	flags    one byte, bit 0 set when the words are big-endian
//	base     two bytes, the address the database was laid at
//
// Eight bytes. What follows is the database exactly as the linker wrote it,
// untouched — this wraps, it does not re-encode, so every other tool still
// sees the same bytes and compare-drb still means what it meant.
const (
	QRunMagic   = "QDB1"
	QRunVersion = 1
	QRunHeader  = 8
)

// QRunBigEndian is the one flag there is so far.
const QRunBigEndian = 1

// QRun wraps a database for qrun.
func QRun(database []byte, base int, bigEndian bool) ([]byte, error) {
	if base < 0 || base > 0xFFFF {
		return nil, fmt.Errorf("a base address is two bytes and %d is not", base)
	}

	out := make([]byte, 0, QRunHeader+len(database))
	out = append(out, QRunMagic...)
	out = append(out, QRunVersion)

	var flags byte
	if bigEndian {
		flags |= QRunBigEndian
	}

	out = append(out, flags)
	out = binary.LittleEndian.AppendUint16(out, uint16(base))

	return append(out, database...), nil
}
