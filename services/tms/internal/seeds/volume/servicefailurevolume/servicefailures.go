package servicefailurevolume

import (
	"context"
	"fmt"
	"math"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
)

const (
	sfChunk           = 5_000
	sfManualRate      = 0.06
	sfManualLateAfter = 15 * volumegen.Minute
	sfLabel           = "Service failures"
)

var (
	sfVoidReasons = []string{
		"Arrival time corrected from the driver's log; the stop was on time.",
		"Customer moved the appointment after dispatch; not a carrier failure.",
		"Duplicate of a failure already raised on this shipment.",
		"Consignee confirmed the late arrival was at their request.",
	}
	sfResolutionNotes = []string{
		"Customer notified with the root cause and the recovery plan.",
		"Driver coached on departure planning for this lane.",
		"Facility detention documented; appointment times adjusted for this location.",
		"Weather delay confirmed with the customer; no chargeback applies.",
		"Carrier scorecard updated and the lane re-tendered to a better performer.",
		"Credit issued to the customer against the next invoice.",
	}
	sfManualNotes = []string{
		"Driver missed the appointment slot; facility worked the load in after the next opening.",
		"Receiver reports the truck was outside the appointment window.",
		"Logged at the customer's request from their carrier scorecard.",
		"Appointment missed while waiting on the prior stop.",
	}
)

type sfPhase struct {
	*volumegen.Generator
}

type sfBuilder struct {
	g       *volumegen.Generator
	now     int64
	grace   int
	pickup  []*servicefailure.ReasonCode
	deliver []*servicefailure.ReasonCode
	rows    []*servicefailure.ServiceFailure
	stamps  []volumegen.Stamp
	counts  map[servicefailure.Status]int
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&sfPhase{Generator: g}).seed(ctx)
}

func (g *sfPhase) seed(ctx context.Context) error {
	control, err := g.DispatchControl(ctx)
	if err != nil {
		return err
	}
	rc := servicefailuredb.ReasonCodes
	codes, err := rc.Where(rc.Active.IsTrue()).
		Order(rc.SortOrder.Asc(), rc.ID.Asc()).
		All(ctx, g.DB)
	if err != nil {
		return fmt.Errorf("load service failure reason codes: %w", err)
	}

	b := &sfBuilder{
		g:      g.Generator,
		now:    volumegen.Unix(g.Now),
		grace:  volumegen.ServiceFailureGraceMinutes(control),
		counts: make(map[servicefailure.Status]int),
	}
	for _, code := range codes {
		if code.AppliesTo.AllowsStopType(shipmentvocab.StopTypePickup) {
			b.pickup = append(b.pickup, code)
		}
		if code.AppliesTo.AllowsStopType(shipmentvocab.StopTypeDelivery) {
			b.deliver = append(b.deliver, code)
		}
	}

	for _, rec := range g.Shipments {
		if rec.Status == shipmentvocab.StatusCanceled {
			continue
		}
		for _, move := range rec.Moves {
			for _, stop := range move.Stops {
				b.consider(rec, move, stop)
			}
		}
	}

	total := len(b.rows)
	for begin := 0; begin < total; begin += sfChunk {
		end := min(begin+sfChunk, total)
		rows, stamps := b.rows[begin:end], b.stamps[begin:end]
		if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
			h := servicefailuredb.ServiceFailures.Handle
			if err := volumegen.InsertRows(ctx, tx, h, rows); err != nil {
				return err
			}
			return volumegen.Backdate(ctx, tx, h, stamps)
		}); err != nil {
			return err
		}
		g.Progress("service failures", end, total)
	}

	g.Report.Add(sfLabel, total)
	for _, status := range []servicefailure.Status{
		servicefailure.StatusOpen, servicefailure.StatusReviewed,
		servicefailure.StatusResolved, servicefailure.StatusVoided,
	} {
		g.Opts.Logf("  • service failures %s: %d", status, b.counts[status])
	}

	return nil
}

func (b *sfBuilder) consider(
	rec *volumegen.ShipmentRecord,
	move *volumegen.MoveRecord,
	stop *volumegen.StopRecord,
) {
	if !stop.Arrived() || stop.Status == shipmentvocab.StopStatusCanceled || stop.WindowEnd <= 0 {
		return
	}

	graceSeconds := int64(b.grace) * volumegen.Minute
	if stop.ActualArrival > stop.WindowEnd+graceSeconds {
		late := lateMinutes(stop.ActualArrival - stop.WindowEnd - graceSeconds)
		b.add(rec, move, stop, &sfDraft{
			kind:        servicefailure.TypeLatePickup,
			source:      servicefailure.SourceDetected,
			cutoff:      stop.WindowEnd,
			grace:       b.grace,
			lateMinutes: late,
		})
		return
	}

	if stop.Appointment && stop.ActualArrival > stop.WindowStart+sfManualLateAfter &&
		b.g.Fake.Chance(sfManualRate) {
		b.add(rec, move, stop, &sfDraft{
			kind:        servicefailure.TypeAppointmentMissed,
			source:      servicefailure.SourceManual,
			cutoff:      stop.WindowStart,
			grace:       b.grace,
			lateMinutes: lateMinutes(stop.ActualArrival - stop.WindowStart),
		})
	}
}

type sfDraft struct {
	kind        servicefailure.Type
	source      servicefailure.Source
	cutoff      int64
	grace       int
	lateMinutes int64
}

