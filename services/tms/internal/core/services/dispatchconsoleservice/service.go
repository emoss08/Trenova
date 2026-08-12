package dispatchconsoleservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	secondsPerHour = int64(3600)
	secondsPerDay  = int64(86400)

	nowThresholdSeconds = int64(4 * 3600)

	defaultCandidateLimit = 25
)

type Params struct {
	fx.In

	ConsoleRepo         repositories.DispatchConsoleRepository
	DispatchControlRepo repositories.DispatchControlRepository
	OrgCacheRepo        repositories.OrganizationCacheRepository
	TenderRepo          repositories.TenderRepository
	CandidateService    *dispatchcandidateservice.Service
	Logger              *zap.Logger
}

type Service struct {
	consoleRepo         repositories.DispatchConsoleRepository
	dispatchControlRepo repositories.DispatchControlRepository
	orgCacheRepo        repositories.OrganizationCacheRepository
	tenderRepo          repositories.TenderRepository
	candidates          *dispatchcandidateservice.Service
	l                   *zap.Logger
}

func New(p Params) *Service {
	return &Service{
		consoleRepo:         p.ConsoleRepo,
		dispatchControlRepo: p.DispatchControlRepo,
		orgCacheRepo:        p.OrgCacheRepo,
		tenderRepo:          p.TenderRepo,
		candidates:          p.CandidateService,
		l:                   p.Logger.Named("service.dispatch-console"),
	}
}

// attachLiveTenders decorates board cards with the tender chip in one batched
// query so a large board never fans out per move.
func (s *Service) attachLiveTenders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moves []*BoardMove,
) error {
	if s.tenderRepo == nil || len(moves) == 0 {
		return nil
	}

	moveIDs := make([]pulid.ID, 0, len(moves))
	for _, move := range moves {
		if move != nil && move.BoardMove != nil && !move.MoveID.IsNil() {
			moveIDs = append(moveIDs, move.MoveID)
		}
	}
	if len(moveIDs) == 0 {
		return nil
	}

	tenders, err := s.tenderRepo.ListLiveByMoveIDs(
		ctx,
		&repositories.ListLiveTendersByMovesRequest{
			TenantInfo: tenantInfo,
			MoveIDs:    moveIDs,
		},
	)
	if err != nil {
		return err
	}
	if len(tenders) == 0 {
		return nil
	}

	summaries := make(map[pulid.ID]*MoveTenderSummary, len(tenders))
	for _, entity := range tenders {
		if entity == nil {
			continue
		}
		summaries[entity.ShipmentMoveID] = summarizeTender(entity)
	}

	for _, move := range moves {
		if move != nil && move.BoardMove != nil {
			move.LiveTender = summaries[move.MoveID]
		}
	}

	return nil
}

