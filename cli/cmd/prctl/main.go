package main

import (
	"fmt"
	"os"

	urfave "github.com/urfave/cli/v2"

	clicontroller "github.com/user/protocol-registry-cli/internal/controllers/cli"
)

func main() {
	h := clicontroller.NewHandler()

	app := &urfave.App{
		Name:  "prctl",
		Usage: "CLI client for Protocol Registry",
		Flags: []urfave.Flag{
			&urfave.StringFlag{
				Name:    "addr",
				Aliases: []string{"a"},
				Value:   "localhost:50051",
				Usage:   "gRPC server address",
			},
		},
		Before:   func(c *urfave.Context) error { return h.Init(c.String("addr")) },
		After:    func(c *urfave.Context) error { return h.Close() },
		Commands: h.Commands(),
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
