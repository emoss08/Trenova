package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
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

type CustomerPaymentService interface {
	Get(ctx context.Context, req *GetCustomerPaymentRequest) (*customerpayment.Payment, error)
	PostAndApply(
		ctx context.Context,
		req *PostCustomerPaymentRequest,
		actor *RequestActor,
	) (*customerpayment.Payment, error)
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
}
