package cli

import (
	"fmt"
	"path/filepath"

	urfave "github.com/urfave/cli/v2"

	"github.com/user/protocol-registry-cli/internal/usecases/publish_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/validate_protocol"
)

func (h *Handler) serverCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "server",
		Usage: "Server-side protocol operations",
		Subcommands: []*urfave.Command{
			h.serverPublishCommand(),
			h.serverValidateCommand(),
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
			&urfave.StringSliceFlag{Name: "versions", Usage: "Versions to publish to (default: [\"default\"])"},
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
				Versions:     c.StringSlice("versions"),
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

func (h *Handler) serverValidateCommand() *urfave.Command {
	return &urfave.Command{
		Name:      "validate",
		Usage:     "Validate a protocol against existing consumers without publishing",
		ArgsUsage: "<protocol>",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "proto-dir", Usage: "Directory with .proto files"},
			&urfave.StringFlag{Name: "entry-point", Value: "default.proto", Usage: "Entry point .proto file (relative to proto-dir)"},
			&urfave.StringSliceFlag{Name: "against-versions", Usage: "Versions to validate against (default: [\"default\"])"},
		},
		Action: func(c *urfave.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: prctl server validate <protocol>")
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

			output, err := h.validateUC.Execute(c.Context, validate_protocol.Input{
				ServiceName:     cfg.Service.Name,
				ProtocolType:    protocolType,
				ProtoDir:        protoDir,
				EntryPoint:      c.String("entry-point"),
				AgainstVersions: c.StringSlice("against-versions"),
			})
			if err != nil {
				return err
			}

			if output.Valid {
				fmt.Println("valid: true")
				return nil
			}

			fmt.Println("valid: false")
			for _, vv := range output.Violations {
				fmt.Printf("\nversion: %s\n", vv.Version)
				for _, cv := range vv.Consumers {
					fmt.Printf("  consumer: %s\n", cv.ConsumerName)
					for _, v := range cv.Violations {
						fmt.Printf("    - %s\n", v)
					}
				}
			}
			return nil
		},
	}
}
