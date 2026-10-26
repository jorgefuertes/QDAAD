package ddbfile

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Opening a database that says what it is, rather than one that has to be
// worked out. See internal/pack/qrun.go for why the container exists and what
// is in it.
//
// Everything readerAt deduces is read here instead, so there is no trying, no
// candidate addresses and no arrangement that happens to fit. A file that is
// not one of these says so and stops.
const (
	qrunMagic   = "QDB1"
	qrunVersion = 2
	qrunHeader  = 10
)

// OpenQRun reads a database in qrun's own container.
func OpenQRun(data []byte) (*Reader, error) {
	if len(data) < qrunHeader {
		return nil, errors.New("too short to hold a header")
	}

	if string(data[:4]) != qrunMagic {
		return nil, errors.New("not a qrun database: the first four bytes are not " + qrunMagic)
	}

	if v := data[4]; v != qrunVersion {
		return nil, fmt.Errorf("this is version %d of the container and we read version %d",
			v, qrunVersion)
	}

	const bigEndian = 1

	body := data[qrunHeader:]

	// The header size is the one thing still read from the database rather
	// than from the container, because it is the adventure's own: version 1
	// has twelve pointers and the later ones have thirteen.
	size := headerSizeV2
	if body[0] == 1 {
		size = headerSizeV1
	}

	r := &Reader{
		data:              body,
		bigEndian:         data[5]&bigEndian != 0,
		headerSize:        size,
		base:              int(binary.LittleEndian.Uint16(data[6:8])),
		unreadableProcess: -1,
		// How many tokens there are is the one thing the database itself
		// cannot say, so here it is said. Below zero means nobody told us and
		// the table has to be worked out the old way.
		tokenCount: int(data[8]),
	}

	if !r.headerAgrees() {
		return nil, errors.New("the header does not describe itself")
	}

	if err := r.readTokens(); err != nil {
		return nil, fmt.Errorf("the compression table: %w", err)
	}

	if err := r.readVocabulary(); err != nil {
		return nil, fmt.Errorf("the vocabulary: %w", err)
	}

	return r, nil
}
