package config

import (
	"os"
	"strconv"
)

// Config holds the kubecommander configuration
type Config struct {
	LogLevel   string
	ServerPort int
	Namespace  string
}

// Load loads configuration from environment variables
func Load() *Config {
	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		port = 8080
	}

	return &Config{
		LogLevel:   getEnv("LOG_LEVEL", "info"),
		ServerPort: port,
		Namespace:  getEnv("NAMESPACE", "kubedial"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
