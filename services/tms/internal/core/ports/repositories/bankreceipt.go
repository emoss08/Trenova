package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetBankReceiptByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetBankReceiptSummaryRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type BankReceiptRepository interface {
	GetByID(ctx context.Context, req GetBankReceiptByIDRequest) (*bankreceipt.Receipt, error)
	ListExceptions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*bankreceipt.Receipt, error)
	GetSummary(ctx context.Context, req GetBankReceiptSummaryRequest) (*bankreceipt.ReconciliationSummary, error)
	Create(ctx context.Context, entity *bankreceipt.Receipt) (*bankreceipt.Receipt, error)
	Update(ctx context.Context, entity *bankreceipt.Receipt) (*bankreceipt.Receipt, error)
}
