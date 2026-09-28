package proposalrecorder

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxPlanTitleChars = 200

type RunOpener interface {
	Create(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
}

type ProposalStore interface {
	Create(ctx context.Context, entity *agent.AgentProposal) (*agent.AgentProposal, error)
}

// Transactor runs the run, the plan and its proposals as one write, so a
// failure part way leaves none of them behind.
type Transactor interface {
	WithTx(ctx context.Context, opts ports.TxOptions, fn func(context.Context, bun.Tx) error) error
}

type Params struct {
	fx.In

	Logger    *zap.Logger
	DB        ports.DBConnection
	Runs      repositories.AgentRunRepository
	Proposals repositories.AgentProposalRepository
	// Trust learns from a tool that ran on its own and failed. Optional, so
	// the recorder still stores proposals where no ledger is wired.
	Trust serviceports.AgentTrustService `optional:"true"`
	// Notifier tells the people who can decide that a background run left
	// something waiting. Optional for the same reason.
	Notifier serviceports.AgentProposalNotifier `optional:"true"`
	// Plans groups a run's pending proposals into one decision. Without it
	// every proposal stands on its own, which is how things worked before.
	Plans PlanStore `optional:"true"`
	// Activity tells connected clients a proposal or plan is waiting.
	Activity serviceports.AgentActivityPublisher `optional:"true"`
	// Watchtower puts what is waiting on a person onto the one feed they
	// read; without it the decision still stands, unannounced.
	Watchtower serviceports.WatchtowerProjector `optional:"true"`
}

// PlanStore is the one write the recorder makes on plans.
type PlanStore interface {
	Create(ctx context.Context, entity *agent.AgentPlan) (*agent.AgentPlan, error)
}

type Service struct {
	logger     *zap.Logger
	tx         Transactor
	runs       RunOpener
	proposals  ProposalStore
	plans      PlanStore
	trust      serviceports.AgentTrustService
	notifier   serviceports.AgentProposalNotifier
	activity   serviceports.AgentActivityPublisher
	watchtower serviceports.WatchtowerProjector
}

func New(p Params) *Service {
	svc := NewWithStores(p.Logger, p.Runs, p.Proposals)
	svc.tx = p.DB
	svc.trust = p.Trust
	svc.notifier = p.Notifier
	svc.plans = p.Plans
	svc.activity = p.Activity
	svc.watchtower = p.Watchtower

	return svc
}

// WithTransactor gives a recorder built with NewWithStores the transaction
// its writes run in. Without one each write stands on its own.
func (s *Service) WithTransactor(tx Transactor) *Service {
	s.tx = tx

	return s
}

// WithPlans gives a recorder built with NewWithStores a plan store.
func (s *Service) WithPlans(plans PlanStore) *Service {
	s.plans = plans

	return s
}

func NewWithStores(logger *zap.Logger, runs RunOpener, proposals ProposalStore) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		logger:    logger.Named("service.proposalrecorder"),
		runs:      runs,
		proposals: proposals,
	}
}

type OpenRunRequest struct {
	AgentType        agent.Type
	SubjectType      agent.SubjectType
	SubjectID        pulid.ID
	Trigger          agent.RunTrigger
	Status           agent.RunStatus
	Model            string
	PromptVersion    string
	InputContextHash string
	Summary          string
	Fingerprint      *agent.Fingerprint
	TraceID          string
	TurnID           pulid.ID
	ParentOwnerKind  agent.RunOwnerKind
	ParentOwnerID    pulid.ID
	DelegateCallID   string
}

type EvidenceFunc func(action serviceports.PendingAction, sourceMessageID pulid.ID) []agent.EvidenceRef

type RecordRequest struct {
	Actor      *serviceports.RequestActor
	Definition *agentdefinition.Definition
	// Run is the run the proposals belong to. When nil, one is opened from Open.
	Run              *agent.AgentRun
	Open             *OpenRunRequest
	Actions          []serviceports.PendingAction
	SourceMessageIDs map[string]pulid.ID
	Evidence         EvidenceFunc
	// Taint is the outside content the run read. The run keeps it, and each
	// proposal decided after the run had read some carries it.
	Taint         *agent.RunTaint
	Delegated     []DelegatedActions
	OpenEmptyRuns bool
	CallOrder     map[string]int
}

