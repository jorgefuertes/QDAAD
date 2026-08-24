package qderror

import "errors"

var (
	ErrRecordNotFound     = errors.New("record not found")
	ErrCannotCreateWithID = errors.New("cannot create record with ID, must be zero")
)
