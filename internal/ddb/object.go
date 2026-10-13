package ddb

import (
	"math"

	qderror "github.com/jorgefuertes/QDAAD/internal/ddb/dberrors"
)

// MAX_WEIGHT is the most an object can weigh: the weight shares its byte with
// the container and wearable flags, so only six bits are left.
const MAX_WEIGHT ID = 63

type Object struct {
	ID      ID
	LabelID ID16
	// Description is the text /OTX gives for this object, kept here rather than
	// in the message store for the same reason a location keeps its own: the
	// text belongs to the thing it describes.
	Description  string
	Name         ID
	Adjective    ID
	Weight       uint8 `valid:"required,min=0,max=63"`
	Container    bool
	Wearable     bool
	Flags        [16]bool
	InitLocation ID
}

type Objects []Object

func NewObjectStore() Objects {
	return Objects{}
}

// Len is how many objects there are.
func (ost Objects) Len() int { return len(ost) }

func (ost Objects) Get(id ID) (*Object, error) {
	for i, obj := range ost {
		if obj.ID == id {
			return &ost[i], nil
		}
	}

	return nil, qderror.ErrObjectNotFound
}

func (ost Objects) GetByLabelID(labelID ID16) (*Object, error) {
	for i, obj := range ost {
		if obj.LabelID == labelID {
			return &ost[i], nil
		}
	}

	return nil, qderror.ErrObjectNotFound
}

func (ost Objects) GetByName(name ID) (*Object, error) {
	for i, obj := range ost {
		if obj.Name == name {
			return &ost[i], nil
		}
	}

	return nil, qderror.ErrObjectNotFound
}

func (ost Objects) GetByNameAdjective(name, adjective ID) (*Object, error) {
	for i, obj := range ost {
		if obj.Name == name && obj.Adjective == adjective {
			return &ost[i], nil
		}
	}

	return nil, qderror.ErrObjectNotFound
}

func (ost Objects) getNextID() (ID, error) {
	for i := 0; i < math.MaxUint8; i++ {
		if _, err := ost.Get(ID(i)); err != nil {
			return ID(i), nil
		}
	}

	return 0, qderror.ErrObjectStoreIsFull
}

// NewLegacy adds an object with the number the source gives it.
//
// A legacy source numbers everything itself and names nothing, so neither the
// duplicate-label check nor the duplicate noun-adjective one that Add makes
// applies: every object carries label zero, and a real adventure may well give
// two objects the same pair of words.
// There is no range to check: an object number is a byte and the type already
// caps it at 255, which is the most a database can hold.
func (ost *Objects) NewLegacy(obj Object) error {
	if _, err := ost.Get(obj.ID); err == nil {
		return qderror.ErrObjectAlreadyExists
	}

	*ost = append(*ost, obj)

	return nil
}

func (ost *Objects) Add(obj Object) error {
	if obj.ID > 0 {
		return qderror.ErrCannotCreateWithID
	}

	id, err := ost.getNextID()
	if err != nil {
		return err
	}

	obj.ID = id

	if _, err = ost.GetByLabelID(obj.LabelID); err == nil {
		return qderror.ErrObjectDuplicatedLabel
	}

	if _, err = ost.GetByNameAdjective(obj.Name, obj.Adjective); err == nil {
		return qderror.ErrObjectDuplicatedNameAdjective
	}

	*ost = append(*ost, obj)

	return nil
}
