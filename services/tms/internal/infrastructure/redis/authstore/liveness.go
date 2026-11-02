package authstore

import (
	"context"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const livenessBatchSize = 500

const (
	createdAtPath           = ".createdAt"
	lastAccessedAtPath      = ".lastAccessedAt"
	accessConfirmedAtPath   = ".accessConfirmedAt"
	expiresAtPath           = ".expiresAt"
	secretHashPath          = ".secretHash"
	previousSecretHashPath  = ".previousSecretHash"
	previousSecretUntilPath = ".previousSecretUntil"
)

var livenessPaths = []string{
	createdAtPath,
	lastAccessedAtPath,
	expiresAtPath,
	secretHashPath,
	previousSecretHashPath,
	previousSecretUntilPath,
}

var unreadableVitalsLog = rate.Sometimes{Interval: time.Minute}

type sessionVitals struct {
	createdAt           int64
	lastAccessedAt      int64
	expiresAt           int64
	secretHash          string
	previousSecretHash  string
	previousSecretUntil int64
}

func (r *sessionRepository) Standings(
	ctx context.Context,
	sessionIDs []pulid.ID,
) ([]sessionports.SessionStanding, error) {
	now := timeutils.NowUnix()
	standings := make([]sessionports.SessionStanding, 0, len(sessionIDs))
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
			standings = append(standings, r.readStanding(batch[index], read, now))
		}
	}

	return standings, nil
}

func (r *sessionRepository) readStanding(
	sessionID pulid.ID,
	read *redis.JSONCmd,
	now int64,
) sessionports.SessionStanding {
	raw, err := read.Result()
	if redishelpers.IsRedisNil(err) {
		return sessionports.SessionStanding{ID: sessionID, Ended: true}
	}

	vitals, ok := sessionVitalsFrom(raw)
	if err != nil || !ok {
		unreadableVitalsLog.Do(func() {
			r.l.Warn(
				"could not read a session's vitals; keeping its streams open",
				zap.String("sessionID", sessionID.String()),
				zap.Error(err),
			)
		})
		return sessionports.SessionStanding{ID: sessionID}
	}

	sess := &session.Session{
		CreatedAt:           vitals.createdAt,
		LastAccessedAt:      vitals.lastAccessedAt,
		ExpiresAt:           vitals.expiresAt,
		SecretHash:          vitals.secretHash,
		PreviousSecretHash:  vitals.previousSecretHash,
		PreviousSecretUntil: vitals.previousSecretUntil,
	}
	sess.Bound(r.policy)
	if sess.IsExpiredAt(r.policy, now) {
		return sessionports.SessionStanding{ID: sessionID, Ended: true}
	}

	return sessionports.SessionStanding{
		ID:           sessionID,
		SecretsKnown: true,
		SecretHashes: sess.SecretHashesAt(now),
	}
}

func sessionVitalsFrom(raw string) (sessionVitals, bool) {
	var reply map[string]any
	if err := sonic.UnmarshalString(raw, &reply); err != nil {
		return sessionVitals{}, false
	}

	createdAt, createdOK := timestampAt(reply, createdAtPath)
	lastAccessedAt, accessedOK := timestampAt(reply, lastAccessedAtPath)
	expiresAt, expiresOK := timestampAt(reply, expiresAtPath)
	secretHash, secretOK := textAt(reply, secretHashPath)
	previousHash, previousOK := textAt(reply, previousSecretHashPath)
	previousUntil, untilOK := numberAt(reply, previousSecretUntilPath)
	if !createdOK || !accessedOK || !expiresOK || !secretOK || !previousOK || !untilOK ||
		secretHash == "" || previousUntil < 0 {
		return sessionVitals{}, false
	}

	return sessionVitals{
		createdAt:           createdAt,
		lastAccessedAt:      lastAccessedAt,
		expiresAt:           expiresAt,
		secretHash:          secretHash,
		previousSecretHash:  previousHash,
		previousSecretUntil: previousUntil,
	}, true
}

func timestampAt(reply map[string]any, path string) (int64, bool) {
	number, ok := numberAt(reply, path)
	if !ok || number <= 0 {
		return 0, false
	}
	return number, true
}

func numberAt(reply map[string]any, path string) (int64, bool) {
	value, ok := valueAt(reply, path)
	if !ok {
		return 0, false
	}
	number, isNumber := value.(float64)
	if !isNumber {
		return 0, false
	}
	return int64(number), true
}

func textAt(reply map[string]any, path string) (string, bool) {
	value, ok := valueAt(reply, path)
	if !ok {
		return "", false
	}
	text, isText := value.(string)
	return text, isText
}

func valueAt(reply map[string]any, path string) (any, bool) {
	value, ok := reply[path]
	if !ok {
		value, ok = reply["$"+path]
	}
	if !ok {
		return nil, false
	}
	if values, isList := value.([]any); isList {
		if len(values) != 1 {
			return nil, false
		}
		value = values[0]
	}
	return value, true
}
