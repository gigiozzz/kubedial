package config

import (
	"os"
	"strconv"
)

// Config holds the kubecommander configuration
type Config struct {
	LogLevel    string
	ServerPort  int
	Namespace   string
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string
}

// Load loads configuration from environment variables
func Load() *Config {
	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		port = 8080
	}

	tlsEnabled, err := strconv.ParseBool(getEnv("TLS_ENABLED", "false"))
	if err != nil {
		tlsEnabled = false
	}

	return &Config{
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		ServerPort:  port,
		Namespace:   getEnv("NAMESPACE", "kubedial"),
		TLSEnabled:  tlsEnabled,
		TLSCertFile: getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  getEnv("TLS_KEY_FILE", ""),
		TLSCAFile:   getEnv("TLS_CA_FILE", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
