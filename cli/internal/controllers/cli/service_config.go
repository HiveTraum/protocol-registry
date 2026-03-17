package cli

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const serviceConfigFile = "service.toml"

type serviceConfig struct {
	Service struct {
		Name string `toml:"name"`
	} `toml:"service"`
}

func readServiceConfig() (serviceConfig, error) {
	var cfg serviceConfig
	if _, err := toml.DecodeFile(serviceConfigFile, &cfg); err != nil {
		if os.IsNotExist(err) {
			return cfg, fmt.Errorf("%s not found; run 'init' first", serviceConfigFile)
		}
		return cfg, fmt.Errorf("reading %s: %w", serviceConfigFile, err)
	}
	if cfg.Service.Name == "" {
		return cfg, fmt.Errorf("%s: service.name is required", serviceConfigFile)
	}
	return cfg, nil
}

func writeServiceConfig(name string) error {
	content := fmt.Sprintf("[service]\nname = %q\n", name)
	return os.WriteFile(serviceConfigFile, []byte(content), 0o644)
}
