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

type FindCustomerPaymentMatchCandidatesRequest struct {
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	ReferenceNumber string                `json:"referenceNumber"`
	AmountMinor     int64                 `json:"amountMinor"`
	ReceiptDate     int64                 `json:"receiptDate"`
}

type CustomerPaymentRepository interface {
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
}
