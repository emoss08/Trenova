package userservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/app/sessionrotation"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"go.uber.org/zap"
)

type SessionChangeResult struct {
	User         *tenant.User
	SessionToken string
	ExpiresAt    int64
}

func (s *Service) endOtherSessionsAfterPasswordChange(
	ctx context.Context,
	scope *SessionScope,
) (signedOut bool, err error) {
	ended, err := s.endOtherSessions(ctx, scope)
	if err != nil {
		return true, s.endEverySessionAfterPasswordChange(ctx, scope, err)
	}
	if ended > 0 {
		s.recordSessionsEnded(
			ctx,
			scope.TenantInfo,
			scope.TenantInfo.UserID,
			iamports.SessionRevocationPasswordChange,
		)
	}

	return false, nil
}

func (s *Service) rotateAfterPasswordChange(
	ctx context.Context,
	scope *SessionScope,
	sessionEpoch int64,
	result *SessionChangeResult,
) error {
	if scope.CurrentSessionID.IsNil() {
		return nil
	}

	token, expiresAt, err := s.rotateSessionSecret(ctx, scope, sessionEpoch)
	if errors.Is(err, sessionports.ErrSessionNotFound) {
		return nil
	}
	if err != nil {
		return s.endEverySessionAfterPasswordChange(ctx, scope, err)
	}

	result.SessionToken = token
	result.ExpiresAt = expiresAt
	return nil
}

func (s *Service) rotateSessionSecret(
	ctx context.Context,
	scope *SessionScope,
	sessionEpoch int64,
) (token string, expiresAt int64, err error) {
	rotated, err := s.rotator.Rotate(ctx, sessionrotation.Request{
		SessionID:         scope.CurrentSessionID,
		UserID:            scope.TenantInfo.UserID,
		CurrentSecretHash: scope.CurrentSecretHash,
		Grace:             session.CredentialChangeGrace,
		Change: func(sess *session.Session) error {
			sess.MustChangePassword = false
			sess.SessionEpoch = sessionEpoch
			return nil
		},
	})
	if err != nil {
		return "", 0, err
	}

	return rotated.Token, rotated.Session.ExpiresAt, nil
}

func (s *Service) endEverySessionAfterPasswordChange(
	ctx context.Context,
	scope *SessionScope,
	cause error,
) error {
	userID := scope.TenantInfo.UserID
	s.l.Error(
		"could not secure the sessions of a password change; ending every session",
		zap.String("userID", userID.String()),
		zap.Error(cause),
	)

	if err := s.sr.DeleteAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("end sessions for user %s: %w", userID, errors.Join(cause, err))
	}
	s.recordSessionsEnded(ctx, scope.TenantInfo, userID, iamports.SessionRevocationPasswordChange)

	return nil
}
