package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trackingevent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TrackingEventVerdict struct {
	ID      pulid.ID
	Outcome trackingevent.Outcome
	Reason  string
}

type ListTrackingEventsByShipmentRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
}

type TrackingEventRepository interface {
	Insert(
		ctx context.Context,
		entity *trackingevent.TrackingEvent,
	) (*trackingevent.TrackingEvent, bool, error)
	ListByMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) ([]*trackingevent.TrackingEvent, error)
	ListByShipment(
		ctx context.Context,
		req *ListTrackingEventsByShipmentRequest,
	) ([]*trackingevent.TrackingEvent, error)
	RecordVerdicts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		verdicts []TrackingEventVerdict,
		at int64,
	) error
}
