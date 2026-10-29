package billingjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs/billingcontract"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// ReportAutoPostFailureActivity hands a failed automatic post to the invoice
// service, which tells the approver the invoice is still a draft.
func (a *Activities) ReportAutoPostFailureActivity(
	ctx context.Context,
	payload *billingcontract.AutoPostFailurePayload,
) error {
	return a.invoicePosting.ReportAutoPostFailure(ctx, &invoiceports.ReportAutoPostFailureRequest{
		InvoiceID: payload.InvoiceID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  payload.OrganizationID,
			BuID:   payload.BusinessUnitID,
			UserID: payload.UserID,
		},
		Reason: payload.Reason,
		Actor: kernelports.AuditActor{
			PrincipalType: payload.PrincipalType,
			PrincipalID:   payload.PrincipalID,
			UserID:        payload.UserID,
			APIKeyID:      payload.APIKeyID,
		},
	})
}
