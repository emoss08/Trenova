package dispatchcandidateservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcost"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatcheligibility"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/hosprojection"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	onTimeShortfallZeroCreditPp = 20.0
	defaultOnTimeTargetPct      = 95.0
	onTimeFailurePenalty        = 0.05
	onTimeHistoryMinStops       = 5
	acceptanceMinDecisions      = 3
	ackLatencyFullCreditMinutes = 15.0
	ackLatencyZeroCreditMinutes = 240.0
	acceptRateShare             = 0.8
	millisPerMinute             = int64(60_000)
	secondsPerMinute            = 60
)

type geoPoint struct {
	lat float64
	lon float64
}

type driverProfile struct {
	driver  *dispatchboardports.BoardDriver
	worker  *worker.Worker
	tractor *tractor.Tractor
	name    string

	hosState    *telematics.WorkerHOSState
	hosKnown    bool
	hosExempt   bool
	hosLimits   hosprojection.Limits
	hosTimeline []hosprojection.DutyInterval

	pto              []dispatcheligibility.PTOWindow
	committedWindows []dispatcheligibility.TimeWindow
	availableAt      int64
	committedDriveMs int64
	nextPTOStart     int64

	origin       geoPoint
	originSource dispatchplanports.OriginSource

	home       geoPoint
	homeKnown  bool
	lastHomeAt int64

	trailerID  pulid.ID
	hasTrailer bool

	weekMilesOverMedian float64
	onTime              dispatchcost.Signal
	safety              dispatchcost.Signal
	acceptance          dispatchcost.Signal
}

func buildProfile(
	snapshot *FleetSnapshot,
	driver *dispatchboardports.BoardDriver,
) *driverProfile {
	w := snapshot.WorkersByID[driver.WorkerID]
	if w == nil {
		return nil
	}

	commitments := snapshot.CommitmentsByWorker[driver.WorkerID]
	profile := &driverProfile{
		driver:           driver,
		worker:           w,
		tractor:          snapshot.TractorByID[driver.TractorID],
		name:             snapshot.DriverName(driver),
		pto:              ptoWindows(snapshot.TimeOffByWorker[driver.WorkerID]),
		committedWindows: commitmentWindows(commitments),
		availableAt:      ProjectedTimeAvailable(commitments, snapshot.Now),
		committedDriveMs: committedDriveMs(commitments, snapshot.Now),
		nextPTOStart: nextApprovedPTOStart(
			snapshot.TimeOffByWorker[driver.WorkerID],
			snapshot.Now,
		),
	}
	profile.trailerID, profile.hasTrailer = currentTrailer(commitments)
	profile.origin, profile.originSource = resolveOrigin(snapshot, driver, profile, commitments)
	if positioning := snapshot.PositionedByWorker[driver.WorkerID]; positioning != nil &&
		profile.originSource == dispatchplanports.OriginPositioned {
		profile.availableAt = max(profile.availableAt, positioning.ArriveAt)
	}
	applyHomeProfile(snapshot, profile)
	applyHOSProfile(snapshot, profile)
	applyHistoryProfile(snapshot, profile)

	return profile
}

func resolveOrigin(
	snapshot *FleetSnapshot,
	driver *dispatchboardports.BoardDriver,
	profile *driverProfile,
	commitments []*dispatchboardports.WorkerCommitment,
) (geoPoint, dispatchplanports.OriginSource) {
	if positioning := snapshot.PositionedByWorker[driver.WorkerID]; positioning != nil {
		return geoPoint{
			lat: positioning.ToLatitude,
			lon: positioning.ToLongitude,
		}, dispatchplanports.OriginPositioned
	}
	if profile.availableAt > snapshot.Now {
		if destination, ok := finalCommitmentDestination(commitments); ok {
			return destination, dispatchplanports.OriginCommitment
		}
	}
	if position := snapshot.PosByTractor[driver.TractorID]; position != nil {
		return geoPoint{
			lat: position.Latitude,
			lon: position.Longitude,
		}, dispatchplanports.OriginPosition
	}
	if t := profile.tractor; t != nil &&
		t.LastKnownLocationLatitude != nil && t.LastKnownLocationLongitude != nil {
		return geoPoint{
			lat: *t.LastKnownLocationLatitude,
			lon: *t.LastKnownLocationLongitude,
		}, dispatchplanports.OriginLastDrop
	}

	return geoPoint{}, dispatchplanports.OriginUnknown
}

func applyHOSProfile(snapshot *FleetSnapshot, profile *driverProfile) {
	state := snapshot.HOSByWorker[profile.driver.WorkerID]
	profile.hosState = state
	profile.hosKnown = snapshot.TelematicsActive && state != nil &&
		snapshot.Now-state.RecordedAt <= dispatcheligibility.HOSStateMaxAgeSeconds
	if p := profile.worker.Profile; p != nil && (p.ELDExempt || p.ShortHaulExempt) {
		profile.hosExempt = true
	}
	if !profile.hosKnown || profile.hosExempt {
		return
	}

	profile.hosLimits = hosprojection.LimitsForRuleset(
		state.RulesetCycle,
		state.RulesetShift,
		state.RulesetJurisdiction,
	)
	profile.hosTimeline = hosprojection.BuildTimeline(
		snapshot.HOSLogsByWorker[profile.driver.WorkerID],
		snapshot.Now,
	)
}

