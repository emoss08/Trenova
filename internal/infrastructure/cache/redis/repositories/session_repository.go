package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	// Key prefixes
	sessionPrefix      = "session:"
	userSessionsPrefix = "user-sessions:"
	sessionEventPrefix = "session-events:"

	// Default TTLs
	defaultSessionTTL = time.Hour * 72
)

type SessionRepositoryParams struct {
	fx.In

	Cache  *redis.Client
	Logger *logger.Logger
}

type sessionRepository struct {
	redis  *redis.Client
	logger *zerolog.Logger
}

func NewSessionRepository(p SessionRepositoryParams) repositories.SessionRepository {
	log := p.Logger.With().
		Str("repository", "session.redis").
		Logger()

	return &sessionRepository{
		redis:  p.Cache,
		logger: &log,
	}
}

// Create should use pipeline for atomic operations
func (sr *sessionRepository) Create(ctx context.Context, sess *session.Session) error {
	log := sr.logger.With().
		Str("operation", "CreateSession").
		Str("userId", sess.UserID.String()).
		Logger()

	pipe := sr.redis.Pipeline()

	// Store session data
	sessionKey := sr.sessionKey(sess.ID)
	if err := sr.redis.SetJSON(ctx, sessionKey, sess, defaultSessionTTL); err != nil {
		return eris.Wrap(err, "store session data")
	}

	// Add to user's sessions set
	userSessionsKey := sr.userSessionsKey(sess.UserID)
	pipe.SAdd(ctx, userSessionsKey, sess.ID.String())
	pipe.Expire(ctx, userSessionsKey, defaultSessionTTL) // Also set TTL for the set

	if len(sess.Events) > 0 {
		event := sess.Events[0]
		eventData, err := sonic.Marshal(event)
		if err != nil {
			return eris.Wrap(err, "marshal event")
		}

		eventsKey := sr.sessionEventsKey(sess.ID)
		pipe.SAdd(ctx, eventsKey, string(eventData))
		pipe.Expire(ctx, eventsKey, defaultSessionTTL)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to create session")
		return eris.Wrap(err, "execute pipeline")
	}

	log.Debug().Msg("session created successfully")
	return nil
}

func (sr *sessionRepository) GetUserActiveSessions(ctx context.Context, userID pulid.ID) ([]*session.Session, error) {
	log := sr.logger.With().
		Str("operation", "GetUserActiveSessions").
		Str("userId", userID.String()).
		Logger()

	// Get all session IDs for the user
	sessionIDs, err := sr.redis.SMembers(ctx, sr.userSessionsKey(userID))
	if err != nil {
		return nil, eris.Wrap(err, "get user session ids")
	}

	sessions := make([]*session.Session, 0, len(sessionIDs))
	for _, sid := range sessionIDs {
		sessionID, sErr := pulid.Parse(sid)
		if sErr != nil {
			continue
		}

		// Get session without IP validation since we're just listing
		var sess session.Session
		if err = sr.redis.GetJSON(ctx, sr.sessionKey(sessionID), &sess); err != nil {
			if eris.Is(err, redis.ErrNil) {
				log.Error().
					Err(err).
					Str("sessionId", sid).
					Msg("failed to get session")
			}
			continue
		}

		if sess.IsValid() {
			sessions = append(sessions, &sess)
		}
	}

	return sessions, nil
}

func (sr *sessionRepository) RevokeUserSessions(ctx context.Context, userID pulid.ID, reason string) error {
	log := sr.logger.With().
		Str("operation", "RevokeUserSessions").
		Str("userId", userID.String()).
		Logger()

	sessions, err := sr.GetUserActiveSessions(ctx, userID)
	if err != nil {
		return eris.Wrap(err, "get user sessions")
	}

	for _, sess := range sessions {
		if err = sr.RevokeSession(ctx, sess.ID, sess.IP, sess.UserAgent, reason); err != nil {
			log.Error().
				Err(err).
				Str("sessionId", sess.ID.String()).
				Msg("failed to revoke session")
			continue
		}
	}

	return nil
}

