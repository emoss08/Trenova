package sessiontest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/sessionrotation"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
)

type UpdateFunc = func(
	context.Context,
	pulid.ID,
	func(*session.Session) error,
) (*session.Session, error)

func UpdateStored(stored *session.Session) UpdateFunc {
	return func(
		_ context.Context,
		_ pulid.ID,
		mutate func(*session.Session) error,
	) (*session.Session, error) {
		next := *stored
		if err := mutate(&next); err != nil {
			return nil, err
		}
		*stored = next
		return stored, nil
	}
}

const TokenSecret = "unit-test-session-secret-with-32-bytes"

func Tokens() *sessiontoken.Issuer {
	return sessiontoken.NewIssuer(TokenSecret)
}

func Rotator(sessions sessionports.SessionRepository) *sessionrotation.Rotator {
	return sessionrotation.New(sessionrotation.Params{Sessions: sessions, Tokens: Tokens()})
}
