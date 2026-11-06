package dispatchautoassignservice

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalexecutor"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	evidenceTypeMove     = "shipment_move"
	evidenceTypeHOS      = "worker_hos_state"
	evidenceTypeDeadhead = "vehicle_position"

	promptVersion = "dispatch-auto-assign-v2"

	assignMoveTool = "dispatch_assign_move"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	DispatchControlRepo dispatchcontrolports.DispatchControlRepository
	AgentControlRepo    tenantports.AgentControlRepository
	DefinitionRepo      agentdefinitionports.AgentDefinitionRepository
	ProposalRepo        agentproposalports.AgentProposalRepository
	Planner             dispatchplanports.Planner
	AgentRunService     agentruntimeports.AgentRunService
	Executor            *proposalexecutor.Service
	WorkerRepo          workerports.WorkerRepository
}

type Service struct {
	l                   *zap.Logger
	dispatchControlRepo dispatchcontrolports.DispatchControlRepository
	agentControlRepo    tenantports.AgentControlRepository
	definitionRepo      agentdefinitionports.AgentDefinitionRepository
	proposalRepo        agentproposalports.AgentProposalRepository
	planner             dispatchplanports.Planner
	runService          agentruntimeports.AgentRunService
	executor            *proposalexecutor.Service
	workerRepo          workerports.WorkerRepository
}

func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.dispatch-auto-assign"),
		dispatchControlRepo: p.DispatchControlRepo,
		agentControlRepo:    p.AgentControlRepo,
		definitionRepo:      p.DefinitionRepo,
		proposalRepo:        p.ProposalRepo,
		planner:             p.Planner,
		runService:          p.AgentRunService,
		executor:            p.Executor,
		workerRepo:          p.WorkerRepo,
	}
}

