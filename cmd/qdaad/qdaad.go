package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jorgefuertes/QDAAD/internal/cmd/check"
	"github.com/jorgefuertes/QDAAD/internal/cmd/compile"
	"github.com/jorgefuertes/QDAAD/internal/cmd/run"
	"github.com/jorgefuertes/QDAAD/internal/version"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:    "QDAAD",
		Usage:   "Queru's DAAD compiler",
		Version: version.Version,
		Commands: []*cli.Command{
			run.Command(),
			compile.Command(),
			check.Command(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
