// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
