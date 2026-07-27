package dispatchconsoleservice

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
)

// decorateMoves derives the triage facts the console renders on each card.
func decorateMoves(moves []*repositories.BoardMove, now int64) []*BoardMove {
	decorated := make([]*BoardMove, 0, len(moves))
	for _, move := range moves {
		decorated = append(decorated, decorateMove(move, now))
	}
	return decorated
}

func decorateMove(move *repositories.BoardMove, now int64) *BoardMove {
	return &BoardMove{
		BoardMove:       move,
		Urgency:         urgencyFor(move, now),
		MinutesToPickup: minutesToPickup(move, now),
		IsCovered:       !move.AssignmentID.IsNil(),
		TotalStopCount:  move.MoveCount,
	}
}

// urgencyFor buckets a move by how soon its pickup is due. A move whose window has
// already opened without coverage is Late — that is the queue a dispatcher works first.
func urgencyFor(move *repositories.BoardMove, now int64) UrgencyBucket {
	if move.OriginWindowStart <= 0 {
		return UrgencyPlanned
	}

	delta := move.OriginWindowStart - now
	switch {
	case delta < 0:
		return UrgencyLate
	case delta <= nowThresholdSeconds:
		return UrgencyNow
	case delta <= secondsPerDay:
		return UrgencyToday
	case delta <= 2*secondsPerDay:
		return UrgencyTomorrow
	default:
		return UrgencyPlanned
	}
}

func minutesToPickup(move *repositories.BoardMove, now int64) int64 {
	if move.OriginWindowStart <= 0 {
		return 0
	}
	return (move.OriginWindowStart - now) / 60
}

// decorateDrivers turns the fleet snapshot into capacity-rail rows, attaching live
// clocks, position, PTA, and the reasons a driver is unavailable.
func decorateDrivers(
	snapshot *dispatchcandidateservice.FleetSnapshot,
	control *dispatchcontrol.DispatchControl,
	now int64,
) []*BoardDriver {
	decorated := make([]*BoardDriver, 0, len(snapshot.Drivers))

	for _, driver := range snapshot.Drivers {
		row := &BoardDriver{
			BoardDriver: driver,
			Commitments: snapshot.CommitmentsByWorker[driver.WorkerID],
			TimeOff:     snapshot.TimeOffByWorker[driver.WorkerID],
		}

		applyHOS(row, snapshot, now)
		applyPosition(row, snapshot)
		applyWorkload(row, snapshot)

		row.ProjectedTimeAvailable = projectedTimeAvailable(row.Commitments, now)
		row.Findings = availabilityFindings(driver, snapshot, control, now)
		row.Availability = availabilityFor(row, now)

		decorated = append(decorated, row)
	}

	return decorated
}

func applyHOS(
	row *BoardDriver,
	snapshot *dispatchcandidateservice.FleetSnapshot,
	now int64,
) {
	state := snapshot.HOSByWorker[row.WorkerID]
	if state == nil {
		return
	}

	row.DutyStatus = string(state.DutyStatus)
	row.DriveRemainingMs = state.DriveRemainingMs
	row.ShiftRemainingMs = state.ShiftRemainingMs
	row.CycleRemainingMs = state.CycleRemainingMs
	row.BreakRemainingMs = state.BreakRemainingMs
	row.HOSRecordedAt = state.RecordedAt
	row.HOSIsStale = now-state.RecordedAt > dispatcheligibility.HOSStateMaxAgeSeconds
}

func applyPosition(row *BoardDriver, snapshot *dispatchcandidateservice.FleetSnapshot) {
	position := snapshot.PosByTractor[row.TractorID]
	if position == nil {
		return
	}

	row.Latitude = &position.Latitude
	row.Longitude = &position.Longitude
	row.FormattedLocation = position.FormattedLocation
	row.PositionAt = position.RecordedAt
}

func applyWorkload(row *BoardDriver, snapshot *dispatchcandidateservice.FleetSnapshot) {
	workload := snapshot.WorkloadByWorker[row.WorkerID]
	if workload == nil {
		return
	}

	row.CommittedMiles = workload.TotalMiles
	row.CommittedRevenue = workload.TotalRevenue
}

// availabilityFindings runs the driver-only rule sets — the ones that do not depend on a
// particular move — so the rail can explain why a driver is unavailable before a
// dispatcher tries to use them.
func availabilityFindings(
	driver *repositories.BoardDriver,
	snapshot *dispatchcandidateservice.FleetSnapshot,
	control *dispatchcontrol.DispatchControl,
	now int64,
) []dispatcheligibility.Finding {
	w := snapshot.WorkersByID[driver.WorkerID]
	if w == nil {
		return nil
	}

	eval := dispatcheligibility.NewEvaluation(4)

	eval.Merge(dispatcheligibility.EvaluateAvailability(dispatcheligibility.AvailabilityInput{
		Worker:  w,
		Control: control,
	}))
	eval.Merge(dispatcheligibility.EvaluateWorkerCompliance(
		dispatcheligibility.WorkerComplianceInput{
			Worker:  w,
			Control: control,
		},
	))
	eval.Merge(dispatcheligibility.EvaluateHOSClocks(dispatcheligibility.HOSInput{
		State:   snapshot.HOSByWorker[driver.WorkerID],
		Control: control,
		Now:     now,
	}))

	return eval.Findings
}

// availabilityFor is the single word the rail shows for a driver's state, chosen in
// priority order: a blocked driver is blocked no matter what their calendar says.
func availabilityFor(row *BoardDriver, now int64) DriverAvailability {
	for i := range row.Findings {
		if row.Findings[i].Severity == dispatcheligibility.SeverityBlock {
			return AvailabilityBlocked
		}
	}

	for _, pto := range row.TimeOff {
		if pto.StartDate <= now && pto.EndDate >= now {
			return AvailabilityTimeOff
		}
	}

	if len(row.Commitments) == 0 {
		return AvailabilityOpen
	}

	if row.ProjectedTimeAvailable-now <= nowThresholdSeconds {
		return AvailabilityFinishing
	}

	return AvailabilityWorking
}

// projectedTimeAvailable is when the driver frees up: the end of their last committed
// move, or now when they hold nothing.
func projectedTimeAvailable(commitments []*repositories.WorkerCommitment, now int64) int64 {
	latest := now
	for _, commitment := range commitments {
		if commitment.WindowEnd > latest {
			latest = commitment.WindowEnd
		}
	}
	return latest
}
