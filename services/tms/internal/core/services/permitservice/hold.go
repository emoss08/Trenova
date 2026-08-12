package permitservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func holdNotes(open []*permit.Requirement) string {
	descriptions := make([]string, 0, len(open))
	for _, requirement := range open {
		descriptions = append(descriptions, requirement.Describe())
	}

	return "Dispatch blocked until permits are recorded: " +
		strings.Join(descriptions, "; ")
}

func (s *service) activePermitHold(
	ctx context.Context,
	entity *shipment.Shipment,
	reason *holdreason.HoldReason,
) *shipment.ShipmentHold {
	result, err := s.holdRepo.ListByShipmentID(ctx, &repositories.ListShipmentHoldsRequest{
		ShipmentID: entity.ID,
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	})
	if err != nil {
		s.l.Warn("failed to list shipment holds", zap.Error(err))
		return nil
	}

	for _, hold := range result.Items {
		if hold == nil || hold.ReleasedAt != nil {
			continue
		}
		if hold.HoldReasonID != nil && *hold.HoldReasonID == reason.ID {
			return hold
		}
	}

	return nil
}

// reconcileHold keeps exactly one PERMIT_REQUIRED hold in step with the open
// requirements. Validation cannot stop a dispatch; a hold can, which is why
// this runs after persistence rather than inside the validator.
func (s *service) reconcileHold(
	ctx context.Context,
	entity *shipment.Shipment,
	assessment *services.PermitAssessment,
	_ *services.RequestActor,
) {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	reason, err := s.permitRepo.FindHoldReasonByCode(
		ctx, tenantInfo, PermitRequiredHoldCode,
	)
	if err != nil || reason == nil {
		s.l.Warn("permit hold reason unavailable; dispatch will not be blocked",
			zap.Error(err))
		return
	}

	existing := s.activePermitHold(ctx, entity, reason)

	if !assessment.HasOpen() {
		if existing == nil {
			return
		}

		now := timeutils.NowUnix()
		existing.ReleasedAt = &now
		if _, rErr := s.holdRepo.Release(ctx, existing); rErr != nil {
			s.l.Error("failed to release permit hold", zap.Error(rErr))
		}
		return
	}

	notes := holdNotes(assessment.Open)

	if existing != nil {
		if existing.Notes == notes {
			return
		}
		existing.Notes = notes
		if _, uErr := s.holdRepo.Update(ctx, existing); uErr != nil {
			s.l.Error("failed to update permit hold", zap.Error(uErr))
		}
		return
	}

	reasonID := reason.ID
	hold := &shipment.ShipmentHold{
		ShipmentID:        entity.ID,
		OrganizationID:    entity.OrganizationID,
		BusinessUnitID:    entity.BusinessUnitID,
		HoldReasonID:      &reasonID,
		Type:              holdreason.HoldTypeCompliance,
		Severity:          holdreason.HoldSeverityBlocking,
		ReasonCode:        PermitRequiredHoldCode,
		Notes:             notes,
		Source:            shipment.HoldSourceRule,
		BlocksDispatch:    true,
		VisibleToCustomer: false,
		StartedAt:         timeutils.NowUnix(),
	}

	if _, cErr := s.holdRepo.Create(ctx, hold); cErr != nil {
		s.l.Error("failed to raise permit hold", zap.Error(cErr))
	}
}
