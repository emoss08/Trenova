package redis

import (
	"context"

	goRedis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Client struct {
	client  *goRedis.Client
	logger  *zerolog.Logger
	options *goRedis.Options
}

// Options is a type alias for go-redis Options.
type Options = goRedis.Options

func NewClient(options *goRedis.Options, logger *zerolog.Logger) *Client {
	client := &Client{options: options, logger: logger}

	if err := client.initialize(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize redis client")
		return nil
	}

	return client
}

func (r *Client) initialize() error {
	client := goRedis.NewClient(r.options)

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		r.logger.Fatal().Err(err).Msg("Failed to initialize redis client")
		return err
	}

	r.client = client

	return nil
}

func (r *Client) CacheByKey(ctx context.Context, key string, value string) error {
	err := r.client.Set(ctx, key, value, 0).Err()
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to cache key")
		return err
	}

	return nil
}

func (r *Client) FetchFromCacheByKey(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to fetch key from cache")
		return "", err
	}

	return val, nil
}

func (r *Client) InvalidateCacheByKey(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to invalidate cache")
		return err
	}

	return nil
}

func (r *Client) Close() error {
	if err := r.client.Close(); err != nil {
		r.logger.Error().Err(err).Msg("Failed to close redis client")
		return err
	}

	return nil
}
