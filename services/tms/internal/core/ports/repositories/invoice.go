package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ListInvoicesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetInvoiceByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetInvoiceByBillingQueueItemIDRequest struct {
	BillingQueueItemID pulid.ID              `json:"billingQueueItemId"`
	TenantInfo         pagination.TenantInfo `json:"tenantInfo"`
}

type CountPostedInvoiceReconciliationDiscrepanciesRequest struct {
	OrgID           pulid.ID        `json:"orgId"`
	BuID            pulid.ID        `json:"buId"`
	PeriodStartDate int64           `json:"periodStartDate"`
	PeriodEndDate   int64           `json:"periodEndDate"`
	ToleranceAmount decimal.Decimal `json:"toleranceAmount"`
}

type InvoiceRepository interface {
	List(
		ctx context.Context,
		req *ListInvoicesRequest,
	) (*pagination.ListResult[*invoice.Invoice], error)
	GetByID(
		ctx context.Context,
		req GetInvoiceByIDRequest,
	) (*invoice.Invoice, error)
	GetByBillingQueueItemID(
		ctx context.Context,
		req GetInvoiceByBillingQueueItemIDRequest,
	) (*invoice.Invoice, error)
	CountPostedReconciliationDiscrepancies(
		ctx context.Context,
		req CountPostedInvoiceReconciliationDiscrepanciesRequest,
	) (int, error)
	Create(
		ctx context.Context,
		entity *invoice.Invoice,
	) (*invoice.Invoice, error)
	Update(
		ctx context.Context,
		entity *invoice.Invoice,
	) (*invoice.Invoice, error)
}
