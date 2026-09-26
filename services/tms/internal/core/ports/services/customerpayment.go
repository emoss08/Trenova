package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CustomerPaymentApplicationInput struct {
	InvoiceID           pulid.ID `json:"invoiceId"`
	AppliedAmountMinor  int64    `json:"appliedAmountMinor"`
	ShortPayAmountMinor int64    `json:"shortPayAmountMinor"`
}

type PostCustomerPaymentRequest struct {
	CustomerID      pulid.ID                           `json:"customerId"`
	PaymentDate     int64                              `json:"paymentDate"`
	AccountingDate  int64                              `json:"accountingDate"`
	AmountMinor     int64                              `json:"amountMinor"`
	PaymentMethod   customerpayment.Method             `json:"paymentMethod"`
	ReferenceNumber string                             `json:"referenceNumber"`
	Memo            string                             `json:"memo"`
	CurrencyCode    string                             `json:"currencyCode"`
	Applications    []*CustomerPaymentApplicationInput `json:"applications"`
	TenantInfo      pagination.TenantInfo              `json:"tenantInfo"`
}

type GetCustomerPaymentRequest struct {
	PaymentID  pulid.ID              `json:"paymentId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ReverseCustomerPaymentRequest struct {
	PaymentID      pulid.ID              `json:"paymentId"`
	AccountingDate int64                 `json:"accountingDate"`
	Reason         string                `json:"reason"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
}

type ApplyCustomerPaymentRequest struct {
	PaymentID      pulid.ID                           `json:"paymentId"`
	AccountingDate int64                              `json:"accountingDate"`
	Applications   []*CustomerPaymentApplicationInput `json:"applications"`
	TenantInfo     pagination.TenantInfo              `json:"tenantInfo"`
}

type CreditMemoApplicationInput struct {
	InvoiceID          pulid.ID `json:"invoiceId"`
	AppliedAmountMinor int64    `json:"appliedAmountMinor"`
}

// ApplyCreditMemoRequest uses a posted credit memo to settle open invoices of
// the same customer.
type ApplyCreditMemoRequest struct {
	CreditMemoID   pulid.ID                      `json:"creditMemoId"`
	AccountingDate int64                         `json:"accountingDate"`
	Applications   []*CreditMemoApplicationInput `json:"applications"`
	TenantInfo     pagination.TenantInfo         `json:"tenantInfo"`
}

type UnapplyCreditMemoApplicationRequest struct {
	ApplicationID  pulid.ID              `json:"applicationId"`
	AccountingDate int64                 `json:"accountingDate"`
	Reason         string                `json:"reason"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
}

// CustomerPaymentPostPreview is a payment as posting it would record it,
// and each invoice it pays, in the order of its applications, before and
// after.
type CustomerPaymentPostPreview struct {
	Payment        *customerpayment.Payment
	InvoicesBefore []*invoice.Invoice
	InvoicesAfter  []*invoice.Invoice
}

type CustomerPaymentChangePreview struct {
	PaymentBefore  *customerpayment.Payment
	PaymentAfter   *customerpayment.Payment
	InvoicesBefore []*invoice.Invoice
	InvoicesAfter  []*invoice.Invoice
	Journal        *JournalPreview
}

type CreditMemoApplicationPreview struct {
	CreditMemoBefore  *invoice.Invoice
	CreditMemoAfter   *invoice.Invoice
	InvoicesBefore    []*invoice.Invoice
	InvoicesAfter     []*invoice.Invoice
	Applications      []*customerpayment.CreditMemoApplication
	ApplicationBefore *customerpayment.CreditMemoApplication
}

type CustomerPaymentService interface {
	List(
		ctx context.Context,
		req *repositories.ListCustomerPaymentsRequest,
	) (*pagination.ListResult[*customerpayment.Payment], error)
	Get(ctx context.Context, req *GetCustomerPaymentRequest) (*customerpayment.Payment, error)
	PostAndApply(
		ctx context.Context,
		req *PostCustomerPaymentRequest,
		actor *RequestActor,
	) (*customerpayment.Payment, error)
	// PreviewPostAndApply validates a post exactly as PostAndApply does and
	// returns the payment and invoices as it would leave them, writing
	// nothing.
	PreviewPostAndApply(
		ctx context.Context,
		req *PostCustomerPaymentRequest,
		actor *RequestActor,
	) (*CustomerPaymentPostPreview, error)
	ApplyUnapplied(
		ctx context.Context,
		req *ApplyCustomerPaymentRequest,
		actor *RequestActor,
	) (*customerpayment.Payment, error)
	Reverse(
		ctx context.Context,
		req *ReverseCustomerPaymentRequest,
		actor *RequestActor,
	) (*customerpayment.Payment, error)
	ApplyCreditMemo(
		ctx context.Context,
		req *ApplyCreditMemoRequest,
		actor *RequestActor,
	) ([]*customerpayment.CreditMemoApplication, error)
	UnapplyCreditMemoApplication(
		ctx context.Context,
		req *UnapplyCreditMemoApplicationRequest,
		actor *RequestActor,
	) (*customerpayment.CreditMemoApplication, error)
	PreviewApplyUnapplied(
		ctx context.Context,
		req *ApplyCustomerPaymentRequest,
		actor *RequestActor,
	) (*CustomerPaymentChangePreview, error)
	PreviewReverse(
		ctx context.Context,
		req *ReverseCustomerPaymentRequest,
		actor *RequestActor,
	) (*CustomerPaymentChangePreview, error)
	PreviewApplyCreditMemo(
		ctx context.Context,
		req *ApplyCreditMemoRequest,
		actor *RequestActor,
	) (*CreditMemoApplicationPreview, error)
	PreviewUnapplyCreditMemoApplication(
		ctx context.Context,
		req *UnapplyCreditMemoApplicationRequest,
		actor *RequestActor,
	) (*CreditMemoApplicationPreview, error)
}
