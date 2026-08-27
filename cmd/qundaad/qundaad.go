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
