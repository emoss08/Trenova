package detentionsync

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// SyncShipmentResult reports what the recalculation did, so callers can decide
// whether shipment charges need rebuilding.
type SyncShipmentResult struct {
	Occurrences []*detention.DetentionOccurrence
	Skipped     int
	TotalAmount decimal.Decimal
}

type computeStopParams struct {
	shipment        *shipment.Shipment
	move            *shipment.ShipmentMove
	stop            *shipment.Stop
	existing        *detention.DetentionOccurrence
	tenantInfo      pagination.TenantInfo
	commodityIDs    []pulid.ID
	location        *time.Location
	payRate         decimal.NullDecimal
	dayAccrued      map[string]decimal.Decimal
	shipmentAccrued decimal.Decimal
	// prices is what the shipment's own contract charges, when a contract
	// priced it. Nil means the organization defaults apply.
	prices *contractPrices
	now    int64
}

func buildOccurrence(
	p computeStopParams,
	policy *detentionpolicy.DetentionPolicy,
	snapshot detentionpolicy.PolicySnapshot,
	computed detentionengine.ComputeResult,
) *detention.DetentionOccurrence {
	policyID := policy.ID

	occurrence := &detention.DetentionOccurrence{
		OrganizationID:     p.tenantInfo.OrgID,
		BusinessUnitID:     p.tenantInfo.BuID,
		ShipmentID:         p.shipment.ID,
		ShipmentMoveID:     p.move.ID,
		StopID:             p.stop.ID,
		CustomerID:         p.shipment.CustomerID,
		LocationID:         p.stop.LocationID,
		DetentionPolicyID:  &policyID,
		PolicySnapshot:     &snapshot,
		CalculationTrace:   computed.Trace,
		StopType:           p.stop.Type,
		ScheduleType:       p.stop.ScheduleType,
		AppointmentStart:   appointmentStart(p.stop),
		AppointmentEnd:     p.stop.ScheduledWindowEnd,
		ArrivedAt:          p.stop.ActualArrival,
		DepartedAt:         p.stop.ActualDeparture,
		ClockStartAt:       computed.ClockStartAt,
		ClockStopAt:        computed.ClockStopAt,
		FreeTimeExpiresAt:  computed.FreeTimeExpiresAt,
		NoticeDueAt:        computed.NoticeDueAt,
		NoticeDeadlineAt:   computed.NoticeDeadlineAt,
		IsOpen:             computed.IsOpen,
		ArrivedLate:        computed.ArrivedLate,
		LateByMinutes:      computed.LateByMinutes,
		FreeMinutesGranted: computed.FreeMinutesGranted,
		RawDwellMinutes:    computed.RawDwellMinutes,
		BillableMinutes:    computed.BillableMinutes,
		RoundedMinutes:     computed.RoundedMinutes,
		BillableUnits:      computed.BillableUnits,
		GrossAmount:        computed.GrossAmount,
		BillableAmount:     computed.BillableAmount,
		DriverPayMinutes:   computed.DriverPayMinutes,
		DriverPayAmount:    decimal.NewNullDecimal(computed.DriverPayAmount),
		NetMargin:          decimal.NewNullDecimal(computed.NetMargin),
		CapApplied:         computed.CapApplied,
		ConvertedToLayover: computed.ConvertedToLayover,
		Currency:           snapshot.Currency,
		Status:             computed.Status,
		NotificationStatus: computed.NotificationStatus,
		SuppressedByGate:   computed.SuppressedByGate,
		RequiresApproval:   computed.RequiresApproval,
	}

	if p.existing != nil {
		occurrence.ID = p.existing.ID
		occurrence.NoticeSentAt = p.existing.NoticeSentAt
		occurrence.ApprovedByID = p.existing.ApprovedByID
		occurrence.ApprovedAt = p.existing.ApprovedAt
		occurrence.EvidenceHead = p.existing.EvidenceHead
		occurrence.AdditionalChargeID = p.existing.AdditionalChargeID
		occurrence.CollectabilityScore = p.existing.CollectabilityScore

		if p.existing.Status == detention.OccurrenceStatusApproved &&
			computed.Status == detention.OccurrenceStatusPending &&
			!computed.RequiresApproval {
			occurrence.Status = detention.OccurrenceStatusApproved
		}
	}

	return occurrence
}

func stopIsBillable(stop *shipment.Stop) bool {
	if stop == nil || stop.Status == shipmentvocab.StopStatusCanceled {
		return false
	}

	if stop.ActualArrival == nil || *stop.ActualArrival <= 0 {
		return false
	}

	if stop.CountDetentionOverride != nil && !*stop.CountDetentionOverride {
		return false
	}

	return true
}

func appointmentStart(stop *shipment.Stop) *int64 {
	if stop.ScheduledWindowStart <= 0 {
		return nil
	}
	start := stop.ScheduledWindowStart
	return &start
}

func shipmentCommodityIDs(entity *shipment.Shipment) []pulid.ID {
	ids := make([]pulid.ID, 0, len(entity.Commodities))
	for _, commodity := range entity.Commodities {
		if commodity == nil || commodity.CommodityID.IsNil() {
			continue
		}
		ids = append(ids, commodity.CommodityID)
	}
	return ids
}
