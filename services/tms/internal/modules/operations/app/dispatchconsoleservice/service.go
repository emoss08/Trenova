package dispatchconsoleservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	nowThresholdSeconds = dispatchboardports.UrgencyNowSeconds

	defaultCandidateLimit = 25
)

type Params struct {
	fx.In

	ConsoleRepo         dispatchboardports.DispatchConsoleRepository
	DispatchControlRepo dispatchcontrolports.DispatchControlRepository
	Organizations       tenantports.OrganizationRepository
	TenderRepo          tenderports.TenderRepository
	CandidateService    *dispatchcandidateservice.Service
	PriceRepo           dispatchplanports.PriceRepository
	Logger              *zap.Logger
}

type Service struct {
	consoleRepo         dispatchboardports.DispatchConsoleRepository
	dispatchControlRepo dispatchcontrolports.DispatchControlRepository
	organizations       tenantports.OrganizationRepository
	tenderRepo          tenderports.TenderRepository
	candidates          *dispatchcandidateservice.Service
	prices              dispatchplanports.PriceRepository
	l                   *zap.Logger
}

func New(p Params) *Service { //nolint:gocritic // fx requires an fx.In parameter struct by value
	return &Service{
		consoleRepo:         p.ConsoleRepo,
		dispatchControlRepo: p.DispatchControlRepo,
		organizations:       p.Organizations,
		tenderRepo:          p.TenderRepo,
		candidates:          p.CandidateService,
		prices:              p.PriceRepo,
		l:                   p.Logger.Named("service.dispatch-console"),
	}
}

func (s *Service) attachLiveTenders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moves []*BoardMove,
) error {
	if len(moves) == 0 {
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
		&tenderports.ListLiveTendersByMovesRequest{
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

func (s *Service) tenantDayStart(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) (int64, error) {
	location, err := orgzone.Location(ctx, s.organizations, tenantInfo)
	if err != nil {
		return 0, err
	}

	return timeutils.DayStartUnix(now, location.String())
}

type boardScope struct {
	control     *dispatchcontrol.DispatchControl
	filter      *dispatchboardports.DispatchBoardFilter
	now         int64
	windowStart int64
	windowEnd   int64
}

func (s *Service) resolveBoardScope(
	ctx context.Context,
	req *GetBoardRequest,
) (*boardScope, error) {
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

	dayStart, err := s.tenantDayStart(ctx, req.TenantInfo, now)
	if err != nil {
		return nil, err
	}

	filter := &dispatchboardports.DispatchBoardFilter{
		TenantInfo:     req.TenantInfo,
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		DayStartUnix:   dayStart,
		FleetCodeIDs:   req.FleetCodeIDs,
		CustomerIDs:    req.CustomerIDs,
		ServiceTypeIDs: req.ServiceTypeIDs,
		WorkerIDs:      req.WorkerIDs,
		Query:          req.Query,
		IncludeCovered: req.IncludeCovered,
		Limit:          req.Limit,
		DueFirst:       req.DueFirst,
		Offset:         max(req.Offset, 0),
	}
	if req.LateOnly {
		filter.LateAt = now
	}

	return &boardScope{
		control:     control,
		filter:      filter,
		now:         now,
		windowStart: windowStart,
		windowEnd:   windowEnd,
	}, nil
}

func (s *Service) decorateBoardMoves(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moves []*dispatchboardports.BoardMove,
	now int64,
) []*BoardMove {
	decorated := decorateMoves(moves, now)
	if err := s.attachLiveTenders(ctx, tenantInfo, decorated); err != nil {
		s.l.Warn(
			"failed to attach live tenders to the dispatch board; rendering without tender chips",
			zap.Error(err),
		)
	}
	if err := s.attachPlanPrices(ctx, tenantInfo, decorated); err != nil {
		s.l.Warn(
			"failed to attach plan prices to the dispatch board; rendering without prices",
			zap.Error(err),
		)
	}

	return decorated
}

func (s *Service) GetBoardMoves(ctx context.Context, req *GetBoardRequest) ([]*BoardMove, error) {
	scope, err := s.resolveBoardScope(ctx, req)
	if err != nil {
		return nil, err
	}

	moves, err := s.consoleRepo.ListBoardMoves(ctx, scope.filter)
	if err != nil {
		return nil, err
	}

	return s.decorateBoardMoves(ctx, req.TenantInfo, moves, scope.now), nil
}

func (s *Service) GetMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*BoardMove, error) {
	move, err := s.loadMove(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}

	return s.decorateBoardMoves(
		ctx,
		tenantInfo,
		[]*dispatchboardports.BoardMove{move},
		timeutils.NowUnix(),
	)[0], nil
}

func (s *Service) GetBoard(ctx context.Context, req *GetBoardRequest) (*Board, error) {
	scope, err := s.resolveBoardScope(ctx, req)
	if err != nil {
		return nil, err
	}

	moves, err := s.consoleRepo.ListBoardMoves(ctx, scope.filter)
	if err != nil {
		return nil, err
	}

	driverFilter := *scope.filter
	driverFilter.Query = ""
	driverFilter.CustomerIDs = nil
	driverFilter.ServiceTypeIDs = nil
	driverFilter.LateAt = 0
	driverFilter.Offset = 0

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo:   req.TenantInfo,
		Filter:       &driverFilter,
		Control:      scope.control,
		CustomerIDs:  dispatchcandidateservice.CustomerIDsOf(moves),
		TrailerIDs:   dispatchcandidateservice.TrailerIDsOf(moves),
		Destinations: dispatchcandidateservice.DestinationsOf(moves),
	})
	if err != nil {
		return nil, err
	}

	summary, err := s.consoleRepo.GetBoardSummary(ctx, scope.filter)
	if err != nil {
		return nil, err
	}

	boardMoves := s.decorateBoardMoves(ctx, req.TenantInfo, moves, scope.now)
	drivers := decorateDrivers(snapshot, scope.control, scope.now)
	attachPlannedMoves(drivers, boardMoves)

	return &Board{
		Moves:            boardMoves,
		Drivers:          drivers,
		DriversTruncated: snapshot.DriversTruncated,
		Summary:          summary,
		WindowStart:      scope.windowStart,
		WindowEnd:        scope.windowEnd,
		GeneratedAt:      scope.now,
	}, nil
}

