package agentdecisionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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

	if err := s.assertHumanCanApprove(ctx, actor); err != nil {
		return nil, err
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

func (s *Service) assertHumanCanApprove(
	ctx context.Context,
	actor *services.RequestActor,
) error {
	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceBillingQueue.String(),
		Operation:      permission.OpApprove,
	})
	if err != nil {
		return err
	}

	if !result.Allowed {
		return errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"You do not have permission to approve billing queue items",
		)
	}

	return nil
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
