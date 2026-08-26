package ddb

import (
	"slices"

	qderror "github.com/jorgefuertes/QDAAD/internal/ddb/dberrors"
)

const MAX_MESSSAGE ID = 254
const MAX_MESSSAGE16 ID16 = 65534

type MessageKind uint8

const (
	SystemMessage MessageKind = iota
	UserMessage
	XMessage
)

type Message struct {
	ID      ID16
	LabelID ID
	Content string
}

type Messages map[MessageKind][]Message

func NewMessageStore() Messages {
	return Messages{
		SystemMessage: make([]Message, 0),
		UserMessage:   make([]Message, 0),
		XMessage:      make([]Message, 0),
	}
}

// AddMessage appends a message to its table and numbers it. Each kind has its
// own numbering, consecutive from 0, as the DDB indexes every text table by an
// ordinal of its own.
func (ms *Messages) AddMessage(kind MessageKind, labelID ID, content string) error {
	messages, exists := (*ms)[kind]
	if !exists {
		return qderror.ErrInvalidMessageKind
	}

	switch kind {
	case SystemMessage, UserMessage:
		// The DDB has a limit of 255 messages per table, numbered 0 to 254
		if len(messages) > MAX_MESSSAGE.Int() {
			return qderror.ErrMessageStoreIsFull
		}
	case XMessage:
		// The XMessage table has no limit (16bit) but the XMB file has a 64KiB limit
		if len(messages) > MAX_MESSSAGE16.Int() {
			return qderror.ErrMessageStoreIsFull
		}

		// Eeach message has a limit of 511 bytes
		if len([]byte(content)) > 511 {
			return qderror.ErrXMessageTooLong
		}
	}

	(*ms)[kind] = append(messages, Message{
		ID:      ID16(len(messages)),
		LabelID: labelID,
		Content: content,
	})

	return nil
}

// GetMessage returns a copy: a message is immutable once defined, so there is
// no way to reach into the store through it.
func (ms Messages) GetMessage(kind MessageKind, id ID16) (string, bool) {
	messages, ok := ms[kind]
	if !ok {
		return "", false
	}

	for _, message := range messages {
		if message.ID == id {
			return message.Content, true
		}
	}

	return "", false
}

// GetMessageByLabelID returns a copy: a message is immutable once defined, so there is no way to reach into the store through it.
func (ms Messages) GetMessageByLabelID(kind MessageKind, labelID ID) (string, bool) {
	messages, ok := ms[kind]
	if !ok {
		return "", false
	}

	for _, message := range messages {
		if message.LabelID == labelID {
			return message.Content, true
		}
	}

	return "", false
}

// GetAllMessages returns a copy of the whole table, in insertion order, so
// that writing to what comes back cannot reach the store.
//
// A shallow clone is enough because Message holds nothing but values, and
// strings are immutable in Go. Adding a slice or a map field to it would
// silently break that.
func (ms Messages) GetAllMessages(kind MessageKind) ([]Message, bool) {
	messages, ok := ms[kind]
	if !ok {
		return nil, false
	}

	return slices.Clone(messages), true
}
