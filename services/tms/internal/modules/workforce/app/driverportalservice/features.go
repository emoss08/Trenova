package driverportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) DashControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*dashcontrol.DashControl, error) {
	return s.dashControlRepo.GetOrCreate(ctx, tenantInfo)
}

// requireFeature loads the org's DashControl and returns a validation error
// when the given capability is switched off for drivers.
func (s *Service) RequireFeature(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	enabled func(*dashcontrol.DashControl) bool,
	message string,
) (*dashcontrol.DashControl, error) {
	control, err := s.DashControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !enabled(control) {
		return nil, errortypes.NewValidationError(
			"feature",
			errortypes.ErrInvalidOperation,
			message,
		)
	}
	return control, nil
}
