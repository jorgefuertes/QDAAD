// Package check reads a source and says what it found, without building
// anything.
//
// It exists because "the source was read without an error" is a much weaker
// claim than it sounds: a condact resolved to the wrong opcode, or a parameter
// taken from the vocabulary where it should have come from the symbol table,
// reads perfectly and compiles to the wrong game. The JSON it writes is the
// intermediate form DAAD READY's two halves talk through, and a second front
// end reading the same source has to produce the same one.
//
// Nothing here knows the compiler of reference exists. Holding the two against
// each other is script/compare-drf.sh, which is a tool of this repository and
// not a part of the compiler.
package check

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/definitions/language"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/urfave/cli/v3"
)

// The two forms a text can come out in. Neither is a dialect of anybody
// else's: they are our own output, read by a person or read by a program.
const (
	textSource  = "source"
	textEncoded = "encoded"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:      "check",
		Usage:     "read a source and report what is in it, without compiling",
		ArgsUsage: "<source.sce>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "machine",
				Category: "compiler flags",
				Aliases:  []string{"m"},
				Value:    "zx",
				Usage: "Machine the source is read for, which changes what it " +
					"compiles to. Valid values are: " + arch.MachineHumanList(),
			},
			&cli.StringFlag{
				Name:     "subtarget",
				Category: "compiler flags",
				Aliases:  []string{"s"},
				Usage:    "Variant of the machine, where it has one.",
			},
			&cli.StringFlag{
				Name:     "language",
				Category: "compiler flags",
				Aliases:  []string{"l"},
				Value:    "spanish",
				Usage:    "Language the interpreter speaks: english or spanish.",
			},
			&cli.UintFlag{
				Name:     "version",
				Category: "compiler flags",
				Aliases:  []string{"v"},
				Value:    3,
				Usage:    "Database format, 2 or 3.",
			},
			&cli.StringSliceFlag{
				Name:     "define",
				Category: "compiler flags",
				Aliases:  []string{"D"},
				Usage:    "Extra symbol to define before reading the source. Repeatable.",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Write the intermediate form instead of a summary.",
			},
			&cli.StringFlag{
				Name:  "text",
				Value: textSource,
				Usage: "What form the texts come out in: " + textSource +
					" keeps the spelling the author used, " + textEncoded +
					" gives the codes a database holds them as.",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Where to write, standard output if left out.",
			},
			&cli.StringFlag{
				Name: "date",
				Usage: "Fix the date the YEARHIGH, YEARLOW, MONTH and DAY " +
					"symbols are worth, as YYYY-MM-DD. Today's if left out. " +
					"Setting it is what makes a build repeatable.",
			},
		},
		Action: checkAction,
	}
}

func checkAction(_ context.Context, cmd *cli.Command) error {
	source := cmd.Args().First()
	if source == "" {
		return fmt.Errorf("no source to read: name one")
	}

	opts, err := options(cmd)
	if err != nil {
		return err
	}

	db, res, err := legacy.Compile(source, opts)
	if err != nil {
		return err
	}

	if !cmd.Bool("json") {
		return summarize(db, res)
	}

	doc := legacy.NewDocument(db, res, opts)

	switch form := cmd.String("text"); form {
	case textSource:
	case textEncoded:
		doc.EncodeTexts()
	default:
		return fmt.Errorf("%q is not a text form: it is %s or %s", form, textSource, textEncoded)
	}

	var out bytes.Buffer
	if err := doc.WriteJSON(&out); err != nil {
		return err
	}

	return write(cmd.String("output"), out.Bytes())
}

func write(to string, body []byte) error {
	if to == "" {
		_, err := os.Stdout.Write(body)

		return err
	}

	return os.WriteFile(to, body, 0o644)
}

func options(cmd *cli.Command) (legacy.Options, error) {
	machine, err := arch.Parse(cmd.String("machine"))
	if err != nil {
		return legacy.Options{}, err
	}

	lang := language.LanguageUndefined.Parse(cmd.String("language"))
	if !lang.IsValid() {
		return legacy.Options{}, fmt.Errorf("%q is not a language, it is english or spanish",
			cmd.String("language"))
	}

	when, err := parseDate(cmd.String("date"))
	if err != nil {
		return legacy.Options{}, err
	}

	opts := legacy.Options{
		Machine:   machine,
		Subtarget: cmd.String("subtarget"),
		Language:  lang,
		Version:   uint8(cmd.Uint("version")),
		Date:      when,
		Extra:     cmd.StringSlice("define"),
	}

	return opts, opts.Validate()
}
