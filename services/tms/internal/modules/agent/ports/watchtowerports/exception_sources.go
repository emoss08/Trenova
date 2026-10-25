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
	StopID       pulid.ID
	MoveID       pulid.ID
	ShipmentID   pulid.ID
	ProNumber    string
	OwnerID      pulid.ID
	StopType     shipment.StopType
	Cutoff       int64
	LocationName string
	City         string
	StateCode    string
}

type ListMissingPODsRequest struct {
	TenantInfo      pagination.TenantInfo
	DeliveredBefore int64
	DeliveredAfter  int64
	ShipmentIDs     []pulid.ID
	Limit           int
}

type MissingPOD struct {
	ShipmentID   pulid.ID
	ProNumber    string
	OwnerID      pulid.ID
	CustomerName string
	DeliveredAt  int64
}

type ListExhaustedTendersRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
	Limit       int
}

type ExhaustedTender struct {
	TenderID    pulid.ID
	MoveID      pulid.ID
	ShipmentID  pulid.ID
	ProNumber   string
	OwnerID     pulid.ID
	ExhaustedAt int64
}

type ExceptionSourceReader interface {
	ListMissedStops(ctx context.Context, req *ListMissedStopsRequest) ([]*MissedStop, error)
	ListMissingPODs(ctx context.Context, req *ListMissingPODsRequest) ([]*MissingPOD, error)
	ListExhaustedTenders(
		ctx context.Context,
		req *ListExhaustedTendersRequest,
	) ([]*ExhaustedTender, error)
}
