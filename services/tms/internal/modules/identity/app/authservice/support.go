package authservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// requireLiveSupport ends a Trenova support session whose grant was revoked or has expired. It
// reads the grant on every request: a revoked grant ends its sessions at the next request on
// any process, however they were deleted. A session whose grant cannot be read is refused
// without being ended, and keeps its cookie, so a database outage ends nobody's support.
func (s *Service) requireLiveSupport(ctx context.Context, sess *session.Session) error {
	err := s.support.RequireLive(ctx, &supportaccessports.RequireLiveRequest{
		OrganizationID:  sess.OrganizationID,
		BusinessUnitID:  sess.BusinessUnitID,
		GrantID:         sess.Support.GrantID,
		PrincipalUserID: sess.UserID,
		Now:             timeutils.NowUnix(),
	})
	if err == nil {
		sess.RoleActivationRequired = false
		return nil
	}
	if !errors.Is(err, supportaccess.ErrGrantNotFound) {
		return fmt.Errorf("%w: %w", sessionports.ErrSessionAccessUnconfirmed, err)
	}

	s.l.Info(
		"ended a Trenova support session whose grant is no longer live",
		zap.String("sessionID", sess.ID.String()),
		zap.String("grantID", sess.Support.GrantID.String()),
	)
	if delErr := s.sr.DeleteAllForUser(ctx, sess.UserID); delErr != nil {
		s.l.Error("failed to end the sessions of an ended support grant", zap.Error(delErr))
	}
	return errSessionEnded
}
