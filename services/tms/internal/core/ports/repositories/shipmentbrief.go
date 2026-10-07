package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetLatestShipmentBriefRequest struct {
	TenantInfo pagination.TenantInfo
	BriefDate  string
}

type DeleteShipmentBriefsBeforeRequest struct {
	BeforeDate string
	Limit      int
}

type ShipmentBriefRepository interface {
	Latest(
		ctx context.Context,
		req *GetLatestShipmentBriefRequest,
	) (*shipmentbrief.Brief, error)
	Insert(ctx context.Context, brief *shipmentbrief.Brief) (*shipmentbrief.Brief, error)
	DeleteBefore(ctx context.Context, req *DeleteShipmentBriefsBeforeRequest) (int, error)
}
