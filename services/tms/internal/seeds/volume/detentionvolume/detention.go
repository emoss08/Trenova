package detentionvolume

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy/detentionpolicydb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment/payassignmentdb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	minute = volumegen.Minute
	hour   = volumegen.Hour
	day    = volumegen.Day

	generousCustomerShare = 5
)

type fate int

const (
	fateWait fate = iota
	fateApprove
	fateBill
	fateWaive
	fateDispute
)

var (
	waiverReasons = []detention.WaiverReason{
		detention.WaiverReasonCustomerGoodwill,
		detention.WaiverReasonDataCorrection,
		detention.WaiverReasonFacilityClosure,
		detention.WaiverReasonWeather,
		detention.WaiverReasonCarrierFault,
		detention.WaiverReasonEquipmentIssue,
		detention.WaiverReasonOther,
	}
	waiverWeights = []int{40, 18, 12, 12, 10, 5, 3}
	waiverNotes   = map[detention.WaiverReason][]string{
		detention.WaiverReasonCustomerGoodwill: {
			"Waived to keep the lane; the shipper asked us to absorb it this once.",
			"Account manager waived it ahead of the quarterly review.",
			"First detention on this account; waived as a courtesy.",
		},
		detention.WaiverReasonDataCorrection: {
			"Departure time corrected from the signed POD; the dock released the truck inside free time.",
			"Geofence exit fired late; the driver's log shows an earlier departure.",
		},
		detention.WaiverReasonFacilityClosure: {
			"Receiver closed the dock early for an inventory count.",
			"Facility shut for a power outage; nobody could unload.",
		},
		detention.WaiverReasonWeather: {
			"Ice storm closed the yard; the delay was nobody's fault.",
			"Thunderstorm stopped all dock work for two hours.",
		},
		detention.WaiverReasonCarrierFault: {
			"Driver arrived without the seal number the receiver needs to check in.",
			"Our paperwork was incomplete; the dock held the truck until it was fixed.",
		},
		detention.WaiverReasonEquipmentIssue: {
			"Reefer unit faulted at the door; the load sat while it was reset.",
		},
		detention.WaiverReasonOther: {
			"Settled as part of a broader rate conversation with the shipper.",
		},
	}
	disputeNotes = []string{
		"Shipper says the truck was outside its appointment window.",
		"Receiver's gate log shows a later arrival than our geofence.",
		"Customer disputes the free time; says the contract grants three hours here.",
		"Consignee says the trailer was dropped, not live unloaded.",
	}
)

type Deriver struct {
	g        *volumegen.Generator
	now      int64
	policies []*detentionpolicy.DetentionPolicy
	payRates map[pulid.ID]decimal.NullDecimal
	counts   map[detention.OccurrenceStatus]int
	total    int
}

type Derived struct {
	Occurrences []*detention.DetentionOccurrence
	Stamps      []volumegen.Stamp
	Billed      decimal.Decimal
}

type Shipment struct {
	Entity       *shipment.Shipment
	CommodityIDs []pulid.ID
	Delivered    bool
	CustomerCode string
}

func Load(ctx context.Context, g *volumegen.Generator) (*Deriver, error) {
	p := detentionpolicydb.DetentionPolicies
	tiers := p.Tiers()
	policies, err := p.Where(p.Status.Eq(detentionpolicy.PolicyStatusActive)).
		Load(tiers.Order(tiers.FromMinute.Asc(), tiers.SortOrder.Asc())).
		Order(p.Priority.Desc(), p.SpecificityScore.Desc(), p.CreatedAt.Asc()).
		All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load detention policies: %w", err)
	}

	rates, err := loadPayRates(ctx, g)
	if err != nil {
		return nil, err
	}

	return &Deriver{
		g:        g,
		now:      volumegen.Unix(g.Now),
		policies: policies,
		payRates: rates,
		counts:   make(map[detention.OccurrenceStatus]int),
	}, nil
}

func loadPayRates(
	ctx context.Context,
	g *volumegen.Generator,
) (map[pulid.ID]decimal.NullDecimal, error) {
	type workerRate struct {
		WorkerID pulid.ID
		Rate     decimal.Decimal
	}

	now := volumegen.Unix(g.Now)
	wpa := payassignmentdb.WorkerPayAssignments
	comp := driverpaydb.PayProfileComponents
	rows, err := keel.Into[workerRate](ctx, g.DB, wpa.Query().
		Join(
			comp,
			comp.PayProfileID.EqCol(wpa.PayProfileID),
			comp.OrganizationID.EqCol(wpa.OrganizationID),
		).
		Pick(wpa.WorkerID, comp.Rate).
		Where(
			wpa.EffectiveFrom.Lte(now),
			keel.Or(wpa.EffectiveTo.IsNull(), wpa.EffectiveTo.Gte(now)),
			comp.Kind.Eq(driverpay.ComponentKindDetention),
			comp.IsActive.IsTrue(),
		))
	if err != nil {
		return nil, fmt.Errorf("load driver detention pay rates: %w", err)
	}

	rates := make(map[pulid.ID]decimal.NullDecimal, len(rows))
	for _, row := range rows {
		rates[row.WorkerID] = decimal.NewNullDecimal(row.Rate)
	}

	return rates, nil
}

