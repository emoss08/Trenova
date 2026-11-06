package dispatchplanservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxPlanMoves   = 5_000
	maxPlanDrivers = 10_000
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	ConsoleRepo         dispatchboardports.DispatchConsoleRepository
	DispatchControlRepo dispatchcontrolports.DispatchControlRepository
	Candidates          *dispatchcandidateservice.Service
	PriceRepo           dispatchplanports.PriceRepository
	Market              dispatchmarketports.Recorder
	Clock               timeutils.Clock
}

type Service struct {
	l           *zap.Logger
	consoleRepo dispatchboardports.DispatchConsoleRepository
	controls    dispatchcontrolports.DispatchControlRepository
	candidates  *dispatchcandidateservice.Service
	prices      dispatchplanports.PriceRepository
	market      dispatchmarketports.Recorder
	clock       timeutils.Clock
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) dispatchplanports.Planner {
	return &Service{
		l:           p.Logger.Named("service.dispatch-plan"),
		consoleRepo: p.ConsoleRepo,
		controls:    p.DispatchControlRepo,
		candidates:  p.Candidates,
		prices:      p.PriceRepo,
		market:      p.Market,
		clock:       p.Clock,
	}
}

func (s *Service) Plan(
	ctx context.Context,
	req *dispatchplanports.Request,
) (*dispatchplanports.Plan, error) {
	control, err := s.controls.GetOrCreate(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return nil, err
	}

	now := s.clock()
	windowStart, windowEnd := dispatchcandidateservice.ResolveWindow(
		req.WindowStart,
		req.WindowEnd,
		control,
		now,
	)
	filter := &dispatchboardports.DispatchBoardFilter{
		TenantInfo:   req.TenantInfo,
		WindowStart:  windowStart,
		WindowEnd:    windowEnd,
		FleetCodeIDs: req.FleetCodeIDs,
		MoveIDs:      req.MoveIDs,
		Limit:        maxPlanMoves,
		DriverLimit:  maxPlanDrivers,
	}

	moves, err := s.consoleRepo.ListBoardMoves(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list moves to plan: %w", err)
	}
	plan := &dispatchplanports.Plan{
		Mode:        control.ResolvedPlanningMode(),
		GeneratedAt: now,
		Pairings:    []*dispatchplanports.Pairing{},
		Uncovered:   []*dispatchplanports.Uncovered{},
		Tours:       []*dispatchplanports.Tour{},
	}
	if len(moves) == 0 {
		return plan, nil
	}

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo:   req.TenantInfo,
		Filter:       filter,
		Control:      control,
		CustomerIDs:  dispatchcandidateservice.CustomerIDsOf(moves),
		TrailerIDs:   dispatchcandidateservice.TrailerIDsOf(moves),
		Destinations: dispatchcandidateservice.DestinationsOf(moves),
		Now:          now,
	})
	if err != nil {
		return nil, err
	}

	if snapshot.DriversTruncated {
		s.l.Warn("dispatch plan read the driver limit; drivers past it were not considered",
			zap.Int("driverLimit", maxPlanDrivers),
			zap.String("organizationId", req.TenantInfo.OrgID.String()))
	}

	graph := s.candidates.BuildGraph(ctx, &dispatchcandidateservice.GraphRequest{
		TenantInfo: req.TenantInfo,
		Moves:      moves,
		Snapshot:   snapshot,
	})

	if err = s.solve(ctx, &solveParams{
		TenantInfo: req.TenantInfo,
		Control:    control,
		Moves:      moves,
		Plan:       plan,
		Graph:      graph,
		Snapshot:   snapshot,
	}); err != nil {
		return nil, err
	}

	if err = s.recordPrices(ctx, req.TenantInfo, moves, plan); err != nil {
		return nil, err
	}

	if control.PlanForEndMarket {
		s.recordMarket(ctx, &marketParams{
			TenantInfo: req.TenantInfo,
			Plan:       plan,
			Graph:      graph,
			Snapshot:   snapshot,
		})
	}

	return plan, nil
}

type solveParams struct {
	TenantInfo pagination.TenantInfo
	Control    *dispatchcontrol.DispatchControl
	Moves      []*dispatchboardports.BoardMove
	Plan       *dispatchplanports.Plan
	Graph      *dispatchcandidateservice.Graph
	Snapshot   *dispatchcandidateservice.FleetSnapshot
}

func (s *Service) solve(ctx context.Context, p *solveParams) error {
	standing, err := s.standingOf(ctx, p.TenantInfo, p.Moves)
	if err != nil {
		return err
	}

	if p.Control.ResolvedPlanningMode() == dispatchcontrol.PlanningModeHorizon {
		return planHorizon(ctx, &horizonParams{
			Plan:       p.Plan,
			Graph:      p.Graph,
			Snapshot:   p.Snapshot,
			Candidates: s.candidates,
			Control:    p.Control,
			Standing:   standing,
		})
	}

	planImmediate(ctx, p.Plan, p.Graph, stabilityOf(
		p.Graph,
		standing,
		p.Snapshot.Model.Economics().ReplanChangeCents,
	))

	return nil
}
