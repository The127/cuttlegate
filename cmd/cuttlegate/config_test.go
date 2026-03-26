package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadSaveRoundTrip(t *testing.T) {
	// Use a temp dir as XDG_CONFIG_HOME.
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Initially empty.
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Server != "" || cfg.Project != "" || cfg.Environment != "" {
		t.Fatalf("expected empty config, got %+v", cfg)
	}

	// Set values and save.
	cfg.Server = "https://example.com"
	cfg.Project = "my-project"
	cfg.Environment = "staging"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(tmp, "cuttlegate", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not found: %v", err)
	}

	// Reload and verify.
	cfg2, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after save: %v", err)
	}
	if cfg2.Server != "https://example.com" {
		t.Errorf("server = %q, want %q", cfg2.Server, "https://example.com")
	}
	if cfg2.Project != "my-project" {
		t.Errorf("project = %q, want %q", cfg2.Project, "my-project")
	}
	if cfg2.Environment != "staging" {
		t.Errorf("environment = %q, want %q", cfg2.Environment, "staging")
	}
}

func TestConfigMissingFileReturnsEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Server != "" {
		t.Errorf("expected empty server, got %q", cfg.Server)
	}
}
