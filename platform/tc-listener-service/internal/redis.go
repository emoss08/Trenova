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
