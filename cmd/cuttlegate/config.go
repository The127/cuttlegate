package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds persisted CLI configuration.
type Config struct {
	Server      string `json:"server,omitempty"`
	Project     string `json:"project,omitempty"`
	Environment string `json:"environment,omitempty"`
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "cuttlegate")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cuttlegate")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

// LoadConfig reads the config file; returns zero value if it doesn't exist.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o600)
}

func cmdConfig(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cuttlegate config <set|get> <key> [value]")
	}
	switch args[0] {
	case "set":
		return cmdConfigSet(args[1:])
	case "get":
		return cmdConfigGet(args[1:])
	default:
		return fmt.Errorf("unknown config subcommand %q — use 'set' or 'get'", args[0])
	}
}

var validConfigKeys = map[string]bool{
	"server":      true,
	"project":     true,
	"environment": true,
}

func cmdConfigSet(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: cuttlegate config set <key> <value>")
	}
	key, value := args[0], args[1]
	if !validConfigKeys[key] {
		return fmt.Errorf("unknown config key %q — valid keys: server, project, environment", key)
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	switch key {
	case "server":
		cfg.Server = value
	case "project":
		cfg.Project = value
	case "environment":
		cfg.Environment = value
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func cmdConfigGet(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: cuttlegate config get <key>")
	}
	key := args[0]
	if !validConfigKeys[key] {
		return fmt.Errorf("unknown config key %q — valid keys: server, project, environment", key)
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	var value string
	switch key {
	case "server":
		value = cfg.Server
	case "project":
		value = cfg.Project
	case "environment":
		value = cfg.Environment
	}
	if value == "" {
		fmt.Fprintf(os.Stderr, "%s is not set\n", key)
	} else {
		fmt.Println(value)
	}
	return nil
}
