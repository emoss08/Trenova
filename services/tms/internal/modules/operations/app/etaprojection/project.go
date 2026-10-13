package etaprojection

import (
	"math"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/hosprojection"
	"github.com/emoss08/trenova/shared/pulid"
)

type cursor struct {
	readyAt   int64
	miles     float64
	driveMs   int64
	rest      hosprojection.Rest
	routed    bool
	planned   bool
	chained   bool
	clamped   bool
	hosWorker pulid.ID
	schedule  *hosprojection.Schedule
	hosNote   hosNote
}

type hosNote string

const (
	hosNoteNone     = hosNote("")
	hosNoteApplied  = hosNote("applied")
	hosNoteMissing  = hosNote("missing")
	hosNoteStale    = hosNote("stale")
	hosNoteExempt   = hosNote("exempt")
	hosNoteCarrier  = hosNote("carrier")
	hosNoteHandover = hosNote("handover")
)

func (p *Plan) Project(estimates []routeestimateports.LegEstimate) []*stopeta.StopETA {
	if len(p.stops) == 0 {
		return nil
	}

	sp := p.in.Shipment
	c := &cursor{readyAt: p.origin.at, chained: p.origin.basis != stopeta.BasisNone}
	p.startHOS(c, p.stops[0].move)

	out := make([]*stopeta.StopETA, 0, len(p.stops))
	for i := range p.stops {
		item := &p.stops[i]
		row := p.baseRow(sp, item)
		p.switchDriver(c, item.move)

		arrival, method := p.arrive(c, i, item, estimates)
		row.DriveMethod = method
		if arrival > 0 {
			departure := p.depart(c, item, arrival)
			row.EstimatedArrival = &arrival
			row.EstimatedDeparture = &departure
			c.readyAt = departure
		}

		row.MilesRemaining = math.Round(c.miles*10) / 10
		row.DriveMinutes = c.driveMs / msPerMinute
		row.RestMinutes = c.rest.RestMs / msPerMinute
		row.InputStale = p.origin.stale || c.clamped
		p.judge(row, item.stop)
		delayed := p.markDelayed(row)
		row.Reason = describe(&reasonInput{
			row:     row,
			origin:  p.origin,
			cursor:  c,
			now:     p.in.Now,
			stop:    item.stop,
			delayed: delayed,
		})
		out = append(out, row)
	}

	return out
}

func (p *Plan) baseRow(sp *shipment.Shipment, item *plannedStop) *stopeta.StopETA {
	row := &stopeta.StopETA{
		OrganizationID:   sp.OrganizationID,
		BusinessUnitID:   sp.BusinessUnitID,
		ShipmentID:       sp.ID,
		ShipmentMoveID:   item.move.ID,
		StopID:           item.stop.ID,
		MoveSequence:     item.move.Sequence,
		StopSequence:     item.stop.Sequence,
		Arrived:          item.stop.ActualArrival != nil,
		Verdict:          stopeta.VerdictUnknown,
		Basis:            p.origin.basis,
		DriveMethod:      stopeta.DriveMethodNone,
		ThresholdMinutes: p.in.Settings.ThresholdMinutes,
		ComputedAt:       p.in.Now,
	}
	if cutoff := item.stop.EffectiveScheduledCutoff(); cutoff > 0 {
		row.WindowEnd = &cutoff
	}
	if p.origin.inputAt > 0 {
		inputAt := p.origin.inputAt
		row.InputAt = &inputAt
	}

	return row
}

func (p *Plan) arrive(
	c *cursor,
	index int,
	item *plannedStop,
	estimates []routeestimateports.LegEstimate,
) (int64, stopeta.DriveMethod) {
	if !c.chained {
		return 0, stopeta.DriveMethodNone
	}

	if index == 0 && p.origin.atStop {
		return *item.stop.ActualArrival, stopeta.DriveMethodNone
	}
	if index == 0 && p.origin.basis == stopeta.BasisCarrierReported {
		return p.origin.reportedArrival, stopeta.DriveMethodReported
	}
	if item.leg < 0 || item.leg >= len(estimates) {
		c.chained = false
		return 0, stopeta.DriveMethodNone
	}

	estimate := estimates[item.leg]
	driveMs := estimate.DriveSeconds * msPerSecond
	elapsedMs := driveMs
	if c.schedule != nil {
		var rest hosprojection.Rest
		elapsedMs, rest = c.schedule.Drive(driveMs)
		c.rest.Add(rest)
	}

	c.miles += estimate.Miles
	c.driveMs += driveMs
	method := stopeta.DriveMethodPlanning
	if estimate.Routed {
		c.routed = true
		method = stopeta.DriveMethodRouted
	} else {
		c.planned = true
	}

	arrival := c.readyAt + elapsedMs/msPerSecond
	if arrival < p.in.Now {
		c.clamped = true
		arrival = p.in.Now
	}

	return arrival, method
}

