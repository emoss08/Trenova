package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/shipmenttracking"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ShipmentEta struct {
	ShipmentID       pulid.ID
	EstimatedArrival *int64
	SlackMinutes     *int64
	Verdict          shipmenttracking.Verdict
	Reason           string
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
