package dispatchmarketports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ListByCellsRequest struct {
	TenantInfo pagination.TenantInfo
	Cells      []int64
}

type Repository interface {
	ListByCells(
		ctx context.Context,
		req *ListByCellsRequest,
	) ([]*dispatchmarket.MarketValue, error)
	Upsert(ctx context.Context, values []*dispatchmarket.MarketValue) error
}

type RecordRequest struct {
	TenantInfo pagination.TenantInfo
	ObservedAt int64
	Values     []dispatchmarket.DriverValue
}

type ValuerRequest struct {
	TenantInfo pagination.TenantInfo
	Points     []dispatchmarket.Point
}

type Recorder interface {
	Record(ctx context.Context, req *RecordRequest) error
}

type ValuerSource interface {
	Valuer(ctx context.Context, req *ValuerRequest) (*dispatchmarket.Valuer, error)
}