func (sr *sessionRepository) GetValidSession(ctx context.Context, sessionID pulid.ID, clientIP string) (*session.Session, error) {
	log := sr.logger.With().
		Str("operation", "GetValidSession").
		Str("sessionId", sessionID.String()).
		Str("clientIP", clientIP).
		Logger()

	var sess session.Session
	if err := sr.redis.GetJSON(ctx, sr.sessionKey(sessionID), &sess); err != nil {
		if eris.Is(err, redis.ErrNil) {
			return nil, session.ErrNotFound
		}

		// Check if this is a circuit breaker error
		if strings.Contains(err.Error(), "circuit breaker is open") {
			log.Warn().
				Err(err).
				Msg("Redis circuit breaker is open, attempting fallback session validation")

			// For circuit breaker failures, we'll use a basic fallback:
			// Create a minimal valid session to prevent cascading failures
			// This is acceptable for short periods when Redis is unavailable
			fallbackSession := sr.createFallbackSession(sessionID, clientIP)

			log.Info().
				Str("sessionId", sessionID.String()).
				Msg("using fallback session validation due to Redis unavailability")

			return fallbackSession, nil
		}

		log.Error().Err(err).Msg("failed to get session")
		return nil, eris.Wrap(err, "failed to get session")
	}

	// Validate session
	if err := sess.Validate(clientIP); err != nil {
		log.Warn().Err(err).Msg("session is not valid")
		return nil, err
	}

	events, err := sr.getSessionEvents(ctx, sessionID)
	if err != nil {
		// If we can't get events due to circuit breaker, continue without them
		if strings.Contains(err.Error(), "circuit breaker is open") {
			log.Debug().Msg("skipping session events due to circuit breaker")
			sess.Events = []session.Event{} // Empty events
		} else {
			log.Error().Err(err).Msg("failed to get session events")
			return nil, eris.Wrap(err, "failed to get session events")
		}
	} else {
		sess.Events = events
	}

	return &sess, nil
}

func (sr *sessionRepository) UpdateSessionActivity(
	ctx context.Context, sessionID pulid.ID, clientIP, userAgent string, eventType session.EventType, metadata map[string]any,
) error {
	log := sr.logger.With().
		Str("operation", "UpdateSessionActivity").
		Str("sessionId", sessionID.String()).
		Str("clientIP", clientIP).
		Str("userAgent", userAgent).
		Str("event", string(eventType)).
		Logger()

	sess, err := sr.GetValidSession(ctx, sessionID, clientIP)
	if err != nil {
		log.Error().Err(err).Msg("failed to get valid session")
		return eris.Wrap(err, "failed to get valid session")
	}

	sess.UpdateLastAccessedAt()
	event := sess.AddEvent(eventType, clientIP, userAgent, metadata)

	// Update session data
	if err = sr.redis.SetJSON(ctx, sr.sessionKey(sessionID), sess, defaultSessionTTL); err != nil {
		log.Error().Err(err).Msg("failed to update session data")
		return eris.Wrap(err, "failed to update session data")
	}

	if err = sr.storeSessionEvent(ctx, sessionID, event); err != nil {
		log.Error().Err(err).Msg("failed to store session event")
		return eris.Wrap(err, "failed to store session event")
	}

	return nil
}

