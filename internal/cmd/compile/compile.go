package compile

import (
	"context"
	"path"

	"github.com/jorgefuertes/QDAAD/internal/definitions/arch"
	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:    "compile",
		Usage:   "compile a .DDB database",
		Aliases: []string{"c"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "machine",
				Category: "compiler flags",
				Aliases:  []string{"m"},
				Usage: "Target machine for the compiled database. " +
					"Valid values are: " + arch.MachineHumanList(),
			},
			&cli.StringFlag{
				Name:     "input",
				Category: "files",
				Aliases:  []string{"i"},
				Usage:    "Main source file to compile.",
			},
			&cli.StringFlag{
				Name:     "output",
				Category: "files",
				Aliases:  []string{"o"},
				Usage:    "Output file for the compiled database, without extension.",
			},
		},
		Action: compileAction,
	}
}

func compileAction(ctx context.Context, cmd *cli.Command) error {
	if path.Ext(cmd.String("input")) == ".sce" {
		db, err := legacy.Analyze(cmd.String("input"))
		if err != nil {
			return err
		}
		_ = db
	}

	return ErrNotImplemented
}
