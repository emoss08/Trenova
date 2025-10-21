package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetValidSessionRequest struct {
	SessionID pulid.ID
	ClientIP  string
}

type UpdateSessionActivityRequest struct {
	SessionID pulid.ID
	ClientIP  string
	UserAgent string
	EventType session.EventType
	Metadata  map[string]any
}

type UpdateSessionOrganizationRequest struct {
	SessionID pulid.ID
	NewOrgID  pulid.ID
}

type RevokeSessionRequest struct {
	SessionID pulid.ID
	ClientIP  string
	UserAgent string
	Reason    string
}

type SessionRepository interface {
	Create(ctx context.Context, session *session.Session) error
	GetValidSession(
		ctx context.Context,
		params GetValidSessionRequest,
	) (*session.Session, error)
	GetUserActiveSessions(ctx context.Context, userID pulid.ID) ([]*session.Session, error)
	UpdateSessionActivity(
		ctx context.Context,
		params *UpdateSessionActivityRequest,
	) error
	UpdateSessionOrganization(
		ctx context.Context,
		params UpdateSessionOrganizationRequest,
	) error
	RevokeSession(ctx context.Context, params RevokeSessionRequest) error
}
