package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
)

type GetStopByIDRequest struct {
	StopID            pulid.ID
	OrgID             pulid.ID
	BuID              pulid.ID
	UserID            pulid.ID
	ExpandStopDetails bool
}

type StopRepository interface {
	GetByID(ctx context.Context, req GetStopByIDRequest) (*shipment.Stop, error)
	BulkInsert(ctx context.Context, stops []*shipment.Stop) ([]*shipment.Stop, error)
	Update(ctx context.Context, stop *shipment.Stop, moveIdx, stopIdx int) (*shipment.Stop, error)
	HandleStopRemovals(
		ctx context.Context,
		tx bun.IDB,
		move *shipment.ShipmentMove,
		existingStops []*shipment.Stop,
		updatedStopIDs map[pulid.ID]struct{},
	) error
}
