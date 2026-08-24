package ddb

type Location struct {
	ID          ID
	LabelID     ID16
	Description string
}

type Locations []Location
