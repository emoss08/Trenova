package shipmenteventports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

type RecordShipmentEventParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ShipmentID     pulid.ID
	MoveID         pulid.ID
	StopID         pulid.ID
	AssignmentID   pulid.ID
	CommentID      pulid.ID
	HoldID         pulid.ID

	Type     shipmentevent.Type
	Severity shipmentevent.Severity
	Summary  string
	Metadata map[string]any

	Actor         kernelports.AuditActor
	ActorLabel    string
	CorrelationID string
	OccurredAt    int64
}

type ShipmentEventService interface {
	Record(ctx context.Context, params *RecordShipmentEventParams) error
	List(
		ctx context.Context,
		req *ListShipmentEventsRequest,
	) ([]*shipmentevent.Event, error)
}

type ShipmentEventObserver interface {
	OnShipmentEvent(ctx context.Context, event *shipmentevent.Event) error
}
