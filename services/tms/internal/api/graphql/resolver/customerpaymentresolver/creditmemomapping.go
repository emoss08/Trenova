package customerpaymentresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func applyCreditMemoRequestFromInput(
	input *gqlmodel.ApplyCreditMemoInput,
	tenantInfo pagination.TenantInfo,
) (*customerpaymentports.ApplyCreditMemoRequest, error) {
	creditMemoID, err := pulid.MustParse(input.CreditMemoID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"creditMemoId",
			errortypes.ErrInvalid,
			"Invalid credit memo",
		)
	}

	applications := make([]*customerpaymentports.CreditMemoApplicationInput, 0, len(input.Applications))
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
		applications = append(applications, &customerpaymentports.CreditMemoApplicationInput{
			InvoiceID:          invoiceID,
			AppliedAmountMinor: int64(app.AppliedAmountMinor),
		})
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &customerpaymentports.ApplyCreditMemoRequest{
		CreditMemoID:   creditMemoID,
		AccountingDate: int64(input.AccountingDate),
		Applications:   applications,
		TenantInfo:     tenantInfo,
	}, nil
}
