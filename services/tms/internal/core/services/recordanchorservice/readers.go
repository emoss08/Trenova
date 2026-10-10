package recordanchorservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmenttracking"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const msPerMinute = 60_000

func shipmentReader(tracking services.ShipmentTrackingReader) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentdefinition.RuntimeAnchor, error) {
		snapshots, err := tracking.TrackingSnapshots(ctx, read.tenant, read.ids, read.timezone)
		if err != nil {
			return nil, err
		}

		out := make(map[pulid.ID]*agentdefinition.RuntimeAnchor, len(snapshots))
		for id, snapshot := range snapshots {
			if snapshot == nil {
				continue
			}
			facts := snapshot.Facts()
			anchor := &agentdefinition.RuntimeAnchor{
				Label: snapshot.ProNumber,
				Facts: make([]agentdefinition.AnchorFact, 0, len(facts)),
			}
			for _, fact := range facts {
				anchor.Facts = append(anchor.Facts, anchorFact(fact))
			}
			out[id] = anchor
		}

		return out, nil
	}
}

func anchorFact(fact shipmenttracking.Fact) agentdefinition.AnchorFact {
	return agentdefinition.AnchorFact{
		Name:   fact.Name,
		Value:  fact.Value,
		SeenAt: fact.SeenAt,
		Stale:  fact.Stale,
	}
}

func invoiceReader(invoices repositories.InvoiceRepository) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentdefinition.RuntimeAnchor, error) {
		found, err := invoices.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
			TenantInfo: read.tenant,
			InvoiceIDs: read.ids,
		})
		if err != nil {
			return nil, err
		}

		out := make(map[pulid.ID]*agentdefinition.RuntimeAnchor, len(found))
		for _, entity := range found {
			if entity == nil {
				continue
			}
			out[entity.ID] = &agentdefinition.RuntimeAnchor{
				Label: entity.Number,
				Facts: invoiceFacts(entity, read),
			}
		}

		return out, nil
	}
}

func invoiceFacts(entity *invoice.Invoice, read *kindRead) []agentdefinition.AnchorFact {
	status := fmt.Sprintf("%s, %s", entity.Status, entity.SettlementStatus)
	if entity.DisputeStatus != "" && entity.DisputeStatus != invoice.DisputeStatusNone {
		status += ", dispute " + string(entity.DisputeStatus)
	}
	facts := []agentdefinition.AnchorFact{{Name: "status", Value: status}}
	if entity.BillToName != "" {
		facts = append(facts, agentdefinition.AnchorFact{Name: "bill to", Value: entity.BillToName})
	}
	if entity.ShipmentProNumber != "" {
		facts = append(facts, agentdefinition.AnchorFact{Name: "shipment", Value: entity.ShipmentProNumber})
	}
	if entity.DueDate != nil && *entity.DueDate > 0 {
		due := timeutils.FormatCalendarDate(*entity.DueDate, read.zone)
		if *entity.DueDate < read.now && entity.SettlementStatus != invoice.SettlementStatusPaid {
			due += ", past due"
		}
		facts = append(facts, agentdefinition.AnchorFact{Name: "due", Value: due})
	}

	return facts
}

func tractorReader(
	tractors repositories.TractorRepository,
	telem repositories.TelematicsRepository,
) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentdefinition.RuntimeAnchor, error) {
		found, err := tractors.GetByIDs(ctx, repositories.GetTractorsByIDsRequest{
			TenantInfo: read.tenant,
			TractorIDs: read.ids,
		})
		if err != nil {
			return nil, err
		}
		positions := map[pulid.ID]*telematics.VehiclePosition{}
		if telem != nil {
			listed, listErr := telem.ListVehiclePositions(ctx, &repositories.ListVehiclePositionsRequest{
				TenantInfo: read.tenant,
				TractorIDs: read.ids,
			})
			if listErr != nil {
				return nil, listErr
			}
			for _, position := range listed {
				if position != nil {
					positions[position.TractorID] = position
				}
			}
		}

		out := make(map[pulid.ID]*agentdefinition.RuntimeAnchor, len(found))
		for _, entity := range found {
			if entity == nil {
				continue
			}
			anchor := &agentdefinition.RuntimeAnchor{
				Label: entity.Code,
				Facts: []agentdefinition.AnchorFact{{Name: "status", Value: string(entity.Status)}},
			}
			if position, ok := positions[entity.ID]; ok {
				anchor.Facts = append(anchor.Facts, positionFact(position, read.now))
			}
			out[entity.ID] = anchor
		}

		return out, nil
	}
}

func positionFact(position *telematics.VehiclePosition, now int64) agentdefinition.AnchorFact {
	where := position.FormattedLocation
	if where == "" {
		where = fmt.Sprintf("%.4f, %.4f", position.Latitude, position.Longitude)
	}
	if position.SpeedMph > 0 {
		where += fmt.Sprintf(", moving at %.0f mph", position.SpeedMph)
	}

	return agentdefinition.AnchorFact{
		Name:   "last position",
		Value:  where,
		SeenAt: position.RecordedAt,
		Stale:  shipmenttracking.PositionStale(position.RecordedAt, now),
	}
}