func (b *sfBuilder) add(
	rec *volumegen.ShipmentRecord,
	move *volumegen.MoveRecord,
	stop *volumegen.StopRecord,
	d *sfDraft,
) {
	f := b.g.Fake
	kind := d.kind
	if kind == servicefailure.TypeLatePickup && !stop.IsOrigin() {
		kind = servicefailure.TypeLateDelivery
	}

	detected := min(stop.ActualArrival+f.Between64(volumegen.Minute, 20*volumegen.Minute), b.now)
	if d.source == servicefailure.SourceManual {
		detected = min(
			max(stop.ActualDeparture, stop.ActualArrival)+
				f.Between64(30*volumegen.Minute, 2*volumegen.Day),
			b.now,
		)
	}

	sf := &servicefailure.ServiceFailure{
		ID:                 volumegen.IDAt("sf_", detected),
		BusinessUnitID:     b.g.Refs.BuID,
		OrganizationID:     b.g.Refs.OrgID,
		ShipmentID:         rec.ID,
		ShipmentMoveID:     move.ID,
		StopID:             stop.ID,
		Type:               kind,
		Source:             d.source,
		Status:             servicefailure.StatusOpen,
		StopType:           stop.Type,
		ScheduledCutoff:    d.cutoff,
		ActualArrival:      stop.ActualArrival,
		GracePeriodMinutes: d.grace,
		LateMinutes:        max(d.lateMinutes, 1),
		DetectedAt:         detected,
		Version:            1,
	}
	if d.source == servicefailure.SourceManual {
		sf.CreatedByID = volumegen.Ptr(b.g.PickUser(b.g.Refs.UserIDs))
		sf.Notes = volumegen.Pick(f, sfManualNotes)
	} else {
		sf.Notes = detectedNote(sf)
	}

	updated := b.lifecycle(sf, stop)
	sf.Normalize()
	b.counts[sf.Status]++
	b.rows = append(b.rows, sf)
	b.stamps = append(b.stamps, volumegen.Stamp{Id: sf.ID, Created: detected, Updated: updated})
}

func (b *sfBuilder) lifecycle(sf *servicefailure.ServiceFailure, stop *volumegen.StopRecord) int64 {
	f := b.g.Fake
	age := b.now - sf.DetectedAt
	var weights []int
	switch {
	case age < 3*volumegen.Day:
		weights = []int{78, 16, 5, 1}
	case age < 14*volumegen.Day:
		weights = []int{30, 32, 33, 5}
	case age < 45*volumegen.Day:
		weights = []int{9, 21, 62, 8}
	default:
		weights = []int{2, 8, 80, 10}
	}

	updated := sf.DetectedAt
	switch f.Weighted(weights) {
	case 1:
		if at, ok := b.after(sf.DetectedAt, 2*volumegen.Hour, 3*volumegen.Day); ok {
			b.review(sf, stop, at)
			updated = at
		}
	case 2:
		reviewed := sf.DetectedAt
		if at, ok := b.after(sf.DetectedAt, volumegen.Hour, 2*volumegen.Day); ok && f.Chance(0.7) {
			b.review(sf, stop, at)
			reviewed = at
		}
		if at, ok := b.after(reviewed, 4*volumegen.Hour, 10*volumegen.Day); ok {
			sf.Status = servicefailure.StatusResolved
			sf.ResolvedAt = volumegen.Ptr(at)
			sf.ResolvedByID = volumegen.Ptr(b.g.PickUser(b.g.Refs.UserIDs))
			sf.InternalNotes = volumegen.Pick(f, sfResolutionNotes)
			b.reason(sf, stop)
			updated = at
		} else if sf.ReviewedAt != nil {
			updated = reviewed
		}
	case 3:
		if at, ok := b.after(sf.DetectedAt, 30*volumegen.Minute, 4*volumegen.Day); ok {
			sf.Status = servicefailure.StatusVoided
			sf.VoidedAt = volumegen.Ptr(at)
			sf.VoidedByID = volumegen.Ptr(b.g.PickUser(b.g.Refs.UserIDs))
			sf.VoidReason = volumegen.Pick(f, sfVoidReasons)
			updated = at
		}
	default:
		if f.Chance(0.2) {
			b.reason(sf, stop)
		}
	}

	return updated
}

func (b *sfBuilder) review(
	sf *servicefailure.ServiceFailure,
	stop *volumegen.StopRecord,
	at int64,
) {
	sf.Status = servicefailure.StatusReviewed
	sf.ReviewedAt = volumegen.Ptr(at)
	sf.ReviewedByID = volumegen.Ptr(b.g.PickUser(b.g.Refs.UserIDs))
	b.reason(sf, stop)
}

func (b *sfBuilder) reason(sf *servicefailure.ServiceFailure, stop *volumegen.StopRecord) {
	codes := b.deliver
	if stop.IsOrigin() {
		codes = b.pickup
	}
	if len(codes) == 0 || sf.ReasonCodeID != nil {
		return
	}
	weights := make([]int, len(codes))
	for i, code := range codes {
		weights[i] = max(int(200-code.SortOrder), 10)
	}
	code := codes[b.g.Fake.Weighted(weights)]
	sf.ReasonCodeID = volumegen.Ptr(code.ID)
}

func (b *sfBuilder) after(from, low, high int64) (int64, bool) {
	at := from + b.g.Fake.Between64(low, high)
	if at > b.now {
		return 0, false
	}
	return at, true
}

func lateMinutes(seconds int64) int64 {
	return int64(math.Ceil(float64(seconds) / float64(volumegen.Minute)))
}

func detectedNote(sf *servicefailure.ServiceFailure) string {
	label := "late delivery"
	if sf.Type == servicefailure.TypeLatePickup {
		label = "late pickup"
	}
	return fmt.Sprintf(
		"Detected %s service failure %d minute(s) after grace.",
		label,
		sf.LateMinutes,
	)
}
