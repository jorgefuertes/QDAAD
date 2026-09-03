package decompiler

import (
	"context"

	"github.com/jorgefuertes/QDAAD/internal/console"
	"github.com/urfave/cli/v3"
)

// RunAction is the entry point of the "decompile" command.
//
// The signature is the one cli/v3 expects: it dropped the v2 *cli.Context and
// an action now takes the context and the command it belongs to, which is
// where the flag values live.
func RunAction(_ context.Context, cmd *cli.Command) error {
	inputFile := cmd.String("input")
	outputDir := cmd.String("output")

	console.Banner("QUNDAAD", console.TitleStyle)

	return Decompile(inputFile, outputDir, Options{
		Binaries: !cmd.Bool("no-binaries"),
	})
}
