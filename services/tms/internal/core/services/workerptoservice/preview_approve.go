package workerptoservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func (s *Service) PreviewApprove(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*services.PTOApprovalPlan, error) {
	current, err := s.repo.GetByID(ctx, &repositories.GetPTOByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	planned := *req
	if err = planTransition(current, &planned, worker.PTOStatusApproved); err != nil {
		return nil, err
	}

	availability, err := s.checkAvailability(ctx, current, current.ID)
	if err != nil {
		return nil, err
	}
	if err = availability.ValidationError(); err != nil {
		return nil, err
	}

	approved := *current
	approved.Status = planned.Status
	approved.ApproverID = planned.UserID
	if availability.Tracked && current.Days.IsPositive() {
		approved.BalanceAfterDays = decimal.NewNullDecimal(
			availability.BalanceDays.Sub(current.Days),
		)
	}

	return &services.PTOApprovalPlan{Current: current, Approved: &approved}, nil
}

func planTransition(
	current *worker.WorkerPTO,
	req *repositories.UpdatePTOStatusRequest,
	target worker.PTOStatus,
) error {
	if !current.Status.CanTransitionTo(target) {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"PTO is {0} and cannot be {1}",
			strings.ToLower(string(current.Status)),
			strings.ToLower(string(target)),
		)
	}

	req.Status = target
	req.Reason = strings.TrimSpace(req.Reason)
	if req.ExpectedVersion == 0 {
		req.ExpectedVersion = current.Version
	}

	return nil
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
