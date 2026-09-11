package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Cosmess/mcpshield/internal/upstream"
)

type Config struct {
	HTTPAddr        string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	MaxBodyBytes    int64
	Upstreams       []upstream.Definition
	AuthIssuer      string
	AuthAudience    string
	AuthJWKSURL     string
	PolicyFile      string
}

func Load() (Config, error) {
	config := Config{
		HTTPAddr:        envOrDefault("MCP_SHIELD_HTTP_ADDR", "127.0.0.1:8080"),
		RequestTimeout:  10 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		MaxBodyBytes:    1 << 20,
	}

	var err error
	if config.RequestTimeout, err = durationEnv("MCP_SHIELD_REQUEST_TIMEOUT", config.RequestTimeout); err != nil {
		return Config{}, err
	}
	if config.ShutdownTimeout, err = durationEnv("MCP_SHIELD_SHUTDOWN_TIMEOUT", config.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	if config.MaxBodyBytes, err = int64Env("MCP_SHIELD_MAX_BODY_BYTES", config.MaxBodyBytes); err != nil {
		return Config{}, err
	}
	if config.RequestTimeout <= 0 {
		return Config{}, fmt.Errorf("MCP_SHIELD_REQUEST_TIMEOUT must be positive")
	}
	if config.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("MCP_SHIELD_SHUTDOWN_TIMEOUT must be positive")
	}
	if config.MaxBodyBytes <= 0 {
		return Config{}, fmt.Errorf("MCP_SHIELD_MAX_BODY_BYTES must be positive")
	}
	if config.HTTPAddr == "" {
		return Config{}, fmt.Errorf("MCP_SHIELD_HTTP_ADDR must not be empty")
	}
	config.AuthIssuer = os.Getenv("MCP_SHIELD_OIDC_ISSUER")
	config.AuthAudience = os.Getenv("MCP_SHIELD_OIDC_AUDIENCE")
	config.AuthJWKSURL = os.Getenv("MCP_SHIELD_OIDC_JWKS_URL")
	config.PolicyFile = os.Getenv("MCP_SHIELD_POLICY_FILE")
	authValues := 0
	for _, value := range []string{config.AuthIssuer, config.AuthAudience, config.AuthJWKSURL} {
		if value != "" {
			authValues++
		}
	}
	if authValues != 0 && authValues != 3 {
		return Config{}, fmt.Errorf("OIDC issuer, audience, and JWKS URL must be configured together")
	}
	if upstreamID := os.Getenv("MCP_SHIELD_UPSTREAM_ID"); upstreamID != "" {
		endpoint := os.Getenv("MCP_SHIELD_UPSTREAM_ENDPOINT")
		if endpoint == "" {
			return Config{}, fmt.Errorf("MCP_SHIELD_UPSTREAM_ENDPOINT must not be empty when MCP_SHIELD_UPSTREAM_ID is set")
		}
		config.Upstreams = []upstream.Definition{{
			ID:              upstreamID,
			Endpoint:        endpoint,
			Enabled:         true,
			Timeout:         config.RequestTimeout,
			ProtocolVersion: "2026-07-28",
		}}
	}
	return config, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration", key)
	}
	return parsed, nil
}

func int64Env(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", key)
	}
	return parsed, nil
}
