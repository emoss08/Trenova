package redis

import (
	"context"
	"time"

	goRedis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Client struct {
	client *goRedis.Client
	addr   string
	logger *zerolog.Logger
}

func NewClient(addr string, logger *zerolog.Logger) *Client {
	client := &Client{addr: addr, logger: logger}

	client.initialize()
	return client
}

func (r *Client) initialize() {
	client := goRedis.NewClient(&goRedis.Options{
		Addr:        r.addr,
		DialTimeout: 5 * time.Second,
	})

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		r.logger.Fatal().Err(err).Msg("Failed to initialize redis client")
	}

	r.client = client
}

func (r *Client) CacheByKey(ctx context.Context, key string, value string) error {
	client := goRedis.NewClient(&goRedis.Options{
		Addr: r.addr,
	})

	err := client.Set(ctx, key, value, 0).Err()
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
