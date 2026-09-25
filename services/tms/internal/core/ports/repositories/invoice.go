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

type GetInvoicesByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"invoiceIds"`
}

// ListInvoicesByShipmentIDsRequest asks for every invoice that bills any of the
// shipments, whether as its header shipment or through a line: that is how a
// split bill's sibling invoices find each other.
type ListInvoicesByShipmentIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	ShipmentIDs []pulid.ID            `json:"-"`
}

type UpdateInvoiceEDISendStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
	MessageID  pulid.ID              `json:"-"`
	Status     invoice.EDISendStatus `json:"-"`
	Error      string                `json:"-"`
	SentAt     *int64                `json:"-"`
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

// ListInvoiceLineChargesRequest asks which shipment charges the invoices' lines
// bill.
type ListInvoiceLineChargesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"-"`
}

// InvoiceLineCharge is one shipment charge an invoice line bills.
type InvoiceLineCharge struct {
	AdditionalChargeID pulid.ID `bun:"additional_charge_id"`
	ShipmentID         pulid.ID `bun:"shipment_id"`
}

// NetBilledByChargeRequest asks what is still billed of each shipment charge
// across every invoice that has not been voided.
type NetBilledByChargeRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ChargeIDs  []pulid.ID            `json:"-"`
}

type InvoiceRepository interface {
	// ListLineCharges returns the distinct shipment charges the invoices'
	// lines bill.
	ListLineCharges(
		ctx context.Context,
		req *ListInvoiceLineChargesRequest,
	) ([]InvoiceLineCharge, error)
	// NetBilledByCharge sums each charge's lines over the invoices that still
	// stand: a voided invoice drops out, a credit memo that credits a charge
	// counts against it, and the credit memo of a full reversal or a write-off
	// does not, because the reversal voids the invoice it credits and a
	// write-off gives up collecting a charge that was billed.
	NetBilledByCharge(
		ctx context.Context,
		req *NetBilledByChargeRequest,
	) (map[pulid.ID]decimal.Decimal, error)
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
	GetByIDs(
		ctx context.Context,
		req GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
	GetByBillingQueueItemID(
		ctx context.Context,
		req GetInvoiceByBillingQueueItemIDRequest,
	) (*invoice.Invoice, error)
	ListByShipmentIDs(
		ctx context.Context,
		req ListInvoicesByShipmentIDsRequest,
	) (map[pulid.ID][]*invoice.Invoice, error)
	// LockForUpdate reads the invoice and its lines under a row lock inside the
	// caller's transaction.
	LockForUpdate(
		ctx context.Context,
		req GetInvoiceByIDRequest,
	) (*invoice.Invoice, error)
	UpdateEDISendStatus(
		ctx context.Context,
		req UpdateInvoiceEDISendStatusRequest,
	) error
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
