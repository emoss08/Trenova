package dispatchplanports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ReplacePricesRequest struct {
	TenantInfo  pagination.TenantInfo
	GeneratedAt int64
	MoveIDs     []pulid.ID
	Prices      []*dispatchplanprice.DispatchPlanPrice
}

type ListPricesRequest struct {
	TenantInfo pagination.TenantInfo
	MoveIDs    []pulid.ID
}

type PriceRepository interface {
	Replace(ctx context.Context, req *ReplacePricesRequest) error
	ListByMoves(
		ctx context.Context,
		req *ListPricesRequest,
	) ([]*dispatchplanprice.DispatchPlanPrice, error)
}
