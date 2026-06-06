package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (r *userResolver) currentOrganization(
	ctx context.Context,
	obj *tenant.User,
) (*tenant.Organization, error) {
	if obj.CurrentOrganizationID.IsNil() {
		return nil, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Organization loader is not configured")
	}

	return loadersForRequest.OrganizationByID.Load(ctx, obj.CurrentOrganizationID.String())()
}
