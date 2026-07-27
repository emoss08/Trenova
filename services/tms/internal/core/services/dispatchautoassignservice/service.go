package dispatchautoassignservice

import (
	"context"
	"fmt"

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

	// maxPlanMoves bounds one optimizer run. The solver is O(n^3); beyond this the run
	// belongs in a scheduled job rather than a request a dispatcher is waiting on.
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

// Plan covers the open board as a whole. It scores every eligible driver against every
// uncovered move, solves the resulting cost matrix for the global optimum, and records
// each chosen pairing as an agent proposal.
//
// Solving globally rather than greedily is the point: taking the best pair repeatedly
// spends the strongest driver on whichever load happens to sort first and leaves later
// loads uncoverable. This is the behavior dispatchers distrust in auto-assignment, and
// avoiding it is what makes the feature usable.
func (s *Service) Plan(ctx context.Context, req *PlanRequest) (*Plan, error) {
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
	filter := s.buildFilter(req, control, now)

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
		return &Plan{
			Assignments:  []*PlannedAssignment{},
			Uncovered:    []*UncoveredMove{},
			ShadowMode:   agentControl.ShadowMode,
			AutonomyTier: resolveTier(agentControl),
			GeneratedAt:  now,
		}, nil
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

	plan := s.solve(moves, snapshot, control, agentControl, now)

	if err = s.recordPlan(ctx, req, agentControl, plan); err != nil {
		return nil, err
	}

	if req.Apply {
		if err = s.applyAutoExecutable(ctx, req, plan); err != nil {
			return nil, err
		}
	}

	return plan, nil
}

func (s *Service) buildFilter(
	req *PlanRequest,
	control *dispatchcontrol.DispatchControl,
	now int64,
) *repositories.DispatchBoardFilter {
	windowStart := req.WindowStart
	if windowStart <= 0 {
		windowStart = now
	}
	windowEnd := req.WindowEnd
	if windowEnd <= 0 {
		windowEnd = now + int64(control.PlanningHorizonHours())*3600
	}

	return &repositories.DispatchBoardFilter{
		TenantInfo:   req.TenantInfo,
		WindowStart:  windowStart,
		WindowEnd:    windowEnd,
		FleetCodeIDs: req.FleetCodeIDs,
		MoveIDs:      req.MoveIDs,
		Limit:        maxPlanMoves,
	}
}

// solve builds the move-by-driver cost matrix and reads the optimum back out of it.
func (s *Service) solve(
	moves []*repositories.BoardMove,
	snapshot *dispatchcandidateservice.FleetSnapshot,
	control *dispatchcontrol.DispatchControl,
	agentControl *tenant.AgentControl,
	now int64,
) *Plan {
	drivers := snapshot.Drivers
	cost := make([][]float64, len(moves))
	scores := make([][]*dispatchcandidateservice.CandidateScore, len(moves))

	for i, move := range moves {
		ranked := s.candidates.RankCandidates(&dispatchcandidateservice.RankRequest{
			Move:           move,
			Snapshot:       snapshot,
			IncludeBlocked: true,
		})

		byWorker := make(map[pulid.ID]*dispatchcandidateservice.CandidateScore, len(ranked))
		for _, score := range ranked {
			byWorker[score.WorkerID] = score
		}

		cost[i] = make([]float64, len(drivers))
		scores[i] = make([]*dispatchcandidateservice.CandidateScore, len(drivers))
		for j, driver := range drivers {
			score := byWorker[driver.WorkerID]
			scores[i][j] = score
			cost[i][j] = costFor(score, control)
		}
	}

	solution := assignmentsolver.Solve(cost)
	return s.buildPlan(moves, solution, scores, control, agentControl, now)
}

func (s *Service) buildPlan(
	moves []*repositories.BoardMove,
	solution assignmentsolver.Result,
	scores [][]*dispatchcandidateservice.CandidateScore,
	control *dispatchcontrol.DispatchControl,
	agentControl *tenant.AgentControl,
	now int64,
) *Plan {
	tier := resolveTier(agentControl)
	threshold := control.ConfidenceThreshold()

	plan := &Plan{
		Assignments:  make([]*PlannedAssignment, 0, len(moves)),
		Uncovered:    make([]*UncoveredMove, 0),
		ShadowMode:   agentControl.ShadowMode,
		AutonomyTier: tier,
		GeneratedAt:  now,
	}

	for i, move := range moves {
		column := solution.RowAssignment[i]
		if column == assignmentsolver.Unassigned {
			plan.Uncovered = append(plan.Uncovered, uncoveredFor(move, scores[i]))
			continue
		}

		score := scores[i][column]
		confidence := confidenceFor(score)

		planned := &PlannedAssignment{
			MoveID:     move.MoveID,
			ProNumber:  move.ProNumber,
			WorkerID:   score.WorkerID,
			WorkerName: score.WorkerName,
			TractorID:  score.TractorID,
			TrailerID:  score.TrailerID,
			Score:      score,
			Confidence: confidence,
			Rationale:  rationaleFor(score, move),
		}

		// Auto-execution needs all three: the tier allows it, the organization is not
		// merely watching, and this particular pairing clears the confidence bar. Anything
		// short of that stays a proposal for a dispatcher rather than being dropped.
		planned.AutoExecutable = tier == agent.TierAutoExecute &&
			!agentControl.ShadowMode &&
			!score.Blocked() &&
			confidence.GreaterThanOrEqual(threshold)

		plan.Assignments = append(plan.Assignments, planned)
		plan.TotalScore += score.Score
	}

	return plan
}

// uncoveredFor explains why a move went uncovered, preferring the closest candidate's
// blocking findings over a generic message.
func uncoveredFor(
	move *repositories.BoardMove,
	scores []*dispatchcandidateservice.CandidateScore,
) *UncoveredMove {
	uncovered := &UncoveredMove{
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

	if best != nil && len(best.Findings) > 0 {
		uncovered.BestBlockedFindings = best.Findings
		uncovered.Reason = fmt.Sprintf(
			"Closest candidate %s was disqualified: %s",
			best.WorkerName,
			best.Findings[0].Message,
		)
	}

	return uncovered
}

// recordPlan opens an agent run and writes one proposal per chosen pairing, so every
// automated decision carries the same audit trail as the billing agent's. Shadow mode
// still records the run: watching what the agent would have done is the entire point of
// that mode.
func (s *Service) recordPlan(
	ctx context.Context,
	req *PlanRequest,
	agentControl *tenant.AgentControl,
	plan *Plan,
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
		proposal, createErr := s.proposalRepo.Create(ctx, &agent.AgentProposal{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			RunID:          run.ID,
			ToolName:       toolNameAssignMove,
			ToolParams:     toolParamsFor(planned),
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

// applyAutoExecutable commits only the pairings that cleared every gate. Everything else
// stays pending for a dispatcher — an assignment the agent was not sure about must remain
// visible rather than being quietly dropped from the board.
func (s *Service) applyAutoExecutable(
	ctx context.Context,
	req *PlanRequest,
	plan *Plan,
) error {
	if plan.ShadowMode {
		return nil
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
			// A pairing that fails at write time stays a proposal rather than failing the
			// whole run: the board is still covered by everything that did succeed, and the
			// dispatcher sees exactly which one needs a human.
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
			return err
		}
	}

	return nil
}

func toolParamsFor(planned *PlannedAssignment) map[string]any {
	params := map[string]any{
		"shipmentMoveId":  planned.MoveID.String(),
		"primaryWorkerId": planned.WorkerID.String(),
		"tractorId":       planned.TractorID.String(),
	}
	if !planned.TrailerID.IsNil() {
		params["trailerId"] = planned.TrailerID.String()
	}
	return params
}

// moveStubFor reconstructs the minimum move context the evidence builder needs from an
// already-planned assignment.
func moveStubFor(planned *PlannedAssignment) *repositories.BoardMove {
	return &repositories.BoardMove{
		MoveID:    planned.MoveID,
		ProNumber: planned.ProNumber,
	}
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

func actorFor(req *PlanRequest) *portservices.RequestActor {
	return &portservices.RequestActor{
		PrincipalType:  portservices.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		UserID:         req.TenantInfo.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
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
