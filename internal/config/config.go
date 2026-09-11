package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	MaxBodyBytes    int64
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
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
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
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}
