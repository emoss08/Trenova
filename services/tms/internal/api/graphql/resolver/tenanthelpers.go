package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

func (r *mutationResolver) applyOrganizationCapabilities(
	ctx context.Context,
	entity *tenant.Organization,
	brokerageEnabled, assetOperationsEnabled *bool,
) error {
	if brokerageEnabled == nil || assetOperationsEnabled == nil {
		current, err := r.organizationService.GetByID(ctx, repositories.GetOrganizationByIDRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.ID,
				BuID:  entity.BusinessUnitID,
			},
		})
		if err != nil {
			return err
		}

		entity.BrokerageEnabled = current.BrokerageEnabled
		entity.AssetOperationsEnabled = current.AssetOperationsEnabled
	}

	if brokerageEnabled != nil {
		entity.BrokerageEnabled = *brokerageEnabled
	}

	if assetOperationsEnabled != nil {
		entity.AssetOperationsEnabled = *assetOperationsEnabled
	}

	return nil
}
