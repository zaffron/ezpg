package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "ezpg", "config.yaml")
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}

	cfg := &Config{
		Settings: DefaultSettings(),
		Path:     path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file - return empty config for home screen
			return cfg, nil
		}

		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	names := make(map[string]bool)
	for i, c := range cfg.Connections {
		if c.Name == "" {
			return fmt.Errorf("config: connection[%d] must have a name", i)
		}

		if names[c.Name] {
			return fmt.Errorf("config: duplicate connection name %q", c.Name)
		}
		names[c.Name] = true

		if c.URL == "" && c.Host == "" {
			return fmt.Errorf("config: connection %q must have either url or host", c.Name)
		}
	}

	if cfg.Settings.DefaultLimit <= 0 {
		cfg.Settings.DefaultLimit = 100
	}

	if cfg.Settings.EditorTabSize <= 0 {
		cfg.Settings.EditorTabSize = 4
	}

	if cfg.Settings.NullDisplay == "" {
		cfg.Settings.NullDisplay = "NULL"
	}

	return nil
}
