package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	repositoryports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BankReceiptMatchSuggestion struct {
	CustomerPaymentID pulid.ID `json:"customerPaymentId"`
	ReferenceNumber   string   `json:"referenceNumber"`
	AmountMinor       int64    `json:"amountMinor"`
	CustomerID        pulid.ID `json:"customerId"`
	Score             int      `json:"score"`
	Reason            string   `json:"reason"`
}

type ImportBankReceiptRequest struct {
	ReceiptDate     int64                 `json:"receiptDate"`
	AmountMinor     int64                 `json:"amountMinor"`
	ReferenceNumber string                `json:"referenceNumber"`
	Memo            string                `json:"memo"`
	BatchID         pulid.ID              `json:"batchId"`
	SkipAudit       bool                  `json:"-"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
}

type MatchBankReceiptRequest struct {
	ReceiptID  pulid.ID              `json:"receiptId"`
	PaymentID  pulid.ID              `json:"paymentId"`
	SkipAudit  bool                  `json:"-"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetBankReceiptRequest struct {
	ReceiptID  pulid.ID              `json:"receiptId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BankReceiptService interface {
	Get(ctx context.Context, req *GetBankReceiptRequest) (*bankreceipt.BankReceipt, error)
	Import(
		ctx context.Context,
		req *ImportBankReceiptRequest,
		actor *RequestActor,
	) (*bankreceipt.BankReceipt, error)
	Match(
		ctx context.Context,
		req *MatchBankReceiptRequest,
		actor *RequestActor,
	) (*bankreceipt.BankReceipt, error)
	ListExceptions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*bankreceipt.BankReceipt, error)
	SuggestMatches(
		ctx context.Context,
		req *GetBankReceiptRequest,
	) ([]*BankReceiptMatchSuggestion, error)
	GetSummary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		asOfDate int64,
	) (*repositoryports.BankReceiptReconciliationSummary, error)
}
