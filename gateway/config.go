package main

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all gateway configuration loaded from environment variables.
type Config struct {
	// APIKey is the Bearer token for authentication.
	APIKey string

	// LlamaServerURL is the internal URL of the llama-server.
	LlamaServerURL string

	// GatewayPort is the port the gateway listens on.
	GatewayPort int
}

// LoadConfig reads configuration from environment variables with sensible defaults.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		APIKey:         os.Getenv("API_KEY"),
		LlamaServerURL: getEnvOrDefault("LLAMA_SERVER_URL", "http://llama-server:8081"),
	}

	// Parse port
	portStr := getEnvOrDefault("GATEWAY_PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid GATEWAY_PORT %q: %w", portStr, err)
	}
	cfg.GatewayPort = port

	// Validate required fields
	if cfg.APIKey == "" || cfg.APIKey == "change-me" {
		return nil, fmt.Errorf("API_KEY must be set (and not 'change-me')")
	}

	return cfg, nil
}

// getEnvOrDefault returns the value of an environment variable, or a default if not set.
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