type DelegatedActions struct {
	Definition *agentdefinition.Definition
	Open       *OpenRunRequest
	Actions    []serviceports.PendingAction
	Taint      *agent.RunTaint
}

type RecordResult struct {
	Run       *agent.AgentRun
	Proposals []*agent.AgentProposal
	// Plan is set when the run's pending proposals were grouped into one.
	Plan      *agent.AgentPlan
	Delegated []DelegatedRecord
}

type DelegatedRecord struct {
	Definition *agentdefinition.Definition
	Run        *agent.AgentRun
	Proposals  []*agent.AgentProposal
}

type runGroup struct {
	definition *agentdefinition.Definition
	run        *agent.AgentRun
	open       *OpenRunRequest
	actions    []serviceports.PendingAction
	taint      *agent.RunTaint
	delegated  bool
	proposals  []*agent.AgentProposal
}

func (g *runGroup) records(openEmpty bool) bool {
	return g.run != nil || len(g.actions) > 0 || (g.delegated && openEmpty)
}

func (req *RecordRequest) groups() []*runGroup {
	groups := make([]*runGroup, 0, len(req.Delegated)+1)
	groups = append(groups, &runGroup{
		definition: req.Definition,
		run:        req.Run,
		open:       req.Open,
		actions:    req.Actions,
		taint:      req.Taint,
	})
	for idx := range req.Delegated {
		delegated := &req.Delegated[idx]
		groups = append(groups, &runGroup{
			definition: delegated.Definition,
			open:       delegated.Open,
			actions:    delegated.Actions,
			taint:      delegated.Taint,
			delegated:  true,
		})
	}

	return groups
}

func (req *RecordRequest) writes() bool {
	if len(req.Actions) > 0 {
		return true
	}
	for idx := range req.Delegated {
		if len(req.Delegated[idx].Actions) > 0 || req.OpenEmptyRuns {
			return true
		}
	}

	return false
}

func (s *Service) Record(ctx context.Context, req *RecordRequest) (*RecordResult, error) {
	if !req.writes() {
		return &RecordResult{Run: req.Run}, nil
	}

	groups := req.groups()
	var plan *agent.AgentPlan
	err := s.inTx(ctx, func(txCtx context.Context) error {
		written, err := s.write(txCtx, req, groups)
		if err != nil {
			return err
		}
		plan = written

		return nil
	})
	if err != nil {
		return nil, err
	}

	all := make([]*agent.AgentProposal, 0, len(req.Actions))
	for _, group := range groups {
		for _, proposal := range group.proposals {
			s.recordAutomaticFailure(ctx, proposal)
		}
		if len(group.proposals) > 0 {
			s.notifyPending(ctx, group.definition, group.run, group.proposals)
		}
		all = append(all, group.proposals...)
	}
	s.announce(ctx, req.Actor, all, plan)
	s.projectPlan(ctx, planDefinition(groups, plan), plan)
	for _, group := range groups {
		s.project(ctx, group.definition, group.proposals)
	}

	return recordResult(groups, plan), nil
}

func recordResult(groups []*runGroup, plan *agent.AgentPlan) *RecordResult {
	own := groups[0]
	result := &RecordResult{Run: own.run, Proposals: own.proposals, Plan: plan}
	if result.Proposals == nil {
		result.Proposals = []*agent.AgentProposal{}
	}
	for _, group := range groups[1:] {
		if group.run == nil {
			continue
		}
		result.Delegated = append(result.Delegated, DelegatedRecord{
			Definition: group.definition,
			Run:        group.run,
			Proposals:  group.proposals,
		})
	}

	return result
}

func planDefinition(groups []*runGroup, plan *agent.AgentPlan) *agentdefinition.Definition {
	if plan == nil {
		return nil
	}
	for _, group := range groups {
		if group.run != nil && group.run.ID == plan.RunID {
			return group.definition
		}
	}

	return nil
}

// inTx runs fn in the recorder's transaction. The run, the plan and every
// proposal are one decision; a failure on the last of them must not leave the
// first ones waiting on a person with the rest missing.
func (s *Service) inTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx == nil {
		return fn(ctx)
	}

	return s.tx.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		return fn(txCtx)
	})
}

