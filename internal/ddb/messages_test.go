package ddb

import (
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb/dberrors"
	"github.com/stretchr/testify/require"
)

// An unassigned kind: the store must reject it instead of quietly opening a
// third text table that nobody would know how to emit.
const unknownKind MessageKind = 200

func TestNewMessageStore(t *testing.T) {
	ms := NewMessageStore()
	require.NotNil(t, ms)
	require.Len(t, ms, 3, "one table per kind")

	for _, kind := range []MessageKind{SystemMessage, UserMessage, XMessage} {
		messages, exists := ms[kind]
		require.True(t, exists, "kind %d", kind)
		require.NotNil(t, messages, "the slice is allocated, not nil")
		require.Empty(t, messages)
	}

	_, exists := ms[unknownKind]
	require.False(t, exists)
}

func TestMessagesAddMessage(t *testing.T) {
	t.Run("numbering is sequential from zero", func(t *testing.T) {
		ms := NewMessageStore()

		for i := range 3 {
			require.NoError(t, ms.AddMessage(UserMessage, ID(i+1), "texto"))
			require.Equal(t, ID16(i), ms[UserMessage][i].ID, "message number %d", i)
		}

		require.Len(t, ms[UserMessage], 3)
	})

	t.Run("each kind is numbered on its own", func(t *testing.T) {
		// The DDB indexes every text table by an ordinal of its own, so system
		// message 0 and user message 0 are two different messages.
		ms := NewMessageStore()

		require.NoError(t, ms.AddMessage(SystemMessage, 1, "sistema cero"))
		require.NoError(t, ms.AddMessage(UserMessage, 2, "usuario cero"))
		require.NoError(t, ms.AddMessage(SystemMessage, 3, "sistema uno"))

		require.Len(t, ms[SystemMessage], 2)
		require.Len(t, ms[UserMessage], 1)

		require.Equal(t, ID16(0), ms[UserMessage][0].ID, "the user table starts at zero too")
		require.Equal(t, ID16(0), ms[SystemMessage][0].ID)
		require.Equal(t, ID16(1), ms[SystemMessage][1].ID)
	})

	t.Run("stores every field as given", func(t *testing.T) {
		ms := NewMessageStore()

		require.NoError(t, ms.AddMessage(UserMessage, 42, "Coges la llave."))
		require.Equal(t, Message{ID: 0, LabelID: 42, Content: "Coges la llave."}, ms[UserMessage][0])
	})

	t.Run("an empty message is a message like any other", func(t *testing.T) {
		ms := NewMessageStore()

		require.NoError(t, ms.AddMessage(UserMessage, 1, ""))
		require.Len(t, ms[UserMessage], 1)
		require.Empty(t, ms[UserMessage][0].Content)
	})

	t.Run("unknown kind", func(t *testing.T) {
		ms := NewMessageStore()

		err := ms.AddMessage(unknownKind, 1, "texto")
		require.ErrorIs(t, err, dberrors.ErrInvalidMessageKind)
		require.Len(t, ms, 3, "no fourth table is opened")
	})

	t.Run("exhausted", func(t *testing.T) {
		ms := NewMessageStore()

		for i := 0; i <= MAX_MESSSAGE.Int(); i++ {
			require.NoError(t, ms.AddMessage(SystemMessage, ID(i), "texto"), "message number %d", i)
		}

		require.Len(t, ms[SystemMessage], MAX_MESSSAGE.Int()+1, "255 messages, numbered 0 to 254")

		err := ms.AddMessage(SystemMessage, 0, "uno de más")
		require.ErrorIs(t, err, dberrors.ErrMessageStoreIsFull)
		require.Len(t, ms[SystemMessage], MAX_MESSSAGE.Int()+1, "nothing new is stored")

		require.NoError(t, ms.AddMessage(UserMessage, 1, "texto"),
			"the other table has its own room")
	})

	t.Run("xmessages are numbered past the byte limit", func(t *testing.T) {
		// The xmessage table is not one of the DDB text tables: nothing counts
		// it in the header, and its ID is a 16-bit offset into the .XMB file.
		ms := NewMessageStore()

		for range MAX_MESSSAGE.Int() + 2 {
			require.NoError(t, ms.AddMessage(XMessage, 1, "texto"))
		}

		last := ms[XMessage][len(ms[XMessage])-1]
		require.Equal(t, ID16(MAX_MESSSAGE.Int()+1), last.ID,
			"an xmessage numbered past 255, where a normal table would be full")
	})

	t.Run("an xmessage is capped at 511 bytes", func(t *testing.T) {
		ms := NewMessageStore()

		require.NoError(t, ms.AddMessage(XMessage, 1, strings.Repeat("a", 511)),
			"exactly 511 fits")

		err := ms.AddMessage(XMessage, 2, strings.Repeat("a", 512))
		require.ErrorIs(t, err, dberrors.ErrXMessageTooLong)
		require.Len(t, ms[XMessage], 1, "the long one is not stored")
	})

	t.Run("the 511 cap is on bytes, not on characters", func(t *testing.T) {
		// A tilde is two bytes in UTF-8, so 300 of them blow the limit even
		// though they are only 300 characters.
		ms := NewMessageStore()

		err := ms.AddMessage(XMessage, 1, strings.Repeat("ñ", 300))
		require.ErrorIs(t, err, dberrors.ErrXMessageTooLong)
	})

	t.Run("the other tables have no length limit", func(t *testing.T) {
		ms := NewMessageStore()

		for _, kind := range []MessageKind{SystemMessage, UserMessage} {
			require.NoError(t, ms.AddMessage(kind, 1, strings.Repeat("a", 2000)),
				"kind %d", kind)
		}
	})
}