func trailerReader(trailers repositories.TrailerRepository) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentdefinition.RuntimeAnchor, error) {
		found, err := trailers.GetByIDs(ctx, repositories.GetTrailersByIDsRequest{
			TenantInfo: read.tenant,
			TrailerIDs: read.ids,
		})
		if err != nil {
			return nil, err
		}

		out := make(map[pulid.ID]*agentdefinition.RuntimeAnchor, len(found))
		for _, entity := range found {
			if entity == nil {
				continue
			}
			out[entity.ID] = &agentdefinition.RuntimeAnchor{
				Label: entity.Code,
				Facts: []agentdefinition.AnchorFact{{Name: "status", Value: string(entity.Status)}},
			}
		}

		return out, nil
	}
}

func workerReader(telem repositories.TelematicsRepository) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentdefinition.RuntimeAnchor, error) {
		states, err := telem.ListWorkerHOSStates(ctx, &repositories.ListWorkerHOSStatesRequest{
			TenantInfo:    read.tenant,
			WorkerIDs:     read.ids,
			IncludeWorker: true,
			Limit:         len(read.ids),
		})
		if err != nil {
			return nil, err
		}

		out := make(map[pulid.ID]*agentdefinition.RuntimeAnchor, len(read.ids))
		for _, state := range states {
			if state == nil {
				continue
			}
			label := read.labels[state.WorkerID]
			if state.Worker != nil {
				label = strings.TrimSpace(state.Worker.FirstName + " " + state.Worker.LastName)
			}
			out[state.WorkerID] = &agentdefinition.RuntimeAnchor{
				Label: label,
				Facts: []agentdefinition.AnchorFact{hoursFact(state, read.now)},
			}
		}
		for _, id := range read.ids {
			if _, ok := out[id]; !ok {
				out[id] = &agentdefinition.RuntimeAnchor{
					Label: read.labels[id],
					Facts: []agentdefinition.AnchorFact{{
						Name:  "driver hours",
						Value: "no ELD has reported hours of service, so they are unknown",
					}},
				}
			}
		}

		return out, nil
	}
}

func hoursFact(state *telematics.WorkerHOSState, now int64) agentdefinition.AnchorFact {
	duty := string(state.DutyStatus)
	if duty == "" {
		duty = "duty status unknown"
	}

	return agentdefinition.AnchorFact{
		Name: "driver hours",
		Value: fmt.Sprintf("%s, %s drive and %s on duty left, %s in the cycle", duty,
			timeutils.FormatLongDurationMs(state.DriveRemainingMs),
			timeutils.FormatLongDurationMs(state.ShiftRemainingMs),
			timeutils.FormatLongDurationMs(state.CycleRemainingMs)),
		SeenAt: state.RecordedAt,
		Stale:  shipmenttracking.HOSStale(state.RecordedAt, now),
	}
}

func detentionReader(
	occurrences repositories.DetentionOccurrenceRepository,
	pricer clockPricer,
) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentdefinition.RuntimeAnchor, error) {
		out := make(map[pulid.ID]*agentdefinition.RuntimeAnchor, len(read.ids))
		for _, id := range read.ids {
			occurrence, err := occurrences.GetByID(ctx, &repositories.GetDetentionOccurrenceByIDRequest{
				OccurrenceID: id,
				TenantInfo:   read.tenant,
			})
			if err != nil {
				if errortypes.IsNotFoundError(err) {
					continue
				}
				return nil, err
			}
			if occurrence == nil {
				continue
			}
			anchor, err := detentionAnchor(ctx, occurrence, pricer, read)
			if err != nil {
				return nil, err
			}
			out[id] = anchor
		}

		return out, nil
	}
}

func detentionAnchor(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	pricer clockPricer,
	read *kindRead,
) (*agentdefinition.RuntimeAnchor, error) {
	label := occurrence.ShipmentProNumber
	if occurrence.LocationName != "" {
		label = strings.TrimSpace(label + " at " + occurrence.LocationName)
	}
	facts := []agentdefinition.AnchorFact{{
		Name:  "status",
		Value: fmt.Sprintf("%s, notice %s", occurrence.Status, occurrence.NotificationStatus),
	}}
	if occurrence.ArrivedAt != nil && *occurrence.ArrivedAt > 0 {
		clock := "arrived " + timeutils.FormatUnixDateTimeIn(*occurrence.ArrivedAt, read.timezone)
		if occurrence.DepartedAt != nil && *occurrence.DepartedAt > 0 {
			clock += ", departed " + timeutils.FormatUnixDateTimeIn(*occurrence.DepartedAt, read.timezone)
		} else {
			clock += ", no departure recorded"
		}
		facts = append(facts, agentdefinition.AnchorFact{Name: "clock", Value: clock})
	}

	minutes, amount := occurrence.BillableMinutes, occurrence.BillableAmount
	billable := agentdefinition.AnchorFact{Name: "billable", SeenAt: occurrence.UpdatedAt}
	if pricer != nil {
		price, err := pricer.PriceOpenClock(ctx, read.tenant, occurrence)
		if err != nil {
			return nil, err
		}
		if price != nil {
			minutes, amount = price.BillableMinutes, price.BillableAmount
			billable.SeenAt = 0
			billable.Name = "billable if the truck left now"
		}
	}
	billable.Value = fmt.Sprintf("%s, %s %s",
		timeutils.FormatLongDurationMs(int64(minutes)*msPerMinute),
		amount.StringFixed(2), occurrence.Currency)
	facts = append(facts, billable)

	return &agentdefinition.RuntimeAnchor{Label: label, Facts: facts}, nil
}
