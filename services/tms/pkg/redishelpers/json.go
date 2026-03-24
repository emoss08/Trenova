package redishelpers

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
)

func GetJSON(ctx context.Context, client *redis.Client, key string, obj any) error {
	val, err := client.JSONGet(ctx, key, ".").Result()
	if err != nil {
		return err
	}

	return sonic.UnmarshalString(val, obj)
}

func SetJSON(
	ctx context.Context,
	client *redis.Client,
	key string,
	obj any,
	ttl time.Duration,
) error {
	data, err := sonic.Marshal(obj)
	if err != nil {
		return err
	}

	if err = client.JSONSet(ctx, key, ".", string(data)).Err(); err != nil {
		return err
	}

	return client.Expire(ctx, key, ttl).Err()
}

func PipelineSetJSON(
	ctx context.Context,
	pipe redis.Pipeliner,
	key string,
	obj any,
	ttl time.Duration,
) error {
	data, err := sonic.Marshal(obj)
	if err != nil {
		return err
	}

	pipe.JSONSet(ctx, key, ".", string(data))
	pipe.Expire(ctx, key, ttl)

	return nil
}
