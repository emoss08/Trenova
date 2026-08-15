package dispatchautoassignservice

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	evidenceTypeMove     = "shipment_move"
	evidenceTypeHOS      = "worker_hos_state"
	evidenceTypeDeadhead = "vehicle_position"

	toolNameAssignMove = "assign_move"

	promptVersion = "dispatch-auto-assign-v1"

	maxPlanMoves = 400
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	ConsoleRepo         repositories.DispatchConsoleRepository
	DispatchControlRepo repositories.DispatchControlRepository
	AgentControlRepo    repositories.AgentControlRepository
	ProposalRepo        repositories.AgentProposalRepository
	CandidateService    *dispatchcandidateservice.Service
	AgentRunService     portservices.AgentRunService
	AssignmentService   portservices.AssignmentService
}

type Service struct {
	l                   *zap.Logger
	consoleRepo         repositories.DispatchConsoleRepository
	dispatchControlRepo repositories.DispatchControlRepository
	agentControlRepo    repositories.AgentControlRepository
	proposalRepo        repositories.AgentProposalRepository
	candidates          *dispatchcandidateservice.Service
	runService          portservices.AgentRunService
	assignments         portservices.AssignmentService
}

func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.dispatch-auto-assign"),
		consoleRepo:         p.ConsoleRepo,
		dispatchControlRepo: p.DispatchControlRepo,
		agentControlRepo:    p.AgentControlRepo,
		proposalRepo:        p.ProposalRepo,
		candidates:          p.CandidateService,
		runService:          p.AgentRunService,
		assignments:         p.AssignmentService,
	}
}