// write stores the runs, the plan and the proposals. Nothing here announces
// what it wrote: that waits until the transaction has committed.
func (s *Service) write(
	ctx context.Context,
	req *RecordRequest,
	groups []*runGroup,
) (*agent.AgentPlan, error) {
	now := timeutils.NowUnix()
	steps := make([]planStep, 0, len(req.Actions))
	for _, group := range groups {
		if !group.records(req.OpenEmptyRuns) {
			continue
		}
		if group.run == nil {
			opened, err := s.openRun(ctx, req.Actor, group)
			if err != nil {
				return nil, err
			}
			group.run = opened
		}

		group.proposals = make([]*agent.AgentProposal, 0, len(group.actions))
		for idx := range group.actions {
			proposal := s.proposalFor(req, group, &group.actions[idx], now)
			group.proposals = append(group.proposals, proposal)
			if proposal.Status == agent.ProposalStatusPending {
				steps = append(steps, planStep{
					proposal: proposal,
					callID:   group.actions[idx].ToolCallID,
					position: len(steps),
				})
			}
		}
	}

	orderSteps(steps, req.CallOrder)
	plan, err := s.openPlan(ctx, req.Actor, groups, steps, now)
	if err != nil {
		return nil, err
	}
	if plan != nil {
		for idx, step := range steps {
			planID := plan.ID
			step.proposal.PlanID = &planID
			step.proposal.PlanStep = idx + 1
			step.proposal.ExpiresAt = plan.ExpiresAt
		}
	}

	for _, group := range groups {
		for idx, proposal := range group.proposals {
			multiErr := errortypes.NewMultiError()
			proposal.Validate(multiErr)
			if multiErr.HasErrors() {
				return nil, multiErr
			}

			created, err := s.proposals.Create(ctx, proposal)
			if err != nil {
				return nil, err
			}
			group.proposals[idx] = created
		}
	}

	return plan, nil
}

func (s *Service) proposalFor(
	req *RecordRequest,
	group *runGroup,
	action *serviceports.PendingAction,
	now int64,
) *agent.AgentProposal {
	sourceMessageID := req.SourceMessageIDs[action.ToolCallID]

	proposal := &agent.AgentProposal{
		ID:              action.ProposalID,
		OrganizationID:  req.Actor.OrganizationID,
		BusinessUnitID:  req.Actor.BusinessUnitID,
		RunID:           group.run.ID,
		TraceID:         action.TraceID,
		SpanID:          action.SpanID,
		StepKey:         action.StepKey,
		ToolName:        action.ToolName,
		ToolParams:      nonNilParams(action.Arguments),
		Rationale:       action.Rationale,
		AutonomyTier:    proposalTier(action.Tier),
		Status:          agent.ProposalStatusPending,
		SourceMessageID: sourceMessageID,
	}
	if req.Evidence != nil {
		proposal.Evidence = req.Evidence(*action, sourceMessageID)
	}
	if action.Target != nil {
		proposal.TargetResource = string(action.Target.Resource)
		proposal.TargetID = action.Target.ID
		proposal.TargetVersion = action.Target.Version
	}
	applyTaint(proposal, *action, group.taint)
	applyExecution(proposal, *action, now)
	applyExecutor(proposal, action, req.Actor)

	return proposal
}

type planStep struct {
	proposal *agent.AgentProposal
	callID   string
	position int
}

func orderSteps(steps []planStep, callOrder map[string]int) {
	if len(callOrder) == 0 {
		return
	}

	slices.SortStableFunc(steps, func(a, b planStep) int {
		return cmp.Compare(a.at(callOrder), b.at(callOrder))
	})
}

func (s planStep) at(callOrder map[string]int) int {
	if position, ok := callOrder[s.callID]; ok {
		return position
	}

	return len(callOrder) + s.position
}

