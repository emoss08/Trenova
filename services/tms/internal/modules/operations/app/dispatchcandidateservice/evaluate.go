package dispatchcandidateservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcost"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatcheligibility"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/hosprojection"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	millisPerSecond    = int64(1_000)
	millisPerHour      = float64(3_600_000)
	tightSlackMinutes  = 90.0
	localComfortMiles  = 150.0
	localHardMiles     = 300.0
	regionalComfortMi  = 500.0
	regionalHardMiles  = 800.0
	otrShortRunMiles   = 100.0
	otrShortRunQuality = 0.4
)

type trip struct {
	emptyMiles          float64
	emptyKnown          bool
	loadedMiles         float64
	emptySeconds        int64
	loadedSeconds       int64
	driveMs             int64
	restMs              int64
	departure           int64
	pickupArrival       int64
	deliveryArrival     int64
	freeAt              int64
	slackMinutes        float64
	appointmentKnown    bool
	windowClosedAt      int64
	earlyMinutes        float64
	pickupLateMinutes   float64
	deliveryLateMinutes float64
}

type evaluation struct {
	profile    *driverProfile
	move       *dispatchboardports.BoardMove
	trailerID  pulid.ID
	trip       trip
	projection *hosprojection.Result
	findings   []dispatchfinding.Finding
	features   dispatchcost.Features
	verdict    string
}

func (e *evaluation) blocked() bool {
	for i := range e.findings {
		if e.findings[i].Severity == dispatchfinding.SeverityBlock {
			return true
		}
	}
	return false
}

type routedLeg struct {
	miles   float64
	seconds int64
}

func evaluate(
	snapshot *FleetSnapshot,
	profile *driverProfile,
	move *dispatchboardports.BoardMove,
	routed *routedLeg,
) *evaluation {
	eval := &evaluation{profile: profile, move: move, trailerID: resolveTrailer(move)}
	eval.trip = planDistances(profile, move, routed)
	eval.projection = projectHOS(snapshot, profile, eval.trip.driveMs)
	scheduleTrip(snapshot, profile, move, eval)

	result := dispatcheligibility.Evaluate(dispatcheligibility.EvaluateInput{
		Candidate: dispatcheligibility.Candidate{
			Worker:           profile.worker,
			Tractor:          profile.tractor,
			Trailer:          snapshot.TrailerByID[eval.trailerID],
			HOSState:         profile.hosState,
			HOSProjection:    eval.projection,
			ApprovedPTO:      profile.pto,
			CommittedWindows: profile.committedWindows,
		},
		Requirements:     requirementsFor(move),
		Control:          snapshot.Control,
		Now:              snapshot.Now,
		TelematicsActive: snapshot.TelematicsActive,
	})
	if finding := appointmentFinding(&eval.trip, snapshot.Control); finding != nil {
		result.Add(finding)
	}
	if !eval.trip.emptyKnown {
		result.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeLocationUnknown,
			Severity: dispatchfinding.SeverityWarn,
			Field:    "workerId",
			Message: "Where this driver is is unknown, so empty miles and arrival " +
				"can't be judged; they are never assigned automatically",
		})
	}
	eval.findings = result.Findings
	eval.verdict = verdictFor(&verdictInput{
		Eval:         result,
		SlackMinutes: eval.trip.slackMinutes,
		SlackKnown:   eval.trip.appointmentKnown,
		HOSKnown:     profile.hosKnown,
		HOSExpected:  snapshot.TelematicsActive,
		Projection:   eval.projection,
	})
	eval.features = featuresFor(snapshot, profile, move, eval)

	return eval
}

func requirementsFor(move *dispatchboardports.BoardMove) dispatcheligibility.MoveRequirements {
	return dispatcheligibility.MoveRequirements{
		MoveID:     move.MoveID,
		ShipmentID: move.ShipmentID,
		Window: dispatcheligibility.TimeWindow{
			Start: move.OriginWindowStart,
			End:   MoveWindowEnd(move),
		},
		HasHazmatCommodities:  move.HasHazmat,
		HasActiveHold:         move.HasActiveHold,
		RequiredTractorTypeID: move.RequiredTractorTypeID,
		RequiredTrailerTypeID: move.RequiredTrailerTypeID,
	}
}

func planDistances(
	profile *driverProfile,
	move *dispatchboardports.BoardMove,
	routed *routedLeg,
) trip {
	t := trip{}

	switch {
	case move.Distance != nil:
		t.loadedMiles = *move.Distance
	case move.OriginLatitude != nil && move.OriginLongitude != nil &&
		move.DestinationLatitude != nil && move.DestinationLongitude != nil:
		t.loadedMiles = geoutils.PlanningRoadMiles(
			*move.OriginLatitude,
			*move.OriginLongitude,
			*move.DestinationLatitude,
			*move.DestinationLongitude,
		)
	}

	switch {
	case routed != nil:
		t.emptyMiles, t.emptyKnown = routed.miles, true
		t.emptySeconds = routed.seconds
	case profile.originSource != dispatchplanports.OriginUnknown &&
		move.OriginLatitude != nil && move.OriginLongitude != nil:
		t.emptyMiles = geoutils.PlanningRoadMiles(
			profile.origin.lat,
			profile.origin.lon,
			*move.OriginLatitude,
			*move.OriginLongitude,
		)
		t.emptyKnown = true
		t.emptySeconds = geoutils.PlanningDriveSeconds(t.emptyMiles)
	}

	t.loadedSeconds = geoutils.PlanningDriveSeconds(t.loadedMiles)
	t.driveMs = (t.emptySeconds + t.loadedSeconds) * millisPerSecond

	return t
}

