package authservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func organizationScope(ctx context.Context, orgID, buID, userID pulid.ID) context.Context {
	return dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		UserID:         userID,
	})
}

func userScope(ctx context.Context, usr *tenant.User) context.Context {
	return organizationScope(ctx, usr.CurrentOrganizationID, usr.BusinessUnitID, usr.ID)
}

func sessionScope(ctx context.Context, sess *session.Session) context.Context {
	return organizationScope(ctx, sess.OrganizationID, sess.BusinessUnitID, sess.UserID)
}

func (s *Service) roleActivationRequired(ctx context.Context, orgID, buID pulid.ID) bool {
	if s.activation == nil {
		return true
	}
	required, err := s.activation.RoleActivationRequired(ctx, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		s.l.Warn(
			"could not read whether the organization requires role activation; requiring it",
			zap.String("organizationID", orgID.String()),
			zap.Error(err),
		)
		return true
	}
	return required
}
