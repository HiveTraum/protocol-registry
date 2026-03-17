package cli

import (
	"fmt"
	"path/filepath"

	"github.com/user/protocol-registry-cli/internal/usecases/publish_protocol"
	urfave "github.com/urfave/cli/v2"
)

func (h *Handler) serverCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "server",
		Usage: "Server-side protocol operations",
		Subcommands: []*urfave.Command{
			h.serverPublishCommand(),
		},
	}
}

func (h *Handler) serverPublishCommand() *urfave.Command {
	return &urfave.Command{
		Name:      "publish",
		Usage:     "Publish the server protocol to the registry",
		ArgsUsage: "<protocol>",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "proto-dir", Usage: "Directory with .proto files"},
			&urfave.StringFlag{Name: "entry-point", Value: "default.proto", Usage: "Entry point .proto file (relative to proto-dir)"},
		},
		Action: func(c *urfave.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: prctl server publish <protocol>")
			}
			protocolType := c.Args().Get(0)

			cfg, err := readServiceConfig()
			if err != nil {
				return err
			}

			protoDir := c.String("proto-dir")
			if protoDir == "" {
				protoDir = filepath.Join("protocols", protocolType, "server")
			}

			output, err := h.publishUC.Execute(c.Context, publish_protocol.Input{
				ServiceName:  cfg.Service.Name,
				ProtocolType: protocolType,
				ProtoDir:     protoDir,
				EntryPoint:   c.String("entry-point"),
			})
			if err != nil {
				return err
			}

			fmt.Printf("service: %s\n", output.ServiceName)
			fmt.Printf("is_new:  %v\n", output.IsNew)
			return nil
		},
	}
}
