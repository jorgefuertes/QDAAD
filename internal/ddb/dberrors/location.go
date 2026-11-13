package dberrors

import "errors"

var (
	ErrLocationNotFound      = errors.New("location not found")
	ErrLocationStoreIsFull   = errors.New("location store is full")
	ErrLocationIDOutOfRange  = errors.New("location ID out of range")
	ErrLocationAlreadyExists = errors.New("location already exists")
)
