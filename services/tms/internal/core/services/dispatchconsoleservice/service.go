package dispatchconsoleservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	secondsPerHour = int64(3600)
	secondsPerDay  = int64(86400)

	// nowThresholdSeconds is how imminent a pickup has to be to demand attention now.
	nowThresholdSeconds = int64(4 * 3600)

	// defaultCandidateLimit keeps the inspector to a list a dispatcher can actually read.
	defaultCandidateLimit = 25
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	ConsoleRepo         repositories.DispatchConsoleRepository
	DispatchControlRepo repositories.DispatchControlRepository
	CandidateService    *dispatchcandidateservice.Service
}

type Service struct {
	l                   *zap.Logger
	consoleRepo         repositories.DispatchConsoleRepository
	dispatchControlRepo repositories.DispatchControlRepository
	candidates          *dispatchcandidateservice.Service
}

func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.dispatch-console"),
		consoleRepo:         p.ConsoleRepo,
		dispatchControlRepo: p.DispatchControlRepo,
		candidates:          p.CandidateService,
	}
}

// GetBoard assembles one console render. The number of queries is fixed: it does not
// grow with the size of the fleet or the board.
func (s *Service) GetBoard(ctx context.Context, req *GetBoardRequest) (*Board, error) {
	control, err := s.dispatchControlRepo.GetOrCreate(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	windowStart, windowEnd := resolveWindow(req.WindowStart, req.WindowEnd, control, now)

	filter := &repositories.DispatchBoardFilter{
		TenantInfo:     req.TenantInfo,
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		FleetCodeIDs:   req.FleetCodeIDs,
		CustomerIDs:    req.CustomerIDs,
		ServiceTypeIDs: req.ServiceTypeIDs,
		WorkerIDs:      req.WorkerIDs,
		Query:          req.Query,
		IncludeCovered: req.IncludeCovered,
		Limit:          req.Limit,
	}

	moves, err := s.consoleRepo.ListBoardMoves(ctx, filter)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo:  req.TenantInfo,
		Filter:      filter,
		Control:     control,
		CustomerIDs: customerIDsOf(moves),
	})
	if err != nil {
		return nil, err
	}

	summary, err := s.consoleRepo.GetBoardSummary(ctx, filter)
	if err != nil {
		return nil, err
	}

	board := &Board{
		Moves:       decorateMoves(moves, now),
		Drivers:     decorateDrivers(snapshot, control, now),
		Summary:     summary,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		GeneratedAt: now,
	}
	board.Summary.AverageDeadhead = averageDeadhead(board.Moves, snapshot)

	return board, nil
}

// GetMoveCandidates ranks the fleet against one move for the inspector panel.
func (s *Service) GetMoveCandidates(
	ctx context.Context,
	req *MoveCandidatesRequest,
) ([]*dispatchcandidateservice.CandidateScore, error) {
	control, err := s.dispatchControlRepo.GetOrCreate(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
	)
	if err != nil {
		return nil, err
	}

	move, err := s.loadMove(ctx, req.TenantInfo, req.MoveID)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.buildSnapshotForMoves(
		ctx,
		req.TenantInfo,
		control,
		req.FleetCodeIDs,
		[]*repositories.BoardMove{move},
	)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultCandidateLimit
	}

	return s.candidates.RankCandidates(&dispatchcandidateservice.RankRequest{
		Move:           move,
		Snapshot:       snapshot,
		Limit:          limit,
		IncludeBlocked: req.IncludeBlocked,
	}), nil
}

// GetDriverMoves is the reverse match: which open moves suit one driver. Dispatchers work
// in both directions — covering a load, and finding work for an idle truck.
func (s *Service) GetDriverMoves(
	ctx context.Context,
	req *DriverMovesRequest,
) ([]*DriverMoveMatch, error) {
	control, err := s.dispatchControlRepo.GetOrCreate(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	windowStart, windowEnd := resolveWindow(req.WindowStart, req.WindowEnd, control, now)

	filter := &repositories.DispatchBoardFilter{
		TenantInfo:  req.TenantInfo,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		WorkerIDs:   []pulid.ID{req.WorkerID},
	}

	moves, err := s.consoleRepo.ListBoardMoves(ctx, &repositories.DispatchBoardFilter{
		TenantInfo:  req.TenantInfo,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	})
	if err != nil {
		return nil, err
	}

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo:  req.TenantInfo,
		Filter:      filter,
		Control:     control,
		CustomerIDs: customerIDsOf(moves),
	})
	if err != nil {
		return nil, err
	}

	matches := make([]*DriverMoveMatch, 0, len(moves))
	for _, move := range moves {
		scores := s.candidates.RankCandidates(&dispatchcandidateservice.RankRequest{
			Move:           move,
			Snapshot:       snapshot,
			IncludeBlocked: true,
		})
		if len(scores) == 0 {
			continue
		}
		matches = append(matches, &DriverMoveMatch{
			Move:  decorateMove(move, now),
			Score: scores[0],
		})
	}

	sortMatches(matches)
	if req.Limit > 0 && len(matches) > req.Limit {
		matches = matches[:req.Limit]
	}

	return matches, nil
}