func (d *Deriver) Ready() bool {
	return len(d.policies) > 0
}

func (d *Deriver) Derive(s *Shipment) *Derived {
	out := &Derived{Billed: decimal.Zero}
	if !d.Ready() || s.Entity.Status == shipmentvocab.StatusCanceled {
		return out
	}

	dayAccrued := map[string]decimal.Decimal{}
	shipmentAccrued := decimal.Zero
	for _, move := range s.Entity.Moves {
		payRate := d.payRate(move)
		for _, stop := range move.Stops {
			occ := d.compute(s, move, stop, payRate, dayAccrued, shipmentAccrued)
			if occ == nil {
				continue
			}
			shipmentAccrued = shipmentAccrued.Add(occ.BillableAmount)
			decided := d.settle(occ, s)
			if occ.Status == detention.OccurrenceStatusBilled {
				out.Billed = out.Billed.Add(occ.BillableAmount)
			}
			d.counts[occ.Status]++
			d.total++
			out.Occurrences = append(out.Occurrences, occ)
			out.Stamps = append(out.Stamps, volumegen.Stamp{
				Id:      occ.ID,
				Created: occ.ClockStartAt,
				Updated: decided,
			})
		}
	}

	return out
}

func (d *Derived) Bind(chargeID pulid.ID) {
	for _, occ := range d.Occurrences {
		if occ.Status != detention.OccurrenceStatusBilled {
			continue
		}
		if chargeID.IsNil() {
			occ.Status = detention.OccurrenceStatusApproved
			continue
		}
		occ.AdditionalChargeID = volumegen.Ptr(chargeID)
	}
}

func (d *Deriver) payRate(move *shipment.ShipmentMove) decimal.NullDecimal {
	if move.Assignment == nil || move.Assignment.PrimaryWorkerID == nil {
		return decimal.NullDecimal{}
	}
	return d.payRates[*move.Assignment.PrimaryWorkerID]
}

func (d *Deriver) compute(
	s *Shipment,
	move *shipment.ShipmentMove,
	stop *shipment.Stop,
	payRate decimal.NullDecimal,
	dayAccrued map[string]decimal.Decimal,
	shipmentAccrued decimal.Decimal,
) *detention.DetentionOccurrence {
	if stop.ActualArrival == nil || stop.Status == shipmentvocab.StopStatusCanceled {
		return nil
	}

	resolution := detentionengine.ResolvePolicy(d.policies, &detentionpolicy.MatchInput{
		CustomerID:     s.Entity.CustomerID,
		LocationID:     stop.LocationID,
		ShipmentTypeID: s.Entity.ShipmentTypeID,
		ServiceTypeID:  s.Entity.ServiceTypeID,
		CommodityIDs:   s.CommodityIDs,
		StopType:       stop.Type,
		ScheduleType:   stop.ScheduleType,
		Timestamp:      *stop.ActualArrival,
	})
	if !resolution.Found() {
		return nil
	}
	policy := resolution.Policy
	accessorial := d.g.Refs.AccessorialByID(policy.AccessorialChargeID)
	if accessorial == nil {
		return nil
	}

	snapshot := detentionpolicy.NewPolicySnapshot(&detentionpolicy.SnapshotInput{
		Policy:          policy,
		StopType:        string(stop.Type),
		FreeMinutes:     policy.FreeMinutesForStopType(stop.Type),
		AccessorialCode: accessorial.Code,
		FlatRate:        accessorial.Amount,
		FlatRateUnit:    detentionengine.TierUnitFromAccessorial(accessorial.RateUnit),
		CapturedAt:      *stop.ActualArrival,
	})
	input := detentionengine.ComputeInput{
		Snapshot:         snapshot,
		StopType:         stop.Type,
		ScheduleType:     stop.ScheduleType,
		ArrivedAt:        stop.ActualArrival,
		DepartedAt:       stop.ActualDeparture,
		AppointmentStart: volumegen.Ptr(stop.ScheduledWindowStart),
		AppointmentEnd:   stop.ScheduledWindowEnd,
		Now:              d.now,
		Location:         d.g.Refs.Timezone,
		DriverPayRate:    payRate,
		DayAccrued:       dayAccrued,
		ShipmentAccrued:  shipmentAccrued,
	}

	computed := detentionengine.Compute(&input)
	if !computed.Applicable {
		return nil
	}
	if sentAt, sent := d.notice(computed); sent {
		input.NoticeSentAt = volumegen.Ptr(sentAt)
		computed = detentionengine.Compute(&input)
	}
	for at, amount := range computed.DayAllocation {
		dayAccrued[at] = dayAccrued[at].Add(amount)
	}

	policyID := policy.ID
	occ := &detention.DetentionOccurrence{
		ID:                 volumegen.IDAt("dto_", computed.ClockStartAt),
		BusinessUnitID:     d.g.Refs.BuID,
		OrganizationID:     d.g.Refs.OrgID,
		ShipmentID:         s.Entity.ID,
		ShipmentMoveID:     move.ID,
		StopID:             stop.ID,
		CustomerID:         s.Entity.CustomerID,
		LocationID:         stop.LocationID,
		DetentionPolicyID:  &policyID,
		PolicySnapshot:     &snapshot,
		CalculationTrace:   computed.Trace,
		StopType:           stop.Type,
		ScheduleType:       stop.ScheduleType,
		Handling:           stop.Handling,
		AppointmentStart:   volumegen.Ptr(stop.ScheduledWindowStart),
		AppointmentEnd:     stop.ScheduledWindowEnd,
		ArrivedAt:          stop.ActualArrival,
		DepartedAt:         stop.ActualDeparture,
		ClockStartAt:       computed.ClockStartAt,
		ClockStopAt:        computed.ClockStopAt,
		FreeTimeExpiresAt:  computed.FreeTimeExpiresAt,
		NoticeDueAt:        computed.NoticeDueAt,
		NoticeDeadlineAt:   computed.NoticeDeadlineAt,
		NoticeSentAt:       input.NoticeSentAt,
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
		Version:            1,
	}
	if err := occ.BeforeCreate(context.Background()); err != nil {
		return nil
	}

	return occ
}

