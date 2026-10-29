package recordanchorservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	msPerMinute = 60_000
	factStatus  = "status"
)

func shipmentReader(tracking shipmenttrackingports.ShipmentTrackingReader) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentturn.RuntimeAnchor, error) {
		snapshots, err := tracking.TrackingSnapshots(ctx, read.tenant, read.ids, read.timezone)
		if err != nil {
			return nil, err
		}

		out := make(map[pulid.ID]*agentturn.RuntimeAnchor, len(snapshots))
		for id, snapshot := range snapshots {
			if snapshot == nil {
				continue
			}
			facts := snapshot.Facts()
			anchor := &agentturn.RuntimeAnchor{
				Label: snapshot.ProNumber,
				Facts: make([]agentturn.AnchorFact, 0, len(facts)),
			}
			for _, fact := range facts {
				anchor.Facts = append(anchor.Facts, anchorFact(fact))
			}
			out[id] = anchor
		}

		return out, nil
	}
}

func anchorFact(fact shipmenttracking.Fact) agentturn.AnchorFact {
	return agentturn.AnchorFact{
		Name:   fact.Name,
		Value:  fact.Value,
		SeenAt: fact.SeenAt,
		Stale:  fact.Stale,
	}
}

func invoiceReader(invoices invoiceports.InvoiceRepository) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentturn.RuntimeAnchor, error) {
		found, err := invoices.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
			TenantInfo: read.tenant,
			InvoiceIDs: read.ids,
		})
		if err != nil {
			return nil, err
		}

		out := make(map[pulid.ID]*agentturn.RuntimeAnchor, len(found))
		for _, entity := range found {
			if entity == nil {
				continue
			}
			out[entity.ID] = &agentturn.RuntimeAnchor{
				Label: entity.Number,
				Facts: invoiceFacts(entity, read),
			}
		}

		return out, nil
	}
}

func invoiceFacts(entity *invoice.Invoice, read *kindRead) []agentturn.AnchorFact {
	status := fmt.Sprintf("%s, %s", entity.Status, entity.SettlementStatus)
	if entity.DisputeStatus != "" && entity.DisputeStatus != invoice.DisputeStatusNone {
		status += ", dispute " + string(entity.DisputeStatus)
	}
	facts := []agentturn.AnchorFact{{Name: factStatus, Value: status}}
	if entity.BillToName != "" {
		facts = append(facts, agentturn.AnchorFact{Name: "bill to", Value: entity.BillToName})
	}
	if entity.ShipmentProNumber != "" {
		facts = append(
			facts,
			agentturn.AnchorFact{Name: "shipment", Value: entity.ShipmentProNumber},
		)
	}
	if entity.DueDate != nil && *entity.DueDate > 0 {
		due := timeutils.FormatCalendarDate(*entity.DueDate, read.zone)
		if *entity.DueDate < read.now && entity.SettlementStatus != invoice.SettlementStatusPaid {
			due += ", past due"
		}
		facts = append(facts, agentturn.AnchorFact{Name: "due", Value: due})
	}

	return facts
}

func tractorReader(
	tractors tractorports.TractorRepository,
	telem telematicsports.TelematicsRepository,
) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentturn.RuntimeAnchor, error) {
		found, err := tractors.GetByIDs(ctx, tractorports.GetTractorsByIDsRequest{
			TenantInfo: read.tenant,
			TractorIDs: read.ids,
		})
		if err != nil {
			return nil, err
		}
		positions := map[pulid.ID]*telematics.VehiclePosition{}
		if telem != nil {
			listed, listErr := telem.ListVehiclePositions(
				ctx,
				&telematicsports.ListVehiclePositionsRequest{
					TenantInfo: read.tenant,
					TractorIDs: read.ids,
				},
			)
			if listErr != nil {
				return nil, listErr
			}
			for _, position := range listed {
				if position != nil {
					positions[position.TractorID] = position
				}
			}
		}

		out := make(map[pulid.ID]*agentturn.RuntimeAnchor, len(found))
		for _, entity := range found {
			if entity == nil {
				continue
			}
			anchor := &agentturn.RuntimeAnchor{
				Label: entity.Code,
				Facts: []agentturn.AnchorFact{{Name: factStatus, Value: string(entity.Status)}},
			}
			if position, ok := positions[entity.ID]; ok {
				anchor.Facts = append(anchor.Facts, positionFact(position, read.now))
			}
			out[entity.ID] = anchor
		}

		return out, nil
	}
}

func positionFact(position *telematics.VehiclePosition, now int64) agentturn.AnchorFact {
	where := position.FormattedLocation
	if where == "" {
		where = fmt.Sprintf("%.4f, %.4f", position.Latitude, position.Longitude)
	}
	if position.SpeedMph > 0 {
		where += fmt.Sprintf(", moving at %.0f mph", position.SpeedMph)
	}

	return agentturn.AnchorFact{
		Name:   "last position",
		Value:  where,
		SeenAt: position.RecordedAt,
		Stale:  shipmenttracking.PositionStale(position.RecordedAt, now),
	}
}

