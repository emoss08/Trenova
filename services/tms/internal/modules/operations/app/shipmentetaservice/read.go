package shipmentetaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) EtasByShipmentIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]*shipmenttrackingports.ShipmentEta, error) {
	out := make(map[pulid.ID]*shipmenttrackingports.ShipmentEta, len(shipmentIDs))
	if len(shipmentIDs) == 0 {
		return out, nil
	}

	byShipment, err := s.storedRows(ctx, tenantInfo, shipmentIDs)
	if err != nil {
		return nil, err
	}
	if len(byShipment) == 0 {
		return out, nil
	}

	attention := make([]pulid.ID, 0, len(byShipment))
	for id, rows := range byShipment {
		if anyNeedsAttention(rows) {
			attention = append(attention, id)
		}
	}
	reasons, err := s.failureReasons(ctx, tenantInfo, attention)
	if err != nil {
		return nil, err
	}

	for id, rows := range byShipment {
		if eta := Summarize(id, rows, reasons[id]); eta != nil {
			out[id] = eta
		}
	}

	return out, nil
}

func (s *Service) TrackingSnapshots(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
	timezone string,
) (map[pulid.ID]*shipmenttracking.Snapshot, error) {
	out := make(map[pulid.ID]*shipmenttracking.Snapshot, len(shipmentIDs))
	if len(shipmentIDs) == 0 {
		return out, nil
	}

	entities, err := s.shipments.ListTrackingShipments(
		ctx,
		&shipmentports.ListTrackingShipmentsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: shipmentIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	sources, err := s.collect(ctx, tenantInfo, entities)
	if err != nil {
		return nil, err
	}
	byShipment, err := s.storedRows(ctx, tenantInfo, shipmentIDs)
	if err != nil {
		return nil, err
	}

	now := s.now().Unix()
	for _, entity := range entities {
		etas := make(map[pulid.ID]*stopeta.StopETA, len(byShipment[entity.ID]))
		for _, row := range byShipment[entity.ID] {
			etas[row.StopID] = row
		}
		out[entity.ID] = shipmenttracking.Build(shipmenttracking.Input{
			Shipment:    entity,
			Assignments: sources.assignments,
			Positions:   sources.positions,
			HOS:         sources.hos,
			StopETAs:    etas,
			Now:         now,
			Timezone:    timezone,
		})
	}

	return out, nil
}

func (s *Service) storedRows(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID][]*stopeta.StopETA, error) {
	rows, err := s.stopEtas.ListByShipmentIDs(ctx, tenantInfo, shipmentIDs)
	if err != nil {
		return nil, err
	}

	byShipment := make(map[pulid.ID][]*stopeta.StopETA, len(shipmentIDs))
	for _, row := range rows {
		byShipment[row.ShipmentID] = append(byShipment[row.ShipmentID], row)
	}

	return byShipment, nil
}

func anyNeedsAttention(rows []*stopeta.StopETA) bool {
	for _, row := range rows {
		if row.NeedsAttention() {
			return true
		}
	}
	return false
}

func Summarize(
	shipmentID pulid.ID,
	rows []*stopeta.StopETA,
	failureReason string,
) *shipmenttrackingports.ShipmentEta {
	if len(rows) == 0 {
		return nil
	}

	governing := Governing(rows)
	eta := &shipmenttrackingports.ShipmentEta{
		ShipmentID:       shipmentID,
		StopID:           governing.StopID,
		EstimatedArrival: governing.EstimatedArrival,
		SlackMinutes:     governing.SlackMinutes,
		Verdict:          governing.Verdict,
		Basis:            governing.Basis,
		InputStale:       governing.InputStale,
		ComputedAt:       governing.ComputedAt,
		Stops:            rows,
	}
	if governing.NeedsAttention() {
		eta.Reason = failureReason
		if eta.Reason == "" {
			eta.Reason = governing.Reason
		}
	}

	return eta
}

func Governing(rows []*stopeta.StopETA) *stopeta.StopETA {
	var next, worst *stopeta.StopETA
	for _, row := range rows {
		if row == nil || row.Arrived {
			continue
		}
		if next == nil {
			next = row
		}
		if row.NeedsAttention() && (worst == nil || row.Verdict.Rank() > worst.Verdict.Rank()) {
			worst = row
		}
	}

	switch {
	case worst != nil:
		return worst
	case next != nil:
		return next
	default:
		return rows[0]
	}
}
