package triageshipmentservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const stopStatusCompleted = string(shipment.StopStatusCompleted)

func (s *Service) ShipmentPeeks(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]*inboundmessageports.ShipmentPeek, error) {
	peeks := make(map[pulid.ID]*inboundmessageports.ShipmentPeek, len(shipmentIDs))
	if len(shipmentIDs) == 0 {
		return peeks, nil
	}

	details, err := s.lookup.PeekDetails(ctx, tenantInfo, shipmentIDs)
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		return peeks, nil
	}

	snapshots, err := s.tracking.TrackingSnapshots(ctx, tenantInfo, shipmentIDs, time.UTC.String())
	if err != nil {
		return nil, err
	}

	summaries, err := s.summaries.ListSummariesByIDs(ctx, &shipmentports.ListShipmentSummariesRequest{
		TenantInfo:  tenantInfo,
		ShipmentIDs: shipmentIDs,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*shipmentports.ShipmentSummary, len(summaries))
	for _, summary := range summaries {
		byID[summary.ShipmentID] = summary
	}

	for id, detail := range details {
		peeks[id] = buildPeek(detail, byID[id], snapshots[id])
	}

	return peeks, nil
}

func buildPeek(
	detail *shipmentports.TriagePeekDetail,
	summary *shipmentports.ShipmentSummary,
	snapshot *shipmenttracking.Snapshot,
) *inboundmessageports.ShipmentPeek {
	peek := &inboundmessageports.ShipmentPeek{
		ID:              detail.ShipmentID,
		Status:          string(detail.Status),
		CustomerName:    detail.CustomerName,
		Equipment:       detail.TrailerType,
		DeliveredAt:     detail.ActualDeliveryDate,
		Stops:           []*inboundmessageports.PeekStop{},
		DocumentsOnFile: []string{},
	}
	if summary != nil {
		peek.ProNumber = summary.ProNumber
		peek.BOL = summary.BOL
		peek.PONumber = summary.PONumber
	}
	if snapshot == nil {
		peek.Progress = deliveredProgress(peek)

		return peek
	}

	if peek.ProNumber == "" {
		peek.ProNumber = snapshot.ProNumber
	}
	if peek.CustomerName == "" {
		peek.CustomerName = snapshot.Customer
	}

	done, total := 0, 0
	for _, move := range snapshot.Moves {
		if peek.DriverName == "" && move.Driver != "" {
			peek.DriverName = move.Driver
		}
		if peek.Equipment == "" && move.Trailer != "" {
			peek.Equipment = move.Trailer
		}
		for i := range move.Stops {
			stop := &move.Stops[i]
			total++
			completed := stop.Status == stopStatusCompleted
			if completed {
				done++
			}
			peek.Stops = append(peek.Stops, peekStop(stop, completed))
			if stop.Late || stop.Overdue {
				peek.Late = true
				peek.LateMinutes = max(peek.LateMinutes, int(stop.LateMinutes))
			}
		}
	}

	if total > 0 {
		peek.Progress = float64(done) / float64(total)
	}
	if estimate := snapshot.Estimate; estimate != nil {
		if estimate.EstimatedArrival > 0 {
			eta := estimate.EstimatedArrival
			peek.ETA = &eta
		}
		if estimate.Verdict == shipmenttracking.VerdictLate {
			peek.Late = true
			if estimate.SlackMinutes < 0 {
				peek.LateMinutes = max(peek.LateMinutes, int(-estimate.SlackMinutes))
			}
		}
	}
	if peek.DeliveredAt != nil {
		peek.Progress = 1
		peek.ETA = nil
	}

	return peek
}

func peekStop(stop *shipmenttracking.StopSnapshot, completed bool) *inboundmessageports.PeekStop {
	out := &inboundmessageports.PeekStop{
		Kind:  stop.Type,
		City:  stop.City,
		State: stop.State,
		Done:  completed,
	}
	if stop.ScheduledStart > 0 {
		at := stop.ScheduledStart
		out.ScheduledAt = &at
	}
	if stop.ActualArrival > 0 {
		at := stop.ActualArrival
		out.ArrivedAt = &at
	}
	if stop.ActualDeparture > 0 {
		at := stop.ActualDeparture
		out.DepartedAt = &at
	}

	return out
}

func deliveredProgress(peek *inboundmessageports.ShipmentPeek) float64 {
	if peek.DeliveredAt != nil {
		return 1
	}

	return 0
}

func (s *Service) CustomerShipments(
	ctx context.Context,
	req *inboundmessageports.TriageCustomerShipmentsRequest,
) (*inboundmessageports.TriageCustomerShipments, error) {
	ids, thisWeek, err := s.lookup.CustomerShipments(ctx, &shipmentports.TriageCustomerShipmentsQuery{
		TenantInfo: req.TenantInfo,
		CustomerID: req.CustomerID,
		WeekStart:  req.WeekStart,
		Limit:      req.Limit,
	})
	if err != nil {
		return nil, err
	}

	return &inboundmessageports.TriageCustomerShipments{ShipmentIDs: ids, ThisWeek: thisWeek}, nil
}
