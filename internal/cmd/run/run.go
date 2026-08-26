package run

import (
	"context"
	"os"

	qderror "github.com/jorgefuertes/QDAAD/internal/ddb/dberrors"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "run a .DDB database",
		Aliases:   []string{"r"},
		ArgsUsage: "<file.DDB>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "headers",
				Aliases: []string{"H"},
				Usage:   "Display headers of the .DDB file",
			},
		},
		Action: action,
	}
}

func action(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() != 1 {
		return qderror.ErrMissingDDBFile
	}

	ddbFileName := cmd.Args().Get(0)

	// open the file
	ddbFile, err := os.Open(ddbFileName)
	if err != nil {
		return qderror.ErrFailedToOpenDDBFile
	}

	defer func() {
		_ = ddbFile.Close()
	}()

	// if cmd.Bool("headers") {
	// 	// Display headers of the .DDB file
	// }

	return nil
}
