package check

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jorgefuertes/QDAAD/internal/console"
	"github.com/jorgefuertes/QDAAD/internal/ddb"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
)

// summarize prints what the source turned out to hold.
//
// It is the answer to "did that work", and the counts are what a person checks
// against the game they know: an adventure with sixty locations that reads as
// having two has lost a section somewhere.
func summarize(db *ddb.DDB, res *legacy.Result) error {
	table := console.NewTable().Header("Section", "Count")

	sysmess, _ := db.Messages().GetAllMessages(ddb.SystemMessage)
	messages, _ := db.Messages().GetAllMessages(ddb.UserMessage)

	for _, row := range []struct {
		name  string
		count int
	}{
		{"Vocabulary", len(*db.Words())},
		{"System messages", len(sysmess)},
		{"Messages", len(messages)},
		{"Locations", db.Locations().Len()},
		{"Objects", db.Objects().Len()},
		{"Connections", len(*db.Conns())},
		{"Processes", db.Processes().Len()},
		{"Entries", entries(db)},
		{"Condacts", condacts(db)},
		{"Symbols", len(res.Symbols)},
		{"Externals", len(res.Linked.Extern)},
	} {
		table.Row(row.name, strconv.Itoa(row.count))
	}

	table.Print()

	if res.Settings.Classic {
		console.Sayln(console.LevelInfo, "the source asks for the classic dialect")
	}

	if res.Settings.Debug {
		console.Sayln(console.LevelInfo, "the source asks for a debug build")
	}

	return nil
}

func entries(db *ddb.DDB) int {
	total := 0
	for _, p := range *db.Processes() {
		total += len(p.Entries)
	}

	return total
}

func condacts(db *ddb.DDB) int {
	total := 0

	for _, p := range *db.Processes() {
		for _, e := range p.Entries {
			total += len(e.Condacts)
		}
	}

	return total
}

// parseDate reads the day the date symbols are worth.
//
// An empty string is the zero time, which the front end reads as "now". That
// is what the compiler of reference always does, and it is why a build is not
// repeatable unless this is given.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	when, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q is not a date, it is written YYYY-MM-DD", s)
	}

	return when, nil
}
