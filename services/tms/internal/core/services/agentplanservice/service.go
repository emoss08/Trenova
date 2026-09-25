// Package agentplanservice decides a plan: several proposals from one run,
// approved once and run in order.
//
// The steps are decided through the same decision service that decides a
// proposal on its own, so each step still gets its own decision row, its own
// audit line, its own permission check at execution and its own entry in the
// trust ledger. What the plan adds is the order, and the stop: the first step
// whose write fails ends the plan, and the steps after it are skipped rather
// than run against a world the failed one was supposed to change.
package agentplanservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// stepDecider is the slice of the decision service a plan uses: decide one
// step and learn whether it ran, and signal the run once at the end.
type stepDecider interface {
	DecideWithOutcome(
		ctx context.Context,
		req *services.DecideAgentProposalRequest,
		actor *services.RequestActor,
	) (*services.DecisionOutcome, error)
	SignalRun(
		ctx context.Context,
		req *services.DecideAgentProposalRequest,
		decision *agent.AgentDecision,
		runID pulid.ID,
	) error
}

type shadowReader interface {
	Organization(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
	ForRun(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runID pulid.ID,
	) (agentshadow.Verdict, error)
}

type runReader interface {
	GetByID(ctx context.Context, req repositories.GetAgentRunByIDRequest) (*agent.AgentRun, error)
}

type threadReader interface {
	GetThread(ctx context.Context, req repositories.GetThreadRequest) (*conversation.Thread, error)
}

type definitionReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type agentAccessChecker interface {
	MayUseAgent(
		ctx context.Context,
		actor *services.RequestActor,
		definition *agentdefinition.Definition,
	) (bool, error)
}

type actionLogger interface {
	LogAction(params *services.LogActionParams, opts ...services.LogOption) error
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	Plans       repositories.AgentPlanRepository
	Proposals   repositories.AgentProposalRepository
	Decisions   services.AgentDecisionService
	Runs        repositories.AgentRunRepository
	Definitions repositories.AgentDefinitionRepository
	Permissions services.PermissionEngine
	// Conversations says whose conversation raised a plan, for a person
	// deciding their own.
	Conversations repositories.ConversationRepository `optional:"true"`
	Shadow        *agentshadow.Resolver
	AuditService  services.AuditService
	Activity      services.AgentActivityPublisher `optional:"true"`
	// Watchtower takes a decided plan off the feed.
	Watchtower services.WatchtowerProjector `optional:"true"`
	// FollowUps has the conversation that raised the plan report its outcome.
	FollowUps services.DecisionFollowUps `optional:"true"`
	// Previews says what the plan's steps would do, as the decider sees it,
	// so an approval approves what was shown and nothing that moved on.
	Previews services.ProposalPreviewService `optional:"true"`
	Metrics  *metrics.Registry               `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	plans      repositories.AgentPlanRepository
	proposals  repositories.AgentProposalRepository
	decisions  stepDecider
	runs       runReader
	threads    threadReader
	agents     definitionReader
	access     agentAccessChecker
	shadow     shadowReader
	audit      actionLogger
	activity   services.AgentActivityPublisher
	watchtower services.WatchtowerProjector
	followUps  services.DecisionFollowUps
	previews   services.ProposalPreviewService
	metrics    *metrics.ProposalPreview
}

func New(p Params) services.AgentPlanService {
	decider, _ := p.Decisions.(stepDecider)

	svc := &Service{
		l:          p.Logger.Named("service.agentplan"),
		plans:      p.Plans,
		proposals:  p.Proposals,
		decisions:  decider,
		shadow:     p.Shadow,
		audit:      p.AuditService,
		activity:   p.Activity,
		watchtower: p.Watchtower,
		followUps:  p.FollowUps,
		previews:   p.Previews,
	}
	if p.Metrics != nil {
		svc.metrics = p.Metrics.ProposalPreview
	}
	if p.Runs != nil {
		svc.runs = p.Runs
	}
	if p.Conversations != nil {
		svc.threads = p.Conversations
	}
	if p.Definitions != nil {
		svc.agents = p.Definitions
	}
	if p.Permissions != nil {
		svc.access = p.Permissions
	}

	return svc
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentPlanByIDRequest,
) (*agent.AgentPlan, error) {
	plan, err := s.plans.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	verdict, err := s.shadow.ForRun(ctx, req.TenantInfo, plan.RunID)
	if err != nil {
		return nil, err
	}
	if verdict.Shadow() {
		return nil, errortypes.NewNotFoundError("Agent plan not found")
	}

	return plan, nil
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentPlanConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentPlan], error) {
	shadow, err := s.shadow.Organization(ctx, req.Filter.TenantInfo)
	if err != nil {
		return nil, err
	}
	if shadow {
		return &pagination.CursorListResult[*agent.AgentPlan]{Items: []*agent.AgentPlan{}}, nil
	}
	req.ExcludeShadowDefinitions = true

	return s.plans.ListConnection(ctx, req)
}

// Decide approves or rejects every pending step of a plan.
//
// An approval runs the steps in order and stops at the first that fails; the
// plan records how far it got. A rejection rejects every pending step. Either
// way the plan is moved out of Pending first, conditionally, so two people
// deciding at once cannot both run it.
func (s *Service) Decide(
	ctx context.Context,
	req *services.DecideAgentPlanRequest,
	actor *services.RequestActor,
) (*agent.AgentPlan, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a human user can decide on agent plans",
		)
	}
	if req.Decision != agent.DecisionAccepted && req.Decision != agent.DecisionRejected {
		return nil, errortypes.NewValidationError(
			"decision",
			errortypes.ErrInvalid,
			"A plan is accepted or rejected as a whole; change a step by deciding it on its own",
		)
	}
	if s.decisions == nil {
		return nil, errortypes.NewBusinessError(
			"Plans cannot be decided: no decision service is wired",
		)
	}

	plan, err := s.plans.GetByID(ctx, repositories.GetAgentPlanByIDRequest{
		ID:         req.PlanID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = decidable(plan); err != nil {
		return nil, err
	}

	verdict, err := s.shadow.ForRun(ctx, req.TenantInfo, plan.RunID)
	if err != nil {
		return nil, err
	}
	if verdict.Shadow() {
		return nil, shadowRefusal(verdict)
	}

	steps, err := s.proposals.ListByPlan(ctx, repositories.ListAgentProposalsByPlanRequest{
		PlanID:     plan.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	shown, err := s.settlePreview(ctx, req, plan, steps, actor)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	claimed := agent.PlanStatusApproved
	if req.Decision == agent.DecisionRejected {
		claimed = agent.PlanStatusRejected
	}
	plan, err = s.plans.UpdateStatus(ctx, repositories.UpdateAgentPlanStatusRequest{
		ID:              plan.ID,
		TenantInfo:      req.TenantInfo,
		Status:          claimed,
		FromStatus:      agent.PlanStatusPending,
		DecidedByUserID: actor.UserID,
		DecidedAt:       now,
	})
	if err != nil {
		return nil, err
	}

	s.clearFromWatchtower(ctx, plan, req.TenantInfo)

	if req.Decision == agent.DecisionRejected {
		s.rejectSteps(ctx, req, steps, actor)
		s.logDecision(plan, actor, "Agent plan rejected")
		s.announce(ctx, plan, actor)
		s.followUp(ctx, plan, req.TenantInfo)

		return plan, nil
	}

	plan = s.runSteps(ctx, &stepRun{
		req:     req,
		plan:    plan,
		steps:   steps,
		actor:   actor,
		preview: shown,
	})
	s.logDecision(plan, actor, "Agent plan approved and executed")
	s.announce(ctx, plan, actor)
	s.followUp(ctx, plan, req.TenantInfo)

	return plan, nil
}

// DecideOwn decides a plan raised in one of the actor's own conversations,
// by an agent they may still use. Past those checks it is the same decision
// as Decide: every step is still decided, and its write still runs only if
// the actor may make it.
func (s *Service) DecideOwn(
	ctx context.Context,
	req *services.DecideAgentPlanRequest,
	actor *services.RequestActor,
) (*agent.AgentPlan, error) {
	if err := s.AssertOwnPlan(ctx, req.PlanID, req.TenantInfo, actor); err != nil {
		return nil, err
	}

	return s.Decide(ctx, req, actor)
}

// AssertOwnPlan refuses a plan not raised in one of the actor's own
// conversations, and one from an agent they may no longer use. Someone
// else's plan is not found rather than forbidden, the way someone else's
// conversation is. The agent checked is the one whose run raised the plan,
// which for a hand-off is the delegate, not the conversation's own agent.
func (s *Service) AssertOwnPlan(
	ctx context.Context,
	planID pulid.ID,
	tenant pagination.TenantInfo,
	actor *services.RequestActor,
) error {
	notYours := errortypes.NewNotFoundError(
		"That plan was not raised in one of your conversations",
	)
	if s.runs == nil || s.threads == nil || actor == nil || actor.UserID.IsNil() {
		return notYours
	}

	plan, err := s.plans.GetByID(ctx, repositories.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}

	run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         plan.RunID,
		TenantInfo: &tenant,
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
		TenantInfo: tenant,
	}); err != nil {
		if errortypes.IsNotFoundError(err) {
			return notYours
		}

		return err
	}

	return s.assertMayUseAgent(ctx, tenant, run, actor)
}

// assertMayUseAgent refuses a plan from an agent the actor may not use. An
// agent removed since, or a check that cannot be made, is a refusal.
func (s *Service) assertMayUseAgent(
	ctx context.Context,
	tenant pagination.TenantInfo,
	run *agent.AgentRun,
	actor *services.RequestActor,
) error {
	gone := errortypes.NewBusinessError(
		"The agent that raised this plan no longer exists, so the plan cannot be decided",
	)
	if run.AgentDefinitionID.IsNil() || s.agents == nil || s.access == nil {
		return gone
	}

	definition, err := s.agents.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         run.AgentDefinitionID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return gone
		}

		return err
	}

	allowed, err := s.access.MayUseAgent(ctx, actor, definition)
	if err != nil {
		return fmt.Errorf("check access to agent %s: %w", definition.ID, err)
	}
	if !allowed {
		return errortypes.NewAuthorizationError(
			"You do not have access to {0}, so you cannot decide its plan. "+
				"An administrator can give one of your roles access to it.",
			definition.Name,
		)
	}

	return nil
}

// followUp has the conversation that raised the plan report how it went,
// once, after every step has run or been skipped. Each step is decided with
// WithinPlan set, so the steps start none of their own.
func (s *Service) followUp(
	ctx context.Context,
	plan *agent.AgentPlan,
	tenant pagination.TenantInfo,
) {
	if s.followUps == nil || plan == nil {
		return
	}

	s.followUps.FollowUp(ctx, services.DecisionFollowUpRequest{
		TenantInfo: tenant,
		RunID:      plan.RunID,
		PlanID:     plan.ID,
	})
}

// clearFromWatchtower takes a decided plan off the feed. Its steps were
// never on it on their own, so nothing else needs clearing.
func (s *Service) clearFromWatchtower(
	ctx context.Context,
	plan *agent.AgentPlan,
	tenant pagination.TenantInfo,
) {
	if s.watchtower == nil || plan == nil {
		return
	}

	s.watchtower.Resolve(ctx, tenant, watchtower.SourceAgentPlan, plan.ID.String())
}

// announce tells connected clients the plan moved, so the queue drops it
// and the pane shows its steps ticking or where it stopped.
func (s *Service) announce(
	ctx context.Context,
	plan *agent.AgentPlan,
	actor *services.RequestActor,
) {
	if s.activity == nil || plan == nil {
		return
	}

	s.activity.PlanChanged(ctx, plan, actor.AuditActorOrSystem(), services.ActivityUpdated)
}

// stepRun is one approved plan being run: its steps, who approved them, and
// the preview they approved.
type stepRun struct {
	req     *services.DecideAgentPlanRequest
	plan    *agent.AgentPlan
	steps   []*agent.AgentProposal
	actor   *services.RequestActor
	preview *shownPlan
}

type recordKey struct {
	resource string
	id       pulid.ID
}

// runSteps decides each pending step in order and stops at the first whose
// write fails. A step that was already decided on its own, or that expired,
// is passed over rather than treated as a failure: it is not the plan's to
// run twice.
//
// A step on a record an earlier step already changed is run against the
// version that step left it at. Its own pin was taken before either ran, so
// comparing against it refused the second of two changes to one record every
// time, although the approver approved both together.
func (s *Service) runSteps(ctx context.Context, r *stepRun) *agent.AgentPlan {
	req, plan := r.req, r.plan
	completed := plan.CompletedSteps
	var last *agent.AgentDecision
	var stepReq *services.DecideAgentProposalRequest
	left := make(map[recordKey]int64, len(r.steps))

	for _, step := range r.steps {
		if step.Status != agent.ProposalStatusPending {
			if step.Status == agent.ProposalStatusExecuted {
				completed++
			}

			continue
		}

		key := recordKey{resource: step.TargetResource, id: step.TargetID}
		stepReq = &services.DecideAgentProposalRequest{
			ProposalID: step.ID,
			Decision:   agent.DecisionAccepted,
			ReasonCode: planReason(req.ReasonCode, plan.ID),
			TenantInfo: req.TenantInfo,
			WithinPlan: true,
		}
		if version, changed := left[key]; changed && step.TargetID.IsNotNil() {
			stepReq.ExpectedTargetVersion = &version
		}
		r.preview.annotate(stepReq, step.ID)

		outcome, err := s.decisions.DecideWithOutcome(ctx, stepReq, r.actor)
		if err == nil && outcome.ExecutionError != nil {
			err = outcome.ExecutionError
		}
		if err != nil {
			return s.failAt(ctx, req, plan, step.PlanStep, completed, err)
		}
		if step.TargetID.IsNotNil() && outcome.ExecutedTargetVersion != nil {
			left[key] = *outcome.ExecutedTargetVersion
		}
		last = outcome.Decision
		completed++
	}

	progressed, err := s.plans.RecordProgress(ctx, repositories.RecordAgentPlanProgressRequest{
		ID:             plan.ID,
		TenantInfo:     req.TenantInfo,
		Status:         agent.PlanStatusCompleted,
		CompletedSteps: completed,
	})
	if err != nil {
		s.l.Error("plan ran but its completion could not be recorded",
			zap.String("plan", plan.ID.String()), zap.Error(err))
		progressed = plan
	}

	if last != nil && stepReq != nil {
		if err = s.decisions.SignalRun(ctx, stepReq, last, plan.RunID); err != nil {
			s.l.Error("failed to signal agent workflow for plan", zap.Error(err))
		}
	}

	return progressed
}

func (s *Service) failAt(
	ctx context.Context,
	req *services.DecideAgentPlanRequest,
	plan *agent.AgentPlan,
	step int,
	completed int,
	cause error,
) *agent.AgentPlan {
	s.l.Warn("agent plan stopped at a failed step",
		zap.String("plan", plan.ID.String()),
		zap.Int("step", step),
		zap.Error(cause),
	)

	if _, err := s.proposals.SkipPendingByPlan(ctx, repositories.SkipPendingByPlanRequest{
		PlanID:     plan.ID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		s.l.Error(
			"failed to skip the steps after a failed one",
			zap.String("plan", plan.ID.String()),
			zap.Error(err),
		)
	}

	failedStep := step
	progressed, err := s.plans.RecordProgress(ctx, repositories.RecordAgentPlanProgressRequest{
		ID:             plan.ID,
		TenantInfo:     req.TenantInfo,
		Status:         agent.PlanStatusFailed,
		CompletedSteps: completed,
		FailedStep:     &failedStep,
		FailureError:   cause.Error(),
	})
	if err != nil {
		s.l.Error(
			"failed to record the plan's failure",
			zap.String("plan", plan.ID.String()),
			zap.Error(err),
		)

		return plan
	}

	return progressed
}

func (s *Service) rejectSteps(
	ctx context.Context,
	req *services.DecideAgentPlanRequest,
	steps []*agent.AgentProposal,
	actor *services.RequestActor,
) {
	var last *agent.AgentDecision
	var stepReq *services.DecideAgentProposalRequest
	for _, step := range steps {
		if step.Status != agent.ProposalStatusPending {
			continue
		}
		stepReq = &services.DecideAgentProposalRequest{
			ProposalID: step.ID,
			Decision:   agent.DecisionRejected,
			ReasonCode: planReason(req.ReasonCode, req.PlanID),
			TenantInfo: req.TenantInfo,
			WithinPlan: true,
		}
		outcome, err := s.decisions.DecideWithOutcome(ctx, stepReq, actor)
		if err != nil {
			s.l.Error("failed to reject a plan step",
				zap.String("proposal", step.ID.String()), zap.Error(err))

			continue
		}
		last = outcome.Decision
	}

	if last != nil && stepReq != nil {
		if err := s.decisions.SignalRun(ctx, stepReq, last, steps[0].RunID); err != nil {
			s.l.Error("failed to signal agent workflow for rejected plan", zap.Error(err))
		}
	}
}

func (s *Service) logDecision(plan *agent.AgentPlan, actor *services.RequestActor, comment string) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActor()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentProposal,
		ResourceID:     plan.ID.String(),
		Operation:      permission.OpApprove,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(plan),
		OrganizationID: plan.OrganizationID,
		BusinessUnitID: plan.BusinessUnitID,
		Critical:       true,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log agent plan audit", zap.Error(err))
	}
}

// planReason keeps the person's reason and names the plan, so a step's
// decision row says it was made as part of a whole.
func planReason(reason string, planID pulid.ID) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "decided_as_plan"
	}

	return reason + " (plan " + planID.String() + ")"
}

func decidable(plan *agent.AgentPlan) error {
	if !plan.Status.Decidable() {
		return errortypes.NewBusinessError(
			"This plan has already been decided: it is {0}",
			strings.ToLower(string(plan.Status)),
		)
	}
	if plan.Expired(timeutils.NowUnix()) {
		return errortypes.NewBusinessError(
			"This plan expired on {0} without a decision. Ask the agent again for a current one",
			timeutils.DescribeUnixDate(plan.ExpiresAt, timeutils.NowUnix()),
		)
	}

	return nil
}

func shadowRefusal(verdict agentshadow.Verdict) error {
	if verdict.Cause == agentshadow.CauseDefinition {
		return errortypes.NewBusinessError(
			"Plans from {0} cannot be actioned while it is in shadow mode. "+
				"Turn off shadow mode on that agent in AI Control, then decide again",
			verdict.AgentName,
		)
	}

	return errortypes.NewBusinessError(
		"Plans cannot be actioned while all agents are paused. " +
			"Turn off \"Pause all agents\" on the AI Control overview, then decide again",
	)
}