func (d *Deriver) notice(computed detentionengine.ComputeResult) (int64, bool) {
	if computed.BillableMinutes <= 0 || computed.FreeTimeExpiresAt <= 0 {
		return 0, false
	}
	f := d.g.Fake
	expiry := computed.FreeTimeExpiresAt
	var sentAt int64
	switch f.Weighted([]int{64, 14, 22}) {
	case 0:
		sentAt = expiry - f.Between64(5*minute, 35*minute)
	case 1:
		sentAt = expiry + f.Between64(20*minute, 3*hour)
	default:
		return 0, false
	}
	if sentAt > d.now {
		return 0, false
	}

	return sentAt, true
}

func (d *Deriver) settle(occ *detention.DetentionOccurrence, s *Shipment) int64 {
	stoppedAt := occ.ClockStartAt
	if occ.ClockStopAt != nil {
		stoppedAt = *occ.ClockStopAt
	}
	if occ.Status != detention.OccurrenceStatusPending ||
		occ.BillableAmount.LessThanOrEqual(decimal.Zero) {
		return stoppedAt
	}

	f := d.g.Fake
	at := min(stoppedAt+f.Between64(2*hour, 6*day), d.now-minute)
	if at <= stoppedAt {
		return stoppedAt
	}
	user := d.g.PickUser(d.g.Refs.UserIDs)

	var err error
	switch d.fateOf(at-stoppedAt, d.now-stoppedAt, s) {
	case fateApprove:
		err = occ.Approve(user, at)
	case fateBill:
		if err = occ.Approve(user, at); err == nil {
			occ.Status = detention.OccurrenceStatusBilled
		}
	case fateWaive:
		reason := waiverReasons[f.Weighted(waiverWeights)]
		err = occ.Waive(reason, volumegen.Pick(f, waiverNotes[reason]), user, at)
	case fateDispute:
		err = occ.Dispute(volumegen.Pick(f, disputeNotes), at)
	case fateWait:
		return stoppedAt
	}
	if err != nil {
		return stoppedAt
	}

	return at
}

func (d *Deriver) fateOf(decidedAfter, age int64, s *Shipment) fate {
	f := d.g.Fake
	if !s.Delivered || age < 2*day {
		if f.Chance(0.3) {
			return fateApprove
		}
		return fateWait
	}

	waive := 16
	if generous(s.CustomerCode) {
		waive = 42
	}
	weights := []int{4, 8, 64, waive, 8}
	if age < 14*day {
		weights = []int{18, 16, 44, waive, 8}
	}
	if decidedAfter < hour {
		weights[fateBill] /= 2
	}

	return fate(f.Weighted(weights))
}

func generous(customerCode string) bool {
	h := fnv.New32a()
	_, _ = h.Write([]byte(customerCode))
	return h.Sum32()%generousCustomerShare == 0
}

func (d *Deriver) Report() {
	d.g.Report.Add("Detention occurrences", d.total)
	for _, status := range []detention.OccurrenceStatus{
		detention.OccurrenceStatusAccruing,
		detention.OccurrenceStatusPending,
		detention.OccurrenceStatusApproved,
		detention.OccurrenceStatusBilled,
		detention.OccurrenceStatusWaived,
		detention.OccurrenceStatusDisputed,
		detention.OccurrenceStatusNotBillable,
	} {
		d.g.Opts.Logf("  • detention %s: %d", status, d.counts[status])
	}
}
