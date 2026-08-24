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
	return assetOperationsEnabled(ctx, orgRepo, tenantInfo, repositories.CapabilityLockNone)
}

func EnsureDriverAssignable(
	ctx context.Context,
	orgRepo repositories.OrganizationRepository,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) error {
	enabled, err := assetOperationsEnabled(
		ctx,
		orgRepo,
		tenantInfo,
		repositories.CapabilityLockShare,
	)
	if err != nil {
		return err
	}
	if !enabled {
		return errortypes.NewBusinessError("Organization does not have asset operations enabled. Cover this shipment move with a carrier instead of assigning a driver").
			WithParam("shipmentMoveId", moveID.String())
	}

	return nil
}

func assetOperationsEnabled(
	ctx context.Context,
	orgRepo repositories.OrganizationRepository,
	tenantInfo pagination.TenantInfo,
	lock repositories.CapabilityLock,
) (bool, error) {
	capabilities, err := orgRepo.GetCapabilities(
		ctx,
		repositories.GetOrganizationCapabilitiesRequest{
			TenantInfo: tenantInfo,
			Lock:       lock,
		},
	)
	if err != nil {
		return false, err
	}

	return capabilities.AssetOperationsEnabled, nil
}
