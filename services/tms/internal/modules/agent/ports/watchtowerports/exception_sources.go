package watchtowerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListMissedStopsRequest struct {
	TenantInfo   pagination.TenantInfo
	StopTypes    []shipment.StopType
	ClosedBefore int64
	ShipmentIDs  []pulid.ID
	Limit        int
}

type MissedStop struct {
	StopID       pulid.ID          `bun:"stop_id"`
	MoveID       pulid.ID          `bun:"move_id"`
	ShipmentID   pulid.ID          `bun:"shipment_id"`
	ProNumber    string            `bun:"pro_number"`
	OwnerID      pulid.ID          `bun:"owner_id"`
	StopType     shipment.StopType `bun:"stop_type"`
	Cutoff       int64             `bun:"cutoff"`
	LocationName string            `bun:"location_name"`
	City         string            `bun:"city"`
	StateCode    string            `bun:"state_code"`
}

type ListMissingPODsRequest struct {
	TenantInfo      pagination.TenantInfo
	DeliveredBefore int64
	DeliveredAfter  int64
	ShipmentIDs     []pulid.ID
	Limit           int
}

type MissingPOD struct {
	ShipmentID   pulid.ID `bun:"shipment_id"`
	ProNumber    string   `bun:"pro_number"`
	OwnerID      pulid.ID `bun:"owner_id"`
	CustomerName string   `bun:"customer_name"`
	DeliveredAt  int64    `bun:"delivered_at"`
}

type ListExhaustedTendersRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
	Limit       int
}

type ExhaustedTender struct {
	TenderID    pulid.ID `bun:"tender_id"`
	MoveID      pulid.ID `bun:"move_id"`
	ShipmentID  pulid.ID `bun:"shipment_id"`
	ProNumber   string   `bun:"pro_number"`
	OwnerID     pulid.ID `bun:"owner_id"`
	ExhaustedAt int64    `bun:"exhausted_at"`
}

type ExceptionSourceReader interface {
	ListMissedStops(ctx context.Context, req *ListMissedStopsRequest) ([]*MissedStop, error)
	ListMissingPODs(ctx context.Context, req *ListMissingPODsRequest) ([]*MissingPOD, error)
	ListExhaustedTenders(
		ctx context.Context,
		req *ListExhaustedTendersRequest,
	) ([]*ExhaustedTender, error)
}
