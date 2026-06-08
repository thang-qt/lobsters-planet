package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	UserAgent string         `yaml:"user_agent"`
	Lobsters  LobstersConfig `yaml:"lobsters"`
	Output    OutputConfig   `yaml:"output"`
}

type LobstersConfig struct {
	UsersURL              string `yaml:"users_url"`
	RequestTimeoutSeconds int    `yaml:"request_timeout_seconds"`
}

type OutputConfig struct {
	StateFile string `yaml:"state_file"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.UserAgent == "" {
		return Config{}, fmt.Errorf("user_agent is required")
	}
	if cfg.Lobsters.UsersURL == "" {
		return Config{}, fmt.Errorf("lobsters.users_url is required")
	}
	if cfg.Output.StateFile == "" {
		return Config{}, fmt.Errorf("output.state_file is required")
	}
	if cfg.Lobsters.RequestTimeoutSeconds <= 0 {
		cfg.Lobsters.RequestTimeoutSeconds = 15
	}
	return cfg, nil
}
