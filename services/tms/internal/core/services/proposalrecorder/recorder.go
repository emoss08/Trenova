package proposalrecorder

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
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

type Params struct {
	fx.In

	Logger    *zap.Logger
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
	svc.trust = p.Trust
	svc.notifier = p.Notifier
	svc.plans = p.Plans
	svc.activity = p.Activity
	svc.watchtower = p.Watchtower

	return svc
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
}

type RecordResult struct {
	Run       *agent.AgentRun
	Proposals []*agent.AgentProposal
	// Plan is set when the run's pending proposals were grouped into one.
	Plan *agent.AgentPlan
}

func (s *Service) Record(ctx context.Context, req *RecordRequest) (*RecordResult, error) {
	if len(req.Actions) == 0 {
		return &RecordResult{Run: req.Run}, nil
	}

	run := req.Run
	if run == nil {
		opened, err := s.openRun(ctx, req)
		if err != nil {
			return nil, err
		}
		run = opened
	}

	now := timeutils.NowUnix()
	proposals := make([]*agent.AgentProposal, 0, len(req.Actions))

	plan, err := s.openPlan(ctx, req, run, now)
	if err != nil {
		return nil, err
	}

	step := 0
	for _, action := range req.Actions {
		sourceMessageID := req.SourceMessageIDs[action.ToolCallID]

		proposal := &agent.AgentProposal{
			OrganizationID:  req.Actor.OrganizationID,
			BusinessUnitID:  req.Actor.BusinessUnitID,
			RunID:           run.ID,
			ToolName:        action.ToolName,
			ToolParams:      nonNilParams(action.Arguments),
			Rationale:       action.Rationale,
			AutonomyTier:    proposalTier(action.Tier),
			Status:          agent.ProposalStatusPending,
			SourceMessageID: sourceMessageID,
		}
		if req.Evidence != nil {
			proposal.Evidence = req.Evidence(action, sourceMessageID)
		}
		if action.Target != nil {
			proposal.TargetResource = string(action.Target.Resource)
			proposal.TargetID = action.Target.ID
			proposal.TargetVersion = action.Target.Version
		}
		applyExecution(proposal, action, now)
		if plan != nil && proposal.Status == agent.ProposalStatusPending {
			step++
			planID := plan.ID
			proposal.PlanID = &planID
			proposal.PlanStep = step
			proposal.ExpiresAt = plan.ExpiresAt
		}

		multiErr := errortypes.NewMultiError()
		proposal.Validate(multiErr)
		if multiErr.HasErrors() {
			return nil, multiErr
		}

		created, err := s.proposals.Create(ctx, proposal)
		if err != nil {
			return nil, err
		}

		s.recordAutomaticFailure(ctx, created)
		proposals = append(proposals, created)
	}

	s.notifyPending(ctx, req.Definition, run, proposals)
	s.announce(ctx, req.Actor, proposals, plan)
	s.project(ctx, req.Definition, proposals, plan)

	return &RecordResult{Run: run, Proposals: proposals, Plan: plan}, nil
}

// openPlan groups a run's pending actions into one decision when there are
// at least two of them. One pending action is a proposal, as before; two or
// more in one run are the agent asking for a sequence, and the order it asked
// in is the order the plan will run them.
func (s *Service) openPlan(
	ctx context.Context,
	req *RecordRequest,
	run *agent.AgentRun,
	now int64,
) (*agent.AgentPlan, error) {
	if s.plans == nil {
		return nil, nil
	}

	pending := 0
	rationale := ""
	for _, action := range req.Actions {
		if action.Executed || action.Simulated {
			continue
		}
		pending++
		if rationale == "" {
			rationale = strings.TrimSpace(action.Rationale)
		}
	}
	if pending < 2 {
		return nil, nil
	}

	plan := &agent.AgentPlan{
		OrganizationID: req.Actor.OrganizationID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		RunID:          run.ID,
		Title:          planTitle(req.Definition, pending),
		Summary:        planSummary(run, rationale),
		Status:         agent.PlanStatusPending,
		StepCount:      pending,
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

func (s *Service) openRun(ctx context.Context, req *RecordRequest) (*agent.AgentRun, error) {
	open := req.Open
	if open == nil {
		open = &OpenRunRequest{}
	}

	now := timeutils.NowUnix()
	status := open.Status
	if status == "" {
		status = agent.RunStatusCompleted
	}

	run := &agent.AgentRun{
		OrganizationID:   req.Actor.OrganizationID,
		BusinessUnitID:   req.Actor.BusinessUnitID,
		AgentType:        open.AgentType,
		SubjectType:      open.SubjectType,
		SubjectID:        open.SubjectID,
		Trigger:          open.Trigger,
		Status:           status,
		ModelIdentifier:  open.Model,
		PromptVersion:    open.PromptVersion,
		InputContextHash: open.InputContextHash,
		Summary:          open.Summary,
		StartedAt:        now,
		CompletedAt:      &now,
	}
	if req.Definition != nil {
		run.AgentDefinitionID = req.Definition.ID
	}
	if run.Trigger == "" {
		run.Trigger = agent.RunTriggerManual
	}

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.runs.Create(ctx, run)
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
	proposal.ExecutedAt = &executedAt
	if action.ExecutionError != "" {
		proposal.Status = agent.ProposalStatusExecutionFailed
		proposal.ExecutionError = action.ExecutionError

		return
	}

	proposal.Status = agent.ProposalStatusExecuted
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

// project puts what is now waiting on a person onto the watchtower. A step
// of a plan is not projected on its own: the plan is the decision.
func (s *Service) project(
	ctx context.Context,
	definition *agentdefinition.Definition,
	proposals []*agent.AgentProposal,
	plan *agent.AgentPlan,
) {
	if s.watchtower == nil {
		return
	}

	name := ""
	if definition != nil {
		name = definition.Name
	}
	if plan != nil {
		s.watchtower.Upsert(ctx, watchtowersources.DescribePlan(plan, name))
	}
	for _, proposal := range proposals {
		if proposal == nil || proposal.Status != agent.ProposalStatusPending ||
			proposal.PlanID != nil {
			continue
		}
		s.watchtower.Upsert(ctx, watchtowersources.DescribeProposal(proposal, name))
	}
}
