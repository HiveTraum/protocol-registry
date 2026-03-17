package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	urfave "github.com/urfave/cli/v2"
)

func (h *Handler) initCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "init",
		Usage: "Initialize service config or protocol template",
		Subcommands: []*urfave.Command{
			h.initServiceCommand(),
			h.initProtocolCommand(),
		},
	}
}

func (h *Handler) initServiceCommand() *urfave.Command {
	return &urfave.Command{
		Name:      "service",
		Usage:     "Create service.toml (for legacy services)",
		ArgsUsage: "<service-name>",
		Action: func(c *urfave.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: prctl init service <service-name>")
			}
			serviceName := c.Args().Get(0)

			if _, err := os.Stat(serviceConfigFile); err == nil {
				return fmt.Errorf("%s already exists", serviceConfigFile)
			}
			if err := writeServiceConfig(serviceName); err != nil {
				return err
			}
			fmt.Printf("created %s\n", serviceConfigFile)
			return nil
		},
	}
}

func (h *Handler) initProtocolCommand() *urfave.Command {
	return &urfave.Command{
		Name:      "protocol",
		Usage:     "Create a protocol template from service.toml",
		ArgsUsage: "<protocol>",
		Action: func(c *urfave.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: prctl init protocol <protocol>")
			}
			protocolType := c.Args().Get(0)

			cfg, err := readServiceConfig()
			if err != nil {
				return err
			}

			serviceName := cfg.Service.Name
			dir := filepath.Join("protocols", protocolType, "server")
			sanitized := strings.ReplaceAll(serviceName, "-", "_")
			path := filepath.Join(dir, sanitized+".proto")

			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("file already exists: %s", path)
			}

			pascal := toPascalCase(serviceName)
			content := fmt.Sprintf(`syntax = "proto3";

package %s.v1;

service %s {
}
`, sanitized, pascal)

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}

			fmt.Printf("created %s\n", path)
			return nil
		},
	}
}

func toPascalCase(s string) string {
	var b strings.Builder
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	}) {
		if len(part) > 0 {
			b.WriteString(strings.ToUpper(part[:1]))
			b.WriteString(part[1:])
		}
	}
	return b.String()
}