func TestMessagesGetMessage(t *testing.T) {
	ms := NewMessageStore()
	require.NoError(t, ms.AddMessage(SystemMessage, 1, "sistema cero"))
	require.NoError(t, ms.AddMessage(SystemMessage, 2, "sistema uno"))
	require.NoError(t, ms.AddMessage(UserMessage, 3, "usuario cero"))

	t.Run("found", func(t *testing.T) {
		content, ok := ms.GetMessage(SystemMessage, 1)
		require.True(t, ok)
		require.Equal(t, "sistema uno", content, "the right one, not the first")
	})

	t.Run("the kind tells two messages with the same ID apart", func(t *testing.T) {
		system, ok := ms.GetMessage(SystemMessage, 0)
		require.True(t, ok)
		require.Equal(t, "sistema cero", system)

		user, ok := ms.GetMessage(UserMessage, 0)
		require.True(t, ok)
		require.Equal(t, "usuario cero", user)
	})

	t.Run("not found", func(t *testing.T) {
		cases := []struct {
			name string
			kind MessageKind
			id   ID16
		}{
			{"unknown ID", SystemMessage, 2},
			// The user table only holds message 0.
			{"right ID, wrong kind", UserMessage, 1},
			{"unknown kind", unknownKind, 0},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				content, ok := ms.GetMessage(c.kind, c.id)
				require.False(t, ok)
				require.Empty(t, content, "no content comes back with a false")
			})
		}
	})
}

func TestMessagesGetMessageByLabelID(t *testing.T) {
	ms := NewMessageStore()
	require.NoError(t, ms.AddMessage(SystemMessage, 7, "sistema siete"))
	require.NoError(t, ms.AddMessage(SystemMessage, 8, "sistema ocho"))

	// The same label in the other table: only the kind tells them apart.
	require.NoError(t, ms.AddMessage(UserMessage, 7, "usuario siete"))

	t.Run("found", func(t *testing.T) {
		content, ok := ms.GetMessageByLabelID(SystemMessage, 8)
		require.True(t, ok)
		require.Equal(t, "sistema ocho", content, "the right one, not the first")
	})

	t.Run("discriminates by kind", func(t *testing.T) {
		system, ok := ms.GetMessageByLabelID(SystemMessage, 7)
		require.True(t, ok)
		require.Equal(t, "sistema siete", system)

		user, ok := ms.GetMessageByLabelID(UserMessage, 7)
		require.True(t, ok)
		require.Equal(t, "usuario siete", user)
	})

	t.Run("returns the first match", func(t *testing.T) {
		// Nothing stops a label from being reused: the store does not check it.
		ms := NewMessageStore()
		require.NoError(t, ms.AddMessage(UserMessage, 5, "el primero"))
		require.NoError(t, ms.AddMessage(UserMessage, 5, "el segundo"))

		content, ok := ms.GetMessageByLabelID(UserMessage, 5)
		require.True(t, ok)
		require.Equal(t, "el primero", content)
	})

	t.Run("not found", func(t *testing.T) {
		cases := []struct {
			name    string
			kind    MessageKind
			labelID ID
		}{
			{"unknown label", SystemMessage, 9},
			{"right label, wrong kind", UserMessage, 8},
			{"unknown kind", unknownKind, 7},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				content, ok := ms.GetMessageByLabelID(c.kind, c.labelID)
				require.False(t, ok)
				require.Empty(t, content, "no content comes back with a false")
			})
		}
	})
}

func TestMessagesGetAllMessages(t *testing.T) {
	t.Run("the whole table, in insertion order", func(t *testing.T) {
		ms := NewMessageStore()
		require.NoError(t, ms.AddMessage(SystemMessage, 1, "cero"))
		require.NoError(t, ms.AddMessage(SystemMessage, 2, "uno"))
		require.NoError(t, ms.AddMessage(UserMessage, 3, "de usuario"))

		messages, ok := ms.GetAllMessages(SystemMessage)
		require.True(t, ok)
		require.Equal(t, []Message{
			{ID: 0, LabelID: 1, Content: "cero"},
			{ID: 1, LabelID: 2, Content: "uno"},
		}, messages, "only its own table")
	})

	t.Run("an empty table", func(t *testing.T) {
		ms := NewMessageStore()

		messages, ok := ms.GetAllMessages(UserMessage)
		require.True(t, ok, "the table exists, it is just empty")
		require.Empty(t, messages)
	})

	t.Run("unknown kind", func(t *testing.T) {
		ms := NewMessageStore()

		messages, ok := ms.GetAllMessages(unknownKind)
		require.False(t, ok)
		require.Nil(t, messages)
	})

	t.Run("the table is a copy: writing to it cannot reach the store", func(t *testing.T) {
		ms := NewMessageStore()
		require.NoError(t, ms.AddMessage(UserMessage, 1, "original"))
		require.NoError(t, ms.AddMessage(UserMessage, 2, "otro"))

		messages, ok := ms.GetAllMessages(UserMessage)
		require.True(t, ok)

		messages[0].Content = "cambiado"
		messages[0].LabelID = 99
		messages[1] = Message{}

		again, ok := ms.GetAllMessages(UserMessage)
		require.True(t, ok)
		require.Equal(t, []Message{
			{ID: 0, LabelID: 1, Content: "original"},
			{ID: 1, LabelID: 2, Content: "otro"},
		}, again, "the store is untouched")
	})

	t.Run("two calls do not share an array", func(t *testing.T) {
		ms := NewMessageStore()
		require.NoError(t, ms.AddMessage(UserMessage, 1, "original"))

		first, ok := ms.GetAllMessages(UserMessage)
		require.True(t, ok)

		second, ok := ms.GetAllMessages(UserMessage)
		require.True(t, ok)

		first[0].Content = "cambiado"
		require.Equal(t, "original", second[0].Content)
	})
}
