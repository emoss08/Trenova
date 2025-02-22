package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetStopByIDRequest struct {
	// ID of the stop
	StopID pulid.ID

	// ID of the organization
	OrgID pulid.ID

	// ID of the business unit
	BuID pulid.ID

	// ID of the user
	UserID pulid.ID

	// Expand stop details (Optional)
	ExpandStopDetails bool
}

type StopRepository interface {
	GetByID(ctx context.Context, req GetStopByIDRequest) (*shipment.Stop, error)
	BulkInsert(ctx context.Context, stops []*shipment.Stop) ([]*shipment.Stop, error)
	Update(ctx context.Context, stop *shipment.Stop, moveIdx, stopIdx int) (*shipment.Stop, error)
}