func (p *Plan) depart(c *cursor, item *plannedStop, arrival int64) int64 {
	dwellSeconds := p.in.Settings.DwellMinutes * secondsPerMinute
	start := max(arrival, item.stop.ScheduledWindowStart)
	departure := start + dwellSeconds
	waitSeconds := start - arrival
	onDutySeconds := dwellSeconds

	if item.stop.ActualArrival != nil {
		if departure < p.in.Now {
			departure = p.in.Now
		}
		waitSeconds = max(start-p.in.Now, 0)
		onDutySeconds = max(departure-max(start, p.in.Now), 0)
	}

	if c.schedule != nil {
		c.schedule.OffDuty(waitSeconds * msPerSecond)
		c.schedule.OnDuty(onDutySeconds * msPerSecond)
	}

	return departure
}

func (p *Plan) judge(row *stopeta.StopETA, stop *shipment.Stop) {
	cutoff := stop.EffectiveScheduledCutoff()
	now := p.in.Now

	if stop.ActualArrival != nil {
		if cutoff <= 0 {
			row.Verdict = stopeta.VerdictOnTime
			return
		}
		slack := floorMinutes(cutoff - *stop.ActualArrival)
		row.SlackMinutes = &slack
		row.Verdict = stopeta.VerdictOnTime
		if *stop.ActualArrival > cutoff {
			row.Verdict = stopeta.VerdictLate
		}
		return
	}

	if cutoff > 0 && now > cutoff {
		reference := now
		if row.EstimatedArrival != nil {
			reference = *row.EstimatedArrival
		}
		slack := floorMinutes(cutoff - reference)
		row.SlackMinutes = &slack
		row.Verdict = stopeta.VerdictLate
		return
	}

	if row.EstimatedArrival == nil {
		return
	}
	if cutoff <= 0 {
		row.Verdict = stopeta.VerdictOnTime
		return
	}

	slack := floorMinutes(cutoff - *row.EstimatedArrival)
	row.SlackMinutes = &slack
	switch {
	case *row.EstimatedArrival > cutoff:
		row.Verdict = stopeta.VerdictLate
	case slack < p.in.Settings.ThresholdMinutes:
		row.Verdict = stopeta.VerdictAtRisk
	default:
		row.Verdict = stopeta.VerdictOnTime
	}
}

func (p *Plan) markDelayed(row *stopeta.StopETA) bool {
	if p.in.Shipment.Status != shipment.StatusDelayed || row.Arrived ||
		row.Verdict != stopeta.VerdictUnknown {
		return false
	}
	row.Verdict = stopeta.VerdictLate
	return true
}

func (p *Plan) startHOS(c *cursor, move *shipment.ShipmentMove) {
	board := p.in.Assignments[move.ID]
	switch {
	case board == nil:
		c.hosNote = hosNoteNone
	case board.CoverageType == string(shipment.MoveCoverageTypeCarrier) ||
		board.AssignedCarrierID.IsNotNil():
		c.hosNote = hosNoteCarrier
	case board.AssignedWorkerID.IsNil():
		c.hosNote = hosNoteNone
	default:
		c.hosWorker = board.AssignedWorkerID
		c.schedule, c.hosNote = p.scheduleFor(board.AssignedWorkerID)
	}
}

func (p *Plan) switchDriver(c *cursor, move *shipment.ShipmentMove) {
	board := p.in.Assignments[move.ID]
	if board == nil || board.AssignedWorkerID.IsNil() || board.AssignedWorkerID == c.hosWorker {
		return
	}
	if c.hosWorker.IsNil() {
		return
	}
	c.schedule = nil
	c.hosNote = hosNoteHandover
}

func (p *Plan) scheduleFor(workerID pulid.ID) (*hosprojection.Schedule, hosNote) {
	if p.in.HOSExempt[workerID] {
		return nil, hosNoteExempt
	}
	state := p.in.HOS[workerID]
	if state == nil {
		return nil, hosNoteMissing
	}
	if p.in.Now-state.RecordedAt > HOSStaleAfterSeconds {
		return nil, hosNoteStale
	}

	limits := hosprojection.LimitsForRuleset(
		state.RulesetCycle,
		state.RulesetShift,
		state.RulesetJurisdiction,
	)

	return hosprojection.NewSchedule(hosprojection.Clocks{
		DriveMs: state.DriveRemainingMs,
		ShiftMs: state.ShiftRemainingMs,
		CycleMs: state.CycleRemainingMs,
		BreakMs: state.BreakRemainingMs,
	}, limits), hosNoteApplied
}

func floorMinutes(seconds int64) int64 {
	if seconds >= 0 {
		return seconds / secondsPerMinute
	}

	return -((-seconds + secondsPerMinute - 1) / secondsPerMinute)
}
