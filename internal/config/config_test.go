package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_NAME", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if cfg.App.Name != "ms-storage" || cfg.HTTP.Port != "8080" || cfg.Log.Level != "info" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadInvalid(t *testing.T) {
	cases := map[string]map[string]string{
		"bad port":   {"HTTP_PORT": "abc"},
		"bad level":  {"LOG_LEVEL": "verbose"},
		"bad format": {"LOG_FORMAT": "yaml"},
		"bad env":    {"APP_ENV": "qa"},
	}

	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			for k, v := range env {
				t.Setenv(k, v)
			}
			_, err := Load()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "invalid configuration") {
				t.Fatalf("expected invalid config error, got %v", err)
			}
		})
	}
}
