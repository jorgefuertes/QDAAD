package qderror

import "errors"

var (
	ErrInvalidMessageKind = errors.New("invalid message kind")
	ErrMessageStoreIsFull = errors.New("message store is full, no more available message IDs")
)
