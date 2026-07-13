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

type ListInvoiceConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"-"`
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

type ListInvoiceEmailAttemptsRequest struct {
	InvoiceID  pulid.ID                 `json:"invoiceId"`
	TenantInfo pagination.TenantInfo    `json:"tenantInfo"`
	Filter     *pagination.QueryOptions `json:"filter"`
}

type UpsertInvoiceAttachmentsRequest struct {
	InvoiceID      pulid.ID              `json:"invoiceId"`
	DocumentIDs    []pulid.ID            `json:"documentIds"`
	OrganizationID pulid.ID              `json:"organizationId"`
	BusinessUnitID pulid.ID              `json:"businessUnitId"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
}

type GetInvoiceDocumentShareTokenRequest struct {
	TokenHash string `json:"-"`
}

type InvoiceRepository interface {
	List(
		ctx context.Context,
		req *ListInvoicesRequest,
	) (*pagination.ListResult[*invoice.Invoice], error)
	ListConnection(
		ctx context.Context,
		req *ListInvoiceConnectionRequest,
	) (*pagination.CursorListResult[*invoice.Invoice], error)
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
	UpsertAttachments(
		ctx context.Context,
		req UpsertInvoiceAttachmentsRequest,
	) ([]*invoice.Attachment, error)
	ListAttachments(
		ctx context.Context,
		req ListInvoiceEmailAttemptsRequest,
	) ([]*invoice.Attachment, error)
	CreateEmailAttempt(
		ctx context.Context,
		attempt *invoice.EmailAttempt,
		attachments []*invoice.EmailAttemptAttachment,
	) (*invoice.EmailAttempt, error)
	ListEmailAttempts(
		ctx context.Context,
		req ListInvoiceEmailAttemptsRequest,
	) (*pagination.ListResult[*invoice.EmailAttempt], error)
	SyncEmailAttemptsForMessage(
		ctx context.Context,
		messageID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) error
	CreateDocumentShareToken(
		ctx context.Context,
		token *invoice.DocumentShareToken,
	) (*invoice.DocumentShareToken, error)
	GetDocumentShareToken(
		ctx context.Context,
		req GetInvoiceDocumentShareTokenRequest,
	) (*invoice.DocumentShareToken, error)
	UpdateDocumentShareToken(
		ctx context.Context,
		token *invoice.DocumentShareToken,
	) (*invoice.DocumentShareToken, error)
}
