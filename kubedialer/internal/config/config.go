package config

import (
	"os"
	"time"
)

// Config holds the kubedialer configuration
type Config struct {
	LogLevel     string
	CommanderURL string
	AgentToken   string
	AgentName    string
	PollInterval time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	pollInterval, err := time.ParseDuration(getEnv("POLL_INTERVAL", "10s"))
	if err != nil {
		pollInterval = 10 * time.Second
	}

	return &Config{
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		CommanderURL: getEnv("COMMANDER_URL", ""),
		AgentToken:   getEnv("AGENT_TOKEN", ""),
		AgentName:    getEnv("AGENT_NAME", ""),
		PollInterval: pollInterval,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
