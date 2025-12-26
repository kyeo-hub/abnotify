package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	// Server settings
	Host string
	Port int

	// Database
	DBPath string

	// WebSocket settings
	WSPingInterval int // seconds
	WSPongTimeout  int // seconds

	// Security
	EnableHTTPS bool
	CertFile    string
	KeyFile     string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Host:           "0.0.0.0",
		Port:           8080,
		DBPath:         "./data/accnotify.db",
		WSPingInterval: 30,
		WSPongTimeout:  60,
		EnableHTTPS:    false,
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	if host := os.Getenv("LSPNOTIFY_HOST"); host != "" {
		cfg.Host = host
	}

	if port := os.Getenv("LSPNOTIFY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if dbPath := os.Getenv("LSPNOTIFY_DB_PATH"); dbPath != "" {
		cfg.DBPath = dbPath
	}

	if pingInterval := os.Getenv("LSPNOTIFY_WS_PING_INTERVAL"); pingInterval != "" {
		if p, err := strconv.Atoi(pingInterval); err == nil {
			cfg.WSPingInterval = p
		}
	}

	if os.Getenv("LSPNOTIFY_ENABLE_HTTPS") == "true" {
		cfg.EnableHTTPS = true
		cfg.CertFile = os.Getenv("LSPNOTIFY_CERT_FILE")
		cfg.KeyFile = os.Getenv("LSPNOTIFY_KEY_FILE")
	}

	return cfg
}