func scheduleTrip(
	snapshot *FleetSnapshot,
	profile *driverProfile,
	move *dispatchboardports.BoardMove,
	eval *evaluation,
) {
	t := &eval.trip
	now := snapshot.Now
	dwell := int64(dwellMinutes(snapshot.Control)) * secondsPerMinute

	if eval.projection != nil {
		t.restMs = max(0, eval.projection.Trip.TotalMs-profile.committedDriveMs-t.driveMs)
	}
	restBefore, restLoaded := splitRest(t.restMs, t.emptySeconds, t.loadedSeconds)

	t.departure = max(now, profile.availableAt)
	t.pickupArrival = t.departure + t.emptySeconds + restBefore
	pickupStart := max(t.pickupArrival, move.OriginWindowStart)
	t.deliveryArrival = pickupStart + dwell + t.loadedSeconds + restLoaded
	t.freeAt = t.deliveryArrival + dwell

	if move.OriginWindowStart > t.pickupArrival {
		t.earlyMinutes = float64(move.OriginWindowStart-t.pickupArrival) / secondsPerMinute
	}

	appointment := move.OriginWindowStart
	if move.OriginWindowEnd != nil && *move.OriginWindowEnd > appointment {
		appointment = *move.OriginWindowEnd
	}
	switch {
	case appointment <= 0:
	case appointment <= now:
		t.windowClosedAt = appointment
	default:
		t.appointmentKnown = true
		t.slackMinutes = float64(appointment-t.pickupArrival) / secondsPerMinute
	}
	if appointment > 0 {
		t.pickupLateMinutes = lateMinutes(t.pickupArrival, max(appointment, now))
	}
	if windowClose := MoveWindowEnd(move); windowClose > 0 {
		t.deliveryLateMinutes = lateMinutes(t.deliveryArrival, max(windowClose, now))
	}
}

func splitRest(restMs, emptySeconds, loadedSeconds int64) (before, loaded int64) {
	total := emptySeconds + loadedSeconds
	if restMs <= 0 || total <= 0 {
		return 0, 0
	}
	restSeconds := restMs / millisPerSecond
	before = restSeconds * emptySeconds / total

	return before, restSeconds - before
}

func lateMinutes(arrival, due int64) float64 {
	if arrival <= due {
		return 0
	}

	return float64(arrival-due) / secondsPerMinute
}

func dwellMinutes(control *dispatchcontrol.DispatchControl) int16 {
	if control == nil || control.EtaDwellMinutes <= 0 {
		return dispatchcontrol.DefaultEtaDwellMinutes
	}

	return control.EtaDwellMinutes
}

func projectHOS(
	snapshot *FleetSnapshot,
	profile *driverProfile,
	tripDriveMs int64,
) *hosprojection.Result {
	if !profile.hosKnown || profile.hosExempt {
		return nil
	}
	state := profile.hosState

	result := hosprojection.Project(hosprojection.Input{
		Now:         snapshot.Now,
		Departure:   snapshot.Now,
		TripDriveMs: profile.committedDriveMs + tripDriveMs,
		Clocks: hosprojection.Clocks{
			DriveMs: state.DriveRemainingMs,
			ShiftMs: state.ShiftRemainingMs,
			CycleMs: state.CycleRemainingMs,
			BreakMs: state.BreakRemainingMs,
		},
		ClocksAt:        state.RecordedAt,
		CycleTomorrowMs: state.CycleTomorrowMs,
		DutyStatus:      state.DutyStatus,
		Limits:          profile.hosLimits,
		Jurisdiction:    state.RulesetJurisdiction,
		Timeline:        profile.hosTimeline,
	})

	return &result
}

const assignmentField = "assignment"

func appointmentFinding(
	t *trip,
	control *dispatchcontrol.DispatchControl,
) *dispatchfinding.Finding {
	if t.windowClosedAt > 0 {
		return &dispatchfinding.Finding{
			Code:     dispatchfinding.CodeWindowPassed,
			Severity: dispatchfinding.SeverityWarn,
			Field:    assignmentField,
			Message: "The pickup window has already closed, so whoever gets there " +
				"soonest is cheapest. Reschedule the pickup if it has not happened yet",
		}
	}
	if !t.appointmentKnown || t.slackMinutes >= 0 {
		return nil
	}

	lateBy := timeutils.FormatLongDurationMs(int64(-t.slackMinutes * float64(millisPerMinute)))
	if control != nil && control.EnforceWorkerPTARestrictions {
		return &dispatchfinding.Finding{
			Code:     dispatchfinding.CodeAppointmentMissed,
			Severity: dispatchfinding.SeverityBlock,
			Field:    assignmentField,
			Message:  fmt.Sprintf("Driver would reach the pickup %s after its window", lateBy),
		}
	}
	return &dispatchfinding.Finding{
		Code:     dispatchfinding.CodeAppointmentAtRisk,
		Severity: dispatchfinding.SeverityWarn,
		Field:    assignmentField,
		Message:  fmt.Sprintf("Driver is projected %s late to the pickup window", lateBy),
	}
}

