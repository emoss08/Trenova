package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetCustomerPaymentByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListCustomerPaymentsRequest struct {
	Filter     *pagination.QueryOptions `json:"filter"`
	CustomerID pulid.ID                 `json:"customerId"`
	Status     customerpayment.Status   `json:"status"`
}

type FindCustomerPaymentMatchCandidatesRequest struct {
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	ReferenceNumber string                `json:"referenceNumber"`
	AmountMinor     int64                 `json:"amountMinor"`
	ReceiptDate     int64                 `json:"receiptDate"`
}

type ListCustomerPaymentConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListApplicationsByInvoiceIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"-"`
}

type GetCreditMemoApplicationRequest struct {
	ID         pulid.ID              `json:"-"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type CustomerPaymentRepository interface {
	List(
		ctx context.Context,
		req *ListCustomerPaymentsRequest,
	) (*pagination.ListResult[*customerpayment.Payment], error)
	ListConnection(
		ctx context.Context,
		req *ListCustomerPaymentConnectionRequest,
	) (*pagination.CursorListResult[*customerpayment.Payment], error)
	GetByID(
		ctx context.Context,
		req GetCustomerPaymentByIDRequest,
	) (*customerpayment.Payment, error)
	FindMatchCandidates(
		ctx context.Context,
		req FindCustomerPaymentMatchCandidatesRequest,
	) ([]*customerpayment.Payment, error)
	FindSuggestedMatchCandidates(
		ctx context.Context,
		req FindCustomerPaymentMatchCandidatesRequest,
	) ([]*customerpayment.Payment, error)
	Create(ctx context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error)
	Update(ctx context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error)
	// ListApplicationsByInvoiceIDs returns every cash application against the
	// invoices, keyed by invoice, with the payment loaded. Reversed payments are
	// included so an invoice's history reads whole.
	ListApplicationsByInvoiceIDs(
		ctx context.Context,
		req *ListApplicationsByInvoiceIDsRequest,
	) (map[pulid.ID][]*customerpayment.Application, error)
	CreateCreditMemoApplications(
		ctx context.Context,
		applications []*customerpayment.CreditMemoApplication,
	) error
	UpdateCreditMemoApplication(
		ctx context.Context,
		application *customerpayment.CreditMemoApplication,
	) (*customerpayment.CreditMemoApplication, error)
	GetCreditMemoApplicationByID(
		ctx context.Context,
		req GetCreditMemoApplicationRequest,
	) (*customerpayment.CreditMemoApplication, error)
	// ListCreditMemoApplicationsByInvoiceIDs keys rows under both the invoice
	// they settle and the credit memo they draw on.
	ListCreditMemoApplicationsByInvoiceIDs(
		ctx context.Context,
		req *ListApplicationsByInvoiceIDsRequest,
	) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error)
}
