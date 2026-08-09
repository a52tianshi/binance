package main

import (
	"flag"
	"fmt"
	"time"
)

type Config struct {
	APIKey        string
	APISecret     string
	APIPassphrase string
	DryRun        bool
	PollInterval  time.Duration
}

func LoadConfig(args []string, getenv func(string) string) (Config, error) {
	fs := flag.NewFlagSet("okx_put_quoter", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "log intended amends without sending them")
	intervalSec := fs.Int("interval", 5, "poll interval in seconds")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg := Config{
		APIKey:        getenv("OKX_API_KEY"),
		APISecret:     getenv("OKX_API_SECRET"),
		APIPassphrase: getenv("OKX_API_PASSPHRASE"),
		DryRun:        *dryRun,
		PollInterval:  time.Duration(*intervalSec) * time.Second,
	}

	if cfg.APIKey == "" || cfg.APISecret == "" || cfg.APIPassphrase == "" {
		return Config{}, fmt.Errorf("OKX_API_KEY, OKX_API_SECRET and OKX_API_PASSPHRASE must be set")
	}
	return cfg, nil
}
