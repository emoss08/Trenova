package tenantresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (r *UserResolver) currentOrganization(
	ctx context.Context,
	obj *tenant.User,
) (*tenant.Organization, error) {
	if obj.CurrentOrganizationID.IsNil() {
		return nil, nil
	}

	if !loaders.Available(ctx) {
		return nil, errortypes.NewDatabaseError("Organization loader is not configured")
	}

	return loaders.OrganizationByID(ctx, obj.CurrentOrganizationID.String())
}

func (r *MutationResolver) applyOrganizationCapabilities(
	ctx context.Context,
	entity *tenant.Organization,
	brokerageEnabled, assetOperationsEnabled *bool,
) error {
	if brokerageEnabled == nil || assetOperationsEnabled == nil {
		current, err := r.OrganizationService.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
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
