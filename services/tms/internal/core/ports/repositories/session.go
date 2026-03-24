package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/shared/pulid"
)

type SessionRepository interface {
	Get(ctx context.Context, sessionID pulid.ID) (*session.Session, error)
	Create(ctx context.Context, session *session.Session) error
	Update(ctx context.Context, session *session.Session) error
	Delete(ctx context.Context, sessionID pulid.ID) error
}
