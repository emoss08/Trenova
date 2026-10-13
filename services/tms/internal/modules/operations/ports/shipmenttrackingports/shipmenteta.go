package shipmenttrackingports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ShipmentEta struct {
	ShipmentID       pulid.ID           `json:"shipmentId"`
	StopID           pulid.ID           `json:"stopId,omitempty"`
	EstimatedArrival *int64             `json:"estimatedArrival"`
	SlackMinutes     *int64             `json:"slackMinutes"`
	Verdict          stopeta.Verdict    `json:"verdict"`
	Basis            stopeta.Basis      `json:"basis,omitempty"`
	InputStale       bool               `json:"inputStale"`
	ComputedAt       int64              `json:"computedAt,omitempty"`
	Reason           string             `json:"reason,omitempty"`
	Stops            []*stopeta.StopETA `json:"stops"`
}

func UnknownShipmentEta(shipmentID pulid.ID) *ShipmentEta {
	return &ShipmentEta{
		ShipmentID: shipmentID,
		Verdict:    stopeta.VerdictUnknown,
		Stops:      []*stopeta.StopETA{},
	}
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
