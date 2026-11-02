package authstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/hashutils"
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
	maxUpdateAttempts  = 3
)

type SessionRepositoryParams struct {
	fx.In

	Client         *redis.Client
	SecurityConfig *securityconfig.Config
	Logger         *zap.Logger
}

type sessionRepository struct {
	client  *redis.Client
	policy  session.Policy
	sealKey []byte
	l       *zap.Logger
}

func NewSessionRepository(p SessionRepositoryParams) sessionports.SessionRepository {
	return newSessionRepository(p)
}

func NewSessionLiveness(p SessionRepositoryParams) sessionports.SessionLiveness {
	return newSessionRepository(p)
}

func newSessionRepository(p SessionRepositoryParams) *sessionRepository {
	return &sessionRepository{
		client: p.Client,
		policy: session.Policy{
			MaxAge:      p.SecurityConfig.Session.MaxAge,
			IdleTimeout: p.SecurityConfig.Session.IdleTimeout,
		},
		sealKey: hashutils.DeriveKey(p.SecurityConfig.Session.Secret, sealKeyLabel),
		l:       p.Logger.Named("redis.session-repository"),
	}
}

func (r *sessionRepository) Get(ctx context.Context, sessionID pulid.ID) (*session.Session, error) {
	log := r.l.With(
		zap.String("operation", "GetSession"),
		zap.String("sessionID", sessionID.String()),
	)

	sess := new(session.Session)
	if err := redishelpers.GetJSON(ctx, r.client, sessionKey(sessionID), sess); err != nil {
		if redishelpers.IsRedisNil(err) {
			return nil, sessionports.ErrSessionNotFound
		}
		log.Error("failed to get session from cache", zap.Error(err))
		return nil, err
	}
	if !r.sealHolds(ctx, sess) {
		return nil, sessionports.ErrSessionNotFound
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
	pipe.Del(ctx, sessionKey(sess.ID))
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

	if err := r.evictOldestSessions(ctx, userSessionsKey, maxSessionsPerUser); err != nil {
		log.Error("failed to evict old sessions", zap.Error(err))
		return err
	}

	if err := r.seal(sess); err != nil {
		log.Error("failed to seal session", zap.Error(err))
		return err
	}

	pipe := r.client.TxPipeline()

	sessionKey := sessionKey(sess.ID)
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

func (r *sessionRepository) Update(
	ctx context.Context,
	sessionID pulid.ID,
	mutate func(*session.Session) error,
) (*session.Session, error) {
	key := sessionKey(sessionID)
	for range maxUpdateAttempts {
		var updated *session.Session
		err := r.client.Watch(ctx, func(tx *redis.Tx) error {
			sess, err := r.readLive(ctx, tx, key)
			if err != nil {
				return err
			}
			identity := sessionIdentityOf(sess)
			if err = mutate(sess); err != nil {
				return err
			}
			if sessionIdentityOf(sess) != identity {
				return errIdentityChanged
			}

			now := timeutils.NowUnix()
			sess.LastAccessedAt = now
			sess.UpdatedAt = now
			if err = r.seal(sess); err != nil {
				return err
			}
			data, err := sonic.MarshalString(sess)
			if err != nil {
				return err
			}

			if _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.JSONSet(ctx, key, ".", data)
				pipe.ExpireAt(ctx, key, time.Unix(sess.ExpiresAt, 0))
				return nil
			}); err != nil {
				return err
			}

			updated = sess
			return nil
		}, key)
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		if err != nil {
			return nil, err
		}

		return updated, nil
	}

	r.l.Warn(
		"gave up updating a session that kept changing",
		zap.String("sessionID", sessionID.String()),
	)
	return nil, fmt.Errorf("update session %s: %w", sessionID, redis.TxFailedErr)
}

var errIdentityChanged = errors.New("a session update may not change who or when the session is")

type sessionIdentity struct {
	id        pulid.ID
	userID    pulid.ID
	createdAt int64
}

func sessionIdentityOf(sess *session.Session) sessionIdentity {
	return sessionIdentity{id: sess.ID, userID: sess.UserID, createdAt: sess.CreatedAt}
}

func (r *sessionRepository) readLive(
	ctx context.Context,
	tx *redis.Tx,
	key string,
) (*session.Session, error) {
	raw, err := tx.JSONGet(ctx, key, ".").Result()
	if redishelpers.IsRedisNil(err) || (err == nil && raw == "") {
		return nil, sessionports.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	sess := new(session.Session)
	if err = sonic.UnmarshalString(raw, sess); err != nil {
		return nil, err
	}
	if !r.sealHolds(ctx, sess) {
		return nil, sessionports.ErrSessionNotFound
	}

	sess.Bound(r.policy)
	if sess.IsExpiredAt(r.policy, timeutils.NowUnix()) {
		r.discard(ctx, sess)
		return nil, sessionports.ErrSessionNotFound
	}

	return sess, nil
}

func (r *sessionRepository) Delete(ctx context.Context, sessionID pulid.ID) error {
	log := r.l.With(
		zap.String("operation", "DeleteSession"),
		zap.String("sessionID", sessionID.String()),
	)

	sess := new(session.Session)
	if err := redishelpers.GetJSON(ctx, r.client, sessionKey(sessionID), sess); err != nil {
		if redishelpers.IsRedisNil(err) {
			return nil
		}
		log.Error("failed to get session", zap.Error(err))
		return err
	}

	pipe := r.client.Pipeline()
	pipe.Del(ctx, sessionKey(sessionID))
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
			log.Warn("skipping unparseable session id", zap.String("member", member))
			continue
		}
		pipe.Del(ctx, sessionKey(sessionID))
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
		reads = append(reads, pipe.JSONGet(ctx, sessionKey(sessionID), "."))
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
		if !r.sealHolds(ctx, sess) {
			stale = append(stale, ids[index])
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

func (r *sessionRepository) ConfirmAccess(
	ctx context.Context,
	sessionID pulid.ID,
	at int64,
) error {
	key := sessionKey(sessionID)
	pipe := r.client.TxPipeline()
	pipe.JSONSet(ctx, key, lastAccessedAtPath, at)
	pipe.JSONSet(ctx, key, accessConfirmedAtPath, at)
	if _, err := pipe.Exec(ctx); err != nil && !redishelpers.IsRedisNil(err) {
		r.l.Warn(
			"failed to record a session's confirmed access",
			zap.String("sessionID", sessionID.String()),
			zap.Error(err),
		)
		return err
	}

	return nil
}

func (r *sessionRepository) Touch(ctx context.Context, sessionID pulid.ID, at int64) error {
	err := r.client.JSONSet(ctx, sessionKey(sessionID), ".lastAccessedAt", at).Err()
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
		sessionKeys[i] = sessionKey(sessionID)
	}

	return r.client.Del(ctx, sessionKeys...).Err()
}

func sessionKey(sessionID pulid.ID) string {
	return fmt.Sprintf("%s:%s", sessionKeyPrefix, sessionID.String())
}

func (r *sessionRepository) getUserSessionsKey(userID pulid.ID) string {
	return fmt.Sprintf("%s:%s", userSessionsPrefix, userID.String())
}
