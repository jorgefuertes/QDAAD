package legacy

import (
	"strings"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/stretchr/testify/require"
)

// An xmessage is written where the text goes and filed somewhere else.
//
// Every other condact's operand is a number or a word. These two take the text
// itself, like MESSAGE does, but what the text goes into is the one table that
// does not live in the database — so the condact keeps an index rather than a
// parameter, and the back end turns that into the offset the text ends up at.
// See xmessage, and internal/back/xmessages.go for the other half.
func TestAnXMessageIsFiledAndNumbered(t *testing.T) {
	db, err := parse(t, strings.Replace(bare,
		"> COGER LLAVE   MESSAGE 0",
		`> COGER LLAVE   XMES "the first"
                XMES "the second"`, 1))
	require.NoError(t, err)

	texts, _ := db.Messages().GetAllMessages(ddb.XMessage)
	require.Len(t, texts, 2)
	require.Equal(t, "the first", texts[0].Content)
	require.Equal(t, "the second", texts[1].Content)

	// And nothing of them is in the tables that do live in the database.
	user, _ := db.Messages().GetAllMessages(ddb.UserMessage)
	require.Len(t, user, 1, "an xmessage went into the user messages")

	condacts := (*db.Processes())[0].Entries[0].Condacts
	require.Equal(t, ddb.OpXMes, condacts[0].Opcode)
	require.Equal(t, ddb.ID16(0), condacts[0].XMessage)
	require.Equal(t, ddb.ID16(1), condacts[1].XMessage)
	require.Empty(t, condacts[0].Params,
		"the index is not a parameter: it does not fit in the byte one holds")
}

// XMESSAGE is XMES with a line break, and that is the whole of the difference.
//
// The break goes in as the escape the character set understands and not as a
// byte, because the byte a line break is stored as — 0x0D — is not the byte a
// text ends with. Appending the wrong one writes a second terminator and the
// break is lost.
func TestXMessageIsXMesAndALineBreak(t *testing.T) {
	db, err := parse(t, strings.Replace(bare,
		"> COGER LLAVE   MESSAGE 0",
		`> COGER LLAVE   XMES "same text"
                XMESSAGE "same text"`, 1))
	require.NoError(t, err)

	texts, _ := db.Messages().GetAllMessages(ddb.XMessage)
	require.Len(t, texts, 2)
	require.Equal(t, "same text", texts[0].Content)
	require.Equal(t, "same text#n", texts[1].Content)

	// Both come out as the one opcode: the back end has one case to handle,
	// which is what the compiler of reference does too.
	condacts := (*db.Processes())[0].Entries[0].Condacts
	require.Equal(t, ddb.OpXMes, condacts[0].Opcode)
	require.Equal(t, ddb.OpXMes, condacts[1].Opcode)
}

// The text is not optional, and an xmessage given a number is a mistake worth
// naming: it reads like MESSAGE and is not.
func TestAnXMessageWantsItsTextWrittenThere(t *testing.T) {
	_, err := parse(t, strings.Replace(bare,
		"> COGER LLAVE   MESSAGE 0", "> COGER LLAVE   XMES 3", 1))

	require.ErrorContains(t, err, "the text of an xmessage")
}

// There are more of them than a byte can count, which is half of why they
// exist: the tables inside the database stop at 255.
func TestXMessagesAreNumberedPastWhatAByteHolds(t *testing.T) {
	var sb strings.Builder

	sb.WriteString("> COGER LLAVE   DONE\n")

	for i := range 300 {
		sb.WriteString("> _     _       XMES \"")
		sb.WriteString(strings.Repeat("x", 1+i%7))
		sb.WriteString("\"\n                DONE\n")
	}

	db, err := parse(t, strings.Replace(bare, "> COGER LLAVE   MESSAGE 0\n                DONE\n",
		sb.String(), 1))
	require.NoError(t, err)

	texts, _ := db.Messages().GetAllMessages(ddb.XMessage)
	require.Len(t, texts, 300)

	// The last one is reachable, which it would not be if the index had gone
	// through a Param on the way.
	last := (*db.Processes())[0].Entries[300].Condacts[0]
	require.Equal(t, ddb.ID16(299), last.XMessage)
}
