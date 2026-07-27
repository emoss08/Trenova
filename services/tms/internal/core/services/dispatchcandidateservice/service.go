package dispatchcandidateservice

import (
	"context"
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// averageLinehaulMph mirrors the constant the shipment-level feasibility check uses,
	// so a driver judged feasible there is judged feasible here.
	averageLinehaulMph = 50.0

	// positionMaxAgeSeconds is how stale a GPS fix may be before deadhead is treated as
	// unknown rather than guessed from a position the truck has long since left.
	positionMaxAgeSeconds = int64(4 * 3600)

	// workloadLookbackSeconds is the rolling window behind the load-balancing factor.
	workloadLookbackSeconds = int64(7 * 24 * 3600)

	// laneExperienceLookbackSeconds bounds how far back customer familiarity counts.
	laneExperienceLookbackSeconds = int64(365 * 24 * 3600)
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	ConsoleRepo    repositories.DispatchConsoleRepository
	TelematicsRepo repositories.TelematicsRepository
}

type Service struct {
	l              *zap.Logger
	consoleRepo    repositories.DispatchConsoleRepository
	telematicsRepo repositories.TelematicsRepository
}

func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.dispatch-candidate"),
		consoleRepo:    p.ConsoleRepo,
		telematicsRepo: p.TelematicsRepo,
	}
}

// FleetSnapshot is everything about a driver pool that does not change from move to
// move. Loading it once and scoring every move against it is what keeps the console a
// handful of queries instead of one per candidate pairing.
type FleetSnapshot struct {
	Drivers     []*repositories.BoardDriver
	WorkersByID map[pulid.ID]*worker.Worker
	TractorByID map[pulid.ID]*tractor.Tractor
	TrailerByID map[pulid.ID]*trailer.Trailer
	HOSByWorker map[pulid.ID]*telematics.WorkerHOSState
	PosByTractor map[pulid.ID]*telematics.VehiclePosition
	CommitmentsByWorker map[pulid.ID][]*repositories.WorkerCommitment
	TimeOffByWorker     map[pulid.ID][]*repositories.WorkerTimeOff
	WorkloadByWorker    map[pulid.ID]*repositories.WorkerWorkload
	LaneExperience      map[laneKey]int
	Control             *dispatchcontrol.DispatchControl
	Now                 int64
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
}

// BuildSnapshot loads the fleet-wide context in a fixed number of batch queries,
// regardless of how many drivers or moves the board holds.
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
		TrailerByID:         make(map[pulid.ID]*trailer.Trailer),
		HOSByWorker:         make(map[pulid.ID]*telematics.WorkerHOSState, len(drivers)),
		PosByTractor:        make(map[pulid.ID]*telematics.VehiclePosition, len(drivers)),
		CommitmentsByWorker: make(map[pulid.ID][]*repositories.WorkerCommitment, len(drivers)),
		TimeOffByWorker:     make(map[pulid.ID][]*repositories.WorkerTimeOff),
		WorkloadByWorker:    make(map[pulid.ID]*repositories.WorkerWorkload, len(drivers)),
		LaneExperience:      make(map[laneKey]int),
		Control:             req.Control,
		Now:                 now,
	}

	if len(drivers) == 0 {
		return snapshot, nil
	}

	workerIDs := make([]pulid.ID, 0, len(drivers))
	tractorIDs := make([]pulid.ID, 0, len(drivers))
	for _, driver := range drivers {
		workerIDs = append(workerIDs, driver.WorkerID)
		if !driver.TractorID.IsNil() {
			tractorIDs = append(tractorIDs, driver.TractorID)
		}
	}

	if err = s.loadWorkersAndEquipment(ctx, req, snapshot, workerIDs, tractorIDs); err != nil {
		return nil, err
	}
	if err = s.loadTelematics(ctx, req.TenantInfo, snapshot, workerIDs, tractorIDs); err != nil {
		return nil, err
	}
	if err = s.loadSchedules(ctx, req, snapshot, workerIDs); err != nil {
		return nil, err
	}

	return snapshot, nil
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

