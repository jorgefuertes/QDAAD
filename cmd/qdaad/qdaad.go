package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jorgefuertes/QDAAD/internal/cmd/run"
	"github.com/urfave/cli/v3"
)

const (
	QDAADVersion = "0.1.0"
)

func main() {
	cmd := &cli.Command{
		Name:    "QDAAD",
		Usage:   "Queru's DAAD compiler",
		Version: QDAADVersion,
		Commands: []*cli.Command{
			run.Command(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
