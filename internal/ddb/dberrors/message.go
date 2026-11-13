package dberrors

import "errors"

var (
	ErrInvalidMessageKind = errors.New("invalid message kind")
	ErrMessageStoreIsFull = errors.New("message store is full, no more available message IDs")
	ErrXMessageTooLong    = errors.New("XMessage content is too long, max 511 bytes")
)
