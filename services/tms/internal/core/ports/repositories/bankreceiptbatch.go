package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BankReceiptBatchSourceOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type GetBankReceiptBatchByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BankReceiptBatchRepository interface {
	GetByID(
		ctx context.Context,
		req GetBankReceiptBatchByIDRequest,
	) (*bankreceiptbatch.BankReceiptBatch, error)
	List(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*bankreceiptbatch.BankReceiptBatch, error)
	Create(
		ctx context.Context,
		entity *bankreceiptbatch.BankReceiptBatch,
	) (*bankreceiptbatch.BankReceiptBatch, error)
	Update(
		ctx context.Context,
		entity *bankreceiptbatch.BankReceiptBatch,
	) (*bankreceiptbatch.BankReceiptBatch, error)
	DistinctSources(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*BankReceiptBatchSourceOption], error)
}
