package driverportalinvitationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/capabilityguard"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) requireAssetOperations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	enabled, err := capabilityguard.AssetOperationsEnabled(ctx, s.orgRepo, tenantInfo)
	if err != nil {
		return err
	}
	if !enabled {
		return errortypes.NewValidationError(
			"feature",
			errortypes.ErrInvalidOperation,
			"Driver portal access requires asset operations. Enable asset operations for this organization before inviting drivers",
		)
	}

	return nil
}
