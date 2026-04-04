package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/protocol-registry-cli/internal/usecases/get_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/register_consumer"
	"github.com/user/protocol-registry-cli/internal/usecases/unregister_consumer"
	urfave "github.com/urfave/cli/v2"
)

func (h *Handler) clientCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "client",
		Usage: "Client-side protocol operations",
		Subcommands: []*urfave.Command{
			h.clientFetchCommand(),
			h.clientRegisterCommand(),
			h.clientUnregisterCommand(),
		},
	}
}

func (h *Handler) clientFetchCommand() *urfave.Command {
	return &urfave.Command{
		Name:      "fetch",
		Usage:     "Download a service's protocol into the clients directory",
		ArgsUsage: "<protocol> <service-name>",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "version", Value: "default", Usage: "Version to fetch"},
		},
		Action: func(c *urfave.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: prctl client fetch <protocol> <service-name>")
			}
			protocolType := c.Args().Get(0)
			serviceName := c.Args().Get(1)

			output, err := h.getUC.Execute(c.Context, get_protocol.Input{
				ServiceName:  serviceName,
				ProtocolType: protocolType,
				Version:      c.String("version"),
			})
			if err != nil {
				return err
			}

			outDir := filepath.Join("protocols", protocolType, "clients", serviceName)
			for _, f := range output.Files {
				dst := filepath.Join(outDir, f.Path)
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(dst, f.Content, 0o644); err != nil {
					return err
				}
				fmt.Printf("wrote %s\n", dst)
			}
			return nil
		},
	}
}

func (h *Handler) clientRegisterCommand() *urfave.Command {
	return &urfave.Command{
		Name:      "register",
		Usage:     "Register as a consumer using local client proto files",
		ArgsUsage: "<protocol> <service-name>",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "entry-point", Value: "default.proto", Usage: "Entry point .proto file (relative to proto-dir)"},
			&urfave.StringSliceFlag{Name: "server-versions", Usage: "Server versions to register against (default: [\"default\"])"},
		},
		Action: func(c *urfave.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: prctl client register <protocol> <service-name>")
			}
			protocolType := c.Args().Get(0)
			serverName := c.Args().Get(1)

			cfg, err := readServiceConfig()
			if err != nil {
				return err
			}

			protoDir := filepath.Join("protocols", protocolType, "clients", serverName)

			output, err := h.registerUC.Execute(c.Context, register_consumer.Input{
				ConsumerName:   cfg.Service.Name,
				ServerName:     serverName,
				ProtocolType:   protocolType,
				ProtoDir:       protoDir,
				EntryPoint:     c.String("entry-point"),
				ServerVersions: c.StringSlice("server-versions"),
			})
			if err != nil {
				return err
			}

			fmt.Printf("consumer: %s\n", output.ConsumerName)
			fmt.Printf("server:   %s\n", output.ServerName)
			fmt.Printf("is_new:   %v\n", output.IsNew)
			return nil
		},
	}
}

func (h *Handler) clientUnregisterCommand() *urfave.Command {
	return &urfave.Command{
		Name:      "unregister",
		Usage:     "Unregister as a consumer of a service's protocol",
		ArgsUsage: "<protocol> <service-name>",
		Flags: []urfave.Flag{
			&urfave.StringSliceFlag{Name: "server-versions", Usage: "Server versions to unregister from (default: [\"default\"])"},
		},
		Action: func(c *urfave.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: prctl client unregister <protocol> <service-name>")
			}
			protocolType := c.Args().Get(0)
			serverName := c.Args().Get(1)

			cfg, err := readServiceConfig()
			if err != nil {
				return err
			}

			if err := h.unregisterUC.Execute(c.Context, unregister_consumer.Input{
				ConsumerName:   cfg.Service.Name,
				ServerName:     serverName,
				ProtocolType:   protocolType,
				ServerVersions: c.StringSlice("server-versions"),
			}); err != nil {
				return err
			}

			fmt.Println("unregistered successfully")
			return nil
		},
	}
}
