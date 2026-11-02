package sessionports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrSessionNotFound          = errors.New("session not found")
	ErrSessionSecretMismatch    = errors.New("session secret does not match")
	ErrSessionAccessUnconfirmed = errors.New("session access could not be confirmed")
)

func KeepsCookie(err error) bool {
	return errors.Is(err, ErrSessionSecretMismatch) || errors.Is(err, ErrSessionAccessUnconfirmed)
}

type SessionRepository interface {
	Get(ctx context.Context, sessionID pulid.ID) (*session.Session, error)
	Create(ctx context.Context, session *session.Session) error
	Update(
		ctx context.Context,
		sessionID pulid.ID,
		mutate func(*session.Session) error,
	) (*session.Session, error)
	Delete(ctx context.Context, sessionID pulid.ID) error
	DeleteAllForUser(ctx context.Context, userID pulid.ID) error
	ListForUser(ctx context.Context, userID pulid.ID) ([]*session.Session, error)
	Touch(ctx context.Context, sessionID pulid.ID, at int64) error
	ConfirmAccess(ctx context.Context, sessionID pulid.ID, at int64) error
}
