package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Discovery DiscoveryConfig         `yaml:"discovery"`
	Defaults  ModuleConfig            `yaml:"defaults"`
	Modules   map[string]ModuleConfig `yaml:"modules"`
}

type DiscoveryConfig struct {
	Exclude []string `yaml:"exclude"`
}

type ModuleConfig struct {
	Update []string `yaml:"update"`
	Checks []string `yaml:"checks"`
	Commit string   `yaml:"commit"`
}

func Load(root string) (Config, error) {
	path := filepath.Join(root, ".modrel.yaml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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
	if override.Update != nil {
		merged.Update = override.Update
	}
	if override.Checks != nil {
		merged.Checks = override.Checks
	}
	if override.Commit != "" {
		merged.Commit = override.Commit
	}
	return merged
}
