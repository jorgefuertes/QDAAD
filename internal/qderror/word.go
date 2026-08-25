package qderror

import "errors"

var (
	ErrInvalidWordKind               = errors.New("invalid word kind")
	ErrWordWithDuplicatedLabel       = errors.New("label already used in words")
	ErrWordConnectionIDsExhausted    = errors.New("no more available word IDs for direction words")
	ErrWordConvertibleIDsExhausted   = errors.New("no more available word IDs for convertible words")
	ErrWordProperNounIDsExhausted    = errors.New("no more available word IDs for proper nouns")
	ErrWordNonPronominalIDsExhausted = errors.New("no more available word IDs for non-pronominal verbs")
	ErrWordGeneralIDsExhausted       = errors.New("no more available word IDs for general words")
	ErrWordNotFound                  = errors.New("word not found")
)
