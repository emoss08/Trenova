package watchtowershipment

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweritem"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxStopVisitTitleLength = 200
)

func TelematicsStopVisitSourceID(event *telematics.TelematicsEvent) string {
	target := "unlocated"
	switch {
	case event.StopID.IsNotNil():
		target = event.StopID.String()
	case event.LocationID.IsNotNil():
		target = event.LocationID.String()
	}

	return event.ShipmentMoveID.String() + ":" + target + ":" + string(event.StopVisit)
}

func DescribeTelematicsStopVisit(
	review *telematicsports.StopReview,
) watchtowerports.WatchtowerItemInput {
	event := review.Event

	visit := "Arrival"
	if event.StopVisit == shipmentvocab.VisitDeparture {
		visit = "Departure"
	}
	title := visit + " not recorded"
	if place := strings.TrimSpace(event.AddressName); place != "" {
		title = visit + " at " + place + " not recorded"
	}

	path := watchtoweritem.PathDispatchConsole
	if review.ShipmentID.IsNotNil() {
		if recordPath, ok := productguide.RecordPath("shipment", review.ShipmentID.String()); ok {
			path = recordPath
		}
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: event.OrganizationID,
			BuID:  event.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceTelematicsStopVisit,
		SourceID:    TelematicsStopVisitSourceID(event),
		Severity:    watchtower.SeverityWarning,
		Title:       stringutils.TruncateRunes(title, maxStopVisitTitleLength),
		Summary:     event.StopOutcomeReason,
		SubjectType: agent.SubjectShipmentMove,
		SubjectID:   event.ShipmentMoveID,
		Path:        path,
		Facts: []watchtower.Fact{
			watchtoweritem.Fact("Place", event.AddressName),
			watchtoweritem.FactAt("Seen", event.OccurredAt),
		},
		OccurredAt: event.OccurredAt,
	}
}

func DescribeNoCheckCall(
	tenant pagination.TenantInfo,
	move *checkcallports.OverdueMove,
) watchtowerports.WatchtowerItemInput {
	window := int64(move.IntervalHours) * 60 * 60
	severity := watchtower.SeverityWarning
	if window > 0 && move.OverdueBy >= window {
		severity = watchtower.SeverityCritical
	}

	title := "No check call"
	if move.ProNumber != "" {
		title = "No check call on " + move.ProNumber
	}

	carrier := stringutils.FirstNonEmpty(move.CarrierName, "The carrier")
	hours := strconv.FormatInt(int64(move.IntervalHours), 10)
	summary := carrier + " has not checked in within the " + hours + "h window for this load."
	if severity == watchtower.SeverityCritical {
		summary = carrier + " has been silent for more than twice the " + hours +
			"h check call window on this load."
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceNoCheckCall,
		SourceID:    move.MoveID.String(),
		Severity:    severity,
		Title:       title,
		Summary:     summary,
		SubjectType: agent.SubjectShipmentMove,
		SubjectID:   move.MoveID,
		Path:        shipmentPath(move.ShipmentID),
		Record:      watchtoweritem.ShipmentRecord(move.ProNumber),
		Facts: []watchtower.Fact{
			watchtoweritem.Fact("Carrier", move.CarrierName),
			watchtoweritem.FactAt("Last contact", move.LastContactAt),
			watchtoweritem.Fact("Window", hours+"h"),
		},
		Trail: []watchtower.TrailEvent{
			{At: move.LastContactAt, Text: "Last heard from " + carrier},
			{At: move.LastContactAt + window, Text: "Check call window closed"},
		},
		OccurredAt: move.LastContactAt + window,
		OwnerID:    move.OwnerID,
	}
}

func DescribeEtaRisk(
	tenant pagination.TenantInfo,
	stop *stopetaports.AttentionStop,
) watchtowerports.WatchtowerItemInput {
	row := stop.ETA
	late := row.Verdict == stopeta.VerdictLate
	severity := watchtower.SeverityWarning
	if late {
		severity = watchtower.SeverityCritical
	}

	place := etaStopPlace(stop)
	title := "ETA at risk"
	summary := "Projected to reach " + place + " with less than " +
		strconv.FormatInt(row.ThresholdMinutes, 10) + " minutes to spare."
	if late {
		title = "Projected late"
		summary = "Projected to reach " + place + " after its window closes."
	}
	if stop.ProNumber != "" {
		title += " on " + stop.ProNumber
	}
	if row.InputStale {
		summary += " The projection is working from old information."
	}

	windowLabel := "Delivery window closes"
	if stop.StopType == shipmentvocab.StopTypePickup {
		windowLabel = "Pickup window closes"
	}
	facts := []watchtower.Fact{
		watchtoweritem.Fact("Stop", place),
		watchtoweritem.FactAtPtr("Window closes", row.WindowEnd),
	}
	if row.EstimatedArrival != nil {
		facts = append(facts, watchtoweritem.FactAt("New ETA", *row.EstimatedArrival))
	}
	if row.MilesRemaining > 0 {
		facts = append(
			facts,
			watchtoweritem.Fact("Miles to go", strconv.FormatFloat(row.MilesRemaining, 'f', 0, 64)),
		)
	}
	trail := make([]watchtower.TrailEvent, 0, 2)
	if row.InputAt != nil {
		trail = append(
			trail,
			watchtower.TrailEvent{At: *row.InputAt, Text: "Last position or check call"},
		)
	}
	if reason := strings.TrimSpace(row.Reason); reason != "" {
		trail = append(trail, watchtower.TrailEvent{Text: reason})
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceEtaAtRisk,
		SourceID:    row.ShipmentID.String(),
		Severity:    severity,
		Title:       title,
		Summary:     summary,
		SubjectType: agent.SubjectShipment,
		SubjectID:   row.ShipmentID,
		Path:        shipmentPath(row.ShipmentID),
		Record:      watchtoweritem.ShipmentRecord(stop.ProNumber),
		Facts:       facts,
		Trail:       trail,
		Deadline:    watchtoweritem.DeadlineAtPtr(row.WindowEnd, windowLabel),
		OccurredAt:  etaRiskOccurredAt(row, late),
		OwnerID:     stop.OwnerID,
	}
}

func etaStopPlace(stop *stopetaports.AttentionStop) string {
	kind := "the " + strings.ToLower(stringutils.FirstNonEmpty(string(stop.StopType), "stop"))
	city := stop.City
	if city != "" && stop.StateCode != "" {
		city += ", " + stop.StateCode
	}
	switch {
	case stop.LocationName != "" && city != "":
		return kind + " at " + stop.LocationName + " in " + city
	case stop.LocationName != "":
		return kind + " at " + stop.LocationName
	case city != "":
		return kind + " in " + city
	default:
		return kind
	}
}

func etaRiskOccurredAt(row *stopeta.StopETA, late bool) int64 {
	if row.WindowEnd == nil {
		return row.ComputedAt
	}
	if late {
		return min(*row.WindowEnd, row.ComputedAt)
	}
	return min(*row.WindowEnd-row.ThresholdMinutes*60, row.ComputedAt)
}
