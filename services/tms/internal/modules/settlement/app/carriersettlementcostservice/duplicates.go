package carriersettlementcostservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
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
		&carriersettlementports.GetLiveCarrierInvoiceMatchByNumberRequest{
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
		carriersettlementports.ListLiveCarrierInvoiceMatchesByAssignmentRequest{
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
	return verdict.Refusal
}

func (s *Service) acceptGuard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	match *carriersettlement.InvoiceMatch,
	withVariance bool,
) (settlementshared.Guard, error) {
	if match.IsDuplicate() {
		original, err := s.invoiceMatchRepo.GetByID(
			ctx,
			carriersettlementports.GetCarrierInvoiceMatchByIDRequest{
				ID:         *match.DuplicateOfMatchID,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil && !errortypes.IsNotFoundError(err) {
			return settlementshared.Guard{}, err
		}
		if original != nil && original.Status != carriersettlement.InvoiceMatchStatusRejected {
			return settlementshared.Refuse(errortypes.NewValidationError(
				"duplicateOfMatchId",
				errortypes.ErrDuplicate,
				"This match repeats invoice {0}, which another match already holds; reject one of them before accepting",
				match.InvoiceNumber,
			)), nil
		}
	}

	if !withVariance {
		return settlementshared.Guard{}, nil
	}

	onLoad, err := s.invoiceMatchRepo.ListLiveByAssignment(
		ctx,
		carriersettlementports.ListLiveCarrierInvoiceMatchesByAssignmentRequest{
			TenantInfo:   tenantInfo,
			AssignmentID: match.CarrierAssignmentID,
		},
	)
	if err != nil {
		return settlementshared.Guard{}, err
	}
	for _, other := range onLoad {
		if other.ID == match.ID ||
			other.Status != carriersettlement.InvoiceMatchStatusResolved ||
			other.AdjustmentCostEventID == nil {
			continue
		}
		return settlementshared.Refuse(errortypes.NewValidationError(
			"carrierAssignmentId",
			errortypes.ErrInvalidOperation,
			"This load's carrier cost was already adjusted for invoice {0}; accepting this variance too would pay the difference twice",
			other.InvoiceNumber,
		)), nil
	}

	return settlementshared.Guard{}, nil
}
