package workerptoservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func (s *Service) PreviewApprove(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*services.WorkerPTOTransitionPreview, error) {
	preview, err := s.previewTransition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusApproved,
		notifyWorker: true,
	})
	if err != nil {
		return nil, err
	}

	availability, err := s.checkAvailability(ctx, preview.Before, preview.Before.ID)
	if err != nil {
		return nil, err
	}
	if availability.Tracked && preview.Before.Days.IsPositive() {
		preview.After.BalanceAfterDays = decimal.NewNullDecimal(
			availability.BalanceDays.Sub(preview.Before.Days),
		)
	}

	return preview, nil
}

func (s *Service) checkAvailability(
	ctx context.Context,
	pto *worker.WorkerPTO,
	excludeID pulid.ID,
) (*ptoledgerservice.AvailabilityResult, error) {
	if s.ledger == nil {
		return &ptoledgerservice.AvailabilityResult{Allowed: true, Days: pto.Days}, nil
	}

	return s.ledger.CheckAvailability(ctx, &ptoledgerservice.AvailabilityRequest{
		TenantInfo:   tenantOf(pto),
		WorkerID:     pto.WorkerID,
		PTOType:      pto.Type,
		Days:         pto.Days,
		StartDate:    pto.StartDate,
		ExcludePTOID: excludeID,
	})
}
