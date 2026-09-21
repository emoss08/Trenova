package agentdecisionservice

import (
	"context"
	"github.com/emoss08/trenova/shared/timeutils"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
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
	Shadow       *agentshadow.Resolver
	Permissions  services.PermissionEngine
	Workflows    services.WorkflowStarter
	Executor     *proposalexecutor.Service
	AuditService services.AuditService
	Trust        services.AgentTrustService
}

type Service struct {
	l            *zap.Logger
	decisionRepo repositories.AgentDecisionRepository
	proposalRepo repositories.AgentProposalRepository
	runRepo      repositories.AgentRunRepository
	shadow       *agentshadow.Resolver
	permissions  services.PermissionEngine
	workflows    services.WorkflowStarter
	executor     *proposalexecutor.Service
	audit        services.AuditService
	trust        services.AgentTrustService
}

func New(p Params) services.AgentDecisionService {
	return &Service{
		l:            p.Logger.Named("service.agentdecision"),
		decisionRepo: p.DecisionRepo,
		proposalRepo: p.ProposalRepo,
		runRepo:      p.RunRepo,
		shadow:       p.Shadow,
		permissions:  p.Permissions,
		workflows:    p.Workflows,
		executor:     p.Executor,
		audit:        p.AuditService,
		trust:        p.Trust,
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

	proposal, err := s.proposalRepo.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         req.ProposalID,
		TenantInfo: &req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if err = decidable(proposal); err != nil {
		return nil, err
	}

	verdict, err := s.shadow.ForRun(ctx, req.TenantInfo, proposal.RunID)
	if err != nil {
		return nil, err
	}
	if verdict.Shadow() {
		return nil, shadowRefusal(verdict)
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

	// Conditional on the proposal still being pending. The check above reads a
	// snapshot; two decisions racing each other both pass it, and without this
	// guard both would execute the tool. The loser now fails on the update
	// instead, with a conflict rather than a second write.
	if _, err = s.proposalRepo.UpdateStatus(ctx, repositories.UpdateAgentProposalStatusRequest{
		ID:         proposal.ID,
		Status:     proposalStatusFor(req.Decision),
		FromStatus: agent.ProposalStatusPending,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	if err = s.signalWorkflow(ctx, proposal.RunID, req, created); err != nil {
		s.l.Error("failed to signal agent workflow", zap.Error(err))
	}

	// An approval that does not act is worse than no approval: the audit trail
	// would say a person authorized something that never happened.
	execErr := s.executeIfApproved(ctx, proposal, req, actor)

	// The ledger learns from the outcome, not the intent: an approval whose
	// write failed is a setback for the tool, whatever the person decided.
	s.recordTrust(ctx, proposal, created, execErr)

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

// shadowRefusal names the switch that is on and where it lives.
//
// The old message said "the agent is in shadow mode" whichever switch it was,
// and the organization-wide pause is on from the day an organization is
// created. People turned shadow off on every agent they had, read the same
// message again, and had no way to learn that the switch they needed was on
// the overview tab under a different name.
func shadowRefusal(verdict agentshadow.Verdict) error {
	if verdict.Cause == agentshadow.CauseDefinition {
		return errortypes.NewBusinessError(
			"Proposals from {0} cannot be actioned while it is in shadow mode. "+
				"Turn off shadow mode on that agent in AI Control, then decide again",
			verdict.AgentName,
		)
	}

	return errortypes.NewBusinessError(
		"Proposals cannot be actioned while all agents are paused. " +
			"Turn off \"Pause all agents\" on the AI Control overview, then decide again",
	)
}

// decidable refuses a proposal that is no longer waiting on anyone.
//
// A decision used to be recorded against whatever state the proposal was in:
// a second click, a retry, another tab, or an API client could accept a
// proposal already rejected, or approve an approved one again and execute its
// write twice. The status is the one fact that says whether there is still a
// decision to make.
func decidable(proposal *agent.AgentProposal) error {
	if proposal.Status != agent.ProposalStatusPending {
		return errortypes.NewBusinessError(
			"This proposal has already been decided: it is {0}",
			strings.ToLower(string(proposal.Status)),
		)
	}

	// The sweeper marks these Expired every quarter hour; between sweeps the
	// clock is the authority, so a proposal is never approved in the minutes
	// after its window closed just because the row had not caught up.
	if proposal.Expired(timeutils.NowUnix()) {
		return errortypes.NewBusinessError(
			"This proposal expired on {0} without a decision. Ask the agent again for a current one",
			timeutils.DescribeUnixDate(proposal.ExpiresAt, timeutils.NowUnix()),
		)
	}

	return nil
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
) error {
	if req.Decision != agent.DecisionAccepted && req.Decision != agent.DecisionModified {
		return nil
	}

	err := s.executor.Execute(ctx, proposal, req.Modifications, actor)
	if err != nil {
		s.l.Error("approved proposal did not execute",
			zap.String("proposal", proposal.ID.String()),
			zap.String("tool", proposal.ToolName),
			zap.Error(err),
		)
	}

	return err
}

// recordTrust is best effort: the decision is the durable fact, and a ledger
// that could not be written is logged rather than allowed to fail it.
func (s *Service) recordTrust(
	ctx context.Context,
	proposal *agent.AgentProposal,
	decision *agent.AgentDecision,
	execErr error,
) {
	if s.trust == nil {
		return
	}

	var err error
	if execErr != nil {
		err = s.trust.RecordExecutionFailure(ctx, proposal)
	} else {
		err = s.trust.RecordDecision(ctx, proposal, decision)
	}
	if err != nil {
		s.l.Error("failed to record decision in the trust ledger",
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
