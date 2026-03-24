package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	sessionKeyPrefix   = "session"
	userSessionsPrefix = "user_sessions"
	maxSessionsPerUser = 5
)

type SessionRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type sessionRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewSessionRepository(p SessionRepositoryParams) repositories.SessionRepository {
	return &sessionRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.session-repository"),
	}
}

func (r *sessionRepository) Get(ctx context.Context, sessionID pulid.ID) (*session.Session, error) {
	log := r.l.With(
		zap.String("operation", "GetSession"),
		zap.String("sessionID", sessionID.String()),
	)

	sess := new(session.Session)
	if err := redishelpers.GetJSON(ctx, r.client, r.getSessionKey(sessionID), sess); err != nil {
		log.Error("failed to get session from cache", zap.Error(err))
		return nil, err
	}

	if err := sess.Validate(); err != nil {
		log.Error("session is not valid", zap.Error(err))
		return nil, err
	}

	return sess, nil
}

func (r *sessionRepository) Create(ctx context.Context, sess *session.Session) error {
	log := r.l.With(
		zap.String("operation", "CreateSession"),
	)

	userSessionsKey := r.getUserSessionsKey(sess.UserID)

	// NOTE: there is a small race window between eviction check and session creation.
	// concurrent logins may briefly exceed maxSessionsPerUser; the next login will clean up.
	if err := r.evictOldestSessions(ctx, userSessionsKey, maxSessionsPerUser); err != nil {
		log.Error("failed to evict old sessions", zap.Error(err))
		return err
	}

	pipe := r.client.Pipeline()

	sessionKey := r.getSessionKey(sess.ID)
	if err := redishelpers.PipelineSetJSON(
		ctx,
		pipe,
		sessionKey,
		sess,
		session.DefaultTTL,
	); err != nil {
		log.Error("failed to marshal session", zap.Error(err))
		return err
	}

	pipe.ZAdd(ctx, userSessionsKey, redis.Z{
		Score:  float64(sess.CreatedAt),
		Member: sess.ID.String(),
	})
	pipe.Expire(ctx, userSessionsKey, session.DefaultTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		log.Error("failed to create session in redis", zap.Error(err))
		return err
	}

	return nil
}

func (r *sessionRepository) Update(ctx context.Context, sess *session.Session) error {
	log := r.l.With(
		zap.String("operation", "UpdateSession"),
		zap.String("sessionID", sess.ID.String()),
	)

	remainingTTL := time.Duration(sess.ExpiresAt-timeutils.NowUnix()) * time.Second
	if remainingTTL <= 0 {
		log.Warn("session has expired, cannot update")
		return nil
	}

	sess.LastAccessedAt = timeutils.NowUnix()

	if err := redishelpers.SetJSON(
		ctx,
		r.client,
		r.getSessionKey(sess.ID),
		sess,
		remainingTTL,
	); err != nil {
		log.Error("failed to update session in redis", zap.Error(err))
		return err
	}

	return nil
}

func (r *sessionRepository) Delete(ctx context.Context, sessionID pulid.ID) error {
	log := r.l.With(
		zap.String("operation", "DeleteSession"),
		zap.String("sessionID", sessionID.String()),
	)

	sess := new(session.Session)
	if err := redishelpers.GetJSON(ctx, r.client, r.getSessionKey(sessionID), sess); err != nil {
		log.Error("failed to get session", zap.Error(err))
		return err
	}

	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.getSessionKey(sessionID))
	pipe.ZRem(ctx, r.getUserSessionsKey(sess.UserID), sessionID.String())

	if _, err := pipe.Exec(ctx); err != nil {
		log.Error("failed to delete session", zap.Error(err))
		return err
	}

	return nil
}

func (r *sessionRepository) evictOldestSessions(
	ctx context.Context,
	userSessionsKey string,
	maxAllowed int64,
) error {
	count, err := r.client.ZCard(ctx, userSessionsKey).Result()
	if err != nil {
		return err
	}

	if count < maxAllowed {
		return nil
	}

	toEvict := count - maxAllowed + 1
	evicted, err := r.client.ZPopMin(ctx, userSessionsKey, toEvict).Result()
	if err != nil {
		return err
	}

	if len(evicted) == 0 {
		return nil
	}

	sessionKeys := make([]string, len(evicted))
	for i, z := range evicted {
		sessionID, _ := pulid.Parse(
			z.Member.(string), //nolint:errcheck // we don't care about the error here
		)
		sessionKeys[i] = r.getSessionKey(sessionID)
	}

	return r.client.Del(ctx, sessionKeys...).Err()
}

func (r *sessionRepository) getSessionKey(sessionID pulid.ID) string {
	return fmt.Sprintf("%s:%s", sessionKeyPrefix, sessionID.String())
}

func (r *sessionRepository) getUserSessionsKey(userID pulid.ID) string {
	return fmt.Sprintf("%s:%s", userSessionsPrefix, userID.String())
}
