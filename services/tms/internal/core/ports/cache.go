package ports

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheConnection interface {
	Client() *redis.Client
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Ping(ctx context.Context) error
	Close() error
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key string, values ...any) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	SAdd(ctx context.Context, key string, members ...any) error
	SRem(ctx context.Context, key string, members ...any) error
	SMembers(ctx context.Context, key string) ([]string, error)
	ZAdd(ctx context.Context, key string, members ...redis.Z) error
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	GetJSON(ctx context.Context, key string, dest any) error
	SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
	SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
	Pipeline() redis.Pipeliner
	ScriptLoad(ctx context.Context, script string) (string, error)
	ScriptExists(ctx context.Context, hashes ...string) ([]bool, error)
	EvalSha(ctx context.Context, sha string, keys []string, args ...any) (any, error)
	Eval(ctx context.Context, script string, keys []string, args ...any) (any, error)
	RunScript(ctx context.Context, scriptName string, keys []string, args ...any) (any, error)
}
