package qderror

import "errors"

var (
	ErrInvalidWordKind             = errors.New("invalid word kind")
	ErrWordDirectionAndConvertible = errors.New("word cannot be both direction and convertible")
	ErrWordWithDuplicatedLabel     = errors.New("label already used in words")
	ErrWordConnectionIDsExhausted  = errors.New("no more available word IDs for direction words")
	ErrWordConvertibleIDsExhausted = errors.New("no more available word IDs for convertible words")
	ErrWordStoreIsFull             = errors.New("word store is full, no more available word IDs")
	ErrWordNotFound                = errors.New("word not found")
)
