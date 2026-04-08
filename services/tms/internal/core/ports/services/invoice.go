package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateInvoiceFromBillingQueueRequest struct {
	BillingQueueItemID pulid.ID
	TenantInfo         pagination.TenantInfo
}

type CreateInvoiceFromBillingQueueResult struct {
	Invoice  *invoice.Invoice
	AutoPost bool
}

type PostInvoiceRequest struct {
	InvoiceID   pulid.ID
	TenantInfo  pagination.TenantInfo
	TriggeredBy string
}

type InvoiceService interface {
	List(
		ctx context.Context,
		req *repositories.ListInvoicesRequest,
	) (*pagination.ListResult[*invoice.Invoice], error)
	GetByID(
		ctx context.Context,
		req repositories.GetInvoiceByIDRequest,
	) (*invoice.Invoice, error)
	CreateFromApprovedBillingQueueItem(
		ctx context.Context,
		req *CreateInvoiceFromBillingQueueRequest,
		actor *RequestActor,
	) (*CreateInvoiceFromBillingQueueResult, error)
	Post(
		ctx context.Context,
		req *PostInvoiceRequest,
		actor *RequestActor,
	) (*invoice.Invoice, error)
	EnqueueAutoPost(
		ctx context.Context,
		entity *invoice.Invoice,
		actor *RequestActor,
	) error
}
