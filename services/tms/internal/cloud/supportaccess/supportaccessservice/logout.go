package supportaccessservice

import (
	"context"

	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessrepository"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func (s *Service) EndForSignOut(ctx context.Context, staffUserID pulid.ID) {
	if !s.Enabled() || staffUserID.IsNil() {
		return
	}

	ended, err := s.repo.EndSessionsForStaff(ctx, supportaccessrepository.EndStaffSessionsRequest{
		StaffUserID: staffUserID,
		Reason:      supportaccess.EndReasonSignedOut,
		Now:         s.nowUnix(),
	})
	if err != nil {
		s.l.Error("failed to end the support sessions of a staff member signing out",
			zap.String("staffUserId", staffUserID.String()),
			zap.Error(err),
		)
		return
	}

	for _, session := range ended {
		s.recordSessionEvent(ctx, &sessionEvent{
			session:   session,
			operation: sessionEnded,
			comment: "Trenova support session of " + session.StaffName + " ended: " +
				endReasonText(supportaccess.EndReasonSignedOut),
			metadata: map[string]any{metadataEndReason: supportaccess.EndReasonSignedOut.String()},
		})
	}
}

type AuthDecoratorParams struct {
	fx.In

	Auth    services.AuthService
	Support *Service
}

type endSupportOnSignOut struct {
	services.AuthService
	support *Service
}

func DecorateAuthService(p AuthDecoratorParams) services.AuthService {
	return &endSupportOnSignOut{AuthService: p.Auth, support: p.Support}
}

func (a *endSupportOnSignOut) Logout(ctx context.Context, sessionID pulid.ID) error {
	sess, lookupErr := a.ValidateSession(ctx, sessionID)

	err := a.AuthService.Logout(ctx, sessionID)

	if lookupErr == nil && sess != nil {
		a.support.EndForSignOut(context.WithoutCancel(ctx), sess.UserID)
	}

	return err
}
