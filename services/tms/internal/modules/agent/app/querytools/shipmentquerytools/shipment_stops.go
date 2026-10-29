package shipmentquerytools

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ShipmentStopSummary struct {
	ID              string `json:"id"`
	Sequence        int64  `json:"sequence"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	LocationID      string `json:"locationId"`
	Location        string `json:"location,omitempty"`
	City            string `json:"city,omitempty"`
	State           string `json:"state,omitempty"`
	Timezone        string `json:"timezone"`
	WindowStart     string `json:"scheduledWindowStart"`
	WindowEnd       string `json:"scheduledWindowEnd,omitempty"`
	ActualArrival   string `json:"actualArrival,omitempty"`
	ActualDeparture string `json:"actualDeparture,omitempty"`
}

func StopSummary(stop *shipment.Stop, fallback string) ShipmentStopSummary {
	stopZone := ""
	if stop.Location != nil {
		stopZone = stop.Location.Timezone
	}
	zone, zoneName := timeutils.ResolveZone(stopZone, fallback)

	out := ShipmentStopSummary{
		ID:              stop.ID.String(),
		Sequence:        stop.Sequence,
		Type:            string(stop.Type),
		Status:          string(stop.Status),
		LocationID:      querytool.PulidString(stop.LocationID),
		Timezone:        zoneName,
		WindowStart:     timeutils.FormatLocalDateTime(stop.ScheduledWindowStart, zone),
		WindowEnd:       localDateTime(stop.ScheduledWindowEnd, zone),
		ActualArrival:   localDateTime(stop.ActualArrival, zone),
		ActualDeparture: localDateTime(stop.ActualDeparture, zone),
	}
	if loc := stop.Location; loc != nil {
		out.Location = strings.TrimSpace(loc.Name)
		out.City = strings.TrimSpace(loc.City)
		if loc.State != nil {
			out.State = loc.State.Abbreviation
		}
	}

	return out
}

func localDateTime(ts *int64, zone *time.Location) string {
	if ts == nil || *ts == 0 {
		return ""
	}

	return timeutils.FormatLocalDateTime(*ts, zone)
}

func HoldNote(hold *shipmenthold.ShipmentHold) string {
	if hold.Source != shipmenthold.HoldSourceUser && hold.Source != shipmenthold.HoldSourceRule {
		return ""
	}

	return strings.TrimSpace(hold.Notes)
}
