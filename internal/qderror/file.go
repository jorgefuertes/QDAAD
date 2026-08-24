package qderror

import "errors"

var (
	ErrMissingDDBFile      = errors.New("missing .DDB file argument")
	ErrFailedToOpenDDBFile = errors.New("failed to open .DDB file")
)
