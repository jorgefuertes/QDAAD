package ddb

import "github.com/jorgefuertes/QDAAD/internal/qderror"

const MaxLocationID ID = 251

type Location struct {
	ID          ID
	LabelID     ID16
	Description string
}

type Locations []Location

func NewLocationStore() Locations {
	return Locations{}
}

func (ls Locations) Get(id ID) (*Location, error) {
	for i, l := range ls {
		if l.ID == id {
			return &ls[i], nil
		}
	}

	return nil, qderror.ErrLocationNotFound
}

func (ls Locations) GetByLabelID(labelID ID16) (*Location, error) {
	for i, l := range ls {
		if l.LabelID == labelID {
			return &ls[i], nil
		}
	}

	return nil, qderror.ErrLocationNotFound
}

func (ls *Locations) Add(labelID ID16, description string) (ID, error) {
	id, err := ls.getNextID()
	if err != nil {
		return 0, err
	}

	*ls = append(*ls, Location{
		ID:          id,
		LabelID:     labelID,
		Description: description,
	})

	return id, nil
}

func (ls Locations) getNextID() (ID, error) {
	for id := ID(0); id < MaxLocationID; id++ {
		if _, err := ls.Get(id); err != nil {
			return id, nil
		}
	}

	return 0, qderror.ErrLocationStoreIsFull
}
