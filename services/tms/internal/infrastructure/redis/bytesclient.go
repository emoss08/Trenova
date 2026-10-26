package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type BytesClient struct {
	client *redis.Client
}

func NewBytesClient(client *redis.Client) BytesClient {
	return BytesClient{client: client}
}

func (c BytesClient) Get(ctx context.Context, key string) (value []byte, found bool, err error) {
	value, err = c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (c BytesClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c BytesClient) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
