package authservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (s *Service) confirmSessionAccess(
	ctx context.Context,
	sess *session.Session,
	now int64,
) error {
	if sess.IssuedBySignIn {
		ended, err := s.sessionAccessEnded(ctx, sess, now)
		if err != nil {
			s.l.Warn(
				"could not confirm a session still has access; checking again on its next request",
				zap.String("sessionID", sess.ID.String()),
				zap.String("userID", sess.UserID.String()),
				zap.Error(err),
			)
			return nil
		}
		if ended {
			s.l.Info(
				"ended a session whose user lost access to its organization",
				zap.String("sessionID", sess.ID.String()),
				zap.String("userID", sess.UserID.String()),
				zap.String("organizationID", sess.OrganizationID.String()),
			)
			if delErr := s.sr.Delete(ctx, sess.ID); delErr != nil {
				s.l.Error("failed to delete a session that lost access", zap.Error(delErr))
			}
			return errortypes.NewAuthenticationError("Session is invalid. Please login again.")
		}
	}

	if err := s.sr.Touch(ctx, sess.ID, now); err == nil {
		sess.LastAccessedAt = now
	}

	return nil
}

func (s *Service) sessionAccessEnded(
	ctx context.Context,
	sess *session.Session,
	now int64,
) (bool, error) {
	user, err := s.ur.GetTenantMember(sessionScope(ctx, sess), tenantports.GetTenantMemberRequest{
		UserID: sess.UserID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  sess.OrganizationID,
			BuID:   sess.BusinessUnitID,
			UserID: sess.UserID,
		},
		Now: now,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return true, nil
		}
		return false, err
	}

	return user.ValidateStatus() != nil, nil
}
