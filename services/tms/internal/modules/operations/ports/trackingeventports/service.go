package trackingeventports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RecordObservedStopEventParams struct {
	TenantInfo   pagination.TenantInfo
	MoveID       pulid.ID
	StopID       pulid.ID
	Kind         shipment.VisitKind
	Source       trackingevent.Source
	SourceKey    string
	MatchMethod  trackingevent.MatchMethod
	EventAt      int64
	Latitude     *float64
	Longitude    *float64
	SourceStatus string
	RawReference string
}

type RecordReportedStopEventParams struct {
	TenantInfo   pagination.TenantInfo
	MoveID       pulid.ID
	StopID       pulid.ID
	Action       shipmentports.StopActualAction
	OccurredAt   *int64
	Source       trackingevent.Source
	SourceKey    string
	ReportedByID pulid.ID
	Latitude     *float64
	Longitude    *float64
}

type StopEventResult struct {
	Event    *trackingevent.TrackingEvent
	Move     *shipment.ShipmentMove
	Outcome  trackingevent.Outcome
	Reason   string
	Replayed bool
}

type TrackingEventService interface {
	RecordObserved(
		ctx context.Context,
		params *RecordObservedStopEventParams,
	) (*StopEventResult, error)
	RecordReported(
		ctx context.Context,
		params *RecordReportedStopEventParams,
	) (*StopEventResult, error)
	Reconcile(ctx context.Context, tenantInfo pagination.TenantInfo, moveID pulid.ID) error
	ListForShipment(
		ctx context.Context,
		req *ListTrackingEventsByShipmentRequest,
	) ([]*trackingevent.TrackingEvent, error)
}
