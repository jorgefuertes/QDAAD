package ddb

import (
	"math"

	"github.com/jorgefuertes/QDAAD/internal/qderror"
)

type Object struct {
	ID           ID
	LabelID      ID16
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
