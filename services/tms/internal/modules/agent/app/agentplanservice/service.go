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
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// stepDecider is the slice of the decision service a plan uses: decide one
// step and learn whether it ran, and signal the run once at the end.
type stepDecider interface {
	DecideWithOutcome(
		ctx context.Context,
		req *agentdecisionports.DecideAgentProposalRequest,
		actor *permissionports.RequestActor,
	) (*agentdecisionports.DecisionOutcome, error)
	SignalRun(
		ctx context.Context,
		req *agentdecisionports.DecideAgentProposalRequest,
		decision *agentdecision.AgentDecision,
		runID pulid.ID,
	) error
}

type shadowReader interface {
	ForRun(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runID pulid.ID,
	) (agentshadow.Verdict, error)
	ForRuns(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runIDs []pulid.ID,
	) (map[pulid.ID]agentshadow.Verdict, error)
}

type runReader interface {
	GetByID(
		ctx context.Context,
		req agentruntimeports.GetAgentRunByIDRequest,
	) (*agent.AgentRun, error)
}

type threadReader interface {
	GetThread(
		ctx context.Context,
		req conversationports.GetThreadRequest,
	) (*conversation.Thread, error)
}

type definitionReader interface {
	GetByID(
		ctx context.Context,
		req agentdefinitionports.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type agentAccessChecker interface {
	MayUseAgent(
		ctx context.Context,
		actor *permissionports.RequestActor,
		definition *agentdefinition.Definition,
	) (bool, error)
}

type actionLogger interface {
	LogAction(params *auditports.LogActionParams, opts ...auditports.LogOption) error
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	Plans       agentplanports.AgentPlanRepository
	Proposals   agentproposalports.AgentProposalRepository
	Decisions   agentdecisionports.AgentDecisionService
	Runs        agentruntimeports.AgentRunRepository
	Definitions agentdefinitionports.AgentDefinitionRepository
	Permissions permissionports.PermissionEngine
	// Conversations says whose conversation raised a plan, for a person
	// deciding their own.
	Conversations conversationports.ConversationRepository `optional:"true"`
	Shadow        *agentshadow.Resolver
	AuditService  auditports.AuditService
	Activity      agentactivityports.AgentActivityPublisher `optional:"true"`
	// Watchtower takes a decided plan off the feed.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
	// FollowUps has the conversation that raised the plan report its outcome.
	FollowUps assistantports.DecisionFollowUps `optional:"true"`
	// Previews says what the plan's steps would do, as the decider sees it,
	// so an approval approves what was shown and nothing that moved on.
	Previews agentruntimeports.ProposalPreviewService `optional:"true"`
	Metrics  *metrics.Registry                        `optional:"true"`
	// Workflows holds an approval from a person's own conversation for its
	// undo window. Without it an approval goes through at once.
	Workflows kernelports.WorkflowStarter `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	plans      agentplanports.AgentPlanRepository
	proposals  agentproposalports.AgentProposalRepository
	decisions  stepDecider
	runs       runReader
	threads    threadReader
	agents     definitionReader
	access     agentAccessChecker
	shadow     shadowReader
	audit      actionLogger
	activity   agentactivityports.AgentActivityPublisher
	watchtower watchtowerports.WatchtowerProjector
	followUps  assistantports.DecisionFollowUps
	previews   agentruntimeports.ProposalPreviewService
	metrics    *aimetrics.ProposalPreview
	workflows  kernelports.WorkflowStarter
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) agentplanports.AgentPlanService {
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
		workflows:  p.Workflows,
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
	req agentplanports.GetAgentPlanByIDRequest,
) (*agentplan.AgentPlan, error) {
	return s.plans.GetByID(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *agentplanports.ListAgentPlanConnectionRequest,
) (*pagination.CursorListResult[*agentplan.AgentPlan], error) {
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
	req *agentplanports.DecideAgentPlanRequest,
	actor *permissionports.RequestActor,
) (*agentplan.AgentPlan, error) {
	prepared, err := s.prepare(ctx, req, actor, agentplan.PlanStatusPending)
	if err != nil {
		return nil, err
	}

	return s.claimAndRun(ctx, req, actor, prepared, agentplan.PlanStatusPending)
}

// preparedPlan is a plan checked for deciding: its steps and, for an
// approval, the preview the decider approved.
type preparedPlan struct {
	plan  *agentplan.AgentPlan
	steps []*agent.AgentProposal
	shown *shownPlan
}

// prepare checks a plan can be decided as asked, by this actor, as the world
// is now. from is the status the plan must be in: Pending for a decision,
// Approving for an approval whose undo window has closed.
func (s *Service) prepare(
	ctx context.Context,
	req *agentplanports.DecideAgentPlanRequest,
	actor *permissionports.RequestActor,
	from agentplan.PlanStatus,
) (*preparedPlan, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a human user can decide on agent plans",
		)
	}
	if req.Decision != agentdecision.DecisionAccepted &&
		req.Decision != agentdecision.DecisionRejected {
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
	if err := agentdecision.CheckDecisionNote(
		agentdecision.NormalizeDecisionNote(req.Note),
	); err != nil {
		return nil, err
	}

	plan, err := s.plans.GetByID(ctx, agentplanports.GetAgentPlanByIDRequest{
		ID:         req.PlanID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if from == agentplan.PlanStatusPending {
		if err = decidable(plan); err != nil {
			return nil, err
		}
	} else if plan.Status != from {
		return nil, errortypes.NewConflictError(
			"This plan is {0}, not waiting to go through",
			strings.ToLower(string(plan.Status)),
		)
	}

	verdict, err := s.shadow.ForRun(ctx, req.TenantInfo, plan.RunID)
	if err != nil {
		return nil, err
	}
	if verdict.Shadow() {
		return nil, shadowRefusal(verdict)
	}

	steps, err := s.proposals.ListByPlan(ctx, agentproposalports.ListAgentProposalsByPlanRequest{
		PlanID:     plan.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = s.refuseShadowedSteps(ctx, req.TenantInfo, plan, steps); err != nil {
		return nil, err
	}

	shown, err := s.settlePreview(ctx, req, plan, steps, actor)
	if err != nil {
		return nil, err
	}

	return &preparedPlan{plan: plan, steps: steps, shown: shown}, nil
}

// claimAndRun moves the plan out of from, conditionally, so two people
// deciding at once cannot both run it, and then rejects or runs its steps.
func (s *Service) claimAndRun(
	ctx context.Context,
	req *agentplanports.DecideAgentPlanRequest,
	actor *permissionports.RequestActor,
	prepared *preparedPlan,
	from agentplan.PlanStatus,
) (*agentplan.AgentPlan, error) {
	now := timeutils.NowUnix()
	claimed := agentplan.PlanStatusApproved
	if req.Decision == agentdecision.DecisionRejected {
		claimed = agentplan.PlanStatusRejected
	}
	plan, err := s.plans.UpdateStatus(ctx, agentplanports.UpdateAgentPlanStatusRequest{
		ID:              prepared.plan.ID,
		TenantInfo:      req.TenantInfo,
		Status:          claimed,
		FromStatus:      from,
		DecidedByUserID: actor.UserID,
		DecidedAt:       now,
		Approval: agentplanports.PlanApproval{
			ReasonCode:    req.ReasonCode,
			Note:          agentdecision.NormalizeDecisionNote(req.Note),
			PreviewDigest: prepared.settledDigest(from),
		},
	})
	if err != nil {
		return nil, err
	}

	s.clearFromWatchtower(ctx, plan, req.TenantInfo)

	if req.Decision == agentdecision.DecisionRejected {
		s.rejectSteps(ctx, req, prepared.steps, actor)
		s.logDecision(plan, prepared.steps, actor, "Agent plan rejected")
		s.announce(ctx, plan, actor, agentactivityports.ActivityUpdated)
		s.followUp(ctx, plan, req.TenantInfo)

		return plan, nil
	}

	run := &stepRun{
		req:     req,
		plan:    plan,
		steps:   prepared.steps,
		actor:   actor,
		preview: prepared.shown,
	}
	comment := "Agent plan approved and executed"
	if prepared.shown.stale() {
		plan = s.refuseStale(ctx, run)
		comment = "Agent plan approval refused: a record it changes moved on since it was proposed"
	} else {
		plan = s.runSteps(ctx, run)
	}
	s.logDecision(plan, prepared.steps, actor, comment)
	action := agentactivityports.ActivityUpdated
	if from == agentplan.PlanStatusApproving {
		action = agentactivityports.ActivityCommitted
	}
	s.announce(ctx, plan, actor, action)
	s.followUp(ctx, plan, req.TenantInfo)

	return plan, nil
}

// DecideOwn decides a plan raised in one of the actor's own conversations,
// by an agent they may still use. Past those checks it is the same decision
// as Decide: every step is still decided, and its write still runs only if
// the actor may make it.
func (s *Service) DecideOwn(
	ctx context.Context,
	req *agentplanports.DecideAgentPlanRequest,
	actor *permissionports.RequestActor,
) (*agentplan.AgentPlan, error) {
	if err := s.AssertOwnPlan(ctx, req.PlanID, req.TenantInfo, actor); err != nil {
		return nil, err
	}

	// An approval from the person's own conversation waits out the undo
	// window; no step runs until the window closes.
	if req.Decision == agentdecision.DecisionAccepted && s.defers() {
		return s.approveDeferred(ctx, req, actor)
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
	actor *permissionports.RequestActor,
) error {
	notYours := errortypes.NewNotFoundError(
		"That plan was not raised in one of your conversations",
	)
	if s.runs == nil || s.threads == nil || actor == nil || actor.UserID.IsNil() {
		return notYours
	}

	plan, err := s.plans.GetByID(ctx, agentplanports.GetAgentPlanByIDRequest{
		ID:         planID,
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}

	run, err := s.runs.GetByID(ctx, agentruntimeports.GetAgentRunByIDRequest{
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

	if _, err = s.threads.GetThread(ctx, conversationports.GetThreadRequest{
		ID:         run.SubjectID,
		UserID:     actor.UserID,
		TenantInfo: tenant,
	}); err != nil {
		if errortypes.IsNotFoundError(err) {
			return notYours
		}

		return err
	}

	runs, err := s.stepRuns(ctx, tenant, plan, run)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return notYours
		}

		return err
	}

	checked := make(map[pulid.ID]struct{}, len(runs))
	for _, stepRun := range runs {
		if stepRun.SubjectType != run.SubjectType || stepRun.SubjectID != run.SubjectID {
			return notYours
		}
		if _, seen := checked[stepRun.AgentDefinitionID]; seen {
			continue
		}
		checked[stepRun.AgentDefinitionID] = struct{}{}
		if err = s.assertMayUseAgent(ctx, tenant, stepRun, actor); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) stepRuns(
	ctx context.Context,
	tenant pagination.TenantInfo,
	plan *agentplan.AgentPlan,
	planRun *agent.AgentRun,
) ([]*agent.AgentRun, error) {
	steps, err := s.proposals.ListByPlan(ctx, agentproposalports.ListAgentProposalsByPlanRequest{
		PlanID:     plan.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	runs := make([]*agent.AgentRun, 0, len(steps)+1)
	runs = append(runs, planRun)
	for _, runID := range otherStepRuns(plan, steps) {
		stepRun, runErr := s.runs.GetByID(ctx, agentruntimeports.GetAgentRunByIDRequest{
			ID:         runID,
			TenantInfo: &tenant,
		})
		if runErr != nil {
			return nil, runErr
		}
		runs = append(runs, stepRun)
	}

	return runs, nil
}

func otherStepRuns(plan *agentplan.AgentPlan, steps []*agent.AgentProposal) []pulid.ID {
	seen := map[pulid.ID]struct{}{plan.RunID: {}}
	runs := make([]pulid.ID, 0, len(steps))
	for _, step := range steps {
		if step == nil {
			continue
		}
		if _, ok := seen[step.RunID]; ok {
			continue
		}
		seen[step.RunID] = struct{}{}
		runs = append(runs, step.RunID)
	}

	return runs
}

func (s *Service) refuseShadowedSteps(
	ctx context.Context,
	tenant pagination.TenantInfo,
	plan *agentplan.AgentPlan,
	steps []*agent.AgentProposal,
) error {
	stepRuns := otherStepRuns(plan, steps)
	verdicts, err := s.shadow.ForRuns(ctx, tenant, append([]pulid.ID{plan.RunID}, stepRuns...))
	if err != nil {
		return err
	}
	if verdict := agentshadow.PlanVerdict(verdicts, plan.RunID, stepRuns); verdict.Shadow() {
		return shadowRefusal(verdict)
	}

	return nil
}

func (s *Service) PlanHolds(
	ctx context.Context,
	tenant pagination.TenantInfo,
	plans []*agentplan.AgentPlan,
) (map[pulid.ID]*assistantproposalports.ProposalHold, error) {
	holds := make(map[pulid.ID]*assistantproposalports.ProposalHold, len(plans))
	waiting := make([]pulid.ID, 0, len(plans))
	for _, plan := range plans {
		if plan != nil && plan.Status == agentplan.PlanStatusPending {
			waiting = append(waiting, plan.ID)
		}
	}
	if len(waiting) == 0 {
		return holds, nil
	}

	steps, err := s.proposals.ListByPlans(ctx, agentproposalports.ListAgentProposalsByPlansRequest{
		PlanIDs:    waiting,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	stepsByPlan := make(map[pulid.ID][]*agent.AgentProposal, len(waiting))
	for _, step := range steps {
		if step.PlanID != nil {
			stepsByPlan[*step.PlanID] = append(stepsByPlan[*step.PlanID], step)
		}
	}

	runs := make([]pulid.ID, 0, len(waiting))
	stepRuns := make(map[pulid.ID][]pulid.ID, len(waiting))
	for _, plan := range plans {
		if plan == nil || plan.Status != agentplan.PlanStatusPending {
			continue
		}
		stepRuns[plan.ID] = otherStepRuns(plan, stepsByPlan[plan.ID])
		runs = append(runs, plan.RunID)
		runs = append(runs, stepRuns[plan.ID]...)
	}
	verdicts, err := s.shadow.ForRuns(ctx, tenant, sliceutils.Dedupe(runs))
	if err != nil {
		return nil, err
	}

	for _, plan := range plans {
		if plan == nil || plan.Status != agentplan.PlanStatusPending {
			continue
		}
		hold := agentshadow.PlanVerdict(verdicts, plan.RunID, stepRuns[plan.ID]).Hold()
		if hold != nil {
			holds[plan.ID] = hold
		}
	}

	return holds, nil
}

// assertMayUseAgent refuses a plan from an agent the actor may not use. An
// agent removed since, or a check that cannot be made, is a refusal.
func (s *Service) assertMayUseAgent(
	ctx context.Context,
	tenant pagination.TenantInfo,
	run *agent.AgentRun,
	actor *permissionports.RequestActor,
) error {
	gone := errortypes.NewBusinessError(
		"The agent that raised this plan no longer exists, so the plan cannot be decided",
	)
	if run.AgentDefinitionID.IsNil() || s.agents == nil || s.access == nil {
		return gone
	}

	definition, err := s.agents.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
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
	plan *agentplan.AgentPlan,
	tenant pagination.TenantInfo,
) {
	if s.followUps == nil || plan == nil {
		return
	}

	s.followUps.FollowUp(ctx, assistantports.DecisionFollowUpRequest{
		TenantInfo: tenant,
		RunID:      plan.RunID,
		PlanID:     plan.ID,
	})
}

// clearFromWatchtower takes a decided plan off the feed. Its steps were
// never on it on their own, so nothing else needs clearing.
func (s *Service) clearFromWatchtower(
	ctx context.Context,
	plan *agentplan.AgentPlan,
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
	plan *agentplan.AgentPlan,
	actor *permissionports.RequestActor,
	action string,
) {
	if s.activity == nil || plan == nil {
		return
	}

	s.activity.PlanChanged(ctx, plan, actor.AuditActorOrSystem(), action)
}

// stepRun is one approved plan being run: its steps, who approved them, and
// the preview they approved.
type stepRun struct {
	req     *agentplanports.DecideAgentPlanRequest
	plan    *agentplan.AgentPlan
	steps   []*agent.AgentProposal
	actor   *permissionports.RequestActor
	preview *shownPlan
}

type recordKey struct {
	resource string
	id       pulid.ID
}

// runSteps decides each pending step in order and stops at the first whose
// write fails, unless the steps are independent. A step that was already
// decided on its own, or that expired, is passed over rather than treated as
// a failure: it is not the plan's to run twice.
//
// A step on a record an earlier step already changed is run against the
// version that step left it at. Its own pin was taken before either ran, so
// comparing against it refused the second of two changes to one record every
// time, although the approver approved both together.
func (s *Service) runSteps(ctx context.Context, r *stepRun) *agentplan.AgentPlan {
	if agentplan.PlanRunModeOf(r.steps) == agentplan.PlanRunModeEachStep {
		return s.runIndependent(ctx, r)
	}

	req, plan := r.req, r.plan
	completed := plan.CompletedSteps
	var last *agentdecision.AgentDecision
	var lastReq *agentdecisionports.DecideAgentProposalRequest
	left := make(map[recordKey]int64, len(r.steps))

	for _, step := range r.steps {
		if step.Status != agent.ProposalStatusPending {
			if step.Status == agent.ProposalStatusExecuted {
				completed++
			}

			continue
		}

		stepReq, outcome, err := s.runStep(ctx, r, step, left)
		if err != nil {
			return s.failAt(ctx, req, plan, step.PlanStep, completed, err)
		}
		last, lastReq = outcome.Decision, stepReq
		completed++
	}

	return s.complete(ctx, r, completed, last, lastReq)
}

type stepFailure struct {
	step  int
	cause error
}

func (s *Service) refuseStale(ctx context.Context, r *stepRun) *agentplan.AgentPlan {
	var last *agentdecision.AgentDecision
	var lastReq *agentdecisionports.DecideAgentProposalRequest
	refusals := make([]stepFailure, 0, 1)
	firstPending := 0

	for _, step := range r.steps {
		if step.Status != agent.ProposalStatusPending {
			continue
		}
		if firstPending == 0 {
			firstPending = step.PlanStep
		}
		if !r.preview.preview.StepFor(step.ID).IsStale() {
			continue
		}

		stepReq := &agentdecisionports.DecideAgentProposalRequest{
			ProposalID: step.ID,
			Decision:   agentdecision.DecisionAccepted,
			ReasonCode: planReason(r.req.ReasonCode, r.plan.ID),
			Note:       r.req.Note,
			TenantInfo: r.req.TenantInfo,
			WithinPlan: true,
		}
		r.preview.annotate(stepReq, step.ID)

		cause := agentplan.ErrTargetChanged
		outcome, err := s.decisions.DecideWithOutcome(ctx, stepReq, r.actor)
		switch {
		case err != nil:
			s.l.Error("a plan step refused over a changed record could not be recorded",
				zap.String("proposal", step.ID.String()), zap.Error(err))
		case outcome.ExecutionError != nil:
			cause = outcome.ExecutionError
			last, lastReq = outcome.Decision, stepReq
		default:
			last, lastReq = outcome.Decision, stepReq
		}
		refusals = append(refusals, stepFailure{step: step.PlanStep, cause: cause})
	}
	if len(refusals) == 0 {
		refusals = append(
			refusals,
			stepFailure{step: firstPending, cause: agentplan.ErrTargetChanged},
		)
	}

	s.signal(ctx, r.plan, last, lastReq)

	return s.failAt(ctx, r.req, r.plan, refusals[0].step, r.plan.CompletedSteps,
		staleFailure(refusals))
}

func staleFailure(refusals []stepFailure) error {
	var b strings.Builder
	b.WriteString(agentplan.ErrPlanTargetChanged.Error())
	for idx, refusal := range refusals {
		if idx == maxReportedStepFailures {
			fmt.Fprintf(&b, "; and %d more", len(refusals)-idx)

			break
		}
		fmt.Fprintf(&b, "; step %d: %s", refusal.step, refusal.cause.Error())
	}

	return errors.New(b.String())
}

func (s *Service) runIndependent(ctx context.Context, r *stepRun) *agentplan.AgentPlan {
	completed := r.plan.CompletedSteps
	var last *agentdecision.AgentDecision
	var lastReq *agentdecisionports.DecideAgentProposalRequest
	failures := make([]stepFailure, 0, len(r.steps))
	left := make(map[recordKey]int64, len(r.steps))

	for _, step := range r.steps {
		if step.Status != agent.ProposalStatusPending {
			if step.Status == agent.ProposalStatusExecuted {
				completed++
			}

			continue
		}

		stepReq, outcome, err := s.runStep(ctx, r, step, left)
		if err != nil {
			failures = append(failures, stepFailure{step: step.PlanStep, cause: err})
			s.l.Warn("an independent plan step failed; the others still run",
				zap.String("plan", r.plan.ID.String()),
				zap.Int("step", step.PlanStep),
				zap.Error(err),
			)

			continue
		}
		last, lastReq = outcome.Decision, stepReq
		completed++
	}

	if len(failures) == 0 {
		return s.complete(ctx, r, completed, last, lastReq)
	}

	s.signal(ctx, r.plan, last, lastReq)

	return s.failAt(ctx, r.req, r.plan, failures[0].step, completed,
		independentFailure(failures, len(r.steps)))
}

const maxReportedStepFailures = 5

func independentFailure(failures []stepFailure, steps int) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%d of %d steps did not run", len(failures), steps)
	for idx, failure := range failures {
		if idx == maxReportedStepFailures {
			fmt.Fprintf(&b, "; and %d more", len(failures)-idx)

			break
		}
		fmt.Fprintf(&b, "; step %d: %s", failure.step, failure.cause.Error())
	}

	return errors.New(b.String())
}

func (s *Service) runStep(
	ctx context.Context,
	r *stepRun,
	step *agent.AgentProposal,
	left map[recordKey]int64,
) (*agentdecisionports.DecideAgentProposalRequest, *agentdecisionports.DecisionOutcome, error) {
	key := recordKey{resource: step.TargetResource, id: step.TargetID}
	stepReq := &agentdecisionports.DecideAgentProposalRequest{
		ProposalID: step.ID,
		Decision:   agentdecision.DecisionAccepted,
		ReasonCode: planReason(r.req.ReasonCode, r.plan.ID),
		Note:       r.req.Note,
		TenantInfo: r.req.TenantInfo,
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
		return nil, nil, err
	}
	if step.TargetID.IsNotNil() && outcome.ExecutedTargetVersion != nil {
		left[key] = *outcome.ExecutedTargetVersion
	}

	return stepReq, outcome, nil
}

func (s *Service) complete(
	ctx context.Context,
	r *stepRun,
	completed int,
	last *agentdecision.AgentDecision,
	lastReq *agentdecisionports.DecideAgentProposalRequest,
) *agentplan.AgentPlan {
	progressed, err := s.plans.RecordProgress(ctx, agentplanports.RecordAgentPlanProgressRequest{
		ID:             r.plan.ID,
		TenantInfo:     r.req.TenantInfo,
		Status:         agentplan.PlanStatusCompleted,
		CompletedSteps: completed,
	})
	if err != nil {
		s.l.Error("plan ran but its completion could not be recorded",
			zap.String("plan", r.plan.ID.String()), zap.Error(err))
		progressed = r.plan
	}

	s.signal(ctx, r.plan, last, lastReq)

	return progressed
}

func (s *Service) signal(
	ctx context.Context,
	plan *agentplan.AgentPlan,
	last *agentdecision.AgentDecision,
	lastReq *agentdecisionports.DecideAgentProposalRequest,
) {
	if last == nil || lastReq == nil {
		return
	}
	if err := s.decisions.SignalRun(ctx, lastReq, last, plan.RunID); err != nil {
		s.l.Error("failed to signal agent workflow for plan", zap.Error(err))
	}
}

func (s *Service) failAt(
	ctx context.Context,
	req *agentplanports.DecideAgentPlanRequest,
	plan *agentplan.AgentPlan,
	step int,
	completed int,
	cause error,
) *agentplan.AgentPlan {
	s.l.Warn("agent plan stopped at a failed step",
		zap.String("plan", plan.ID.String()),
		zap.Int("step", step),
		zap.Error(cause),
	)

	if _, err := s.proposals.SkipPendingByPlan(ctx, agentproposalports.SkipPendingByPlanRequest{
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
	progressed, err := s.plans.RecordProgress(ctx, agentplanports.RecordAgentPlanProgressRequest{
		ID:             plan.ID,
		TenantInfo:     req.TenantInfo,
		Status:         agentplan.PlanStatusFailed,
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
	req *agentplanports.DecideAgentPlanRequest,
	steps []*agent.AgentProposal,
	actor *permissionports.RequestActor,
) {
	var last *agentdecision.AgentDecision
	var stepReq *agentdecisionports.DecideAgentProposalRequest
	for _, step := range steps {
		if step.Status != agent.ProposalStatusPending {
			continue
		}
		stepReq = &agentdecisionports.DecideAgentProposalRequest{
			ProposalID: step.ID,
			Decision:   agentdecision.DecisionRejected,
			ReasonCode: planReason(req.ReasonCode, req.PlanID),
			Note:       req.Note,
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

func (s *Service) logDecision(
	plan *agentplan.AgentPlan,
	steps []*agent.AgentProposal,
	actor *permissionports.RequestActor,
	comment string,
) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActor()
	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentProposal,
		ResourceID:     plan.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   decisionState(plan, steps),
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

func decidable(plan *agentplan.AgentPlan) error {
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

// decisionState is the plan as decided and, when any step was decided after
// its run read outside content, that content's marks: the trail says what
// may have steered the changes, a script's batch above all.
func decisionState(plan *agentplan.AgentPlan, steps []*agent.AgentProposal) map[string]any {
	state := jsonutils.MustToJSON(plan)
	tainted := false
	taint := &agent.RunTaint{}
	for _, step := range steps {
		if step != nil && step.Tainted {
			tainted = true
			taint.Merge(step.Taint)
		}
	}
	if !tainted {
		return state
	}
	state["tainted"] = true
	if taint.Tainted() {
		state["taint"] = jsonutils.MustToJSON(taint)
	}

	return state
}
