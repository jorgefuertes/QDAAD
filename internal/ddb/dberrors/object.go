package dberrors

import "errors"

var (
	ErrObjectNotFound                = errors.New("object not found")
	ErrObjectStoreIsFull             = errors.New("object store is full, no more available object IDs")
	ErrObjectDuplicatedLabel         = errors.New("label already used in objects")
	ErrObjectDuplicatedNameAdjective = errors.New("duplicated name and adjective combination")
)