func applyHistoryProfile(snapshot *FleetSnapshot, profile *driverProfile) {
	workerID := profile.driver.WorkerID

	if workload := snapshot.WorkloadByWorker[workerID]; workload != nil {
		profile.weekMilesOverMedian = workload.TotalMiles - snapshot.medianWeekMiles
	}

	if stats := snapshot.OnTimeByWorker[workerID]; stats != nil &&
		stats.CompletedStops >= onTimeHistoryMinStops {
		target := defaultOnTimeTargetPct
		if c := snapshot.Control; c != nil && c.ServiceFailureTarget != nil &&
			*c.ServiceFailureTarget > 0 {
			target = *c.ServiceFailureTarget
		}
		pct := 100 * float64(stats.OnTimeStops) / float64(stats.CompletedStops)
		quality := 1 - (target-pct)/onTimeShortfallZeroCreditPp -
			onTimeFailurePenalty*float64(stats.DriverFaultFailures)
		profile.onTime = dispatchcost.Known(min(quality, 1))
	}

	if snapshot.TelematicsActive {
		violations := 0.0
		if stats := snapshot.ViolationsByWorker[workerID]; stats != nil {
			violations = float64(stats.ViolationCount)
		}
		profile.safety = dispatchcost.SafetyQuality(violations)
	}

	if stats := snapshot.AcceptanceByWorker[workerID]; stats != nil {
		decisions := stats.Accepted + stats.Declined
		if decisions >= acceptanceMinDecisions {
			latency := 1.0
			if minutes := stats.AvgAckLatencySeconds / secondsPerMinute; minutes > 0 {
				latency = invertedRamp(
					minutes,
					ackLatencyFullCreditMinutes,
					ackLatencyZeroCreditMinutes,
				)
			}
			rate := float64(stats.Accepted) / float64(decisions)
			profile.acceptance = dispatchcost.Known(
				acceptRateShare*rate + (1-acceptRateShare)*latency,
			)
		}
	}
}

func committedDriveMs(commitments []*dispatchboardports.WorkerCommitment, now int64) int64 {
	var total int64
	for _, commitment := range commitments {
		switch {
		case commitment.RemainingDriveMinutes > 0:
			total += commitment.RemainingDriveMinutes * millisPerMinute
		case commitment.Distance != nil && commitment.WindowEnd > now:
			total += geoutils.PlanningDriveSeconds(*commitment.Distance) * 1000
		}
	}

	return total
}

func ProjectedTimeAvailable(commitments []*dispatchboardports.WorkerCommitment, now int64) int64 {
	latest := now
	for _, commitment := range commitments {
		latest = max(latest, commitmentEnd(commitment))
	}
	return latest
}

func commitmentEnd(commitment *dispatchboardports.WorkerCommitment) int64 {
	return max(commitment.WindowEnd, commitment.ExpectedEnd)
}

func finalCommitmentDestination(
	commitments []*dispatchboardports.WorkerCommitment,
) (geoPoint, bool) {
	var latest *dispatchboardports.WorkerCommitment
	for _, commitment := range commitments {
		if latest == nil || commitmentEnd(commitment) > commitmentEnd(latest) {
			latest = commitment
		}
	}
	if latest == nil || latest.DestLatitude == nil || latest.DestLongitude == nil {
		return geoPoint{}, false
	}
	return geoPoint{lat: *latest.DestLatitude, lon: *latest.DestLongitude}, true
}

func currentTrailer(commitments []*dispatchboardports.WorkerCommitment) (pulid.ID, bool) {
	var (
		latest    int64
		trailerID pulid.ID
		found     bool
	)
	for _, commitment := range commitments {
		if commitment.TrailerID.IsNil() {
			continue
		}
		if !found || commitmentEnd(commitment) > latest {
			latest = commitmentEnd(commitment)
			trailerID = commitment.TrailerID
			found = true
		}
	}
	return trailerID, found
}

func nextApprovedPTOStart(timeOff []*dispatchboardports.WorkerTimeOff, now int64) int64 {
	next := int64(0)
	for _, pto := range timeOff {
		if pto.StartDate <= now || pto.StartDate > now+ptoLookaheadSeconds {
			continue
		}
		if next == 0 || pto.StartDate < next {
			next = pto.StartDate
		}
	}
	return next
}

func ptoWindows(items []*dispatchboardports.WorkerTimeOff) []dispatcheligibility.PTOWindow {
	if len(items) == 0 {
		return nil
	}
	windows := make([]dispatcheligibility.PTOWindow, 0, len(items))
	for _, item := range items {
		windows = append(windows, dispatcheligibility.PTOWindow{
			Window: dispatcheligibility.TimeWindow{
				Start: item.StartDate,
				End:   PTOInclusiveEnd(item.EndDate),
			},
			Type: item.Type,
		})
	}
	return windows
}

func commitmentWindows(
	items []*dispatchboardports.WorkerCommitment,
) []dispatcheligibility.TimeWindow {
	if len(items) == 0 {
		return nil
	}
	windows := make([]dispatcheligibility.TimeWindow, 0, len(items))
	for _, item := range items {
		windows = append(windows, dispatcheligibility.TimeWindow{
			Start: item.WindowStart,
			End:   commitmentEnd(item),
		})
	}
	slices.SortFunc(windows, func(a, b dispatcheligibility.TimeWindow) int {
		return int(a.Start - b.Start)
	})
	return windows
}

func invertedRamp(value, best, worst float64) float64 {
	if worst <= best {
		return 0
	}
	if value <= best {
		return 1
	}
	if value >= worst {
		return 0
	}
	return 1 - (value-best)/(worst-best)
}
