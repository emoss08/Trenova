package dispatchcandidateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	positionMaxAgeSeconds = int64(4 * 3600)

	workloadLookbackSeconds = int64(7 * 24 * 3600)
	homeLookbackSeconds     = int64(60 * 24 * 3600)

	laneExperienceLookbackSeconds = int64(365 * 24 * 3600)

	onTimeHistoryLookbackSeconds = int64(90 * 24 * 3600)

	acceptanceLookbackSeconds = int64(180 * 24 * 3600)

	safetyLookbackSeconds = int64(90 * 24 * 3600)

	hosLogLookbackSeconds = int64(9 * 24 * 3600)

	ptoLookaheadSeconds = int64(14 * 24 * 3600)
)

type Params struct {
	fx.In

	ConsoleRepo     dispatchboardports.DispatchConsoleRepository
	TelematicsRepo  telematicsports.TelematicsRepository
	IntegrationRepo integrationports.IntegrationRepository
	TractorRepo     tractorports.TractorRepository
	RouteEstimator  routeestimateports.RouteEstimator
	Style           dispatchlearningports.MultiplierSource
	Market          dispatchmarketports.ValuerSource
	Logger          *zap.Logger
}

type Service struct {
	l               *zap.Logger
	consoleRepo     dispatchboardports.DispatchConsoleRepository
	telematicsRepo  telematicsports.TelematicsRepository
	integrationRepo integrationports.IntegrationRepository
	tractorRepo     tractorports.TractorRepository
	routes          routeestimateports.RouteEstimator
	style           dispatchlearningports.MultiplierSource
	market          dispatchmarketports.ValuerSource
}

func New(p Params) *Service {
	return &Service{
		l:               p.Logger.Named("service.dispatch-candidates"),
		consoleRepo:     p.ConsoleRepo,
		telematicsRepo:  p.TelematicsRepo,
		integrationRepo: p.IntegrationRepo,
		tractorRepo:     p.TractorRepo,
		routes:          p.RouteEstimator,
		style:           p.Style,
		market:          p.Market,
	}
}

type noEndMarket struct{}

func (noEndMarket) EndMarketCents(float64, float64, int64) int64 { return 0 }

func (s *Service) telematicsActive(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	records, err := s.integrationRepo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	return integration.HasEnabledTelematics(records), nil
}

type SnapshotRequest struct {
	TenantInfo   pagination.TenantInfo
	Filter       *dispatchboardports.DispatchBoardFilter
	Control      *dispatchcontrol.DispatchControl
	CustomerIDs  []pulid.ID
	TrailerIDs   []pulid.ID
	Destinations []dispatchmarket.Point
	Now          int64
}

func (s *Service) BuildSnapshot(
	ctx context.Context,
	req *SnapshotRequest,
) (*FleetSnapshot, error) {
	page, err := s.consoleRepo.ListBoardDrivers(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	drivers := page.Drivers

	now := req.Now
	if now <= 0 {
		now = timeutils.NowUnix()
	}
	model, endMarket, err := s.costInputs(ctx, req)
	if err != nil {
		return nil, err
	}
	snapshot := &FleetSnapshot{
		Drivers:          drivers,
		DriversTruncated: page.Truncated,
		WorkersByID:      make(map[pulid.ID]*worker.Worker, len(drivers)),
		TractorByID:      make(map[pulid.ID]*tractor.Tractor, len(drivers)),
		TrailerByID:      make(map[pulid.ID]*trailer.Trailer, len(drivers)),
		HOSByWorker:      make(map[pulid.ID]*telematics.WorkerHOSState, len(drivers)),
		HOSLogsByWorker:  make(map[pulid.ID][]*telematics.WorkerHOSLog, len(drivers)),
		PosByTractor:     make(map[pulid.ID]*telematics.VehiclePosition, len(drivers)),
		CommitmentsByWorker: make(
			map[pulid.ID][]*dispatchboardports.WorkerCommitment,
			len(drivers),
		),
		TimeOffByWorker:  make(map[pulid.ID][]*dispatchboardports.WorkerTimeOff, len(drivers)),
		WorkloadByWorker: make(map[pulid.ID]*dispatchboardports.WorkerWorkload, len(drivers)),
		OnTimeByWorker:   make(map[pulid.ID]*dispatchboardports.WorkerOnTimeStats, len(drivers)),
		AcceptanceByWorker: make(
			map[pulid.ID]*dispatchboardports.WorkerAcceptanceStats,
			len(drivers),
		),
		ViolationsByWorker: make(
			map[pulid.ID]*telematicsports.WorkerHOSViolationStats,
			len(drivers),
		),
		NamesByWorker:  make(map[pulid.ID]string, len(drivers)),
		LaneExperience: make(map[laneKey]int, len(drivers)),
		Control:        req.Control,
		Now:            now,
		Model:          model,
		EndMarket:      endMarket,
	}

	active, err := s.telematicsActive(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	snapshot.TelematicsActive = active

	if len(drivers) == 0 {
		snapshot.Prepare()
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

	snapshot.Prepare()

	return snapshot, nil
}

func (s *Service) loadPerformance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snapshot *FleetSnapshot,
	workerIDs []pulid.ID,
) error {
	onTime, err := s.consoleRepo.ListWorkerOnTimeStats(ctx, &dispatchboardports.ListWorkloadRequest{
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
		&dispatchboardports.ListWorkloadRequest{
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

	tractors, err := s.tractorRepo.GetByIDs(ctx, tractorports.GetTractorsByIDsRequest{
		TenantInfo: req.TenantInfo,
		TractorIDs: tractorIDs,
		TractorRelationIncludes: tractorports.TractorRelationIncludes{
			IncludeLastKnownLocation: true,
		},
	})
	if err != nil {
		return err
	}
	for _, t := range tractors {
		snapshot.TractorByID[t.ID] = t
	}

	_, trailers, err := s.consoleRepo.ListEquipmentByIDs(
		ctx,
		&dispatchboardports.ListEquipmentByIDsRequest{
			TenantInfo: req.TenantInfo,
			TrailerIDs: snapshotTrailerIDs(req, snapshot),
		},
	)
	if err != nil {
		return err
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
	if !snapshot.TelematicsActive {
		return nil
	}

	states, err := s.telematicsRepo.ListWorkerHOSStates(
		ctx,
		&telematicsports.ListWorkerHOSStatesRequest{
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

	logs, err := s.telematicsRepo.ListWorkerHOSLogs(ctx, &telematicsports.ListWorkerHOSLogsRequest{
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
		&telematicsports.ListWorkerHOSViolationStatsRequest{
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
		&telematicsports.ListVehiclePositionsRequest{
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
	windows := &dispatchboardports.ListWorkerWindowsRequest{
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

	pastTimeOff, err := s.consoleRepo.ListWorkerTimeOff(
		ctx,
		&dispatchboardports.ListWorkerWindowsRequest{
			TenantInfo:  req.TenantInfo,
			WorkerIDs:   workerIDs,
			WindowStart: snapshot.Now - homeLookbackSeconds,
			WindowEnd:   snapshot.Now,
		},
	)
	if err != nil {
		return err
	}
	snapshot.LastTimeOffEnd = lastTimeOffEnds(pastTimeOff, snapshot.Now)

	workload, err := s.consoleRepo.ListWorkerWorkload(ctx, &dispatchboardports.ListWorkloadRequest{
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
			&dispatchboardports.ListLaneExperienceRequest{
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
