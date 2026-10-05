package supportaccessservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const (
	startWindow    = time.Hour
	startKeyPrefix = "support_session_starts"
)

type StartLimiter interface {
	Allow(ctx context.Context, staffUserID pulid.ID, limit int) (bool, time.Duration, error)
}

type RedisStartLimiterParams struct {
	fx.In

	Client *redis.Client
}

type RedisStartLimiter struct {
	client *redis.Client
}

func NewRedisStartLimiter(p RedisStartLimiterParams) *RedisStartLimiter {
	return &RedisStartLimiter{client: p.Client}
}

func (l *RedisStartLimiter) Allow(
	ctx context.Context,
	staffUserID pulid.ID,
	limit int,
) (bool, time.Duration, error) {
	window := time.Now().Unix() / int64(startWindow.Seconds())
	key := fmt.Sprintf("%s:%s:%d", startKeyPrefix, staffUserID, window)

	pipe := l.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, startWindow)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, fmt.Errorf("count support session starts: %w", err)
	}

	if incr.Val() > int64(limit) {
		retry := time.Duration((window+1)*int64(startWindow.Seconds())-time.Now().Unix()) * time.Second
		return false, retry, nil
	}

	return true, 0, nil
}
