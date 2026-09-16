package agentdecisionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/proposalexecutor"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DecisionRepo repositories.AgentDecisionRepository
	ProposalRepo repositories.AgentProposalRepository
	RunRepo      repositories.AgentRunRepository
	Control      services.AgentControlService
	Permissions  services.PermissionEngine
	Workflows    services.WorkflowStarter
	Executor     *proposalexecutor.Service
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	decisionRepo repositories.AgentDecisionRepository
	proposalRepo repositories.AgentProposalRepository
	runRepo      repositories.AgentRunRepository
	control      services.AgentControlService
	permissions  services.PermissionEngine
	workflows    services.WorkflowStarter
	executor     *proposalexecutor.Service
	audit        services.AuditService
}

func New(p Params) services.AgentDecisionService {
	return &Service{
		l:            p.Logger.Named("service.agentdecision"),
		decisionRepo: p.DecisionRepo,
		proposalRepo: p.ProposalRepo,
		runRepo:      p.RunRepo,
		control:      p.Control,
		permissions:  p.Permissions,
		workflows:    p.Workflows,
		executor:     p.Executor,
		audit:        p.AuditService,
	}
}

func (s *Service) Decide(
	ctx context.Context,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) (*agent.AgentDecision, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a human user can decide on agent proposals",
		)
	}

	control, err := s.control.Get(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if control.ShadowMode {
		return nil, errortypes.NewBusinessError(
			"Agent proposals cannot be actioned while the organization is in shadow mode",
		)
	}

	proposal, err := s.proposalRepo.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         req.ProposalID,
		TenantInfo: &req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	decision := &agent.AgentDecision{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		ProposalID:      &proposal.ID,
		DecidedByUserID: actor.UserID,
		Decision:        req.Decision,
		Modifications:   req.Modifications,
		ReasonCode:      req.ReasonCode,
	}

	me := errortypes.NewMultiError()
	decision.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	created, err := s.decisionRepo.Create(ctx, decision)
	if err != nil {
		return nil, err
	}

	if _, err = s.proposalRepo.UpdateStatus(ctx, repositories.UpdateAgentProposalStatusRequest{
		ID:         proposal.ID,
		Status:     proposalStatusFor(req.Decision),
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	if err = s.signalWorkflow(ctx, proposal.RunID, req, created); err != nil {
		s.l.Error("failed to signal agent workflow", zap.Error(err))
	}

	// An approval that does not act is worse than no approval: the audit trail
	// would say a person authorized something that never happened.
	s.executeIfApproved(ctx, proposal, req, actor)

	auditActor := actor.AuditActor()
	if err = s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentProposal,
		ResourceID:     proposal.GetID().String(),
		Operation:      permission.OpApprove,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(created),
		OrganizationID: created.OrganizationID,
		BusinessUnitID: created.BusinessUnitID,
		Critical:       true,
	}, auditservice.WithComment("Human decision recorded for agent proposal")); err != nil {
		s.l.Error("failed to log agent decision audit", zap.Error(err))
	}

	return created, nil
}

// executeIfApproved runs the tool behind an accepted or modified proposal.
//
// Execution failure does not fail the decision. The person's judgement was
// recorded and is not invalidated by the write going wrong afterwards; the
// failure is stored on the proposal so they can see it and retry or escalate.
// Rolling the decision back would lose the one durable fact in the exchange.
func (s *Service) executeIfApproved(
	ctx context.Context,
	proposal *agent.AgentProposal,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) {
	if req.Decision != agent.DecisionAccepted && req.Decision != agent.DecisionModified {
		return
	}

	if err := s.executor.Execute(ctx, proposal, req.Modifications, actor); err != nil {
		s.l.Error("approved proposal did not execute",
			zap.String("proposal", proposal.ID.String()),
			zap.String("tool", proposal.ToolName),
			zap.Error(err),
		)
	}
}

func (s *Service) signalWorkflow(
	ctx context.Context,
	runID pulid.ID,
	req *services.DecideAgentProposalRequest,
	decision *agent.AgentDecision,
) error {
	run, err := s.runRepo.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if run.WorkflowID == "" {
		return nil
	}

	return s.workflows.SignalWorkflow(ctx, run.WorkflowID, "", agentjobs.AgentDecisionSignalName,
		agentjobs.DecisionSignal{
			ProposalID:      *decision.ProposalID,
			Decision:        decision.Decision,
			DecidedByUserID: decision.DecidedByUserID,
			ReasonCode:      decision.ReasonCode,
		})
}

func proposalStatusFor(decision agent.DecisionType) agent.ProposalStatus {
	switch decision {
	case agent.DecisionAccepted:
		return agent.ProposalStatusAccepted
	case agent.DecisionModified:
		return agent.ProposalStatusModified
	case agent.DecisionRejected:
		return agent.ProposalStatusRejected
	default:
		return agent.ProposalStatusRejected
	}
}
