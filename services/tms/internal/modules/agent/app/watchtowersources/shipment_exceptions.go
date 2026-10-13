package watchtowersources

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	secondsPerMinute = 60
	secondsPerHour   = 3600
	// podLookbackDays bounds how far back a missing POD is still chased. A
	// load delivered three months ago without paperwork is a billing write
	// off, not something for today's queue.
	podLookbackDays = 90
	// criticalAfterGraceMultiple is how many graces past due an item goes
	// critical: a stop two hours past a thirty-minute grace, a POD three
	// days past a day's grace.
	criticalAfterGraceMultiple = 4
	minimumCriticalSeconds     = 2 * secondsPerHour
)

type ShipmentExceptionParams struct {
	fx.In

	Reader   watchtowerports.ExceptionSourceReader
	Controls dispatchcontrolports.DispatchControlRepository
}

type shipmentExceptions struct {
	reader   watchtowerports.ExceptionSourceReader
	controls dispatchcontrolports.DispatchControlRepository
	now      func() int64
}

func newShipmentExceptions(p ShipmentExceptionParams) shipmentExceptions {
	return shipmentExceptions{reader: p.Reader, controls: p.Controls, now: timeutils.NowUnix}
}

func (e shipmentExceptions) policy(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (dispatchcontrol.ExceptionPolicy, error) {
	control, err := e.controls.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID},
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return (*dispatchcontrol.DispatchControl)(nil).ExceptionPolicy(), nil
		}
		return dispatchcontrol.ExceptionPolicy{}, err
	}
	return control.ExceptionPolicy(), nil
}

func pickupTypes() []shipment.StopType {
	return []shipment.StopType{shipment.StopTypePickup, shipment.StopTypeSplitPickup}
}

func deliveryTypes() []shipment.StopType {
	return []shipment.StopType{shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery}
}

func (e shipmentExceptions) missedStops(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
	shipmentIDs []pulid.ID,
) ([]watchtowerports.WatchtowerItemInput, error) {
	policy, err := e.policy(ctx, tenant)
	if err != nil {
		return nil, err
	}
	types := pickupTypes()
	if kind == watchtower.SourceMissedDelivery {
		types = deliveryTypes()
	}
	now := e.now()
	grace := int64(policy.MissedStopGrace) * secondsPerMinute
	stops, err := e.reader.ListMissedStops(ctx, &watchtowerports.ListMissedStopsRequest{
		TenantInfo:   tenant,
		StopTypes:    types,
		ClosedBefore: now - grace,
		ShipmentIDs:  shipmentIDs,
	})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(stops))
	for _, stop := range stops {
		if stop != nil && stop.StopID.IsNotNil() {
			items = append(items, DescribeMissedStop(tenant, kind, stop, grace, now))
		}
	}
	return items, nil
}

func (e shipmentExceptions) missingPODs(
	ctx context.Context,
	tenant pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) ([]watchtowerports.WatchtowerItemInput, error) {
	policy, err := e.policy(ctx, tenant)
	if err != nil {
		return nil, err
	}
	now := e.now()
	grace := int64(policy.PODGraceHours) * secondsPerHour
	missing, err := e.reader.ListMissingPODs(ctx, &watchtowerports.ListMissingPODsRequest{
		TenantInfo:      tenant,
		DeliveredBefore: now - grace,
		DeliveredAfter:  now - podLookbackDays*24*secondsPerHour,
		ShipmentIDs:     shipmentIDs,
	})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(missing))
	for _, row := range missing {
		if row != nil && row.ShipmentID.IsNotNil() {
			items = append(items, DescribeMissingPOD(tenant, row, grace, now))
		}
	}
	return items, nil
}

func (e shipmentExceptions) exhaustedTenders(
	ctx context.Context,
	tenant pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) ([]watchtowerports.WatchtowerItemInput, error) {
	tenders, err := e.reader.ListExhaustedTenders(ctx, &watchtowerports.ListExhaustedTendersRequest{
		TenantInfo:  tenant,
		ShipmentIDs: shipmentIDs,
	})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(tenders))
	for _, row := range tenders {
		if row != nil && row.TenderID.IsNotNil() {
			items = append(items, DescribeExhaustedTender(tenant, row))
		}
	}
	return items, nil
}

func (e shipmentExceptions) forKind(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
	shipmentIDs []pulid.ID,
) ([]watchtowerports.WatchtowerItemInput, error) {
	if kind == watchtower.SourceMissedPickup || kind == watchtower.SourceMissedDelivery {
		return e.missedStops(ctx, tenant, kind, shipmentIDs)
	}
	if kind == watchtower.SourcePodMissing {
		return e.missingPODs(ctx, tenant, shipmentIDs)
	}
	if kind == watchtower.SourceTenderExhausted {
		return e.exhaustedTenders(ctx, tenant, shipmentIDs)
	}
	return []watchtowerports.WatchtowerItemInput{}, nil
}

func ShipmentExceptionKinds() []watchtower.SourceKind {
	return []watchtower.SourceKind{
		watchtower.SourceMissedPickup,
		watchtower.SourceMissedDelivery,
		watchtower.SourcePodMissing,
		watchtower.SourceTenderExhausted,
	}
}

type ShipmentExceptionSource struct {
	kind       watchtower.SourceKind
	exceptions shipmentExceptions
}

