package dispatchcandidateservice

import (
	"context"
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/internal/core/services/hosprojection"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	averageLinehaulMph = 50.0

	roadCircuityFactor = 1.2

	positionMaxAgeSeconds = int64(4 * 3600)

	workloadLookbackSeconds = int64(7 * 24 * 3600)

	laneExperienceLookbackSeconds = int64(365 * 24 * 3600)

	onTimeHistoryLookbackSeconds = int64(90 * 24 * 3600)

	acceptanceLookbackSeconds = int64(180 * 24 * 3600)

	safetyLookbackSeconds = int64(90 * 24 * 3600)

	hosLogLookbackSeconds = int64(9 * 24 * 3600)

	ptoLookaheadSeconds = int64(14 * 24 * 3600)
)

type Params struct {
	fx.In

	ConsoleRepo     repositories.DispatchConsoleRepository
	TelematicsRepo  repositories.TelematicsRepository
	IntegrationRepo repositories.IntegrationRepository
}

type Service struct {
	consoleRepo     repositories.DispatchConsoleRepository
	telematicsRepo  repositories.TelematicsRepository
	integrationRepo repositories.IntegrationRepository
}

func New(p Params) *Service {
	return &Service{
		consoleRepo:     p.ConsoleRepo,
		telematicsRepo:  p.TelematicsRepo,
		integrationRepo: p.IntegrationRepo,
	}
}

// telematicsActive reports whether the organization has an enabled, supported
// telematics integration. Without one, hours-of-service must not be weighed at all:
// no HOS factor, no HOS findings, and no unknown-verdict penalty for missing data.
func (s *Service) telematicsActive(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	if s.integrationRepo == nil || s.telematicsRepo == nil {
		return false, nil
	}
	records, err := s.integrationRepo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	return integration.HasEnabledTelematics(records), nil
}

type FleetSnapshot struct {
	Drivers             []*repositories.BoardDriver
	WorkersByID         map[pulid.ID]*worker.Worker
	TractorByID         map[pulid.ID]*tractor.Tractor
	TrailerByID         map[pulid.ID]*trailer.Trailer
	HOSByWorker         map[pulid.ID]*telematics.WorkerHOSState
	HOSLogsByWorker     map[pulid.ID][]*telematics.WorkerHOSLog
	PosByTractor        map[pulid.ID]*telematics.VehiclePosition
	CommitmentsByWorker map[pulid.ID][]*repositories.WorkerCommitment
	TimeOffByWorker     map[pulid.ID][]*repositories.WorkerTimeOff
	WorkloadByWorker    map[pulid.ID]*repositories.WorkerWorkload
	OnTimeByWorker      map[pulid.ID]*repositories.WorkerOnTimeStats
	AcceptanceByWorker  map[pulid.ID]*repositories.WorkerAcceptanceStats
	ViolationsByWorker  map[pulid.ID]*repositories.WorkerHOSViolationStats
	NamesByWorker       map[pulid.ID]string
	LaneExperience      map[laneKey]int
	Control             *dispatchcontrol.DispatchControl
	TelematicsActive    bool
	Now                 int64
}

func (s *FleetSnapshot) driverName(driver *repositories.BoardDriver) string {
	if name, ok := s.NamesByWorker[driver.WorkerID]; ok {
		return name
	}
	return driver.FirstName + " " + driver.LastName
}

type laneKey struct {
	WorkerID   pulid.ID
	CustomerID pulid.ID
}

type SnapshotRequest struct {
	TenantInfo  pagination.TenantInfo
	Filter      *repositories.DispatchBoardFilter
	Control     *dispatchcontrol.DispatchControl
	CustomerIDs []pulid.ID
	TrailerIDs  []pulid.ID
}

