package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Discovery DiscoveryConfig         `toml:"discovery"`
	Defaults  ModuleConfig            `toml:"defaults"`
	Modules   map[string]ModuleConfig `toml:"modules"`
}

type DiscoveryConfig struct {
	Excludes []string `toml:"excludes"`
}

type ModuleConfig struct {
	Updates []string `toml:"updates"`
	Checks  []string `toml:"checks"`
	Commit  string   `toml:"commit"`
}

func Load(root string) (Config, error) {
	path := filepath.Join(root, ".modrel.toml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) ForModule(name string) ModuleConfig {
	merged := c.Defaults
	if len(merged.Checks) == 0 {
		merged.Checks = []string{"go test ./..."}
	}
	if c.Modules == nil {
		return merged
	}
	override, ok := c.Modules[name]
	if !ok {
		return merged
	}
	if override.Updates != nil {
		merged.Updates = override.Updates
	}
	if override.Checks != nil {
		merged.Checks = override.Checks
	}
	if override.Commit != "" {
		merged.Commit = override.Commit
	}
	return merged
}
