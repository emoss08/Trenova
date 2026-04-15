package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetBankReceiptBatchByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BankReceiptBatchRepository interface {
	GetByID(ctx context.Context, req GetBankReceiptBatchByIDRequest) (*bankreceiptbatch.Batch, error)
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*bankreceiptbatch.Batch, error)
	Create(ctx context.Context, entity *bankreceiptbatch.Batch) (*bankreceiptbatch.Batch, error)
	Update(ctx context.Context, entity *bankreceiptbatch.Batch) (*bankreceiptbatch.Batch, error)
	DistinctSources(ctx context.Context, req *pagination.SelectQueryRequest) (*pagination.ListResult[*bankreceiptbatch.SourceOption], error)
}
