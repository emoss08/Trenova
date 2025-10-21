package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	sessionPrefix      = "session:"
	userSessionsPrefix = "user_sessions:"
	defaultSessionTTL  = 24 * time.Hour
)

type SessionRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type sessionRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewSessionRepository(p SessionRepositoryParams) repositories.SessionRepository {
	return &sessionRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.session-repository"),
	}
}

func (sr *sessionRepository) GetValidSession(
	ctx context.Context,
	req repositories.GetValidSessionRequest,
) (*session.Session, error) {
	log := sr.l.With(zap.String("operation", "GetValidSession"))

	sess := new(session.Session)
	if err := sr.cache.GetJSON(ctx, sr.getSessionKey(req.SessionID), sess); err != nil {
		log.Error("failed to get session from cache", zap.Error(err))
		return nil, err
	}

	if err := sess.Validate(req.ClientIP); err != nil {
		log.Warn("invalid session", zap.String("session_id", req.SessionID.String()))
		return nil, err
	}

	return sess, nil
}

func (sr *sessionRepository) GetUserActiveSessions(
	ctx context.Context,
	userID pulid.ID,
) ([]*session.Session, error) {
	log := sr.l.With(zap.String("operation", "GetUserActiveSessions"))

	sessionIDs, err := sr.cache.SMembers(ctx, sr.getUserSessionsKey(userID))
	if err != nil {
		log.Error("failed to get user active sessions", zap.Error(err))
		return nil, err
	}

	sessions := make([]*session.Session, 0, len(sessionIDs))
	for _, sID := range sessionIDs {
		sessionID, sErr := pulid.Parse(sID)
		if sErr != nil {
			continue
		}

		sess := new(session.Session)
		if err = sr.cache.GetJSON(ctx, sr.getSessionKey(sessionID), sess); err != nil {
			continue
		}

		if sess.IsValid() {
			sessions = append(sessions, sess)
		}
	}

	return sessions, nil
}

func (sr *sessionRepository) Create(ctx context.Context, sess *session.Session) error {
	log := sr.l.With(zap.String("operation", "Create"))

	pipe := sr.cache.Client().Pipeline()

	sessionKey := sr.getSessionKey(sess.ID)
	if err := sr.cache.SetJSON(ctx, sessionKey, sess, defaultSessionTTL); err != nil {
		log.Error("failed to set session in cache", zap.Error(err))
		return err
	}

	userSessionsKey := sr.getUserSessionsKey(sess.UserID)
	pipe.SAdd(ctx, userSessionsKey, sess.ID.String())
	pipe.Expire(ctx, userSessionsKey, defaultSessionTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		log.Error("failed to execute pipeline", zap.Error(err))
		return err
	}

	return nil
}

func (sr *sessionRepository) UpdateSessionActivity(
	ctx context.Context,
	req *repositories.UpdateSessionActivityRequest,
) error {
	log := sr.l.With(zap.String("operation", "UpdateSessionActivity"))

	sess, err := sr.GetValidSession(ctx, repositories.GetValidSessionRequest{
		SessionID: req.SessionID,
		ClientIP:  req.ClientIP,
	})
	if err != nil {
		log.Error("failed to get valid session", zap.Error(err))
		return err
	}

	sess.UpdateLastAccessedAt()

	if err = sr.cache.SetJSON(ctx, sr.getSessionKey(sess.ID), sess, defaultSessionTTL); err != nil {
		log.Error("failed to set session in cache", zap.Error(err))
		return err
	}

	return nil
}

func (sr *sessionRepository) UpdateSessionOrganization(
	ctx context.Context,
	req repositories.UpdateSessionOrganizationRequest,
) error {
	log := sr.l.With(zap.String("operation", "UpdateSessionOrganization"))

	sess := new(session.Session)
	if err := sr.cache.GetJSON(ctx, sr.getSessionKey(req.SessionID), sess); err != nil {
		log.Error("failed to get session", zap.Error(err))
		return err
	}

	sess.OrganizationID = req.NewOrgID
	sess.UpdatedAt = utils.NowUnix()

	if err := sr.cache.SetJSON(ctx, sr.getSessionKey(sess.ID), sess, defaultSessionTTL); err != nil {
		log.Error("failed to set session in cache", zap.Error(err))
		return err
	}

	return nil
}

func (sr *sessionRepository) RevokeSession(
	ctx context.Context,
	req repositories.RevokeSessionRequest,
) error {
	log := sr.l.With(zap.String("operation", "RevokeSession"))

	sess, err := sr.GetValidSession(ctx, repositories.GetValidSessionRequest{
		SessionID: req.SessionID,
		ClientIP:  req.ClientIP,
	})
	if err != nil {
		log.Error("failed to get valid session", zap.Error(err))
		return err
	}

	sess.Revoke()

	if err = sr.cache.SetJSON(ctx, sr.getSessionKey(sess.ID), sess, defaultSessionTTL); err != nil {
		log.Error("failed to set session in cache", zap.Error(err))
		return err
	}

	if err = sr.cache.SRem(ctx, sr.getUserSessionsKey(sess.UserID), sess.ID.String()); err != nil {
		log.Error("failed to remove session from user sessions", zap.Error(err))
		return err
	}

	return nil
}

func (sr *sessionRepository) getSessionKey(sessionID pulid.ID) string {
	return fmt.Sprintf("%s%s", sessionPrefix, sessionID.String())
}

func (sr *sessionRepository) getUserSessionsKey(userID pulid.ID) string {
	return fmt.Sprintf("%s%s", userSessionsPrefix, userID.String())
}
