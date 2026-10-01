package authservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
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