func (s *ShipmentExceptionSource) Kind() watchtower.SourceKind { return s.kind }

func (s *ShipmentExceptionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	return s.exceptions.forKind(ctx, tenant, s.kind, nil)
}

func newShipmentExceptionSource(
	kind watchtower.SourceKind,
	p ShipmentExceptionParams,
) watchtowerports.WatchtowerSource {
	return &ShipmentExceptionSource{kind: kind, exceptions: newShipmentExceptions(p)}
}

func NewMissedPickupSource(p ShipmentExceptionParams) watchtowerports.WatchtowerSource {
	return newShipmentExceptionSource(watchtower.SourceMissedPickup, p)
}

func NewMissedDeliverySource(p ShipmentExceptionParams) watchtowerports.WatchtowerSource {
	return newShipmentExceptionSource(watchtower.SourceMissedDelivery, p)
}

func NewPodMissingSource(p ShipmentExceptionParams) watchtowerports.WatchtowerSource {
	return newShipmentExceptionSource(watchtower.SourcePodMissing, p)
}

func NewTenderExhaustedSource(p ShipmentExceptionParams) watchtowerports.WatchtowerSource {
	return newShipmentExceptionSource(watchtower.SourceTenderExhausted, p)
}

func shipmentPath(shipmentID pulid.ID) string {
	if shipmentID.IsNotNil() {
		if recordPath, ok := productguide.RecordPath("shipment", shipmentID.String()); ok {
			return recordPath
		}
	}
	return pathDispatchConsole
}

func pastDueSeverity(overdueSeconds, grace int64) watchtower.Severity {
	threshold := max(grace*criticalAfterGraceMultiple, minimumCriticalSeconds)
	if overdueSeconds >= threshold {
		return watchtower.SeverityCritical
	}
	return watchtower.SeverityWarning
}

func onPro(title, proNumber string) string {
	if proNumber == "" {
		return title
	}
	return title + " on " + proNumber
}

func missedStopPlace(stop *watchtowerports.MissedStop) string {
	city := stop.City
	if city != "" && stop.StateCode != "" {
		city += ", " + stop.StateCode
	}
	switch {
	case stop.LocationName != "" && city != "":
		return stop.LocationName + " in " + city
	case stop.LocationName != "":
		return stop.LocationName
	case city != "":
		return city
	default:
		return "the stop"
	}
}

func DescribeMissedStop(
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
	stop *watchtowerports.MissedStop,
	grace, now int64,
) watchtowerports.WatchtowerItemInput {
	what := "pickup"
	title := "Missed pickup"
	if kind == watchtower.SourceMissedDelivery {
		what = "delivery"
		title = "Missed delivery"
	}
	late := max(now-stop.Cutoff, 0)
	summary := "The " + what + " window at " + missedStopPlace(stop) + " closed " +
		timeutils.DescribeAgo(stop.Cutoff, now) + " and no arrival has been recorded."

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  kind,
		SourceID:    stop.StopID.String(),
		Severity:    pastDueSeverity(late-grace, grace),
		Title:       onPro(title, stop.ProNumber),
		Summary:     summary,
		SubjectType: agent.SubjectShipment,
		SubjectID:   stop.ShipmentID,
		Path:        shipmentPath(stop.ShipmentID),
		OccurredAt:  stop.Cutoff + grace,
		OwnerID:     stop.OwnerID,
	}
}

func DescribeMissingPOD(
	tenant pagination.TenantInfo,
	row *watchtowerports.MissingPOD,
	grace, now int64,
) watchtowerports.WatchtowerItemInput {
	since := max(now-row.DeliveredAt, 0)
	ago := timeutils.DescribeAgo(row.DeliveredAt, now)
	summary := "Delivered " + ago + " and no proof of delivery is on file."
	if row.CustomerName != "" {
		summary = "Delivered for " + row.CustomerName + " " + ago +
			" and no proof of delivery is on file."
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourcePodMissing,
		SourceID:    row.ShipmentID.String(),
		Severity:    pastDueSeverity(since-grace, grace),
		Title:       onPro("POD missing", row.ProNumber),
		Summary:     summary,
		SubjectType: agent.SubjectShipment,
		SubjectID:   row.ShipmentID,
		Path:        shipmentPath(row.ShipmentID),
		OccurredAt:  row.DeliveredAt + grace,
		OwnerID:     row.OwnerID,
	}
}

func DescribeExhaustedTender(
	tenant pagination.TenantInfo,
	row *watchtowerports.ExhaustedTender,
) watchtowerports.WatchtowerItemInput {
	return watchtowerports.WatchtowerItemInput{
		TenantInfo: tenant,
		SourceKind: watchtower.SourceTenderExhausted,
		SourceID:   row.TenderID.String(),
		Severity:   watchtower.SeverityCritical,
		Title:      onPro("Tender exhausted", row.ProNumber),
		Summary: "Every carrier in the routing guide passed or let the offer lapse, " +
			"and the move is still uncovered.",
		SubjectType: agent.SubjectShipment,
		SubjectID:   row.ShipmentID,
		Path:        shipmentPath(row.ShipmentID),
		OccurredAt:  row.ExhaustedAt,
		OwnerID:     row.OwnerID,
	}
}
