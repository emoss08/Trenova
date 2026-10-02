package carriersettlementservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func errDuplicateInvoiceNumber(invoiceNumber string) error {
	return errortypes.NewValidationError(
		"invoiceNumber",
		errortypes.ErrDuplicate,
		"Invoice {0} from this carrier is already matched; reject that match first if this one replaces it",
		invoiceNumber,
	)
}

func isDuplicateRefusal(err error) bool {
	if fieldErr, ok := errors.AsType[*errortypes.Error](err); ok {
		return fieldErr.Code == errortypes.ErrDuplicate
	}
	return false
}

func (s *Service) screenDuplicateMatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
	invoiceNumber string,
	assignmentID pulid.ID,
) (*pulid.ID, error) {
	live, err := s.invoiceMatchRepo.GetLiveByCarrierInvoiceNumber(
		ctx,
		&repositories.GetLiveCarrierInvoiceMatchByNumberRequest{
			TenantInfo:    tenantInfo,
			CarrierID:     carrierID,
			InvoiceNumber: invoiceNumber,
		},
	)
	if err != nil {
		return nil, err
	}
	if live != nil {
		return nil, errDuplicateInvoiceNumber(invoiceNumber)
	}

	onLoad, err := s.invoiceMatchRepo.ListLiveByAssignment(
		ctx,
		repositories.ListLiveCarrierInvoiceMatchesByAssignmentRequest{
			TenantInfo:   tenantInfo,
			AssignmentID: assignmentID,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(onLoad) == 0 {
		return nil, nil //nolint:nilnil // no earlier match on the load is the usual case
	}
	first := onLoad[0].ID
	return &first, nil
}

type guard struct {
	refusal error
}

func (g guard) refused() bool { return g.refusal != nil }

func refuse(err error) guard { return guard{refusal: err} }

func (s *Service) checkAcceptable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	match *carriersettlement.InvoiceMatch,
	withVariance bool,
) error {
	verdict, err := s.acceptGuard(ctx, tenantInfo, match, withVariance)
	if err != nil {
		return err
	}
	return verdict.refusal
}

func (s *Service) acceptGuard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	match *carriersettlement.InvoiceMatch,
	withVariance bool,
) (guard, error) {
	if match.IsDuplicate() {
		original, err := s.invoiceMatchRepo.GetByID(
			ctx,
			repositories.GetCarrierInvoiceMatchByIDRequest{
				ID:         *match.DuplicateOfMatchID,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil && !errortypes.IsNotFoundError(err) {
			return guard{}, err
		}
		if original != nil && original.Status != carriersettlement.InvoiceMatchStatusRejected {
			return refuse(errortypes.NewValidationError(
				"duplicateOfMatchId",
				errortypes.ErrDuplicate,
				"This match repeats invoice {0}, which another match already holds; reject one of them before accepting",
				match.InvoiceNumber,
			)), nil
		}
	}

	if !withVariance {
		return guard{}, nil
	}

	onLoad, err := s.invoiceMatchRepo.ListLiveByAssignment(
		ctx,
		repositories.ListLiveCarrierInvoiceMatchesByAssignmentRequest{
			TenantInfo:   tenantInfo,
			AssignmentID: match.CarrierAssignmentID,
		},
	)
	if err != nil {
		return guard{}, err
	}
	for _, other := range onLoad {
		if other.ID == match.ID ||
			other.Status != carriersettlement.InvoiceMatchStatusResolved ||
			other.AdjustmentCostEventID == nil {
			continue
		}
		return refuse(errortypes.NewValidationError(
			"carrierAssignmentId",
			errortypes.ErrInvalidOperation,
			"This load's carrier cost was already adjusted for invoice {0}; accepting this variance too would pay the difference twice",
			other.InvoiceNumber,
		)), nil
	}

	return guard{}, nil
}

func (s *Service) approvalHoldGuard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) (guard, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return guard{}, err
	}
	if !control.HoldUntilInvoiceMatched {
		return guard{}, nil
	}

	awaiting, err := s.costEventRepo.CountAwaitingInvoiceMatch(ctx, tenantInfo, settlementID)
	if err != nil {
		return guard{}, err
	}
	if awaiting == 0 {
		return guard{}, nil
	}
	return refuse(errortypes.NewValidationError(
		"lines",
		errortypes.ErrInvalidOperation,
		"{0} load(s) on this settlement are still waiting on a resolved carrier invoice match; resolve the matches or recalculate the settlement before approving",
		awaiting,
	)), nil
}
