package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ImportBankReceiptRequest struct {
	ReceiptDate     int64                 `json:"receiptDate"`
	AmountMinor     int64                 `json:"amountMinor"`
	ReferenceNumber string                `json:"referenceNumber"`
	Memo            string                `json:"memo"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
}

type MatchBankReceiptRequest struct {
	ReceiptID  pulid.ID              `json:"receiptId"`
	PaymentID  pulid.ID              `json:"paymentId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetBankReceiptRequest struct {
	ReceiptID  pulid.ID              `json:"receiptId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BankReceiptService interface {
	Get(ctx context.Context, req *GetBankReceiptRequest) (*bankreceipt.Receipt, error)
	Import(
		ctx context.Context,
		req *ImportBankReceiptRequest,
		actor *RequestActor,
	) (*bankreceipt.Receipt, error)
	Match(
		ctx context.Context,
		req *MatchBankReceiptRequest,
		actor *RequestActor,
	) (*bankreceipt.Receipt, error)
	ListExceptions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*bankreceipt.Receipt, error)
	SuggestMatches(
		ctx context.Context,
		req *GetBankReceiptRequest,
	) ([]*bankreceipt.MatchSuggestion, error)
	GetSummary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		asOfDate int64,
	) (*bankreceipt.ReconciliationSummary, error)
}
