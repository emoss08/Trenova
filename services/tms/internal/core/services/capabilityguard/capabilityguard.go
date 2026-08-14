package capabilityguard

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func AssetOperationsEnabled(
	ctx context.Context,
	orgRepo repositories.OrganizationRepository,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	org, err := orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return false, err
	}

	return org.AssetOperationsEnabled, nil
}

func EnsureDriverAssignable(
	ctx context.Context,
	orgRepo repositories.OrganizationRepository,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) error {
	enabled, err := AssetOperationsEnabled(ctx, orgRepo, tenantInfo)
	if err != nil {
		return err
	}
	if !enabled {
		return errortypes.NewBusinessError("Organization does not have asset operations enabled. Cover this shipment move with a carrier instead of assigning a driver").
			WithParam("shipmentMoveId", moveID.String())
	}

	return nil
}