// openPlan groups a turn's pending actions into one decision when there are
// at least two of them, whichever agent filed each. One pending action is a
// proposal, as before; two or more are a sequence, run in the order the
// conversation asked for them.
func (s *Service) openPlan(
	ctx context.Context,
	actor *serviceports.RequestActor,
	groups []*runGroup,
	steps []planStep,
	now int64,
) (*agent.AgentPlan, error) {
	if s.plans == nil || len(steps) < 2 {
		return nil, nil
	}

	owner := groups[0]
	for _, group := range groups {
		if slices.ContainsFunc(group.proposals, func(p *agent.AgentProposal) bool {
			return p.Status == agent.ProposalStatusPending
		}) {
			owner = group
			break
		}
	}
	rationale := ""
	for _, step := range steps {
		if rationale = strings.TrimSpace(step.proposal.Rationale); rationale != "" {
			break
		}
	}

	plan := &agent.AgentPlan{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		RunID:          owner.run.ID,
		Title:          planTitle(owner.definition, len(steps)),
		Summary:        planSummary(owner.run, rationale),
		Status:         agent.PlanStatusPending,
		StepCount:      len(steps),
		ExpiresAt:      now + int64(agent.DefaultProposalTTL.Seconds()),
	}

	multiErr := errortypes.NewMultiError()
	plan.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.plans.Create(ctx, plan)
}

func planTitle(definition *agentdefinition.Definition, steps int) string {
	title := fmt.Sprintf("%d changes", steps)
	if definition != nil && strings.TrimSpace(definition.Name) != "" {
		title = definition.Name + ": " + title
	}

	return stringutils.Ellipsize(title, maxPlanTitleChars)
}

func planSummary(run *agent.AgentRun, rationale string) string {
	if run != nil && strings.TrimSpace(run.Summary) != "" {
		return run.Summary
	}

	return rationale
}

