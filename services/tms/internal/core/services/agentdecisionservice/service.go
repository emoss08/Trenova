package agentdecisionservice

import (
	"context"
	"github.com/emoss08/trenova/shared/timeutils"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/proposalexecutor"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
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
	// Conversations says whose conversation raised a proposal, for a person
	// deciding their own.
	Conversations repositories.ConversationRepository `optional:"true"`
	Shadow        *agentshadow.Resolver
	Permissions   services.PermissionEngine
	Workflows     services.WorkflowStarter
	Executor      *proposalexecutor.Service
	AuditService  services.AuditService
	Trust         services.AgentTrustService
	Memories      services.AgentMemoryService     `optional:"true"`
	Activity      services.AgentActivityPublisher `optional:"true"`
	// Watchtower takes a decided proposal off the feed.
	Watchtower services.WatchtowerProjector `optional:"true"`
	// FollowUps has the conversation that raised a proposal report what came
	// of deciding it.
	FollowUps services.DecisionFollowUps `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	decisionRepo repositories.AgentDecisionRepository
	proposalRepo repositories.AgentProposalRepository
	runRepo      repositories.AgentRunRepository
	threads      repositories.ConversationRepository
	shadow       *agentshadow.Resolver
	permissions  services.PermissionEngine
	workflows    services.WorkflowStarter
	executor     *proposalexecutor.Service
	audit        services.AuditService
	trust        services.AgentTrustService
	memories     services.AgentMemoryService
	activity     services.AgentActivityPublisher
	watchtower   services.WatchtowerProjector
	followUps    services.DecisionFollowUps
}

func New(p Params) services.AgentDecisionService {
	return &Service{
		l:            p.Logger.Named("service.agentdecision"),
		decisionRepo: p.DecisionRepo,
		proposalRepo: p.ProposalRepo,
		runRepo:      p.RunRepo,
		threads:      p.Conversations,
		shadow:       p.Shadow,
		permissions:  p.Permissions,
		workflows:    p.Workflows,
		executor:     p.Executor,
		audit:        p.AuditService,
		trust:        p.Trust,
		memories:     p.Memories,
		activity:     p.Activity,
		watchtower:   p.Watchtower,
		followUps:    p.FollowUps,
	}
}

func (s *Service) Decide(
	ctx context.Context,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) (*agent.AgentDecision, error) {
	outcome, err := s.DecideWithOutcome(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return outcome.Decision, nil
}

func (s *Service) DecideOwn(
	ctx context.Context,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) (*agent.AgentDecision, error) {
	if err := s.assertOwnProposal(ctx, req, actor); err != nil {
		return nil, err
	}

	return s.Decide(ctx, req, actor)
}

// assertOwnProposal refuses a proposal not raised in one of the actor's own
// conversations. Someone else's is not found rather than forbidden, the way
// someone else's conversation is.
func (s *Service) assertOwnProposal(
	ctx context.Context,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) error {
	notYours := errortypes.NewNotFoundError(
		"That proposal was not raised in one of your conversations",
	)
	if s.threads == nil || actor == nil || actor.UserID.IsNil() {
		return notYours
	}

	proposal, err := s.proposalRepo.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         req.ProposalID,
		TenantInfo: &req.TenantInfo,
	})
	if err != nil {
		return err
	}

	run, err := s.runRepo.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         proposal.RunID,
		TenantInfo: &req.TenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return notYours
		}

		return err
	}
	if run.SubjectType != agent.SubjectAssistantThread || run.SubjectID.IsNil() {
		return notYours
	}

	if _, err = s.threads.GetThread(ctx, repositories.GetThreadRequest{
		ID:         run.SubjectID,
		UserID:     actor.UserID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		if errortypes.IsNotFoundError(err) {
			return notYours
		}

		return err
	}

	return nil
}

func (s *Service) DecideWithOutcome(
	ctx context.Context,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) (*services.DecisionOutcome, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a human user can decide on agent proposals",
		)
	}
	if req.TenantInfo.OrgID != actor.OrganizationID || req.TenantInfo.BuID != actor.BusinessUnitID {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"A proposal can only be decided within the decider's own organization",
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

	req, err = s.settleModifications(ctx, proposal, req, actor)
	if err != nil {
		return nil, err
	}

	ctx, span := aitrace.StartDecide(ctx, &aitrace.DecideSpec{
		Operation:         aitrace.DecideOperationDecide,
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		ProposalID:        proposal.ID,
		RunID:             proposal.RunID,
		ToolName:          proposal.ToolName,
		Decision:          string(req.Decision),
		UserID:            actor.UserID,
		ReasonCode:        reasonCodeFor(req.Decision, req.ReasonCode),
		ModificationCount: len(req.Modifications),
		ProposalTraceID:   proposal.TraceID,
		ProposalSpanID:    proposal.SpanID,
	})
	defer span.End()

	decision := &agent.AgentDecision{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		ProposalID:      &proposal.ID,
		DecidedByUserID: actor.UserID,
		Decision:        req.Decision,
		Modifications:   req.Modifications,
		ReasonCode:      reasonCodeFor(req.Decision, req.ReasonCode),
	}
	decision.TraceID, _ = aitrace.IDs(ctx)

	me := errortypes.NewMultiError()
	decision.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	created, err := s.decisionRepo.Create(ctx, decision)
	if err != nil {
		aitrace.MarkFailed(span, aitrace.OutcomeFailed)

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
		aitrace.MarkFailed(span, aitrace.OutcomeFailed)

		return nil, err
	}

	s.clearFromWatchtower(ctx, proposal, req.TenantInfo)

	// A plan signals its run's workflow once, for all its steps; a step
	// signalling on its own would reach a workflow the first step already
	// released.
	if !req.WithinPlan {
		if err = s.signalWorkflow(ctx, proposal.RunID, req, created); err != nil {
			s.l.Error("failed to signal agent workflow", zap.Error(err))
		}
	}

	// An approval that does not act is worse than no approval: the audit trail
	// would say a person authorized something that never happened.
	execErr := s.executeIfApproved(ctx, proposal, req, actor)

	// The ledger learns from the outcome, not the intent: an approval whose
	// write failed is a setback for the tool, whatever the person decided.
	s.recordTrust(ctx, proposal, created, execErr)

	// A change or a refusal with a reason is the best correction an agent
	// can get, and the memory keeps it for the next run to read.
	s.recordCorrection(ctx, proposal, created)

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

	s.announce(ctx, proposal, req.TenantInfo, auditActor)

	// Last, once the change has run or failed: the report is of the outcome,
	// not of the click. A plan's steps are reported once, by the plan.
	if !req.WithinPlan && s.followUps != nil {
		s.followUps.FollowUp(ctx, services.DecisionFollowUpRequest{
			TenantInfo: req.TenantInfo,
			RunID:      proposal.RunID,
			ProposalID: proposal.ID,
		})
	}

	return &services.DecisionOutcome{Decision: created, ExecutionError: execErr}, nil
}

// settleModifications turns what the form sent back into what the decision
// records. A form returns every field; only the values that differ from
// the proposal are changes. With none, the person approved as proposed and
// the decision says so, since a "modified" decision with nothing modified
// would be a lie in the audit trail. With some, they are checked against
// the tool before anything is recorded, so a change that could not run is
// refused as a validation error rather than approved and then failed.
func (s *Service) settleModifications(
	ctx context.Context,
	proposal *agent.AgentProposal,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) (*services.DecideAgentProposalRequest, error) {
	if req.Decision != agent.DecisionModified {
		return req, nil
	}

	settled := *req
	settled.Modifications = toolschema.Changed(proposal.ToolParams, req.Modifications)
	if len(settled.Modifications) == 0 {
		settled.Decision = agent.DecisionAccepted
		settled.Modifications = nil

		return &settled, nil
	}

	if s.executor != nil {
		if _, err := s.executor.CheckModifications(ctx, proposal, settled.Modifications, actor); err != nil {
			return nil, err
		}
	}

	return &settled, nil
}

// SignalRun tells a run's workflow that its proposals were decided. The
// plan service calls it once after deciding every step.
func (s *Service) SignalRun(
	ctx context.Context,
	req *services.DecideAgentProposalRequest,
	decision *agent.AgentDecision,
	runID pulid.ID,
) error {
	return s.signalWorkflow(ctx, runID, req, decision)
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

func (s *Service) recordCorrection(
	ctx context.Context,
	proposal *agent.AgentProposal,
	decision *agent.AgentDecision,
) {
	if s.memories == nil {
		return
	}

	if _, err := s.memories.RecordCorrection(ctx, proposal, decision); err != nil {
		s.l.Error("failed to record a correction from the decision",
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

// reasonCodeFor fills in a code when the decision came without one. The
// chat card asks for a click, not a reason, and a decision refused for the
// lack of a reason nobody was asked for is a button that does nothing. A
// filled-in code is a code, not a reason: memory ignores it as such.
func reasonCodeFor(decision agent.DecisionType, reason string) string {
	if trimmed := strings.TrimSpace(reason); trimmed != "" {
		return trimmed
	}

	switch decision {
	case agent.DecisionAccepted:
		return "approved_without_reason"
	case agent.DecisionModified:
		return "modified_without_reason"
	default:
		return "rejected_without_reason"
	}
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

// announce publishes the proposal as it now is, after the decision and any
// execution, so a queue drops it and a pane shows the outcome. The record
// is re-read because execution moved its status past the decision.
func (s *Service) announce(
	ctx context.Context,
	proposal *agent.AgentProposal,
	tenant pagination.TenantInfo,
	actor services.AuditActor,
) {
	if s.activity == nil {
		return
	}

	current, err := s.proposalRepo.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         proposal.ID,
		TenantInfo: &tenant,
	})
	if err != nil {
		s.l.Warn("decided proposal could not be re-read for its announcement",
			zap.String("proposal", proposal.ID.String()), zap.Error(err))
		current = proposal
	}

	s.activity.ProposalChanged(ctx, current, actor, services.ActivityUpdated)
}

// clearFromWatchtower takes a decided proposal off the feed. It is called
// after the decision is recorded, whatever the outcome: approved, rejected
// or modified, it is no longer waiting on anyone.
func (s *Service) clearFromWatchtower(
	ctx context.Context,
	proposal *agent.AgentProposal,
	tenant pagination.TenantInfo,
) {
	if s.watchtower == nil || proposal == nil {
		return
	}

	s.watchtower.Resolve(ctx, tenant, watchtower.SourceAgentProposal, proposal.ID.String())
}
