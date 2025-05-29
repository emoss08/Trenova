package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type SessionRepository interface {
	Create(ctx context.Context, session *session.Session) error
	GetValidSession(
		ctx context.Context,
		sessionID pulid.ID,
		clientIP string,
	) (*session.Session, error)
	GetUserActiveSessions(ctx context.Context, userID pulid.ID) ([]*session.Session, error)
	UpdateSessionActivity(
		ctx context.Context,
		sessionID pulid.ID,
		clientIP, userAgent string,
		eventType session.EventType,
		metadata map[string]any,
	) error
	RevokeSession(ctx context.Context, sessionID pulid.ID, ip, userAgent, reason string) error
}
