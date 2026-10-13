package ddb

import (
	"os"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
)

type (
	ID   uint8
	ID16 uint16
)

func (id ID) Int() int {
	return int(id)
}

func (id ID16) Int() int {
	return int(id)
}

type DDB struct {
	Version       uint8             `valid:"required,min=1,max=3"`
	MachineID     arch.Machine      `valid:"required"`
	Language      language.Language `valid:"required"`
	VideoMode     byte              // MSX2
	ExternVectors []uint32          // 16b in 8bit machines, 32b in >=16bit machines
	// NullWord is the character /CTL declares, which ends up in byte 2 of the
	// binary. It is a relic, but it is the source's to say and the back end's
	// to write, so it travels here rather than being worked out again.
	NullWord byte
	data     Data `valid:"required"`
}

type Data struct {
	labels    Labels
	words     Words
	locations Locations
	conns     Conns
	objects   Objects
	processes Processes
	messages  Messages
}

func New() *DDB {
	return &DDB{
		data: Data{
			labels:    NewLabelStore(),
			words:     NewWordStore(),
			locations: NewLocationStore(),
			conns:     NewConnStore(),
			objects:   NewObjectStore(),
			processes: NewProcessStore(),
			messages:  NewMessageStore(),
		},
	}
}

// The stores, reachable so that something outside the package can fill a
// database in. They come back as pointers because building means appending, and
// appending to a copy would not reach the database.
//
// New is what creates them, so a DDB made any other way has none: these are not
// lazily built on purpose, to keep a half-made database from looking whole.
func (d *DDB) Words() *Words         { return &d.data.words }
func (d *DDB) Locations() *Locations { return &d.data.locations }
func (d *DDB) Conns() *Conns         { return &d.data.conns }
func (d *DDB) Objects() *Objects     { return &d.data.objects }
func (d *DDB) Processes() *Processes { return &d.data.processes }
func (d *DDB) Messages() *Messages   { return &d.data.messages }
func (d *DDB) Labels() *Labels       { return &d.data.labels }

func Open(filePath string) (*DDB, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = f.Close()
	}()

	d := new(DDB)

	return d, nil
}
