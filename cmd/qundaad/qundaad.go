package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jorgefuertes/QDAAD/cmd/qundaad/decompiler"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:    "qundaad",
		Usage:   "Queru's DAAD decompiler",
		Version: decompiler.VERSION,
		Commands: []*cli.Command{
			{
				Name:  "decompile",
				Usage: "Decompile a DAAD file into its source code",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "input",
						Aliases:  []string{"i"},
						Usage:    "Input DAAD file to decompile",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Usage:    "Output directory for the decompiled source code",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "no-binaries",
						Aliases: []string{"B"},
						Usage: "Write only what can be read, leaving out the character sets, " +
							"screens and image archives as they came",
					},
					&cli.IntFlag{
						Name:    "scale",
						Aliases: []string{"s"},
						Value:   1,
						Usage: "Also write the vector drawings enlarged by this many times, " +
							"beside the ones at their original size",
					},
					&cli.BoolFlag{
						Name: "no-pictures",
						Usage: "Leave out the condacts that load and show an illustration, for " +
							"building without them. Not part of --v3: version 3 draws pictures too",
					},
					&cli.BoolFlag{
						Name:    "v3",
						Aliases: []string{"3"},
						Usage: "Write a source a version 3 interpreter will run, rather than an " +
							"account of what the database holds. Only a format 1 database has " +
							"anything to convert, and the source it writes no longer recompiles " +
							"into the database it came from",
					},
				},
				Action: decompiler.RunAction,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