func (s *Service) Plan(
	ctx context.Context,
	req *portservices.DispatchPlanRequest,
) (*portservices.DispatchPlan, error) {
	control, err := s.dispatchControlRepo.GetOrCreate(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
	)
	if err != nil {
		return nil, err
	}

	agentControl, err := s.agentControlRepo.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if !control.EnableAutoAssignment {
		return nil, errortypes.NewBusinessError(
			"Auto assignment is disabled for this organization",
		)
	}

	now := timeutils.NowUnix()
	filter := buildFilter(req, control, now)

	moves, err := s.consoleRepo.ListBoardMoves(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(moves) > maxPlanMoves {
		moves = moves[:maxPlanMoves]
		s.l.Warn(
			"auto-assign plan truncated to the per-run move cap",
			zap.Int("cap", maxPlanMoves),
		)
	}

	if len(moves) == 0 {
		return emptyPlan(control, agentControl, now), nil
	}

	snapshot, err := s.candidates.BuildSnapshot(ctx, &dispatchcandidateservice.SnapshotRequest{
		TenantInfo:  req.TenantInfo,
		Filter:      filter,
		Control:     control,
		CustomerIDs: dispatchcandidateservice.CustomerIDsOf(moves),
		TrailerIDs:  dispatchcandidateservice.TrailerIDsOf(moves),
	})
	if err != nil {
		return nil, err
	}

	plan := s.solve(&solveParams{
		Moves:        moves,
		Snapshot:     snapshot,
		Control:      control,
		AgentControl: agentControl,
		Now:          now,
	})

	if err = s.recordPlan(ctx, req, agentControl, plan); err != nil {
		return nil, err
	}

	if req.Apply {
		s.applyAutoExecutable(ctx, req, plan)
	}

	return plan, nil
}

func emptyPlan(
	control *dispatchcontrol.DispatchControl,
	agentControl *tenant.AgentControl,
	now int64,
) *portservices.DispatchPlan {
	return &portservices.DispatchPlan{
		Assignments:  []*portservices.DispatchPlannedAssignment{},
		Uncovered:    []*portservices.DispatchUncoveredMove{},
		Tours:        []*portservices.DispatchTour{},
		PlanningMode: control.ResolvedPlanningMode().String(),
		ShadowMode:   agentControl.ShadowMode,
		AutonomyTier: resolveTier(agentControl),
		GeneratedAt:  now,
	}
}

func buildFilter(
	req *portservices.DispatchPlanRequest,
	control *dispatchcontrol.DispatchControl,
	now int64,
) *repositories.DispatchBoardFilter {
	windowStart, windowEnd := dispatchcandidateservice.ResolveWindow(
		req.WindowStart,
		req.WindowEnd,
		control,
		now,
	)

	return &repositories.DispatchBoardFilter{
		TenantInfo:   req.TenantInfo,
		WindowStart:  windowStart,
		WindowEnd:    windowEnd,
		FleetCodeIDs: req.FleetCodeIDs,
		MoveIDs:      req.MoveIDs,
		Limit:        maxPlanMoves,
	}
}

type solveParams struct {
	Moves        []*repositories.BoardMove
	Snapshot     *dispatchcandidateservice.FleetSnapshot
	Control      *dispatchcontrol.DispatchControl
	AgentControl *tenant.AgentControl
	Now          int64
}

func (s *Service) solve(p *solveParams) *portservices.DispatchPlan {
	if len(p.Snapshot.Drivers) == 0 {
		return uncoveredOnlyPlan(p)
	}

	if p.Control.ResolvedPlanningMode() == dispatchcontrol.PlanningModeHorizon {
		return s.solveHorizon(p)
	}

	return s.solveImmediate(p)
}

func uncoveredOnlyPlan(p *solveParams) *portservices.DispatchPlan {
	plan := emptyPlan(p.Control, p.AgentControl, p.Now)
	plan.Uncovered = make([]*portservices.DispatchUncoveredMove, 0, len(p.Moves))
	for _, move := range p.Moves {
		plan.Uncovered = append(plan.Uncovered, uncoveredFor(move, nil, p.Control))
	}
	return plan
}

func (s *Service) solveImmediate(p *solveParams) *portservices.DispatchPlan {
	drivers := p.Snapshot.Drivers

	cost := make([][]float64, len(p.Moves))
	scores := make([][]*dispatchcandidateservice.CandidateScore, len(p.Moves))
	byWorker := make(map[pulid.ID]*dispatchcandidateservice.CandidateScore, len(drivers))

	for i, move := range p.Moves {
		ranked := s.candidates.RankCandidates(&dispatchcandidateservice.RankRequest{
			Move:           move,
			Snapshot:       p.Snapshot,
			IncludeBlocked: true,
		})

		clear(byWorker)
		for _, score := range ranked {
			byWorker[score.WorkerID] = score
		}

		cost[i] = make([]float64, len(drivers))
		scores[i] = make([]*dispatchcandidateservice.CandidateScore, len(drivers))
		for j, driver := range drivers {
			score := byWorker[driver.WorkerID]
			scores[i][j] = score
			cost[i][j] = costFor(score, p.Control)
		}
	}

	return buildPlan(&buildPlanParams{
		Moves:        p.Moves,
		Solution:     assignmentsolver.Solve(cost),
		Scores:       scores,
		Control:      p.Control,
		AgentControl: p.AgentControl,
		Now:          p.Now,
	})
}

type buildPlanParams struct {
	Moves        []*repositories.BoardMove
	Solution     assignmentsolver.Result
	Scores       [][]*dispatchcandidateservice.CandidateScore
	Control      *dispatchcontrol.DispatchControl
	AgentControl *tenant.AgentControl
	Now          int64
}

func buildPlan(p *buildPlanParams) *portservices.DispatchPlan {
	tier := resolveTier(p.AgentControl)
	threshold := p.Control.ConfidenceThreshold()

	plan := &portservices.DispatchPlan{
		Assignments:  make([]*portservices.DispatchPlannedAssignment, 0, len(p.Moves)),
		Uncovered:    make([]*portservices.DispatchUncoveredMove, 0, len(p.Moves)),
		Tours:        []*portservices.DispatchTour{},
		PlanningMode: dispatchcontrol.PlanningModeImmediate.String(),
		ShadowMode:   p.AgentControl.ShadowMode,
		AutonomyTier: tier,
		GeneratedAt:  p.Now,
	}

	for i, move := range p.Moves {
		column := assignmentsolver.Unassigned
		if i < len(p.Solution.RowAssignment) {
			column = p.Solution.RowAssignment[i]
		}
		if column == assignmentsolver.Unassigned {
			plan.Uncovered = append(plan.Uncovered, uncoveredFor(move, p.Scores[i], p.Control))
			continue
		}

		score := p.Scores[i][column]
		planned := plannedAssignmentFor(&plannedAssignmentParams{
			Move:       move,
			Score:      score,
			Tier:       tier,
			Threshold:  threshold,
			ShadowMode: p.AgentControl.ShadowMode,
		})

		plan.Assignments = append(plan.Assignments, planned)
		plan.TotalScore += score.Score
	}

	return plan
}

func uncoveredFor(
	move *repositories.BoardMove,
	scores []*dispatchcandidateservice.CandidateScore,
	control *dispatchcontrol.DispatchControl,
) *portservices.DispatchUncoveredMove {
	uncovered := &portservices.DispatchUncoveredMove{
		MoveID:              move.MoveID,
		ProNumber:           move.ProNumber,
		Reason:              "No eligible driver was available for this move",
		BestBlockedFindings: []dispatcheligibility.Finding{},
	}

	var best *dispatchcandidateservice.CandidateScore
	for _, score := range scores {
		if score == nil {
			continue
		}
		if best == nil || score.Score > best.Score {
			best = score
		}
	}

	if best == nil {
		return uncovered
	}

	switch {
	case best.Blocked():
		uncovered.BestBlockedFindings = best.Findings
		uncovered.Reason = fmt.Sprintf(
			"Closest candidate %s was disqualified",
			best.WorkerName,
		)
	case deadheadExceedsLimit(best, control):
		uncovered.Reason = fmt.Sprintf(
			"Closest candidate %s exceeds the %d mile deadhead limit (%.0f empty miles)",
			best.WorkerName,
			*control.AutoAssignMaxDeadheadMiles,
			*best.DeadheadMiles,
		)
	default:
		uncovered.Reason = fmt.Sprintf(
			"Closest candidate %s was eligible but no feasible pairing remained "+
				"after higher-value assignments were made",
			best.WorkerName,
		)
	}

	return uncovered
}

func (s *Service) recordPlan(
	ctx context.Context,
	req *portservices.DispatchPlanRequest,
	agentControl *tenant.AgentControl,
	plan *portservices.DispatchPlan,
) error {
	if len(plan.Assignments) == 0 {
		return nil
	}

	run, err := s.runService.StartInline(ctx, &portservices.StartInlineAgentRunRequest{
		AgentType:     agent.TypeDispatchAssignment,
		SubjectType:   agent.SubjectShipmentMove,
		SubjectID:     plan.Assignments[0].MoveID,
		PromptVersion: promptVersion,
		Summary: fmt.Sprintf(
			"Dispatch auto-assign proposed coverage for %d move(s)",
			len(plan.Assignments),
		),
		TenantInfo: req.TenantInfo,
	}, actorFor(req))
	if err != nil {
		return err
	}
	plan.RunID = run.ID

	tier := resolveTier(agentControl)
	for _, planned := range plan.Assignments {
		toolParams, paramsErr := toolParamsFor(planned)
		if paramsErr != nil {
			return paramsErr
		}

		proposal, createErr := s.proposalRepo.Create(ctx, &agent.AgentProposal{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			RunID:          run.ID,
			ToolName:       toolNameAssignMove,
			ToolParams:     toolParams,
			Confidence:     planned.Confidence,
			Rationale:      planned.Rationale,
			Evidence:       evidenceFor(planned.Score, moveStubFor(planned)),
			AutonomyTier:   tier,
			Status:         agent.ProposalStatusPending,
		})
		if createErr != nil {
			return createErr
		}
		planned.ProposalID = proposal.ID
	}

	return nil
}

func (s *Service) applyAutoExecutable(
	ctx context.Context,
	req *portservices.DispatchPlanRequest,
	plan *portservices.DispatchPlan,
) {
	if plan.ShadowMode {
		return
	}

	for _, planned := range plan.Assignments {
		if !planned.AutoExecutable {
			continue
		}

		assignReq := &repositories.AssignShipmentMoveRequest{
			TenantInfo:      req.TenantInfo,
			ShipmentMoveID:  planned.MoveID,
			PrimaryWorkerID: planned.WorkerID,
			TractorID:       planned.TractorID,
		}
		if !planned.TrailerID.IsNil() {
			trailerID := planned.TrailerID
			assignReq.TrailerID = &trailerID
		}

		if _, err := s.assignments.AssignToMove(ctx, assignReq); err != nil {
			planned.AutoExecutable = false
			s.l.Warn(
				"auto-assign proposal could not be executed and remains pending",
				zap.String("moveId", planned.MoveID.String()),
				zap.String("workerId", planned.WorkerID.String()),
				zap.Error(err),
			)
			continue
		}

		if _, err := s.proposalRepo.UpdateStatus(
			ctx,
			repositories.UpdateAgentProposalStatusRequest{
				ID:         planned.ProposalID,
				TenantInfo: req.TenantInfo,
				Status:     agent.ProposalStatusAccepted,
			},
		); err != nil {
			s.l.Error(
				"auto-assign executed but the proposal status could not be updated",
				zap.String("moveId", planned.MoveID.String()),
				zap.String("proposalId", planned.ProposalID.String()),
				zap.Error(err),
			)
		}
	}
}

type assignMoveToolParams struct {
	ShipmentMoveID  string `json:"shipmentMoveId"`
	PrimaryWorkerID string `json:"primaryWorkerId"`
	TractorID       string `json:"tractorId"`
	TrailerID       string `json:"trailerId,omitempty"`
}

func toolParamsFor(planned *portservices.DispatchPlannedAssignment) (map[string]any, error) {
	params := assignMoveToolParams{
		ShipmentMoveID:  planned.MoveID.String(),
		PrimaryWorkerID: planned.WorkerID.String(),
		TractorID:       planned.TractorID.String(),
	}
	if !planned.TrailerID.IsNil() {
		params.TrailerID = planned.TrailerID.String()
	}

	raw, err := sonic.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal assign move tool params: %w", err)
	}

	out := make(map[string]any, 4)
	if err = sonic.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal assign move tool params: %w", err)
	}

	return out, nil
}

func resolveTier(agentControl *tenant.AgentControl) agent.AutonomyTier {
	if agentControl == nil || !agentControl.DispatchAgentEnabled {
		return agent.TierPropose
	}
	tier := agent.AutonomyTier(agentControl.DispatchAutonomyTier)
	if !tier.IsValid() {
		return agent.TierPropose
	}
	return tier
}

func actorFor(req *portservices.DispatchPlanRequest) *portservices.RequestActor {
	return &portservices.RequestActor{
		PrincipalType:  portservices.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		UserID:         req.TenantInfo.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
}
