package dberrors

import "errors"

var (
	ErrProcessStoreIsFull  = errors.New("process store is full, no more available process IDs")
	ErrProcessIDOutOfRange = errors.New("process ID out of range")
	ErrProcessNoProcess    = errors.New("no process to add the entry to")
	ErrProcessNoEntry      = errors.New("no entry to add the condact to")
	ErrInvalidOpcode       = errors.New("invalid condact opcode")
	ErrCondactParamCount   = errors.New("wrong number of parameters for the condact")
)
