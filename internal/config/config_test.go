package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"MCP_SHIELD_HTTP_ADDR", "MCP_SHIELD_REQUEST_TIMEOUT", "MCP_SHIELD_SHUTDOWN_TIMEOUT", "MCP_SHIELD_MAX_BODY_BYTES"} {
		t.Setenv(key, "")
	}
	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.HTTPAddr != "127.0.0.1:8080" || config.MaxBodyBytes <= 0 {
		t.Fatalf("unexpected defaults: %+v", config)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	secretValue := "not-a-duration-secret-value"
	t.Setenv("MCP_SHIELD_REQUEST_TIMEOUT", secretValue)
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid duration error")
	} else if strings.Contains(err.Error(), secretValue) {
		t.Fatalf("Load() error exposed configuration value: %v", err)
	}
}

func TestLoadRejectsNonPositiveBodyLimit(t *testing.T) {
	t.Setenv("MCP_SHIELD_MAX_BODY_BYTES", "0")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want body limit error")
	}
}
