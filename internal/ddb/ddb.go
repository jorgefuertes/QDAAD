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
	data          Data              `valid:"required"`
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
