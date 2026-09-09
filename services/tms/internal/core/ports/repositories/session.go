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
	// DeleteAllForUser ends every session a user has open. A password reset has to
	// reach the sessions somebody else may already be holding, or the reset does not
	// actually take the account back.
	DeleteAllForUser(ctx context.Context, userID pulid.ID) error
}
