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

package internal

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// RedisClient provides functionalities to initialize and manage a Redis client.
type RedisClient struct {
	logger *zerolog.Logger // Logger for logging messages.
}

// NewRedisClient creates a new RedisClient instance with the provided logger.
//
// Parameters:
//
//	logger - logger instance for logging messages
//
// Returns:
//
//	*RedisClient - a new RedisClient instance
func NewRedisClient(logger *zerolog.Logger) *RedisClient {
	return &RedisClient{logger: logger}
}

// Initialize initializes a Redis client using the address specified in the environment
// variable "REDIS_ADDR". It pings the Redis server to ensure the connection is successful.
//
// Returns:
//
//	*redis.Client - a new Redis client instance
//	error - an error if the initialization or ping fails
func (r *RedisClient) Initialize() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: EnvVar("REDIS_ADDR"),
	})

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		r.logger.Fatal().Err(err).Msg("Failed to initialize redis client")
		return nil, err
	}

	return client, nil
}