type verdictInput struct {
	Eval         *dispatcheligibility.Evaluation
	SlackMinutes float64
	SlackKnown   bool
	HOSKnown     bool
	HOSExpected  bool
	Projection   *hosprojection.Result
}

func verdictFor(in *verdictInput) string {
	switch {
	case in.Eval.Blocked():
		return telematics.FeasibilityVerdictInfeasible
	case in.HOSExpected && !in.HOSKnown:
		return telematics.FeasibilityVerdictUnknown
	case in.Projection != nil && !in.Projection.Feasible:
		return telematics.FeasibilityVerdictInfeasible
	case in.SlackKnown && in.SlackMinutes < 0:
		return telematics.FeasibilityVerdictInfeasible
	case in.SlackKnown && in.SlackMinutes < tightSlackMinutes:
		return telematics.FeasibilityVerdictTight
	case in.Projection != nil && in.Projection.Strategy != hosprojection.StrategyCurrentClocks:
		return telematics.FeasibilityVerdictTight
	default:
		return telematics.FeasibilityVerdictFeasible
	}
}

func featuresFor(
	snapshot *FleetSnapshot,
	profile *driverProfile,
	move *dispatchboardports.BoardMove,
	eval *evaluation,
) dispatchcost.Features {
	t := eval.trip
	features := dispatchcost.Features{
		EmptyMiles:          t.emptyMiles,
		PickupLateMinutes:   t.pickupLateMinutes,
		DeliveryLateMinutes: t.deliveryLateMinutes,
		EarlyMinutes:        t.earlyMinutes,
		RestHours:           float64(t.restMs) / millisPerHour,
		WeekMilesOverMedian: profile.weekMilesOverMedian,
		HomeOverdueDays: homeOverdueDays(
			profile,
			t.freeAt,
			snapshot.Model.Economics().HomeTimeTargetDays,
		),
		HomewardShare: homewardShare(profile, move),
		CustomerRuns: dispatchcost.CustomerRunsQuality(
			snapshot.LaneExperience[laneKey{profile.driver.WorkerID, move.CustomerID}],
		),
		OnTimeHistory:     profile.onTime,
		Safety:            profile.safety,
		Acceptance:        profile.acceptance,
		TrailerContinuity: trailerContinuity(profile, eval.trailerID),
		HaulFit:           haulFit(profile.driver.DriverType, t.emptyMiles+t.loadedMiles),
		TimeOffBuffer:     timeOffBuffer(profile, t.freeAt),
	}
	if move.DestinationLatitude != nil && move.DestinationLongitude != nil {
		features.EndMarketCents = snapshot.EndMarket.EndMarketCents(
			*move.DestinationLatitude,
			*move.DestinationLongitude,
			t.freeAt,
		)
	}

	return features
}

func trailerContinuity(profile *driverProfile, needed pulid.ID) dispatchcost.Signal {
	switch {
	case needed.IsNil():
		return dispatchcost.Known(1)
	case profile.hasTrailer && profile.trailerID == needed:
		return dispatchcost.Known(1)
	default:
		return dispatchcost.Known(0)
	}
}

func timeOffBuffer(profile *driverProfile, freeAt int64) dispatchcost.Signal {
	if profile.nextPTOStart <= 0 {
		return dispatchcost.Known(1)
	}

	return dispatchcost.TimeOffBufferQuality(float64(profile.nextPTOStart-freeAt) / 3_600)
}

func haulFit(driverType worker.DriverType, tripMiles float64) dispatchcost.Signal {
	if tripMiles <= 0 {
		return dispatchcost.Known(1)
	}

	switch driverType {
	case worker.DriverTypeLocal:
		return dispatchcost.Known(invertedRamp(tripMiles, localComfortMiles, localHardMiles))
	case worker.DriverTypeRegional:
		return dispatchcost.Known(invertedRamp(tripMiles, regionalComfortMi, regionalHardMiles))
	case worker.DriverTypeOTR, worker.DriverTypeTeam:
		if tripMiles < otrShortRunMiles {
			return dispatchcost.Known(otrShortRunQuality)
		}
		return dispatchcost.Known(1)
	default:
		return dispatchcost.Signal{}
	}
}

func resolveTrailer(move *dispatchboardports.BoardMove) pulid.ID {
	if !move.AssignedTrailerID.IsNil() {
		return move.AssignedTrailerID
	}
	return move.PreviousMoveTrailerID
}
