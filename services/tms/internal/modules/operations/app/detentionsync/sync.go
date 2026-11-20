package detentionsync

import (
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
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

// stopPlan is what a billable stop needs before it is computed. A stop whose
// stored occurrence is frozen has no policy: it is carried as it stands.
type stopPlan struct {
	move     *shipment.ShipmentMove
	stop     *shipment.Stop
	existing *detention.DetentionOccurrence
	policy   *detentionpolicy.DetentionPolicy
	payRate  decimal.NullDecimal
}

type computeStopParams struct {
	shipment        *shipment.Shipment
	move            *shipment.ShipmentMove
	stop            *shipment.Stop
	existing        *detention.DetentionOccurrence
	policy          *detentionpolicy.DetentionPolicy
	accessorial     *accessorialcharge.AccessorialCharge
	tenantInfo      pagination.TenantInfo
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
	p *computeStopParams,
	snapshot *detentionpolicy.PolicySnapshot,
	computed *detentionengine.ComputeResult,
) *detention.DetentionOccurrence {
	policyID := p.policy.ID

	occurrence := &detention.DetentionOccurrence{
		OrganizationID:     p.tenantInfo.OrgID,
		BusinessUnitID:     p.tenantInfo.BuID,
		ShipmentID:         p.shipment.ID,
		ShipmentMoveID:     p.move.ID,
		StopID:             p.stop.ID,
		CustomerID:         p.shipment.CustomerID,
		LocationID:         p.stop.LocationID,
		DetentionPolicyID:  &policyID,
		PolicySnapshot:     snapshot,
		CalculationTrace:   computed.Trace,
		StopType:           p.stop.Type,
		ScheduleType:       p.stop.ScheduleType,
		Handling:           p.stop.Handling,
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
		occurrence.CreatedAt = p.existing.CreatedAt
		occurrence.Version = p.existing.Version
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

func billableStopIDs(entity *shipment.Shipment) []pulid.ID {
	ids := make([]pulid.ID, 0, 4)
	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stopIsBillable(stop) {
				ids = append(ids, stop.ID)
			}
		}
	}

	return ids
}

// movePrimaryWorker is the driver paid for the move, or nil when it has none.
func movePrimaryWorker(move *shipment.ShipmentMove) pulid.ID {
	if move == nil || move.Assignment == nil || move.Assignment.PrimaryWorkerID == nil {
		return pulid.Nil
	}

	return *move.Assignment.PrimaryWorkerID
}

// billableMoveWorkers lists, once each, the primary drivers of the moves that
// have a billable stop: the drivers whose detention pay the sync compares.
func billableMoveWorkers(entity *shipment.Shipment) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(entity.Moves))
	ids := make([]pulid.ID, 0, len(entity.Moves))
	for _, move := range entity.Moves {
		workerID := movePrimaryWorker(move)
		if workerID.IsNil() || !slices.ContainsFunc(move.Stops, stopIsBillable) {
			continue
		}
		if _, ok := seen[workerID]; ok {
			continue
		}
		seen[workerID] = struct{}{}
		ids = append(ids, workerID)
	}

	return ids
}

// planAccessorialIDs lists, once each, the accessorial charges the plan's
// resolved policies price detention with.
func planAccessorialIDs(plan []*stopPlan) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(plan))
	ids := make([]pulid.ID, 0, len(plan))
	for _, sp := range plan {
		if sp.policy == nil {
			continue
		}
		id := sp.policy.AccessorialChargeID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids
}

type resolveStopParams struct {
	shipment        *shipment.Shipment
	stop            *shipment.Stop
	commodityIDs    []pulid.ID
	candidates      []*detentionpolicy.DetentionPolicy
	defaultPolicyID pulid.ID
}

// resolveStopPolicy picks the policy that governs a stop from the shipment's
// candidates, or nil when none does.
func resolveStopPolicy(p *resolveStopParams) *detentionpolicy.DetentionPolicy {
	resolution := detentionengine.ResolvePolicy(
		candidatesForStop(p.candidates, p.stop.LocationID),
		&detentionpolicy.MatchInput{
			CustomerID:     p.shipment.CustomerID,
			LocationID:     p.stop.LocationID,
			ShipmentTypeID: p.shipment.ShipmentTypeID,
			ServiceTypeID:  p.shipment.ServiceTypeID,
			CommodityIDs:   p.commodityIDs,
			StopType:       p.stop.Type,
			ScheduleType:   p.stop.ScheduleType,
			Timestamp:      *p.stop.ActualArrival,
		},
		p.defaultPolicyID,
	)
	if !resolution.Found() {
		return nil
	}

	return resolution.Policy
}

func resolutionCandidatesRequest(
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) (*detentionports.GetResolutionCandidatesRequest, bool) {
	req := &detentionports.GetResolutionCandidatesRequest{
		TenantInfo: tenantInfo,
		CustomerID: entity.CustomerID,
	}
	seen := make(map[pulid.ID]struct{})

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		for _, stop := range move.Stops {
			if !stopIsBillable(stop) {
				continue
			}

			arrival := *stop.ActualArrival
			if len(seen) == 0 || arrival < req.ArrivedFrom {
				req.ArrivedFrom = arrival
			}
			if len(seen) == 0 || arrival > req.ArrivedTo {
				req.ArrivedTo = arrival
			}

			if _, ok := seen[stop.LocationID]; !ok {
				seen[stop.LocationID] = struct{}{}
				req.LocationIDs = append(req.LocationIDs, stop.LocationID)
			}
		}
	}

	return req, len(req.LocationIDs) > 0
}

// candidatesForStop keeps the shipment's candidates that a stop at this
// facility could see, in the order the repository returned them. Policies
// scoped to the shipment's other facilities drop out here, as they would from
// a read for this stop alone.
func candidatesForStop(
	candidates []*detentionpolicy.DetentionPolicy,
	locationID pulid.ID,
) []*detentionpolicy.DetentionPolicy {
	out := make([]*detentionpolicy.DetentionPolicy, 0, len(candidates))
	for _, policy := range candidates {
		if policy == nil {
			continue
		}
		if policy.LocationID != nil && !policy.LocationID.IsNil() &&
			*policy.LocationID != locationID {
			continue
		}
		out = append(out, policy)
	}

	return out
}
