// Command jsoncmp holds two intermediate-form files against each other and
// says where they disagree.
//
// It is a development tool and not one of this program's commands, which is
// why it lives beside the script that drives it rather than under cmd/: it is
// the Go half of compare-drf.sh, the way the rest of that harness is the shell
// half. Nothing in the compiler calls it, nothing in the compiler depends on
// the other front end existing, and nothing ships it.
//
// It compares meaning, not text. The two files are written by different
// programs and will never match byte for byte: the keys come out in a
// different order, the numbers are indented differently, and a literal string
// filed in a different order gets a different message number. What has to
// match is what the adventure does — which word is which, which text a condact
// shows, which entry runs. So each section is read into a map keyed by what
// names it, and compared that way.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: jsoncmp <theirs.json> <ours.json>")
		os.Exit(2)
	}

	theirs, err := load(os.Args[1], true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", os.Args[1], err)
		os.Exit(2)
	}

	ours, err := load(os.Args[2], false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", os.Args[2], err)
		os.Exit(2)
	}

	report := Compare(theirs, ours)
	if report.Empty() {
		return
	}

	fmt.Print(report)
	os.Exit(1)
}
