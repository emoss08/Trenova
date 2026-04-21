package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BankReceiptExceptionAging struct {
	CurrentCount    int64 `json:"currentCount"`
	CurrentAmount   int64 `json:"currentAmount"`
	Days1To3Count   int64 `json:"days1To3Count"`
	Days1To3Amount  int64 `json:"days1To3Amount"`
	Days4To7Count   int64 `json:"days4To7Count"`
	Days4To7Amount  int64 `json:"days4To7Amount"`
	DaysOver7Count  int64 `json:"daysOver7Count"`
	DaysOver7Amount int64 `json:"daysOver7Amount"`
}

type BankReceiptReconciliationSummary struct {
	AsOfDate              int64                    `json:"asOfDate"`
	ImportedCount         int64                    `json:"importedCount"`
	ImportedAmount        int64                    `json:"importedAmount"`
	MatchedCount          int64                    `json:"matchedCount"`
	MatchedAmount         int64                    `json:"matchedAmount"`
	ExceptionCount        int64                    `json:"exceptionCount"`
	ExceptionAmount       int64                    `json:"exceptionAmount"`
	ActiveWorkItemCount   int64                    `json:"activeWorkItemCount"`
	AssignedWorkItemCount int64                    `json:"assignedWorkItemCount"`
	InReviewWorkItemCount int64                    `json:"inReviewWorkItemCount"`
	ExceptionAging        BankReceiptExceptionAging `json:"exceptionAging"`
}

type GetBankReceiptByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetBankReceiptSummaryRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type ListBankReceiptsByImportBatchRequest struct {
	BatchID    pulid.ID              `json:"batchId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BankReceiptRepository interface {
	GetByID(ctx context.Context, req GetBankReceiptByIDRequest) (*bankreceipt.BankReceipt, error)
	ListByImportBatchID(
		ctx context.Context,
		req ListBankReceiptsByImportBatchRequest,
	) ([]*bankreceipt.BankReceipt, error)
	ListExceptions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*bankreceipt.BankReceipt, error)
	GetSummary(
		ctx context.Context,
		req GetBankReceiptSummaryRequest,
	) (*BankReceiptReconciliationSummary, error)
	Create(ctx context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error)
	Update(ctx context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error)
}
