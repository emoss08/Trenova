package factoringoverviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CustomersRequest struct {
	Filter             *pagination.QueryOptions
	Cursor             pagination.CursorInfo
	FactoringCompanyID pulid.ID
	Stage              factoringports.CustomerNoticeStage
}

func (s *Service) Customers(
	ctx context.Context,
	req *CustomersRequest,
) (*factoringports.FactorCustomerPage, error) {
	switch req.Stage {
	case factoringports.CustomerNoticeAll,
		factoringports.CustomerNoticeNeeded,
		factoringports.CustomerNoticeAwaiting,
		factoringports.CustomerNoticeAcknowledged:
	default:
		return nil, errortypes.NewValidationError(
			"stage",
			errortypes.ErrInvalid,
			"Choose a notice stage: needs a notice, awaiting or acknowledged",
		)
	}

	return s.customers.List(ctx, &factoringports.ListFactorCustomersRequest{
		Filter:             req.Filter,
		Cursor:             req.Cursor,
		FactoringCompanyID: req.FactoringCompanyID,
		Stage:              req.Stage,
		At:                 s.now(),
	})
}
