package dberrors

import "errors"

var (
	ErrLocationNotFound    = errors.New("location not found")
	ErrLocationStoreIsFull = errors.New("location store is full")
)
