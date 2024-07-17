// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
