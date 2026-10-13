package checkcallports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListCheckCallsByShipmentRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	Limit      int
}

type MonitoredMove struct {
	MoveID                pulid.ID
	ShipmentID            pulid.ID
	ProNumber             string
	CarrierID             pulid.ID
	CarrierName           string
	CustomerIntervalHours *int16
	LastContactAt         int64
}

type CheckCallRepository interface {
	Insert(
		ctx context.Context,
		entity *checkcall.CheckCall,
	) (*checkcall.CheckCall, bool, error)
	ListByShipment(
		ctx context.Context,
		req *ListCheckCallsByShipmentRequest,
	) ([]*checkcall.CheckCall, error)
	LatestForMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) (*checkcall.CheckCall, error)
	ListMonitoredMoves(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		limit int,
	) ([]*MonitoredMove, error)
}

type RevokeTrackingLinksRequest struct {
	TenantInfo pagination.TenantInfo
	MoveID     pulid.ID
	At         int64
}

type TrackingLinkRepository interface {
	Create(ctx context.Context, entity *checkcall.TrackingLink) error
	GetByHash(ctx context.Context, tokenHash string) (*checkcall.TrackingLink, error)
	GetActiveForMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
		now int64,
	) (*checkcall.TrackingLink, error)
	RevokeForMove(ctx context.Context, req *RevokeTrackingLinksRequest) (int, error)
	Touch(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID, at int64) error
	PurgeDead(ctx context.Context, before int64, limit int) (int, error)
}