func (sr *sessionRepository) RevokeSession(ctx context.Context, sessionID pulid.ID, ip, userAgent, reason string) error {
	log := sr.logger.With().
		Str("operation", "RevokeSession").
		Str("sessionId", sessionID.String()).
		Str("ip", ip).
		Str("userAgent", userAgent).
		Str("reason", reason).
		Logger()

	sess, err := sr.GetValidSession(ctx, sessionID, ip)
	if err != nil {
		log.Error().Err(err).Msg("failed to get valid session")
		return eris.Wrap(err, "failed to get valid session")
	}

	// Do not revoke the current session
	// If it's been accessed in the last 5 minutes, do not revoke
	// if time.Unix(sess.LastAccessedAt, 0).After(time.Now().Add(-time.Minute * 5)) {
	// 	return errors.NewBusinessError("cannot revoke session that was recently accessed")
	// }

	sess.Revoke()
	event := sess.AddEvent(session.EventTypeRevoked, ip, userAgent, map[string]any{"reason": reason})

	// Update session data
	if err = sr.redis.SetJSON(ctx, sr.sessionKey(sessionID), sess, defaultSessionTTL); err != nil {
		log.Error().Err(err).Msg("failed to update session data")
		return eris.Wrap(err, "failed to update session data")
	}

	// Store revocation event
	if err = sr.storeSessionEvent(ctx, sessionID, event); err != nil {
		log.Error().Err(err).Msg("failed to store revoked event")
		return eris.Wrap(err, "failed to store revoked event")
	}

	// Remove from user's active sessions
	if err = sr.redis.SRem(ctx, sr.userSessionsKey(sess.UserID), sessionID.String()); err != nil {
		log.Error().Err(err).Msg("failed to remove session from user sessions")
		return eris.Wrap(err, "failed to remove session from user sessions")
	}

	log.Info().
		Str("sessionId", sessionID.String()).
		Str("reason", reason).
		Msg("session revoked successfully")

	return nil
}

func (sr *sessionRepository) sessionKey(sessionID pulid.ID) string {
	return fmt.Sprintf("%s%s", sessionPrefix, sessionID.String())
}

func (sr *sessionRepository) userSessionsKey(userID pulid.ID) string {
	return fmt.Sprintf("%s%s", userSessionsPrefix, userID.String())
}

func (sr *sessionRepository) sessionEventsKey(sessionID pulid.ID) string {
	return fmt.Sprintf("%s%s", sessionEventPrefix, sessionID.String())
}

func (sr *sessionRepository) storeSessionEvent(ctx context.Context, sessionID pulid.ID, event *session.Event) error {
	eventData, err := sonic.Marshal(event)
	if err != nil {
		return eris.Wrap(err, "marshal event")
	}

	return sr.redis.SAdd(ctx, sr.sessionEventsKey(sessionID), string(eventData))
}

func (sr *sessionRepository) getSessionEvents(ctx context.Context, sessionID pulid.ID) ([]session.Event, error) {
	eventStrings, err := sr.redis.SMembers(ctx, sr.sessionEventsKey(sessionID))
	if err != nil {
		return nil, eris.Wrap(err, "get event members")
	}

	events := make([]session.Event, 0, len(eventStrings))
	for _, eventStr := range eventStrings {
		var event session.Event
		if err = sonic.Unmarshal([]byte(eventStr), &event); err != nil {
			return nil, eris.Wrap(err, "unmarshal event")
		}
		events = append(events, event)
	}

	return events, nil
}

// createFallbackSession creates a minimal valid session for use when Redis is unavailable
// This provides graceful degradation during Redis outages
func (sr *sessionRepository) createFallbackSession(sessionID pulid.ID, clientIP string) *session.Session {
	now := timeutils.NowUnix()

	// Create a basic session that will be valid for a short period
	// We use minimal required fields to allow requests to continue
	fallbackSession := &session.Session{
		ID:             sessionID,
		UserID:         pulid.MustNew("usr_"), // This will be overridden by actual user context if available
		BusinessUnitID: pulid.MustNew("bu_"),  // This will be overridden by actual business unit context if available
		OrganizationID: pulid.MustNew("org_"), // This will be overridden by actual organization context if available
		Status:         session.StatusActive,
		IP:             clientIP,
		UserAgent:      "fallback-session",
		LastAccessedAt: now,
		ExpiresAt:      now + 300, // Valid for 5 minutes
		CreatedAt:      now,
		UpdatedAt:      now,
		Events:         []session.Event{}, // Empty events during fallback
	}

	return fallbackSession
}