// notifyPending is best effort. The proposals are stored; a notice that could
// not be sent is logged, not allowed to fail the run that produced them.
func (s *Service) notifyPending(
	ctx context.Context,
	definition *agentdefinition.Definition,
	run *agent.AgentRun,
	proposals []*agent.AgentProposal,
) {
	if s.notifier == nil {
		return
	}

	if err := s.notifier.NotifyPending(ctx, serviceports.PendingProposalsNotice{
		Definition: definition,
		Run:        run,
		Proposals:  proposals,
	}); err != nil {
		s.logger.Error("failed to notify deciders of pending proposals",
			zap.String("run", run.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) openRun(
	ctx context.Context,
	actor *serviceports.RequestActor,
	group *runGroup,
) (*agent.AgentRun, error) {
	open := group.open
	if open == nil {
		open = &OpenRunRequest{}
	}

	now := timeutils.NowUnix()
	status := open.Status
	if status == "" {
		status = agent.RunStatusCompleted
	}

	run := &agent.AgentRun{
		OrganizationID:   actor.OrganizationID,
		BusinessUnitID:   actor.BusinessUnitID,
		AgentType:        open.AgentType,
		SubjectType:      open.SubjectType,
		SubjectID:        open.SubjectID,
		Trigger:          open.Trigger,
		Status:           status,
		ModelIdentifier:  open.Model,
		PromptVersion:    open.PromptVersion,
		InputContextHash: open.InputContextHash,
		Summary:          open.Summary,
		Fingerprint:      open.Fingerprint,
		StartedAt:        now,
		CompletedAt:      &now,
		TraceID:          open.TraceID,
		TurnID:           open.TurnID,
		ParentOwnerKind:  open.ParentOwnerKind,
		ParentOwnerID:    open.ParentOwnerID,
		DelegateCallID:   open.DelegateCallID,
	}
	if group.definition != nil {
		run.AgentDefinitionID = group.definition.ID
	}
	if run.Trigger == "" {
		run.Trigger = agent.RunTriggerManual
	}
	run.RecordTaint(group.taint, now)

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.runs.Create(ctx, run)
}

// applyTaint records where the write reaches, what held it, and whether the
// run had read outside content when it was decided. A proposal decided before
// the run read any is clean, though the run it belongs to is not.
func applyTaint(
	proposal *agent.AgentProposal,
	action serviceports.PendingAction,
	taint *agent.RunTaint,
) {
	proposal.EgressClass = action.Egress
	proposal.HeldBy = slices.Clone(action.HeldBy)
	if proposal.HeldBy == nil {
		proposal.HeldBy = []string{}
	}
	if !action.Tainted {
		return
	}
	proposal.Tainted = true
	proposal.Taint = taint.Clone()
}

func applyExecution(proposal *agent.AgentProposal, action serviceports.PendingAction, now int64) {
	if action.Simulated {
		simulatedAt := now
		proposal.SimulatedAt = &simulatedAt
		proposal.Simulation = action.Simulation
		proposal.Status = agent.ProposalStatusSimulated

		return
	}

	if !action.Executed {
		return
	}

	executedAt := now
	if action.ExecutedAt > 0 {
		executedAt = action.ExecutedAt
	}
	proposal.ExecutedAt = &executedAt
	if action.ExecutionError != "" {
		proposal.Status = agent.ProposalStatusExecutionFailed
		proposal.ExecutionError = action.ExecutionError

		return
	}

	proposal.Status = agent.ProposalStatusExecuted
	proposal.ExecutionResult = action.ExecutionResult.Bounded()
}

func applyExecutor(
	proposal *agent.AgentProposal,
	action *serviceports.PendingAction,
	actor *serviceports.RequestActor,
) {
	if !action.Executed && !action.Simulated {
		return
	}
	if actor != nil && actor.PrincipalType == serviceports.PrincipalTypeUser {
		proposal.ExecutedByUserID = actor.UserID
	}
	if action.Executed && !action.Simulated && action.ExecutionError == "" {
		proposal.ExecutedTargetVersion = intutils.ClonePointer(action.ExecutedVersion)
	}
}

func proposalTier(tier agent.AutonomyTier) agent.AutonomyTier {
	if tier == "" {
		return agent.TierPropose
	}

	return tier
}

func nonNilParams(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}

	return params
}

// recordAutomaticFailure tells the ledger about a tool that ran without asking
// and did not go through. A person never decided on it, so nothing else would
// count the failure against the tier that let it run unasked.
func (s *Service) recordAutomaticFailure(ctx context.Context, proposal *agent.AgentProposal) {
	if s.trust == nil || proposal.Status != agent.ProposalStatusExecutionFailed {
		return
	}

	if err := s.trust.RecordExecutionFailure(ctx, proposal); err != nil {
		s.logger.Error("failed to record automatic execution failure in the trust ledger",
			zap.String("proposal", proposal.ID.String()),
			zap.String("tool", proposal.ToolName),
			zap.Error(err),
		)
	}
}

// announce tells connected clients what the run left waiting. A proposal
// that ran on its own is announced too: the ledger it landed in is on the
// same screens.
func (s *Service) announce(
	ctx context.Context,
	actor *serviceports.RequestActor,
	proposals []*agent.AgentProposal,
	plan *agent.AgentPlan,
) {
	if s.activity == nil {
		return
	}

	auditActor := actor.AuditActorOrSystem()
	for _, proposal := range proposals {
		s.activity.ProposalChanged(ctx, proposal, auditActor, serviceports.ActivityCreated)
	}
	if plan != nil {
		s.activity.PlanChanged(ctx, plan, auditActor, serviceports.ActivityCreated)
	}
}

func (s *Service) projectPlan(
	ctx context.Context,
	definition *agentdefinition.Definition,
	plan *agent.AgentPlan,
) {
	if s.watchtower == nil || plan == nil {
		return
	}

	s.watchtower.Upsert(ctx, watchtowersources.DescribePlan(plan, definitionName(definition)))
}

// project puts what is now waiting on a person onto the watchtower. A step
// of a plan is not projected on its own: the plan is the decision.
func (s *Service) project(
	ctx context.Context,
	definition *agentdefinition.Definition,
	proposals []*agent.AgentProposal,
) {
	if s.watchtower == nil {
		return
	}

	name := definitionName(definition)
	for _, proposal := range proposals {
		if proposal == nil || proposal.Status != agent.ProposalStatusPending ||
			proposal.PlanID != nil {
			continue
		}
		s.watchtower.Upsert(ctx, watchtowersources.DescribeProposal(proposal, name))
	}
}

func definitionName(definition *agentdefinition.Definition) string {
	if definition == nil {
		return ""
	}

	return definition.Name
}
