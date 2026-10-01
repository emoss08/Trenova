package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func applyCreditMemoRequestFromInput(
	input *gqlmodel.ApplyCreditMemoInput,
	tenantInfo pagination.TenantInfo,
) (*serviceports.ApplyCreditMemoRequest, error) {
	creditMemoID, err := pulid.MustParse(input.CreditMemoID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"creditMemoId",
			errortypes.ErrInvalid,
			"Invalid credit memo",
		)
	}

	applications := make([]*serviceports.CreditMemoApplicationInput, 0, len(input.Applications))
	multiErr := errortypes.NewMultiError()
	for idx, app := range input.Applications {
		if app == nil {
			multiErr.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrRequired, "Application is required")
			continue
		}
		invoiceID, parseErr := pulid.MustParse(app.InvoiceID)
		if parseErr != nil {
			multiErr.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invalid invoice")
			continue
		}
		applications = append(applications, &serviceports.CreditMemoApplicationInput{
			InvoiceID:          invoiceID,
			AppliedAmountMinor: int64(app.AppliedAmountMinor),
		})
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &serviceports.ApplyCreditMemoRequest{
		CreditMemoID:   creditMemoID,
		AccountingDate: int64(input.AccountingDate),
		Applications:   applications,
		TenantInfo:     tenantInfo,
	}, nil
}
