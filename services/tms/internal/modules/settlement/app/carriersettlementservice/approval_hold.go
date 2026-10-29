package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) approvalHoldGuard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) (settlementshared.Guard, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return settlementshared.Guard{}, err
	}
	if !control.HoldUntilInvoiceMatched {
		return settlementshared.Guard{}, nil
	}

	awaiting, err := s.costEventRepo.CountAwaitingInvoiceMatch(ctx, tenantInfo, settlementID)
	if err != nil {
		return settlementshared.Guard{}, err
	}
	if awaiting == 0 {
		return settlementshared.Guard{}, nil
	}
	return settlementshared.Refuse(errortypes.NewValidationError(
		"lines",
		errortypes.ErrInvalidOperation,
		"{0} load(s) on this settlement are still waiting on a resolved carrier invoice match; resolve the matches or recalculate the settlement before approving",
		awaiting,
	)), nil
}