func (s *Service) BuildSnapshot(
	ctx context.Context,
	req *SnapshotRequest,
) (*FleetSnapshot, error) {
	drivers, err := s.consoleRepo.ListBoardDrivers(ctx, req.Filter)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	snapshot := &FleetSnapshot{
		Drivers:             drivers,
		WorkersByID:         make(map[pulid.ID]*worker.Worker, len(drivers)),
		TractorByID:         make(map[pulid.ID]*tractor.Tractor, len(drivers)),
		TrailerByID:         make(map[pulid.ID]*trailer.Trailer, len(drivers)),
		HOSByWorker:         make(map[pulid.ID]*telematics.WorkerHOSState, len(drivers)),
		HOSLogsByWorker:     make(map[pulid.ID][]*telematics.WorkerHOSLog, len(drivers)),
		PosByTractor:        make(map[pulid.ID]*telematics.VehiclePosition, len(drivers)),
		CommitmentsByWorker: make(map[pulid.ID][]*repositories.WorkerCommitment, len(drivers)),
		TimeOffByWorker:     make(map[pulid.ID][]*repositories.WorkerTimeOff, len(drivers)),
		WorkloadByWorker:    make(map[pulid.ID]*repositories.WorkerWorkload, len(drivers)),
		OnTimeByWorker:      make(map[pulid.ID]*repositories.WorkerOnTimeStats, len(drivers)),
		AcceptanceByWorker:  make(map[pulid.ID]*repositories.WorkerAcceptanceStats, len(drivers)),
		ViolationsByWorker:  make(map[pulid.ID]*repositories.WorkerHOSViolationStats, len(drivers)),
		NamesByWorker:       make(map[pulid.ID]string, len(drivers)),
		LaneExperience:      make(map[laneKey]int, len(drivers)),
		Control:             req.Control,
		Now:                 now,
	}

	active, err := s.telematicsActive(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	snapshot.TelematicsActive = active

	if len(drivers) == 0 {
		return snapshot, nil
	}

	workerIDs := make([]pulid.ID, 0, len(drivers))
	tractorIDs := make([]pulid.ID, 0, len(drivers))
	for _, driver := range drivers {
		workerIDs = append(workerIDs, driver.WorkerID)
		snapshot.NamesByWorker[driver.WorkerID] = driver.FirstName + " " + driver.LastName
		if !driver.TractorID.IsNil() {
			tractorIDs = append(tractorIDs, driver.TractorID)
		}
	}

	if err = s.loadSchedules(ctx, req, snapshot, workerIDs); err != nil {
		return nil, err
	}
	if err = s.loadWorkersAndEquipment(ctx, req, snapshot, workerIDs, tractorIDs); err != nil {
		return nil, err
	}
	if err = s.loadPerformance(ctx, req.TenantInfo, snapshot, workerIDs); err != nil {
		return nil, err
	}
	if err = s.loadTelematics(ctx, req.TenantInfo, snapshot, workerIDs, tractorIDs); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func (s *Service) loadPerformance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snapshot *FleetSnapshot,
	workerIDs []pulid.ID,
) error {
	onTime, err := s.consoleRepo.ListWorkerOnTimeStats(ctx, &repositories.ListWorkloadRequest{
		TenantInfo: tenantInfo,
		WorkerIDs:  workerIDs,
		Since:      snapshot.Now - onTimeHistoryLookbackSeconds,
	})
	if err != nil {
		return err
	}
	for _, item := range onTime {
		snapshot.OnTimeByWorker[item.WorkerID] = item
	}

	acceptance, err := s.consoleRepo.ListWorkerAcceptanceStats(
		ctx,
		&repositories.ListWorkloadRequest{
			TenantInfo: tenantInfo,
			WorkerIDs:  workerIDs,
			Since:      snapshot.Now - acceptanceLookbackSeconds,
		},
	)
	if err != nil {
		return err
	}
	for _, item := range acceptance {
		snapshot.AcceptanceByWorker[item.WorkerID] = item
	}

	return nil
}

func (s *Service) loadWorkersAndEquipment(
	ctx context.Context,
	req *SnapshotRequest,
	snapshot *FleetSnapshot,
	workerIDs, tractorIDs []pulid.ID,
) error {
	workers, err := s.consoleRepo.ListWorkersByIDs(ctx, req.TenantInfo, workerIDs)
	if err != nil {
		return err
	}
	for _, w := range workers {
		snapshot.WorkersByID[w.ID] = w
	}

	tractors, trailers, err := s.consoleRepo.ListEquipmentByIDs(
		ctx,
		&repositories.ListEquipmentByIDsRequest{
			TenantInfo: req.TenantInfo,
			TractorIDs: tractorIDs,
			TrailerIDs: snapshotTrailerIDs(req, snapshot),
		},
	)
	if err != nil {
		return err
	}
	for _, t := range tractors {
		snapshot.TractorByID[t.ID] = t
	}
	for _, t := range trailers {
		snapshot.TrailerByID[t.ID] = t
	}

	return nil
}

func snapshotTrailerIDs(req *SnapshotRequest, snapshot *FleetSnapshot) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(req.TrailerIDs)+len(snapshot.CommitmentsByWorker))
	ids := make([]pulid.ID, 0, len(req.TrailerIDs)+len(snapshot.CommitmentsByWorker))

	appendID := func(id pulid.ID) {
		if id.IsNil() {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	for _, id := range req.TrailerIDs {
		appendID(id)
	}
	for _, commitments := range snapshot.CommitmentsByWorker {
		for _, commitment := range commitments {
			appendID(commitment.TrailerID)
		}
	}

	return ids
}

func (s *Service) loadTelematics(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snapshot *FleetSnapshot,
	workerIDs, tractorIDs []pulid.ID,
) error {
	if s.telematicsRepo == nil || !snapshot.TelematicsActive {
		return nil
	}

	states, err := s.telematicsRepo.ListWorkerHOSStates(
		ctx,
		&repositories.ListWorkerHOSStatesRequest{
			TenantInfo: tenantInfo,
			WorkerIDs:  workerIDs,
		},
	)
	if err != nil {
		return err
	}
	for _, state := range states {
		snapshot.HOSByWorker[state.WorkerID] = state
	}

	logs, err := s.telematicsRepo.ListWorkerHOSLogs(ctx, &repositories.ListWorkerHOSLogsRequest{
		TenantInfo: tenantInfo,
		WorkerIDs:  workerIDs,
		Since:      snapshot.Now - hosLogLookbackSeconds,
	})
	if err != nil {
		return err
	}
	for _, entry := range logs {
		snapshot.HOSLogsByWorker[entry.WorkerID] = append(
			snapshot.HOSLogsByWorker[entry.WorkerID],
			entry,
		)
	}

	violations, err := s.telematicsRepo.ListWorkerHOSViolationStats(
		ctx,
		&repositories.ListWorkerHOSViolationStatsRequest{
			TenantInfo: tenantInfo,
			WorkerIDs:  workerIDs,
			Since:      snapshot.Now - safetyLookbackSeconds,
		},
	)
	if err != nil {
		return err
	}
	for _, item := range violations {
		snapshot.ViolationsByWorker[item.WorkerID] = item
	}

	if len(tractorIDs) == 0 {
		return nil
	}

	positions, err := s.telematicsRepo.ListVehiclePositions(
		ctx,
		&repositories.ListVehiclePositionsRequest{
			TenantInfo:    tenantInfo,
			TractorIDs:    tractorIDs,
			MaxAgeSeconds: positionMaxAgeSeconds,
		},
	)
	if err != nil {
		return err
	}
	for _, position := range positions {
		snapshot.PosByTractor[position.TractorID] = position
	}

	return nil
}

func (s *Service) loadSchedules(
	ctx context.Context,
	req *SnapshotRequest,
	snapshot *FleetSnapshot,
	workerIDs []pulid.ID,
) error {
	windows := &repositories.ListWorkerWindowsRequest{
		TenantInfo:  req.TenantInfo,
		WorkerIDs:   workerIDs,
		WindowStart: req.Filter.WindowStart,
		WindowEnd:   req.Filter.WindowEnd,
	}

	commitments, err := s.consoleRepo.ListWorkerCommitments(ctx, windows)
	if err != nil {
		return err
	}
	for _, commitment := range commitments {
		snapshot.CommitmentsByWorker[commitment.WorkerID] = append(
			snapshot.CommitmentsByWorker[commitment.WorkerID],
			commitment,
		)
	}

	timeOff, err := s.consoleRepo.ListWorkerTimeOff(ctx, windows)
	if err != nil {
		return err
	}
	for _, pto := range timeOff {
		snapshot.TimeOffByWorker[pto.WorkerID] = append(snapshot.TimeOffByWorker[pto.WorkerID], pto)
	}

	workload, err := s.consoleRepo.ListWorkerWorkload(ctx, &repositories.ListWorkloadRequest{
		TenantInfo: req.TenantInfo,
		WorkerIDs:  workerIDs,
		Since:      snapshot.Now - workloadLookbackSeconds,
	})
	if err != nil {
		return err
	}
	for _, item := range workload {
		snapshot.WorkloadByWorker[item.WorkerID] = item
	}

	if len(req.CustomerIDs) > 0 {
		experience, expErr := s.consoleRepo.ListWorkerLaneExperience(
			ctx,
			&repositories.ListLaneExperienceRequest{
				TenantInfo:  req.TenantInfo,
				WorkerIDs:   workerIDs,
				CustomerIDs: req.CustomerIDs,
				Since:       snapshot.Now - laneExperienceLookbackSeconds,
			},
		)
		if expErr != nil {
			return expErr
		}
		for _, item := range experience {
			snapshot.LaneExperience[laneKey{item.WorkerID, item.CustomerID}] = item.MoveCount
		}
	}

	return nil
}

type RankRequest struct {
	Move           *repositories.BoardMove
	Snapshot       *FleetSnapshot
	Limit          int
	IncludeBlocked bool
}

func (s *Service) RankCandidates(req *RankRequest) []*CandidateScore {
	if req.Move == nil || req.Snapshot == nil {
		return nil
	}

	weights := req.Snapshot.Control.ResolvedScoringWeights()
	requirements := requirementsFor(req.Move)

	results := make([]*CandidateScore, 0, len(req.Snapshot.Drivers))
	for _, driver := range req.Snapshot.Drivers {
		score := s.scoreDriver(&scoreDriverParams{
			Driver:       driver,
			Move:         req.Move,
			Requirements: requirements,
			Snapshot:     req.Snapshot,
			Weights:      weights,
		})
		if score == nil {
			continue
		}
		if score.Blocked() && !req.IncludeBlocked {
			continue
		}
		results = append(results, score)
	}

	sortCandidates(results)

	if req.Limit > 0 && len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results
}

func requirementsFor(move *repositories.BoardMove) dispatcheligibility.MoveRequirements {
	return dispatcheligibility.MoveRequirements{
		MoveID:                move.MoveID,
		ShipmentID:            move.ShipmentID,
		Window:                moveWindow(move),
		HasHazmatCommodities:  move.HasHazmat,
		HasActiveHold:         move.HasActiveHold,
		RequiredTractorTypeID: move.RequiredTractorTypeID,
		RequiredTrailerTypeID: move.RequiredTrailerTypeID,
	}
}

func moveWindow(move *repositories.BoardMove) dispatcheligibility.TimeWindow {
	return dispatcheligibility.TimeWindow{
		Start: move.OriginWindowStart,
		End:   MoveWindowEnd(move),
	}
}

func sortCandidates(results []*CandidateScore) {
	sort.SliceStable(results, func(i, j int) bool {
		if c := CompareRank(results[i], results[j]); c != 0 {
			return c < 0
		}
		return results[i].WorkerName < results[j].WorkerName
	})
}

type scoreDriverParams struct {
	Driver       *repositories.BoardDriver
	Move         *repositories.BoardMove
	Requirements dispatcheligibility.MoveRequirements
	Snapshot     *FleetSnapshot
	Weights      map[dispatchcontrol.ScoringFactor]float64
}

func (s *Service) scoreDriver(p *scoreDriverParams) *CandidateScore {
	w := p.Snapshot.WorkersByID[p.Driver.WorkerID]
	if w == nil {
		return nil
	}

	trip := s.computeTrip(p.Driver, p.Move, p.Snapshot)

	hosState := p.Snapshot.HOSByWorker[p.Driver.WorkerID]
	hosKnown := p.Snapshot.TelematicsActive && hosState != nil &&
		p.Snapshot.Now-hosState.RecordedAt <= dispatcheligibility.HOSStateMaxAgeSeconds
	projection := s.projectHOS(p.Driver, p.Snapshot, trip, hosKnown)

	trailerID := resolveTrailer(p.Move)
	candidate := dispatcheligibility.Candidate{
		Worker:           w,
		Tractor:          p.Snapshot.TractorByID[p.Driver.TractorID],
		Trailer:          p.Snapshot.TrailerByID[trailerID],
		HOSState:         hosState,
		HOSProjection:    projection,
		ApprovedPTO:      ptoWindows(p.Snapshot.TimeOffByWorker[p.Driver.WorkerID]),
		CommittedWindows: commitmentWindows(p.Snapshot.CommitmentsByWorker[p.Driver.WorkerID]),
	}

	eval := dispatcheligibility.Evaluate(dispatcheligibility.EvaluateInput{
		Candidate:        candidate,
		Requirements:     p.Requirements,
		Control:          p.Snapshot.Control,
		Now:              p.Snapshot.Now,
		TelematicsActive: p.Snapshot.TelematicsActive,
	})

	if finding := appointmentFinding(trip, p.Snapshot.Control); finding != nil {
		eval.Add(*finding)
	}

	result := &CandidateScore{
		WorkerID:   p.Driver.WorkerID,
		WorkerName: p.Snapshot.driverName(p.Driver),
		TractorID:  p.Driver.TractorID,
		TrailerID:  trailerID,
		MoveID:     p.Move.MoveID,
		Findings:   eval.Findings,
	}

	result.DeadheadMiles = trip.deadheadMiles
	result.EstimatedDriveMs = trip.driveMs
	result.ProjectedArrival = trip.projectedArrival
	result.MinutesOfSlack = int64(trip.slackMinutes)
	result.ProjectedAvailable = ProjectedTimeAvailable(
		p.Snapshot.CommitmentsByWorker[p.Driver.WorkerID],
		p.Snapshot.Now,
	)

	if hosState != nil {
		result.DriveRemainingMs = hosState.DriveRemainingMs
		result.ShiftRemainingMs = hosState.ShiftRemainingMs
		result.CycleRemainingMs = hosState.CycleRemainingMs
	}
	if projection != nil {
		result.HOSStrategy = string(projection.Strategy)
		result.HOSRestStartDeadline = projection.RestStartDeadline
		result.HOSProjectedDriveMs = projection.DriveAvailableMs
		result.HOSProjectedShiftMs = projection.ShiftAvailableMs
		result.HOSProjectedCycleMs = projection.CycleAvailableMs
	}

	result.Verdict = verdictFor(&verdictInput{
		Eval:         eval,
		SlackMinutes: trip.slackMinutes,
		HOSKnown:     hosKnown,
		HOSExpected:  p.Snapshot.TelematicsActive,
		Projection:   projection,
	})
	result.Score, result.Factors = buildFactors(
		factorInputFor(&factorContext{
			driver:     p.Driver,
			move:       p.Move,
			snapshot:   p.Snapshot,
			trip:       trip,
			hosState:   hosState,
			hosKnown:   hosKnown,
			projection: projection,
		}),
		p.Weights,
	)

	return result
}

// projectHOS runs the forward hours-of-service projection for one candidate. It
// returns nil when the projection cannot be trusted: no telematics integration, no or
// stale clock data, or an ELD-exempt driver whose hours are not tracked.
func (s *Service) projectHOS(
	driver *repositories.BoardDriver,
	snapshot *FleetSnapshot,
	trip tripEstimate,
	hosKnown bool,
) *hosprojection.Result {
	if !hosKnown {
		return nil
	}
	state := snapshot.HOSByWorker[driver.WorkerID]
	if state == nil {
		return nil
	}
	if w := snapshot.WorkersByID[driver.WorkerID]; w != nil && w.Profile != nil &&
		(w.Profile.ELDExempt || w.Profile.ShortHaulExempt) {
		return nil
	}

	limits := hosprojection.LimitsForRuleset(
		state.RulesetCycle,
		state.RulesetShift,
		state.RulesetJurisdiction,
	)
	result := hosprojection.Project(hosprojection.Input{
		Now:         snapshot.Now,
		Departure:   trip.departure,
		TripDriveMs: trip.driveMs,
		Clocks: hosprojection.Clocks{
			DriveMs: state.DriveRemainingMs,
			ShiftMs: state.ShiftRemainingMs,
			CycleMs: state.CycleRemainingMs,
			BreakMs: state.BreakRemainingMs,
		},
		ClocksAt:        state.RecordedAt,
		CycleTomorrowMs: state.CycleTomorrowMs,
		DutyStatus:      state.DutyStatus,
		Limits:          limits,
		Jurisdiction:    state.RulesetJurisdiction,
		Timeline: hosprojection.BuildTimeline(
			snapshot.HOSLogsByWorker[driver.WorkerID],
			snapshot.Now,
		),
	})
	return &result
}

// appointmentFinding gives EnforceWorkerPTARestrictions its teeth: when a driver's
// projected availability puts them past the pickup window, the flag decides whether
// that hard-blocks the candidate or merely warns the dispatcher.
func appointmentFinding(
	trip tripEstimate,
	control *dispatchcontrol.DispatchControl,
) *dispatcheligibility.Finding {
	if !trip.appointmentKnown || trip.slackMinutes >= 0 {
		return nil
	}

	lateBy := timeutils.FormatLongDurationMs(int64(-trip.slackMinutes * 60_000))
	if control != nil && control.EnforceWorkerPTARestrictions {
		return &dispatcheligibility.Finding{
			Code:     dispatcheligibility.CodeAppointmentMissed,
			Severity: dispatcheligibility.SeverityBlock,
			Field:    "assignment",
			Message: fmt.Sprintf(
				"Driver's projected availability is %s past the pickup window",
				lateBy,
			),
		}
	}
	return &dispatcheligibility.Finding{
		Code:     dispatcheligibility.CodeAppointmentAtRisk,
		Severity: dispatcheligibility.SeverityWarn,
		Field:    "assignment",
		Message: fmt.Sprintf(
			"Driver is projected %s late to the pickup window",
			lateBy,
		),
	}
}

type tripEstimate struct {
	deadheadMiles     *float64
	loadedMiles       float64
	totalMiles        float64
	driveMs           int64
	departure         int64
	projectedArrival  int64
	projectedComplete int64
	slackMinutes      float64
	appointmentKnown  bool
}

func (s *Service) computeTrip(
	driver *repositories.BoardDriver,
	move *repositories.BoardMove,
	snapshot *FleetSnapshot,
) tripEstimate {
	estimate := tripEstimate{}

	if move.Distance != nil {
		estimate.loadedMiles = *move.Distance
	}
	estimate.totalMiles = estimate.loadedMiles

	if origin, ok := deadheadOrigin(driver, snapshot); ok &&
		move.OriginLatitude != nil && move.OriginLongitude != nil {
		deadhead := geoutils.HaversineMiles(
			origin.lat,
			origin.lon,
			*move.OriginLatitude,
			*move.OriginLongitude,
		) * roadCircuityFactor
		estimate.deadheadMiles = &deadhead
		estimate.totalMiles += deadhead
	}

	if estimate.totalMiles > 0 {
		estimate.driveMs = int64(estimate.totalMiles / averageLinehaulMph * 3_600_000)
	}

	deadheadMs := int64(0)
	if estimate.deadheadMiles != nil {
		deadheadMs = int64(*estimate.deadheadMiles / averageLinehaulMph * 3_600_000)
	}

	estimate.departure = max(
		snapshot.Now,
		ProjectedTimeAvailable(snapshot.CommitmentsByWorker[driver.WorkerID], snapshot.Now),
	)
	estimate.projectedArrival = estimate.departure + deadheadMs/1000
	estimate.projectedComplete = estimate.departure + estimate.driveMs/1000

	appointment := move.OriginWindowStart
	if move.OriginWindowEnd != nil && *move.OriginWindowEnd > appointment {
		appointment = *move.OriginWindowEnd
	}
	if appointment > 0 {
		estimate.appointmentKnown = true
		estimate.slackMinutes = float64(appointment-estimate.projectedArrival) / 60
	}

	return estimate
}

type geoPoint struct {
	lat float64
	lon float64
}

// deadheadOrigin picks where the empty leg to the pickup really starts. A driver who
// is mid-run will depart from their final commitment's destination, not from wherever
// their tractor happens to be pinging right now; only idle drivers use the live
// position. With neither, deadhead is unknown and the factor is omitted.
func deadheadOrigin(
	driver *repositories.BoardDriver,
	snapshot *FleetSnapshot,
) (geoPoint, bool) {
	commitments := snapshot.CommitmentsByWorker[driver.WorkerID]
	if ProjectedTimeAvailable(commitments, snapshot.Now) > snapshot.Now {
		if destination, ok := finalCommitmentDestination(commitments); ok {
			return destination, true
		}
	}

	if position := snapshot.PosByTractor[driver.TractorID]; position != nil {
		return geoPoint{lat: position.Latitude, lon: position.Longitude}, true
	}
	return geoPoint{}, false
}

func finalCommitmentDestination(commitments []*repositories.WorkerCommitment) (geoPoint, bool) {
	var latest *repositories.WorkerCommitment
	for _, commitment := range commitments {
		if latest == nil || commitment.WindowEnd > latest.WindowEnd {
			latest = commitment
		}
	}
	if latest == nil || latest.DestLatitude == nil || latest.DestLongitude == nil {
		return geoPoint{}, false
	}
	return geoPoint{lat: *latest.DestLatitude, lon: *latest.DestLongitude}, true
}

type factorContext struct {
	driver     *repositories.BoardDriver
	move       *repositories.BoardMove
	snapshot   *FleetSnapshot
	trip       tripEstimate
	hosState   *telematics.WorkerHOSState
	hosKnown   bool
	projection *hosprojection.Result
}

func factorInputFor(fc *factorContext) *factorInput {
	in := &factorInput{
		deadheadMiles: fc.trip.deadheadMiles,
		slackMinutes:  fc.trip.slackMinutes,
		slackKnown:    fc.trip.appointmentKnown,
		customerName:  fc.move.CustomerName,
	}

	applyHOSInput(in, fc)

	if needed := resolveTrailer(fc.move); !needed.IsNil() {
		if current, ok := currentTrailer(fc.snapshot.CommitmentsByWorker[fc.driver.WorkerID]); ok {
			in.trailerKnown = true
			in.trailerContinues = current == needed
		}
	}

	if !fc.driver.FleetCodeID.IsNil() && !fc.driver.TractorFleetID.IsNil() {
		in.fleetKnown = true
		in.fleetMatches = fc.driver.FleetCodeID == fc.driver.TractorFleetID
	}

	in.driverTypeFit, in.driverTypeLabel = driverTypeFit(fc.driver.DriverType, fc.trip.totalMiles)

	if fc.snapshot.WorkloadByWorker != nil {
		in.workloadKnown = true
		if workload := fc.snapshot.WorkloadByWorker[fc.driver.WorkerID]; workload != nil {
			in.committedMiles = workload.TotalMiles
			in.moveCount = float64(workload.MoveCount)
			if workload.LastEndedAt > 0 {
				in.daysOutKnown = true
				in.daysOut = float64(fc.snapshot.Now-workload.LastEndedAt) / 86400
			}
		}
	}
	in.openAssignments = float64(fc.driver.OpenAssignments)

	in.laneMoves = float64(
		fc.snapshot.LaneExperience[laneKey{fc.driver.WorkerID, fc.move.CustomerID}],
	)

	applyPerformanceInput(in, fc)

	return in
}

// applyHOSInput feeds the hours factor. With a fresh state and a projection, the raw
// input is the simulated post-trip margin under the driver's best rest strategy; with
// a fresh state but no projection it falls back to the instantaneous clock margin.
// Stale states and organizations without telematics contribute nothing.
func applyHOSInput(in *factorInput, fc *factorContext) {
	if !fc.hosKnown || fc.hosState == nil {
		return
	}

	in.hosKnown = true
	if fc.projection != nil {
		if fc.projection.Feasible {
			in.hosMarginMs = float64(fc.projection.Trip.MarginMs)
		} else {
			in.hosMarginMs = 0
		}
		in.hosDetail = fc.projection.Detail
		return
	}
	in.hosMarginMs = float64(min(
		fc.hosState.DriveRemainingMs,
		fc.hosState.ShiftRemainingMs,
	) - fc.trip.driveMs)
}

func applyPerformanceInput(in *factorInput, fc *factorContext) {
	if stats := fc.snapshot.OnTimeByWorker[fc.driver.WorkerID]; stats != nil &&
		stats.CompletedStops >= onTimeHistoryMinStops {
		in.onTimeKnown = true
		in.onTimePct = 100 * float64(stats.OnTimeStops) / float64(stats.CompletedStops)
		in.onTimeStops = stats.CompletedStops
		in.driverFaultFailures = stats.DriverFaultFailures
		in.onTimeTargetPct = defaultOnTimeTargetPct
		if fc.snapshot.Control != nil && fc.snapshot.Control.ServiceFailureTarget != nil &&
			*fc.snapshot.Control.ServiceFailureTarget > 0 {
			in.onTimeTargetPct = *fc.snapshot.Control.ServiceFailureTarget
		}
	}

	if fc.snapshot.TelematicsActive {
		in.safetyKnown = true
		if stats := fc.snapshot.ViolationsByWorker[fc.driver.WorkerID]; stats != nil {
			in.violationCount = float64(stats.ViolationCount)
		}
	}

	if stats := fc.snapshot.AcceptanceByWorker[fc.driver.WorkerID]; stats != nil {
		decisions := stats.Accepted + stats.Declined
		if decisions >= acceptanceMinDecisions {
			in.acceptanceKnown = true
			in.acceptRate = float64(stats.Accepted) / float64(decisions)
			in.ackDecisions = decisions
			in.ackLatencyMinutes = stats.AvgAckLatencySeconds / 60
		}
	}

	if nextPTO := nextApprovedPTOStart(
		fc.snapshot.TimeOffByWorker[fc.driver.WorkerID],
		fc.snapshot.Now,
	); nextPTO > 0 {
		in.ptoKnown = true
		in.ptoGapHours = float64(nextPTO-projectedTripCompletion(fc)) / 3600
	}
}

// projectedTripCompletion is when the driver would actually be done with the trip:
// the simulated plan when a projection exists (it includes breaks and resets), the
// flat drive-time estimate otherwise.
func projectedTripCompletion(fc *factorContext) int64 {
	if fc.projection != nil && fc.projection.Feasible && fc.projection.Trip.Arrival > 0 {
		return fc.projection.Trip.Arrival
	}
	return fc.trip.projectedComplete
}

// nextApprovedPTOStart finds the next approved time-off window starting within the
// lookahead. Windows already underway are excluded — those hard-block availability.
func nextApprovedPTOStart(timeOff []*repositories.WorkerTimeOff, now int64) int64 {
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

func currentTrailer(commitments []*repositories.WorkerCommitment) (pulid.ID, bool) {
	var (
		latest    int64
		trailerID pulid.ID
		found     bool
	)
	for _, commitment := range commitments {
		if commitment.TrailerID.IsNil() {
			continue
		}
		if !found || commitment.WindowEnd > latest {
			latest = commitment.WindowEnd
			trailerID = commitment.TrailerID
			found = true
		}
	}
	return trailerID, found
}

func resolveTrailer(move *repositories.BoardMove) pulid.ID {
	if !move.AssignedTrailerID.IsNil() {
		return move.AssignedTrailerID
	}
	return move.PreviousMoveTrailerID
}

func ptoWindows(items []*repositories.WorkerTimeOff) []dispatcheligibility.PTOWindow {
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

func commitmentWindows(items []*repositories.WorkerCommitment) []dispatcheligibility.TimeWindow {
	if len(items) == 0 {
		return nil
	}
	windows := make([]dispatcheligibility.TimeWindow, 0, len(items))
	for _, item := range items {
		windows = append(windows, dispatcheligibility.TimeWindow{
			Start: item.WindowStart,
			End:   item.WindowEnd,
		})
	}
	return windows
}