func (s *Service) Plan(
	ctx context.Context,
	req *dispatchconsoleports.DispatchPlanRequest,
) (*dispatchconsoleports.DispatchPlan, error) {
	control, err := s.dispatchControlRepo.GetOrCreate(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
	)
	if err != nil {
		return nil, err
	}
	if !control.EnableAutoAssignment {
		return nil, errortypes.NewBusinessError(
			"Auto assignment is disabled for this organization",
		)
	}

	policy, err := s.loadPolicy(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	planned, err := s.planner.Plan(ctx, &dispatchplanports.Request{
		TenantInfo:   req.TenantInfo,
		WindowStart:  req.WindowStart,
		WindowEnd:    req.WindowEnd,
		FleetCodeIDs: req.FleetCodeIDs,
		MoveIDs:      req.MoveIDs,
	})
	if err != nil {
		return nil, err
	}

	plan := &dispatchconsoleports.DispatchPlan{
		Mode:        planned.Mode,
		GeneratedAt: planned.GeneratedAt,
		Assignments: make(
			[]*dispatchconsoleports.DispatchPlannedAssignment,
			0,
			len(planned.Pairings),
		),
		Uncovered:      planned.Uncovered,
		Tours:          planned.Tours,
		CostCents:      planned.CostCents,
		UncoveredCents: planned.UncoveredCents,
		ShadowMode:     policy.ShadowMode,
		AutonomyTier:   policy.Tier,
	}
	threshold := control.ConfidenceThreshold()
	for _, pairing := range planned.Pairings {
		plan.Assignments = append(plan.Assignments, plannedAssignmentFor(&plannedAssignmentParams{
			Pairing:    pairing,
			Tier:       policy.Tier,
			Threshold:  threshold,
			ShadowMode: policy.ShadowMode,
		}))
	}

	proposals, err := s.recordPlan(ctx, req, policy, plan)
	if err != nil {
		return nil, err
	}
	if req.Apply {
		s.applyAutoExecutable(ctx, req, plan, proposals)
	}

	return plan, nil
}

func (s *Service) recordPlan(
	ctx context.Context,
	req *dispatchconsoleports.DispatchPlanRequest,
	policy Policy,
	plan *dispatchconsoleports.DispatchPlan,
) ([]*agent.AgentProposal, error) {
	if len(plan.Assignments) == 0 {
		return nil, nil
	}

	run, err := s.runService.StartInline(ctx, &agentruntimeports.StartInlineAgentRunRequest{
		AgentType:         agent.TypeDispatchAssignment,
		AgentDefinitionID: policy.DefinitionID,
		SubjectType:       agent.SubjectShipmentMove,
		SubjectID:         plan.Assignments[0].MoveID,
		PromptVersion:     promptVersion,
		Summary: fmt.Sprintf(
			"Dispatch planning proposed coverage for %d move(s)",
			len(plan.Assignments),
		),
		Trigger:    triggerOf(req),
		TenantInfo: req.TenantInfo,
	}, actorFor(req))
	if err != nil {
		return nil, err
	}
	plan.RunID = run.ID

	proposals := make([]*agent.AgentProposal, 0, len(plan.Assignments))
	for _, planned := range plan.Assignments {
		toolParams, paramsErr := toolParamsFor(planned.Candidate)
		if paramsErr != nil {
			return nil, paramsErr
		}

		proposal := &agent.AgentProposal{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			RunID:          run.ID,
			ToolName:       assignMoveTool,
			ToolParams:     toolParams,
			Confidence:     planned.Confidence,
			Rationale:      planned.Rationale,
			Evidence:       evidenceFor(planned.Pairing),
			AutonomyTier:   policy.Tier,
			Status:         agent.ProposalStatusPending,
			TargetResource: string(permission.ResourceShipmentMove),
			TargetID:       planned.MoveID,
			TargetVersion:  planned.MoveVersion,
		}
		proposal.EgressClass = s.egressOf(proposal)
		proposals = append(proposals, proposal)
	}

	recorded, err := s.proposalRepo.ReplacePendingForTargets(
		ctx,
		&agentproposalports.ReplaceTargetedProposalsRequest{
			TenantInfo:     req.TenantInfo,
			ToolName:       assignMoveTool,
			TargetResource: string(permission.ResourceShipmentMove),
			Proposals:      proposals,
		},
	)
	if err != nil {
		return nil, err
	}
	for i, planned := range plan.Assignments {
		planned.ProposalID = recorded[i].ID
	}

	return recorded, nil
}

func triggerOf(req *dispatchconsoleports.DispatchPlanRequest) agent.RunTrigger {
	if req.Trigger.IsValid() {
		return req.Trigger
	}

	return agent.RunTriggerManual
}

func (s *Service) egressOf(proposal *agent.AgentProposal) agent.EgressClass {
	egress, ok := s.executor.CallEgress(proposal, nil)
	if !ok {
		s.l.Warn("the auto-assign tool is not registered; its proposal is held for review",
			zap.String("tool", proposal.ToolName))

		return ""
	}

	return egress
}

func (s *Service) applyAutoExecutable(
	ctx context.Context,
	req *dispatchconsoleports.DispatchPlanRequest,
	plan *dispatchconsoleports.DispatchPlan,
	proposals []*agent.AgentProposal,
) {
	if plan.ShadowMode {
		return
	}

	actor := actorFor(req)
	for i, planned := range plan.Assignments {
		if !planned.AutoExecutable || i >= len(proposals) {
			continue
		}

		if err := s.executor.Execute(ctx, proposals[i], nil, actor); err != nil {
			planned.AutoExecutable = false
			s.l.Warn(
				"auto-assign proposal could not be executed and remains pending",
				zap.String("moveId", planned.MoveID.String()),
				zap.String("workerId", planned.Candidate.WorkerID.String()),
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

func toolParamsFor(candidate *dispatchplanports.CandidateScore) (map[string]any, error) {
	params := assignMoveToolParams{
		ShipmentMoveID:  candidate.MoveID.String(),
		PrimaryWorkerID: candidate.WorkerID.String(),
		TractorID:       candidate.TractorID.String(),
	}
	if !candidate.TrailerID.IsNil() {
		params.TrailerID = candidate.TrailerID.String()
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

func actorFor(req *dispatchconsoleports.DispatchPlanRequest) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		UserID:         req.TenantInfo.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
}
