package capabilityguard

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func AssetOperationsEnabled(
	ctx context.Context,
	orgRepo tenantports.OrganizationRepository,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	return assetOperationsEnabled(ctx, orgRepo, tenantInfo, tenantports.CapabilityLockNone)
}

func EnsureDriverAssignable(
	ctx context.Context,
	orgRepo tenantports.OrganizationRepository,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	lock tenantports.CapabilityLock,
) error {
	enabled, err := assetOperationsEnabled(ctx, orgRepo, tenantInfo, lock)
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
	orgRepo tenantports.OrganizationRepository,
	tenantInfo pagination.TenantInfo,
	lock tenantports.CapabilityLock,
) (bool, error) {
	capabilities, err := orgRepo.GetCapabilities(
		ctx,
		tenantports.GetOrganizationCapabilitiesRequest{
			TenantInfo: tenantInfo,
			Lock:       lock,
		},
	)
	if err != nil {
		return false, err
	}

	return capabilities.AssetOperationsEnabled, nil
}
