package ddb

import qderror "github.com/jorgefuertes/QDAAD/internal/ddb/dberrors"

const (
	MAX_LOCATION_ID ID = 251
	LOC_NOT_CREATED ID = 252
	LOC_WORN        ID = 253
	LOC_CARRIED     ID = 254
	LOC_HERE        ID = 255
)

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
	for id := ID(0); id <= MAX_LOCATION_ID; id++ {
		if _, err := ls.Get(id); err != nil {
			return id, nil
		}
	}

	return 0, qderror.ErrLocationStoreIsFull
}
