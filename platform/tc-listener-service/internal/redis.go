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