func (s *Service) loadTelematics(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snapshot *FleetSnapshot,
	workerIDs, tractorIDs []pulid.ID,
) error {
	if s.telematicsRepo == nil {
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

// RankRequest asks for the candidates for a single move, ordered best first.
type RankRequest struct {
	Move     *repositories.BoardMove
	Snapshot *FleetSnapshot
	Limit    int
	// IncludeBlocked keeps disqualified drivers in the result so the console can explain
	// why an expected driver is missing rather than silently omitting them.
	IncludeBlocked bool
}

// RankCandidates scores every driver in the snapshot against one move and returns them
// ordered best first. Blocked candidates always sort last, whatever their score.
func (s *Service) RankCandidates(req *RankRequest) []*CandidateScore {
	if req.Move == nil || req.Snapshot == nil {
		return nil
	}

	weights := req.Snapshot.Control.ResolvedScoringWeights()
	requirements := requirementsFor(req.Move)

	results := make([]*CandidateScore, 0, len(req.Snapshot.Drivers))
	for _, driver := range req.Snapshot.Drivers {
		score := s.scoreDriver(driver, req.Move, requirements, req.Snapshot, weights)
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
		RequiredTractorTypeID: move.RequiredTractorTypeID,
		RequiredTrailerTypeID: move.RequiredTrailerTypeID,
	}
}

func moveWindow(move *repositories.BoardMove) dispatcheligibility.TimeWindow {
	end := move.DestinationWindowStart
	if move.DestinationWindowEnd != nil && *move.DestinationWindowEnd > end {
		end = *move.DestinationWindowEnd
	}
	return dispatcheligibility.TimeWindow{Start: move.OriginWindowStart, End: end}
}

func sortCandidates(results []*CandidateScore) {
	sort.SliceStable(results, func(i, j int) bool {
		leftBlocked, rightBlocked := results[i].Blocked(), results[j].Blocked()
		if leftBlocked != rightBlocked {
			return rightBlocked
		}
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].WorkerName < results[j].WorkerName
	})
}

func (s *Service) scoreDriver(
	driver *repositories.BoardDriver,
	move *repositories.BoardMove,
	requirements dispatcheligibility.MoveRequirements,
	snapshot *FleetSnapshot,
	weights map[dispatchcontrol.ScoringFactor]float64,
) *CandidateScore {
	w := snapshot.WorkersByID[driver.WorkerID]
	if w == nil {
		return nil
	}

	trailerID := resolveTrailer(move)
	candidate := dispatcheligibility.Candidate{
		Worker:           w,
		Tractor:          snapshot.TractorByID[driver.TractorID],
		Trailer:          snapshot.TrailerByID[trailerID],
		HOSState:         snapshot.HOSByWorker[driver.WorkerID],
		ApprovedPTO:      ptoWindows(snapshot.TimeOffByWorker[driver.WorkerID]),
		CommittedWindows: commitmentWindows(snapshot.CommitmentsByWorker[driver.WorkerID]),
	}

	eval := dispatcheligibility.Evaluate(dispatcheligibility.EvaluateInput{
		Candidate:    candidate,
		Requirements: requirements,
		Control:      snapshot.Control,
		Now:          snapshot.Now,
	})

	result := &CandidateScore{
		WorkerID:   driver.WorkerID,
		WorkerName: fmt.Sprintf("%s %s", driver.FirstName, driver.LastName),
		TractorID:  driver.TractorID,
		TrailerID:  trailerID,
		MoveID:     move.MoveID,
		Findings:   eval.Findings,
	}

	trip := s.computeTrip(driver, move, snapshot)
	result.DeadheadMiles = trip.deadheadMiles
	result.EstimatedDriveMs = trip.driveMs
	result.ProjectedArrival = trip.projectedArrival
	result.MinutesOfSlack = int64(trip.slackMinutes)
	result.ProjectedAvailable = projectedTimeAvailable(
		snapshot.CommitmentsByWorker[driver.WorkerID],
		snapshot.Now,
	)

	hosState := snapshot.HOSByWorker[driver.WorkerID]
	hosKnown := hosState != nil &&
		snapshot.Now-hosState.RecordedAt <= dispatcheligibility.HOSStateMaxAgeSeconds
	if hosState != nil {
		result.DriveRemainingMs = hosState.DriveRemainingMs
		result.ShiftRemainingMs = hosState.ShiftRemainingMs
		result.CycleRemainingMs = hosState.CycleRemainingMs
	}

	result.Verdict = verdictFor(eval, trip.slackMinutes, hosKnown)
	result.Score, result.Factors = buildFactors(
		s.factorInputFor(driver, move, snapshot, trip, hosState),
		weights,
	)

	return result
}

type tripEstimate struct {
	deadheadMiles    *float64
	loadedMiles      float64
	totalMiles       float64
	driveMs          int64
	projectedArrival int64
	slackMinutes     float64
}

// computeTrip estimates empty miles from the tractor's last known position to the
// pickup, then projects arrival against the appointment window. Deadhead stays nil when
// there is no fresh position, so the score omits the factor rather than inventing one.
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

	position := snapshot.PosByTractor[driver.TractorID]
	if position != nil && move.OriginLatitude != nil && move.OriginLongitude != nil {
		deadhead := geoutils.HaversineMiles(
			position.Latitude,
			position.Longitude,
			*move.OriginLatitude,
			*move.OriginLongitude,
		)
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
	estimate.projectedArrival = snapshot.Now + deadheadMs/1000

	appointment := move.OriginWindowStart
	if move.OriginWindowEnd != nil && *move.OriginWindowEnd > appointment {
		appointment = *move.OriginWindowEnd
	}
	if appointment > 0 {
		estimate.slackMinutes = float64(appointment-estimate.projectedArrival) / 60
	}

	return estimate
}

func (s *Service) factorInputFor(
	driver *repositories.BoardDriver,
	move *repositories.BoardMove,
	snapshot *FleetSnapshot,
	trip tripEstimate,
	hosState *telematics.WorkerHOSState,
) factorInput {
	in := factorInput{
		deadheadMiles: trip.deadheadMiles,
		slackMinutes:  trip.slackMinutes,
		customerName:  move.CustomerName,
	}

	if hosState != nil {
		in.hosMarginMs = float64(minInt64(
			hosState.DriveRemainingMs,
			hosState.ShiftRemainingMs,
		) - trip.driveMs)
	}

	// Continuity is a property of the driver, not of the move: it holds when the trailer
	// this driver is already pulling is the one the move needs, so no swap is required.
	if needed := resolveTrailer(move); !needed.IsNil() {
		if current, ok := currentTrailer(snapshot.CommitmentsByWorker[driver.WorkerID]); ok {
			in.trailerKnown = true
			in.trailerContinues = current == needed
		}
	}

	if !driver.FleetCodeID.IsNil() && !driver.TractorFleetID.IsNil() {
		in.fleetKnown = true
		in.fleetMatches = driver.FleetCodeID == driver.TractorFleetID
	}

	in.driverTypeFit, in.driverTypeLabel = driverTypeFit(driver.DriverType, trip.totalMiles)

	if workload := snapshot.WorkloadByWorker[driver.WorkerID]; workload != nil {
		in.committedMiles = workload.TotalMiles
		if workload.LastEndedAt > 0 {
			in.daysOutKnown = true
			in.daysOut = float64(snapshot.Now-workload.LastEndedAt) / 86400
		}
	}

	in.laneMoves = float64(snapshot.LaneExperience[laneKey{driver.WorkerID, move.CustomerID}])

	return in
}

// projectedTimeAvailable is the moment a driver frees up: the end of their last
// committed move, or now when they hold nothing. This is the PTA dispatchers plan
// around and what makes the PTA restriction setting meaningful.
func projectedTimeAvailable(commitments []*repositories.WorkerCommitment, now int64) int64 {
	latest := now
	for _, commitment := range commitments {
		if commitment.WindowEnd > latest {
			latest = commitment.WindowEnd
		}
	}
	return latest
}

// currentTrailer is the trailer behind the driver's latest committed move, which is what
// they will physically still be pulling when they pick the next load up.
func currentTrailer(commitments []*repositories.WorkerCommitment) (pulid.ID, bool) {
	var (
		latest    int64
		trailerID pulid.ID
		found     bool
	)
	for _, commitment := range commitments {
		if commitment.TrailerID.IsNil() || commitment.WindowEnd < latest {
			continue
		}
		latest = commitment.WindowEnd
		trailerID = commitment.TrailerID
		found = true
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
			Window: dispatcheligibility.TimeWindow{Start: item.StartDate, End: item.EndDate},
			Type:   item.Type,
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

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
