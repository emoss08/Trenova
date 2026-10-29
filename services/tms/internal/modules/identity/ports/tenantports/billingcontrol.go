package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetBillingControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BillingControlRepository interface {
	GetByOrgID(
		ctx context.Context,
		orgID pulid.ID,
	) (*billingcontrol.BillingControl, error)
	Update(
		ctx context.Context,
		bc *billingcontrol.BillingControl,
	) (*billingcontrol.BillingControl, error)
	// ListAutomaticTransfer lists, across every organization, the billing
	// controls that move ready freight into billing on their own.
	ListAutomaticTransfer(ctx context.Context) ([]*billingcontrol.BillingControl, error)
	// ListPaymentReminders lists, across every organization, the billing
	// controls that email customers about overdue invoices.
	ListPaymentReminders(ctx context.Context) ([]*billingcontrol.BillingControl, error)
	// ListMonthlyStatements lists, across every organization, the billing
	// controls that email customers a statement each month.
	ListMonthlyStatements(ctx context.Context) ([]*billingcontrol.BillingControl, error)
}
