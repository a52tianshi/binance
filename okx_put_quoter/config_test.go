package main

import (
	"testing"
	"time"
)

func TestLoadConfig_DefaultsAndEnv(t *testing.T) {
	env := map[string]string{
		"OKX_API_KEY":        "key123",
		"OKX_API_SECRET":     "secret123",
		"OKX_API_PASSPHRASE": "pass123",
	}
	getenv := func(k string) string { return env[k] }

	cfg, err := LoadConfig([]string{}, getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "key123" || cfg.APISecret != "secret123" || cfg.APIPassphrase != "pass123" {
		t.Fatalf("credentials not loaded from env: %+v", cfg)
	}
	if cfg.DryRun != false {
		t.Fatalf("expected dry-run default false, got true")
	}
	if cfg.PollInterval != 5*time.Second {
		t.Fatalf("expected default interval 5s, got %v", cfg.PollInterval)
	}
}

func TestLoadConfig_FlagsOverride(t *testing.T) {
	env := map[string]string{
		"OKX_API_KEY":        "key123",
		"OKX_API_SECRET":     "secret123",
		"OKX_API_PASSPHRASE": "pass123",
	}
	getenv := func(k string) string { return env[k] }

	cfg, err := LoadConfig([]string{"--dry-run", "--interval", "10"}, getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DryRun {
		t.Fatalf("expected dry-run true")
	}
	if cfg.PollInterval != 10*time.Second {
		t.Fatalf("expected interval 10s, got %v", cfg.PollInterval)
	}
}

func TestLoadConfig_MissingCredentials(t *testing.T) {
	getenv := func(k string) string { return "" }
	_, err := LoadConfig([]string{}, getenv)
	if err == nil {
		t.Fatalf("expected error for missing credentials")
	}
}
