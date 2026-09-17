package integrationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

type carrierIntelRoleChange struct {
	tenant  pagination.TenantInfo
	typ     integration.Type
	enabled bool
	role    string
}

func (c *carrierIntelRoleChange) wantsFallback() bool {
	return c.typ == integration.TypeFMCSAQCMobile && c.role == integration.CarrierIntelRoleFallback
}

func (s *Service) checkCarrierIntelRole(ctx context.Context, change *carrierIntelRoleChange) error {
	if s.carrierIntelControls == nil || !change.typ.SupportsCarrierIntelligence() ||
		!change.enabled || change.wantsFallback() {
		return nil
	}
	control, err := s.carrierIntelControls.GetOrCreate(ctx, change.tenant)
	if err != nil {
		return err
	}
	current, ok := control.PrimaryType()
	if !ok || current == change.typ {
		return nil
	}
	record, err := s.repo.GetByType(ctx, change.tenant, current)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if !record.Enabled {
		return nil
	}
	return errortypes.NewBusinessError(
		"{0} is already the primary carrier intelligence provider. Disable it, or switch the primary provider from Carrier Monitoring, before enabling {1} as primary",
		current.String(),
		change.typ.String(),
	)
}

func (s *Service) syncCarrierIntelRole(ctx context.Context, change *carrierIntelRoleChange) error {
	if s.carrierIntelControls == nil || !change.typ.SupportsCarrierIntelligence() {
		return nil
	}
	control, err := s.carrierIntelControls.GetOrCreate(ctx, change.tenant)
	if err != nil {
		return err
	}

	if !applyCarrierIntelRole(control, change) {
		return nil
	}
	_, err = s.carrierIntelControls.Update(ctx, control)
	return err
}

func applyCarrierIntelRole(
	control *carrierintel.CarrierIntelControl,
	change *carrierIntelRoleChange,
) bool {
	typ := change.typ
	isPrimary := control.PrimaryProvider != nil && *control.PrimaryProvider == typ
	isFallback := control.FallbackProvider != nil && *control.FallbackProvider == typ

	switch {
	case !change.enabled:
		dirty := false
		if isPrimary {
			control.PrimaryProvider = nil
			control.PolicyVersion++
			dirty = true
		}
		if isFallback {
			control.FallbackProvider = nil
			dirty = true
		}
		return dirty
	case change.wantsFallback():
		dirty := false
		if isPrimary {
			control.PrimaryProvider = nil
			control.PolicyVersion++
			dirty = true
		}
		if !isFallback {
			control.FallbackProvider = &typ
			dirty = true
		}
		return dirty
	default:
		dirty := false
		if isFallback {
			control.FallbackProvider = nil
			dirty = true
		}
		if _, hasPrimary := control.PrimaryType(); !hasPrimary {
			control.PrimaryProvider = &typ
			control.PolicyVersion++
			dirty = true
		}
		return dirty
	}
}
