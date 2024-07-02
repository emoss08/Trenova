package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"kafka/internal"
	"kafka/internal/services"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize the context
	ctx := context.Background()

	// Configure the file rotatelogs
	logPath := os.ExpandEnv("$HOME/logs/table_change_alert_listener.log") // Use a directory within the home directory
	rotator, err := rotatelogs.New(
		fmt.Sprintf("%s.%s", logPath, "%Y%m%d%H%M"),
		rotatelogs.WithLinkName(logPath),          // Generate a symlink to the latest log file
		rotatelogs.WithRotationTime(24*time.Hour), // Rotate every 24 hours
		rotatelogs.WithMaxAge(7*24*time.Hour),     // Keep logs for 7 days
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create rotatelogs")
	}

	// Configure zerolog with rotatelogs
	// logger := zerolog.New(rotator).With().Timestamp().Logger()

	// You can also output to both console and file by combining outputs
	multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stderr}, rotator)
	logger := zerolog.New(multi).With().Timestamp().Logger()

	// Initialize the database connection.
	db, err := internal.InitDB(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}

	// Initialize the redis connection.
	redisClient, err := internal.NewRedisClient(&logger).Initialize()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize redis client")
	}

	s := services.NewSubscriptionService(db, &logger, redisClient)

	services.StartListener(s, &logger)
}