func (s *Service) tenantDayStart(ctx context.Context, orgID pulid.ID, now int64) int64 {
	timezone := ""
	if s.orgCacheRepo != nil {
		if org, err := s.orgCacheRepo.GetByID(ctx, orgID); err == nil && org != nil {
			timezone = org.Timezone
		}
	}

	dayStart, err := timeutils.DayStartUnix(now, timezone)
	if err != nil {
		return 0
	}
	return dayStart
}

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
	windowStart, windowEnd := dispatchcandidateservice.ResolveWindow(
		req.WindowStart,
		req.WindowEnd,
		control,
		now,
	)

	filter := &repositories.DispatchBoardFilter{
		TenantInfo:     req.TenantInfo,
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		DayStartUnix:   s.tenantDayStart(ctx, req.TenantInfo.OrgID, now),
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

	driverFilter := *filter
	driverFilter.Query = ""
	driverFilter.CustomerIDs = nil
	driverFilter.ServiceTypeIDs = nil

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo:  req.TenantInfo,
		Filter:      &driverFilter,
		Control:     control,
		CustomerIDs: dispatchcandidateservice.CustomerIDsOf(moves),
		TrailerIDs:  dispatchcandidateservice.TrailerIDsOf(moves),
	})
	if err != nil {
		return nil, err
	}

	summary, err := s.consoleRepo.GetBoardSummary(ctx, filter)
	if err != nil {
		return nil, err
	}

	decoratedMoves := decorateMoves(moves, now)
	if err = s.attachLiveTenders(ctx, req.TenantInfo, decoratedMoves); err != nil {
		s.l.Warn("failed to attach live tenders to the dispatch board; rendering without tender chips",
			zap.Error(err))
	}

	return &Board{
		Moves:       decoratedMoves,
		Drivers:     decorateDrivers(snapshot, control, now),
		Summary:     summary,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		GeneratedAt: now,
	}, nil
}

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

	snapshot, err := s.buildSnapshotForMoves(ctx, &snapshotForMovesParams{
		TenantInfo:   req.TenantInfo,
		Control:      control,
		FleetCodeIDs: req.FleetCodeIDs,
		Moves:        []*repositories.BoardMove{move},
	})
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
	windowStart, windowEnd := dispatchcandidateservice.ResolveWindow(
		req.WindowStart,
		req.WindowEnd,
		control,
		now,
	)

	moveFilter := &repositories.DispatchBoardFilter{
		TenantInfo:  req.TenantInfo,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	}

	moves, err := s.consoleRepo.ListBoardMoves(ctx, moveFilter)
	if err != nil {
		return nil, err
	}

	driverFilter := *moveFilter
	driverFilter.WorkerIDs = []pulid.ID{req.WorkerID}

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo:  req.TenantInfo,
		Filter:      &driverFilter,
		Control:     control,
		CustomerIDs: dispatchcandidateservice.CustomerIDsOf(moves),
		TrailerIDs:  dispatchcandidateservice.TrailerIDsOf(moves),
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

	now := timeutils.NowUnix()
	moves := []*repositories.BoardMove{move}
	windowStart, windowEnd := spanOf(moves, now)

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo: req.TenantInfo,
		Filter: &repositories.DispatchBoardFilter{
			TenantInfo:  req.TenantInfo,
			WorkerIDs:   []pulid.ID{req.WorkerID},
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
		},
		Control:     control,
		CustomerIDs: dispatchcandidateservice.CustomerIDsOf(moves),
		TrailerIDs:  dispatchcandidateservice.TrailerIDsOf(moves),
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

type snapshotForMovesParams struct {
	TenantInfo   pagination.TenantInfo
	Control      *dispatchcontrol.DispatchControl
	FleetCodeIDs []pulid.ID
	Moves        []*repositories.BoardMove
}

func (s *Service) buildSnapshotForMoves(
	ctx context.Context,
	p *snapshotForMovesParams,
) (*dispatchcandidateservice.FleetSnapshot, error) {
	windowStart, windowEnd := spanOf(p.Moves, timeutils.NowUnix())

	return s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo: p.TenantInfo,
		Filter: &repositories.DispatchBoardFilter{
			TenantInfo:   p.TenantInfo,
			FleetCodeIDs: p.FleetCodeIDs,
			WindowStart:  windowStart,
			WindowEnd:    windowEnd,
		},
		Control:     p.Control,
		CustomerIDs: dispatchcandidateservice.CustomerIDsOf(p.Moves),
		TrailerIDs:  dispatchcandidateservice.TrailerIDsOf(p.Moves),
	})
}

func firstNonNil(preferred, fallback pulid.ID) pulid.ID {
	if !preferred.IsNil() {
		return preferred
	}
	return fallback
}

func spanOf(moves []*repositories.BoardMove, now int64) (spanStart, spanEnd int64) {
	var start, end int64
	for _, move := range moves {
		if move.OriginWindowStart > 0 && (start == 0 || move.OriginWindowStart < start) {
			start = move.OriginWindowStart
		}
		if moveEnd := dispatchcandidateservice.MoveWindowEnd(move); moveEnd > end {
			end = moveEnd
		}
	}
	if start == 0 {
		start = now
	}
	if end < start {
		end = start
	}
	return start, end
}

func sortMatches(matches []*DriverMoveMatch) {
	sort.SliceStable(matches, func(i, j int) bool {
		if c := dispatchcandidateservice.CompareRank(matches[i].Score, matches[j].Score); c != 0 {
			return c < 0
		}
		return matches[i].Move.OriginWindowStart < matches[j].Move.OriginWindowStart
	})
}
