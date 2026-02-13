package config

import (
	"os"
	"strconv"
	"strings"
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

	// APNs settings (for iOS devices)
	APNSKeyID      string
	APNSTeamID     string
	APNSPrivateKey string // PEM-encoded private key
	APNSProduction bool   // true for production, false for development
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Host:           "0.0.0.0",
		Port:           8080,
		DBPath:         "./data/abnotify.db",
		WSPingInterval: 30,
		WSPongTimeout:  60,
		EnableHTTPS:    false,
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	if host := os.Getenv("ABNOTIFY_HOST"); host != "" {
		cfg.Host = host
	}

	if port := os.Getenv("ABNOTIFY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if dbPath := os.Getenv("ABNOTIFY_DB_PATH"); dbPath != "" {
		cfg.DBPath = dbPath
	}

	if pingInterval := os.Getenv("ABNOTIFY_WS_PING_INTERVAL"); pingInterval != "" {
		if p, err := strconv.Atoi(pingInterval); err == nil {
			cfg.WSPingInterval = p
		}
	}

	if os.Getenv("ABNOTIFY_ENABLE_HTTPS") == "true" {
		cfg.EnableHTTPS = true
		cfg.CertFile = os.Getenv("ABNOTIFY_CERT_FILE")
		cfg.KeyFile = os.Getenv("ABNOTIFY_KEY_FILE")
	}

	// APNs configuration
	cfg.APNSKeyID = os.Getenv("APNS_KEY_ID")
	cfg.APNSTeamID = os.Getenv("APNS_TEAM_ID")
	// Handle \n escape sequences in private key (for environment variables)
	cfg.APNSPrivateKey = strings.ReplaceAll(os.Getenv("APNS_PRIVATE_KEY"), "\\n", "\n")
	cfg.APNSProduction = os.Getenv("APNS_PRODUCTION") != "false"

	return cfg
}