func trailerReader(trailers trailerports.TrailerRepository) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentturn.RuntimeAnchor, error) {
		found, err := trailers.GetByIDs(ctx, trailerports.GetTrailersByIDsRequest{
			TenantInfo: read.tenant,
			TrailerIDs: read.ids,
		})
		if err != nil {
			return nil, err
		}

		out := make(map[pulid.ID]*agentturn.RuntimeAnchor, len(found))
		for _, entity := range found {
			if entity == nil {
				continue
			}
			out[entity.ID] = &agentturn.RuntimeAnchor{
				Label: entity.Code,
				Facts: []agentturn.AnchorFact{{Name: factStatus, Value: string(entity.Status)}},
			}
		}

		return out, nil
	}
}

func workerReader(telem telematicsports.TelematicsRepository) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentturn.RuntimeAnchor, error) {
		states, err := telem.ListWorkerHOSStates(ctx, &telematicsports.ListWorkerHOSStatesRequest{
			TenantInfo:    read.tenant,
			WorkerIDs:     read.ids,
			IncludeWorker: true,
			Limit:         len(read.ids),
		})
		if err != nil {
			return nil, err
		}

		out := make(map[pulid.ID]*agentturn.RuntimeAnchor, len(read.ids))
		for _, state := range states {
			if state == nil {
				continue
			}
			label := read.labels[state.WorkerID]
			if state.Worker != nil {
				label = strings.TrimSpace(state.Worker.FirstName + " " + state.Worker.LastName)
			}
			out[state.WorkerID] = &agentturn.RuntimeAnchor{
				Label: label,
				Facts: []agentturn.AnchorFact{hoursFact(state, read.now)},
			}
		}
		for _, id := range read.ids {
			if _, ok := out[id]; !ok {
				out[id] = &agentturn.RuntimeAnchor{
					Label: read.labels[id],
					Facts: []agentturn.AnchorFact{{
						Name:  "driver hours",
						Value: "no ELD has reported hours of service, so they are unknown",
					}},
				}
			}
		}

		return out, nil
	}
}

func hoursFact(state *telematics.WorkerHOSState, now int64) agentturn.AnchorFact {
	duty := string(state.DutyStatus)
	if duty == "" {
		duty = "duty status unknown"
	}

	return agentturn.AnchorFact{
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
	occurrences detentionports.DetentionOccurrenceRepository,
	pricer clockPricer,
) kindReader {
	return func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentturn.RuntimeAnchor, error) {
		out := make(map[pulid.ID]*agentturn.RuntimeAnchor, len(read.ids))
		for _, id := range read.ids {
			occurrence, err := occurrences.GetByID(
				ctx,
				&detentionports.GetDetentionOccurrenceByIDRequest{
					OccurrenceID: id,
					TenantInfo:   read.tenant,
				},
			)
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
) (*agentturn.RuntimeAnchor, error) {
	label := occurrence.ShipmentProNumber
	if occurrence.LocationName != "" {
		label = strings.TrimSpace(label + " at " + occurrence.LocationName)
	}
	facts := []agentturn.AnchorFact{{
		Name:  factStatus,
		Value: fmt.Sprintf("%s, notice %s", occurrence.Status, occurrence.NotificationStatus),
	}}
	if occurrence.ArrivedAt != nil && *occurrence.ArrivedAt > 0 {
		clock := "arrived " + timeutils.FormatUnixDateTimeIn(*occurrence.ArrivedAt, read.timezone)
		if occurrence.DepartedAt != nil && *occurrence.DepartedAt > 0 {
			clock += ", departed " + timeutils.FormatUnixDateTimeIn(
				*occurrence.DepartedAt,
				read.timezone,
			)
		} else {
			clock += ", no departure recorded"
		}
		facts = append(facts, agentturn.AnchorFact{Name: "clock", Value: clock})
	}

	minutes, amount := occurrence.BillableMinutes, occurrence.BillableAmount
	billable := agentturn.AnchorFact{Name: "billable", SeenAt: occurrence.UpdatedAt}
	if pricer != nil {
		price, err := pricer.PriceOpenClock(ctx, read.tenant, occurrence)
		switch {
		case errors.Is(err, detentionreview.ErrClockNotPriced):
		case err != nil:
			return nil, err
		default:
			minutes, amount = price.BillableMinutes, price.BillableAmount
			billable.SeenAt = 0
			billable.Name = "billable if the truck left now"
		}
	}
	billable.Value = fmt.Sprintf("%s, %s %s",
		timeutils.FormatLongDurationMs(int64(minutes)*msPerMinute),
		amount.StringFixed(2), occurrence.Currency)
	facts = append(facts, billable)

	return &agentturn.RuntimeAnchor{Label: label, Facts: facts}, nil
}
