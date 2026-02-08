package provider

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogging initializes the zerolog logger and returns a context with the logger attached.
// It creates a new background context and calls InitLoggingWithContext.
func InitLogging() context.Context {
	return InitLoggingWithContext(context.Background())
}

// InitLoggingWithContext initializes the zerolog logger based on LOG_LEVEL environment variable
// and returns a context with the logger attached using zerolog's native context integration.
func InitLoggingWithContext(ctx context.Context) context.Context {
	level := os.Getenv("LOG_LEVEL")

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Caller().Logger()

	log.Logger = logger

	return logger.WithContext(ctx)
}

// FromContext returns the logger from the context using zerolog's log.Ctx().
// If no logger is found in context, it returns the global logger (log.Logger).
func FromContext(ctx context.Context) *zerolog.Logger {
	l := log.Ctx(ctx)
	if l.GetLevel() == zerolog.Disabled {
		return &log.Logger
	}
	return l
}
