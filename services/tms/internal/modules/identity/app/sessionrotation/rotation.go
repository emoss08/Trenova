package sessionrotation

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Sessions sessionports.SessionRepository
	Tokens   *sessiontoken.Issuer
}

type Rotator struct {
	sessions sessionports.SessionRepository
	tokens   *sessiontoken.Issuer
}

func New(p Params) *Rotator {
	return &Rotator{sessions: p.Sessions, tokens: p.Tokens}
}

type Request struct {
	SessionID         pulid.ID
	UserID            pulid.ID
	CurrentSecretHash string
	Grace             time.Duration
	Change            func(*session.Session) error
}

type Rotated struct {
	Session *session.Session
	Token   string
}

func (r *Rotator) Rotate(ctx context.Context, req Request) (*Rotated, error) {
	issued, err := r.tokens.Issue(req.SessionID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("issue session token: %w", err)
	}

	updated, err := r.sessions.Update(ctx, req.SessionID, func(sess *session.Session) error {
		if sess.UserID != req.UserID {
			return sessionports.ErrSessionNotFound
		}
		if req.CurrentSecretHash == "" || sess.SecretHash != req.CurrentSecretHash {
			return errortypes.NewSessionContextStaleError(
				sess.UserID,
				sess.OrganizationID,
				sess.BusinessUnitID,
			)
		}
		if req.Change != nil {
			if changeErr := req.Change(sess); changeErr != nil {
				return changeErr
			}
		}
		sess.ReplaceSecret(issued.SecretHash, timeutils.NowUnix(), req.Grace)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Rotated{Session: updated, Token: issued.Token}, nil
}