// PreviewAssignment scores one explicit pairing without writing anything. This backs the
// drag-drop pre-flight, so a dispatcher sees the consequences before committing.
func (s *Service) PreviewAssignment(
	ctx context.Context,
	req *PreviewAssignmentRequest,
) (*AssignmentPreview, error) {
	control, err := s.dispatchControlRepo.GetOrCreate(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
	)
	if err != nil {
		return nil, err
	}

	move, err := s.loadMove(ctx, req.TenantInfo, req.MoveID)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo: req.TenantInfo,
		Filter: &repositories.DispatchBoardFilter{
			TenantInfo:  req.TenantInfo,
			WorkerIDs:   []pulid.ID{req.WorkerID},
			WindowStart: move.OriginWindowStart,
			WindowEnd:   move.DestinationWindowStart,
		},
		Control:     control,
		CustomerIDs: []pulid.ID{move.CustomerID},
	})
	if err != nil {
		return nil, err
	}

	scores := s.candidates.RankCandidates(&dispatchcandidateservice.RankRequest{
		Move:           move,
		Snapshot:       snapshot,
		IncludeBlocked: true,
	})
	if len(scores) == 0 {
		return nil, errortypes.NewNotFoundError(
			"Driver is not available in your organization's dispatch pool",
		)
	}

	score := scores[0]
	preview := &AssignmentPreview{
		MoveID:    req.MoveID,
		WorkerID:  req.WorkerID,
		TractorID: firstNonNil(req.TractorID, score.TractorID),
		TrailerID: firstNonNil(req.TrailerID, score.TrailerID),
		Score:     score,
		Blocked:   score.Blocked(),
	}
	preview.RequiresOverride = preview.Blocked &&
		control.ComplianceEnforcementLevel.ShouldBlock()

	return preview, nil
}

func (s *Service) loadMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*repositories.BoardMove, error) {
	moves, err := s.consoleRepo.ListBoardMoves(ctx, &repositories.DispatchBoardFilter{
		TenantInfo:     tenantInfo,
		MoveIDs:        []pulid.ID{moveID},
		IncludeCovered: true,
		Limit:          1,
	})
	if err != nil {
		return nil, err
	}
	if len(moves) == 0 {
		return nil, errortypes.NewNotFoundError("Shipment move not found within your organization")
	}
	return moves[0], nil
}

func (s *Service) buildSnapshotForMoves(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *dispatchcontrol.DispatchControl,
	fleetCodeIDs []pulid.ID,
	moves []*repositories.BoardMove,
) (*dispatchcandidateservice.FleetSnapshot, error) {
	windowStart, windowEnd := spanOf(moves)

	return s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo: tenantInfo,
		Filter: &repositories.DispatchBoardFilter{
			TenantInfo:   tenantInfo,
			FleetCodeIDs: fleetCodeIDs,
			WindowStart:  windowStart,
			WindowEnd:    windowEnd,
		},
		Control:     control,
		CustomerIDs: customerIDsOf(moves),
	})
}

func firstNonNil(preferred, fallback pulid.ID) pulid.ID {
	if !preferred.IsNil() {
		return preferred
	}
	return fallback
}

func customerIDsOf(moves []*repositories.BoardMove) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(moves))
	ids := make([]pulid.ID, 0, len(moves))
	for _, move := range moves {
		if move.CustomerID.IsNil() {
			continue
		}
		if _, ok := seen[move.CustomerID]; ok {
			continue
		}
		seen[move.CustomerID] = struct{}{}
		ids = append(ids, move.CustomerID)
	}
	return ids
}

func spanOf(moves []*repositories.BoardMove) (int64, int64) {
	var start, end int64
	for _, move := range moves {
		if move.OriginWindowStart > 0 && (start == 0 || move.OriginWindowStart < start) {
			start = move.OriginWindowStart
		}
		moveEnd := move.DestinationWindowStart
		if move.DestinationWindowEnd != nil && *move.DestinationWindowEnd > moveEnd {
			moveEnd = *move.DestinationWindowEnd
		}
		if moveEnd > end {
			end = moveEnd
		}
	}
	return start, end
}

// resolveWindow falls back to the organization's configured planning horizon so the
// board and the optimizer always plan over the same span.
func resolveWindow(
	start, end int64,
	control *dispatchcontrol.DispatchControl,
	now int64,
) (int64, int64) {
	if start <= 0 {
		start = now
	}
	if end <= 0 {
		end = now + int64(control.PlanningHorizonHours())*secondsPerHour
	}
	return start, end
}

func sortMatches(matches []*DriverMoveMatch) {
	sort.SliceStable(matches, func(i, j int) bool {
		left, right := matches[i], matches[j]
		leftBlocked, rightBlocked := left.Score.Blocked(), right.Score.Blocked()
		if leftBlocked != rightBlocked {
			return rightBlocked
		}
		if left.Score.Score != right.Score.Score {
			return left.Score.Score > right.Score.Score
		}
		return left.Move.OriginWindowStart < right.Move.OriginWindowStart
	})
}

// averageDeadhead reports the empty miles already committed on the board: for every
// covered move, how far its assigned truck currently sits from the pickup. Moves whose
// truck has no fresh position are skipped rather than counted as zero, which would
// flatter the number.
func averageDeadhead(
	moves []*BoardMove,
	snapshot *dispatchcandidateservice.FleetSnapshot,
) float64 {
	if len(moves) == 0 || snapshot == nil {
		return 0
	}

	total, count := 0.0, 0
	for _, move := range moves {
		if move.AssignedTractorID.IsNil() ||
			move.OriginLatitude == nil ||
			move.OriginLongitude == nil {
			continue
		}
		position := snapshot.PosByTractor[move.AssignedTractorID]
		if position == nil {
			continue
		}
		total += geoutils.HaversineMiles(
			position.Latitude,
			position.Longitude,
			*move.OriginLatitude,
			*move.OriginLongitude,
		)
		count++
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}
