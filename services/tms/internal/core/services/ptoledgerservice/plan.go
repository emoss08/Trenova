package ptoledgerservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

func validateAdjust(req *AdjustRequest) error {
	if req.AmountDays.IsZero() {
		return errortypes.NewValidationError(
			"amountDays",
			errortypes.ErrInvalid,
			"Amount cannot be zero",
		)
	}
	if req.Note == "" {
		return errortypes.NewValidationError(
			"note",
			errortypes.ErrRequired,
			"A note is required for manual adjustments",
		)
	}
	return nil
}

// AdjustPlan is the entry Adjust would post and the balance on either side of
// it, as the ledger stands now.
type AdjustPlan struct {
	Entry         *worker.WorkerPTOLedgerEntry
	BalanceBefore decimal.Decimal
	BalanceAfter  decimal.Decimal
}

// PlanAdjust checks an adjustment as Adjust does and projects it onto the
// worker's current balance of that type, posting nothing.
func (s *Service) PlanAdjust(ctx context.Context, req *AdjustRequest) (*AdjustPlan, error) {
	if err := validateAdjust(req); err != nil {
		return nil, err
	}

	balances, err := s.ledgerRepo.ListBalances(ctx, &repositories.ListPTOBalancesRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.WorkerID,
	})
	if err != nil {
		return nil, err
	}
	before := decimal.Zero
	for _, bal := range balances {
		if bal != nil && bal.PTOType == req.PTOType {
			before = bal.BalanceDays
		}
	}

	effectiveAt := req.EffectiveAt
	if effectiveAt == 0 {
		effectiveAt = time.Now().Unix()
	}
	after := before.Add(req.AmountDays)
	entry := &worker.WorkerPTOLedgerEntry{
		OrganizationID:   req.TenantInfo.OrgID,
		BusinessUnitID:   req.TenantInfo.BuID,
		WorkerID:         req.WorkerID,
		PTOType:          req.PTOType,
		EntryType:        worker.PTOLedgerEntryAdjustment,
		AmountDays:       req.AmountDays,
		BalanceAfterDays: after,
		EffectiveAt:      effectiveAt,
		Note:             req.Note,
		ActorType:        worker.PTOLedgerActorUser,
		CreatedByID:      req.UserID,
	}
	multiErr := errortypes.NewMultiError()
	entry.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return &AdjustPlan{Entry: entry, BalanceBefore: before, BalanceAfter: after}, nil
}