func (s *Service) GetMoveCandidates(
	ctx context.Context,
	req *MoveCandidatesRequest,
) ([]*dispatchplanports.CandidateScore, error) {
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
		Moves:        []*dispatchboardports.BoardMove{move},
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

	moveFilter := &dispatchboardports.DispatchBoardFilter{
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
		TenantInfo:   req.TenantInfo,
		Filter:       &driverFilter,
		Control:      control,
		CustomerIDs:  dispatchcandidateservice.CustomerIDsOf(moves),
		TrailerIDs:   dispatchcandidateservice.TrailerIDsOf(moves),
		Destinations: dispatchcandidateservice.DestinationsOf(moves),
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
	moves := []*dispatchboardports.BoardMove{move}
	windowStart, windowEnd := spanOf(moves, now)

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo: req.TenantInfo,
		Filter: &dispatchboardports.DispatchBoardFilter{
			TenantInfo:  req.TenantInfo,
			WorkerIDs:   []pulid.ID{req.WorkerID},
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
		},
		Control:      control,
		CustomerIDs:  dispatchcandidateservice.CustomerIDsOf(moves),
		TrailerIDs:   dispatchcandidateservice.TrailerIDsOf(moves),
		Destinations: dispatchcandidateservice.DestinationsOf(moves),
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
) (*dispatchboardports.BoardMove, error) {
	moves, err := s.consoleRepo.ListBoardMoves(ctx, &dispatchboardports.DispatchBoardFilter{
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
	Moves        []*dispatchboardports.BoardMove
}

func (s *Service) buildSnapshotForMoves(
	ctx context.Context,
	p *snapshotForMovesParams,
) (*dispatchcandidateservice.FleetSnapshot, error) {
	windowStart, windowEnd := spanOf(p.Moves, timeutils.NowUnix())

	return s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo: p.TenantInfo,
		Filter: &dispatchboardports.DispatchBoardFilter{
			TenantInfo:   p.TenantInfo,
			FleetCodeIDs: p.FleetCodeIDs,
			WindowStart:  windowStart,
			WindowEnd:    windowEnd,
			Near:         pickupOf(p.Moves),
		},
		Control:      p.Control,
		CustomerIDs:  dispatchcandidateservice.CustomerIDsOf(p.Moves),
		TrailerIDs:   dispatchcandidateservice.TrailerIDsOf(p.Moves),
		Destinations: dispatchcandidateservice.DestinationsOf(p.Moves),
	})
}

func pickupOf(moves []*dispatchboardports.BoardMove) *geoutils.Point {
	if len(moves) != 1 {
		return nil
	}
	move := moves[0]
	if move.OriginLatitude == nil || move.OriginLongitude == nil {
		return nil
	}

	return &geoutils.Point{Latitude: *move.OriginLatitude, Longitude: *move.OriginLongitude}
}

func firstNonNil(preferred, fallback pulid.ID) pulid.ID {
	if !preferred.IsNil() {
		return preferred
	}
	return fallback
}

func spanOf(moves []*dispatchboardports.BoardMove, now int64) (spanStart, spanEnd int64) {
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
		if c := dispatchplanports.CompareRank(matches[i].Score, matches[j].Score); c != 0 {
			return c < 0
		}
		return matches[i].Move.OriginWindowStart < matches[j].Move.OriginWindowStart
	})
}
