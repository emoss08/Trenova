package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	DefaultUnattributedMovesPageSize = 200
	MaxUnattributedMovesPageSize     = 1000
)

type ListJurisdictionMilesByMoveIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MoveIDs    []pulid.ID            `json:"moveIds"`
}

type UnattributedMovesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Start      int64                 `json:"start"`
	End        int64                 `json:"end"`
	Limit      int                   `json:"limit"`
	Offset     int                   `json:"offset"`
}

func (r UnattributedMovesRequest) PageSize() int {
	if r.Limit <= 0 {
		return DefaultUnattributedMovesPageSize
	}
	if r.Limit > MaxUnattributedMovesPageSize {
		return MaxUnattributedMovesPageSize
	}
	return r.Limit
}

type UnattributedMovesPage struct {
	MoveIDs    []pulid.ID      `json:"moveIds"`
	TotalMoves int             `json:"totalMoves"`
	TotalMiles decimal.Decimal `json:"totalMiles"`
}

type ShipmentMoveJurisdictionMileRepository interface {
	ReplaceForMove(ctx context.Context, move *shipment.ShipmentMove) error
	ListByMoveIDs(
		ctx context.Context,
		req ListJurisdictionMilesByMoveIDsRequest,
	) ([]*shipment.ShipmentMoveJurisdictionMile, error)
	ListUnattributedMoves(
		ctx context.Context,
		req UnattributedMovesRequest,
	) (*UnattributedMovesPage, error)
}
