package shipmenttrackingports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ShipmentEta struct {
	ShipmentID       pulid.ID
	StopID           pulid.ID
	EstimatedArrival *int64
	SlackMinutes     *int64
	Verdict          stopeta.Verdict
	Basis            stopeta.Basis
	InputStale       bool
	ComputedAt       int64
	Reason           string
	Stops            []*stopeta.StopETA
}

type ShipmentEtaReader interface {
	EtasByShipmentIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) (map[pulid.ID]*ShipmentEta, error)
}

type ShipmentTrackingReader interface {
	TrackingSnapshots(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
		timezone string,
	) (map[pulid.ID]*shipmenttracking.Snapshot, error)
}
