package authstore

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
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

	Client         *redis.Client
	SecurityConfig *securityconfig.Config
	Logger         *zap.Logger
}

type sessionRepository struct {
	client *redis.Client
	policy session.Policy
	l      *zap.Logger
}

func NewSessionRepository(p SessionRepositoryParams) sessionports.SessionRepository {
	return &sessionRepository{
		client: p.Client,
		policy: session.Policy{
			MaxAge:      p.SecurityConfig.Session.MaxAge,
			IdleTimeout: p.SecurityConfig.Session.IdleTimeout,
		},
		l: p.Logger.Named("redis.session-repository"),
	}
}

func (r *sessionRepository) Get(ctx context.Context, sessionID pulid.ID) (*session.Session, error) {
	log := r.l.With(
		zap.String("operation", "GetSession"),
		zap.String("sessionID", sessionID.String()),
	)

	sess := new(session.Session)
	if err := redishelpers.GetJSON(ctx, r.client, r.getSessionKey(sessionID), sess); err != nil {
		if redishelpers.IsRedisNil(err) {
			return nil, sessionports.ErrSessionNotFound
		}
		log.Error("failed to get session from cache", zap.Error(err))
		return nil, err
	}

	sess.Bound(r.policy)
	if sess.IsExpiredAt(r.policy, timeutils.NowUnix()) {
		log.Debug("session has ended")
		r.discard(ctx, sess)
		return nil, sessionports.ErrSessionNotFound
	}

	return sess, nil
}

func (r *sessionRepository) discard(ctx context.Context, sess *session.Session) {
	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.getSessionKey(sess.ID))
	pipe.ZRem(ctx, r.getUserSessionsKey(sess.UserID), sess.ID.String())
	if _, err := pipe.Exec(ctx); err != nil {
		r.l.Warn(
			"failed to discard an ended session",
			zap.String("sessionID", sess.ID.String()),
			zap.Error(err),
		)
	}
}

func (r *sessionRepository) Create(ctx context.Context, sess *session.Session) error {
	log := r.l.With(
		zap.String("operation", "CreateSession"),
	)

	now := timeutils.NowUnix()
	if err := sess.ValidateAt(r.policy, now); err != nil {
		return err
	}
	ttl := max(time.Duration(sess.ExpiresAt-now)*time.Second, time.Second)

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
		ttl,
	); err != nil {
		log.Error("failed to marshal session", zap.Error(err))
		return err
	}

	pipe.ZAdd(ctx, userSessionsKey, redis.Z{
		Score:  float64(sess.CreatedAt),
		Member: sess.ID.String(),
	})
	pipe.Expire(ctx, userSessionsKey, ttl)

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

func (r *sessionRepository) DeleteAllForUser(ctx context.Context, userID pulid.ID) error {
	log := r.l.With(
		zap.String("operation", "DeleteAllUserSessions"),
		zap.String("userID", userID.String()),
	)

	userSessionsKey := r.getUserSessionsKey(userID)
	members, err := r.client.ZRange(ctx, userSessionsKey, 0, -1).Result()
	if err != nil {
		log.Error("failed to list user sessions", zap.Error(err))
		return err
	}

	pipe := r.client.Pipeline()
	for _, member := range members {
		sessionID, parseErr := pulid.Parse(member)
		if parseErr != nil {
			// A member that will not parse cannot be turned into a session key, but it
			// also cannot authenticate anything. Drop it with the set below rather than
			// abandoning the sessions that follow it.
			log.Warn("skipping unparseable session id", zap.String("member", member))
			continue
		}
		pipe.Del(ctx, r.getSessionKey(sessionID))
	}
	pipe.Del(ctx, userSessionsKey)

	if _, err = pipe.Exec(ctx); err != nil {
		log.Error("failed to delete user sessions", zap.Error(err))
		return err
	}

	return nil
}

func (r *sessionRepository) ListForUser(
	ctx context.Context,
	userID pulid.ID,
) ([]*session.Session, error) {
	log := r.l.With(
		zap.String("operation", "ListForUser"),
		zap.String("userID", userID.String()),
	)

	userSessionsKey := r.getUserSessionsKey(userID)
	members, err := r.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   userSessionsKey,
		Start: 0,
		Stop:  -1,
		Rev:   true,
	}).Result()
	if err != nil {
		log.Error("failed to list user sessions", zap.Error(err))
		return nil, err
	}
	if len(members) == 0 {
		return []*session.Session{}, nil
	}

	pipe := r.client.Pipeline()
	reads := make([]*redis.JSONCmd, 0, len(members))
	ids := make([]string, 0, len(members))
	for _, member := range members {
		sessionID, parseErr := pulid.Parse(member)
		if parseErr != nil {
			continue
		}
		reads = append(reads, pipe.JSONGet(ctx, r.getSessionKey(sessionID), "."))
		ids = append(ids, member)
	}
	if _, err = pipe.Exec(ctx); err != nil && !redishelpers.IsRedisNil(err) {
		log.Error("failed to read user sessions", zap.Error(err))
		return nil, err
	}

	now := timeutils.NowUnix()
	sessions := make([]*session.Session, 0, len(reads))
	stale := make([]any, 0)
	for index, read := range reads {
		raw, readErr := read.Result()
		if readErr != nil || raw == "" {
			stale = append(stale, ids[index])
			continue
		}
		sess := new(session.Session)
		if decodeErr := sonic.UnmarshalString(raw, sess); decodeErr != nil {
			log.Warn("skipping undecodable session", zap.String("member", ids[index]))
			continue
		}
		sess.Bound(r.policy)
		if sess.UserID != userID || sess.IsExpiredAt(r.policy, now) {
			stale = append(stale, ids[index])
			continue
		}
		sessions = append(sessions, sess)
	}

	if len(stale) > 0 {
		if remErr := r.client.ZRem(ctx, userSessionsKey, stale...).Err(); remErr != nil {
			log.Warn("failed to prune stale session ids", zap.Error(remErr))
		}
	}

	return sessions, nil
}

func (r *sessionRepository) Touch(ctx context.Context, sessionID pulid.ID, at int64) error {
	err := r.client.JSONSet(ctx, r.getSessionKey(sessionID), ".lastAccessedAt", at).Err()
	if err != nil && !redishelpers.IsRedisNil(err) {
		r.l.Warn(
			"failed to touch session",
			zap.String("sessionID", sessionID.String()),
			zap.Error(err),
		)
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
