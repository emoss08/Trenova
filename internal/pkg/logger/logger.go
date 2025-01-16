package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/pkg/config"
)

type Logger struct {
	*zerolog.Logger
}

func NewLogger(cfg *config.Config) *Logger {
	var output zerolog.ConsoleWriter

	if cfg.App.Environment == "development" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	} else {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    true,
		}
	}

	// Parse log level from config
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.App.LogLevel))
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	logger := zerolog.New(output).
		With().
		Caller().
		Timestamp().
		Str("app", cfg.App.Name).
		Str("version", cfg.App.Version).
		Str("env", cfg.App.Environment).
		Logger()

	return &Logger{Logger: &logger}
}
