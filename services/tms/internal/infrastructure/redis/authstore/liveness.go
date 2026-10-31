package authstore

import (
	"context"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const livenessBatchSize = 500

const (
	createdAtPath      = ".createdAt"
	lastAccessedAtPath = ".lastAccessedAt"
	expiresAtPath      = ".expiresAt"
)

var livenessPaths = []string{createdAtPath, lastAccessedAtPath, expiresAtPath}

var unreadableClockLog = rate.Sometimes{Interval: time.Minute}

type sessionClock struct {
	createdAt      int64
	lastAccessedAt int64
	expiresAt      int64
}

func (r *sessionRepository) Ended(
	ctx context.Context,
	sessionIDs []pulid.ID,
) ([]pulid.ID, error) {
	now := timeutils.NowUnix()
	ended := make([]pulid.ID, 0)
	for batch := range slices.Chunk(sessionIDs, livenessBatchSize) {
		pipe := r.client.Pipeline()
		reads := make([]*redis.JSONCmd, len(batch))
		for index, sessionID := range batch {
			reads[index] = pipe.JSONGet(ctx, sessionKey(sessionID), livenessPaths...)
		}
		if _, err := pipe.Exec(ctx); err != nil && !redishelpers.IsRedisNil(err) {
			return nil, err
		}
		for index, read := range reads {
			if r.readEnded(read, now) {
				ended = append(ended, batch[index])
			}
		}
	}

	return ended, nil
}

func (r *sessionRepository) readEnded(read *redis.JSONCmd, now int64) bool {
	raw, err := read.Result()
	if redishelpers.IsRedisNil(err) {
		return true
	}

	clock, ok := sessionClockFrom(raw)
	if err != nil || !ok {
		unreadableClockLog.Do(func() {
			r.l.Warn(
				"could not read a session's clock; keeping its streams open",
				zap.String("reply", raw),
				zap.Error(err),
			)
		})
		return false
	}

	sess := &session.Session{
		CreatedAt:      clock.createdAt,
		LastAccessedAt: clock.lastAccessedAt,
		ExpiresAt:      clock.expiresAt,
	}
	sess.Bound(r.policy)
	return sess.IsExpiredAt(r.policy, now)
}

func sessionClockFrom(raw string) (sessionClock, bool) {
	var reply map[string]any
	if err := sonic.UnmarshalString(raw, &reply); err != nil {
		return sessionClock{}, false
	}

	createdAt, createdOK := timestampAt(reply, createdAtPath)
	lastAccessedAt, accessedOK := timestampAt(reply, lastAccessedAtPath)
	expiresAt, expiresOK := timestampAt(reply, expiresAtPath)
	if !createdOK || !accessedOK || !expiresOK {
		return sessionClock{}, false
	}

	return sessionClock{
		createdAt:      createdAt,
		lastAccessedAt: lastAccessedAt,
		expiresAt:      expiresAt,
	}, true
}

func timestampAt(reply map[string]any, path string) (int64, bool) {
	value, ok := reply[path]
	if !ok {
		value, ok = reply["$"+path]
	}
	if !ok {
		return 0, false
	}
	if values, isList := value.([]any); isList {
		if len(values) != 1 {
			return 0, false
		}
		value = values[0]
	}

	number, isNumber := value.(float64)
	if !isNumber || number <= 0 {
		return 0, false
	}
	return int64(number), true
}
