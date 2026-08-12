// Package capabilityguard holds the organization-capability gates shared by the
// services that refuse asset-side work for organizations which do not run their
// own drivers and equipment.
package capabilityguard

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// AssetOperationsEnabled reports whether the organization dispatches its own
// drivers and equipment rather than covering freight through carriers only.
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

// EnsureDriverAssignable blocks driver coverage of a shipment move while the
// organization has asset operations switched off.
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
