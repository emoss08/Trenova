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
	logPath := os.ExpandEnv("$HOME/logs/table_change_alert_listener.log") // TODO: We might want this to be set via an environment variable.
	rotator, err := rotatelogs.New(
		fmt.Sprintf("%s.%s", logPath, "%Y%m%d%H%M"),
		rotatelogs.WithLinkName(logPath),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(7*24*time.Hour),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create rotatelogs")
	}

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
