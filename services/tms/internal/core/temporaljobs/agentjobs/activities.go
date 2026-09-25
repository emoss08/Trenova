package agentjobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentscoring"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxSummaryChars = 2000
	promptVersion   = "agent-definition/v2"
)

type ActivitiesParams struct {
	fx.In

	Logger        *zap.Logger
	Definitions   repositories.AgentDefinitionRepository
	Controls      repositories.AgentControlRepository
	RunRepo       repositories.AgentRunRepository
	ProposalRepo  repositories.AgentProposalRepository
	Runs          serviceports.AgentRunService
	Runtime       *agentruntime.Service
	Steps         serviceports.RunStepLedger
	Trajectory    serviceports.AgentRunEventRecorder `optional:"true"`
	Contexts      serviceports.RuntimeContextBuilder
	Recorder      *proposalrecorder.Service
	Subjects      serviceports.AgentSubjectDescriber
	Activity      serviceports.AgentActivityPublisher `optional:"true"`
	Notifier      serviceports.AgentProposalNotifier  `optional:"true"`
	Plans         repositories.AgentPlanRepository    `optional:"true"`
	Evaluations   repositories.AgentEvaluationRepository
	EvalCases     repositories.AgentEvalCaseRepository
	CaseService   serviceports.AgentEvalCaseService `optional:"true"`
	Users         repositories.UserRepository
	QueryTools    serviceports.AgentQueryToolRegistry `optional:"true"`
	ActionTools   serviceports.AgentToolRegistry      `optional:"true"`
	Decisions     repositories.AgentDecisionRepository
	Conversations repositories.ConversationRepository `optional:"true"`
	// Watchtower puts a run that could not finish on the feed, so a
	// failure nobody was watching for still reaches someone.
	Watchtower serviceports.WatchtowerProjector `optional:"true"`
	Schedules  *DefinitionSchedules
}

type Activities struct {
	logger        *zap.Logger
	definitions   repositories.AgentDefinitionRepository
	controls      repositories.AgentControlRepository
	runRepo       repositories.AgentRunRepository
	proposalRepo  repositories.AgentProposalRepository
	runs          serviceports.AgentRunService
	runtime       *agentruntime.Service
	steps         serviceports.RunStepLedger
	trajectory    serviceports.AgentRunEventRecorder
	contexts      serviceports.RuntimeContextBuilder
	recorder      *proposalrecorder.Service
	notifier      serviceports.AgentProposalNotifier
	plans         repositories.AgentPlanRepository
	evaluations   repositories.AgentEvaluationRepository
	evalCases     repositories.AgentEvalCaseRepository
	caseService   serviceports.AgentEvalCaseService
	users         repositories.UserRepository
	scorer        *agentscoring.Scorer
	decisions     repositories.AgentDecisionRepository
	conversations repositories.ConversationRepository
	subjects      serviceports.AgentSubjectDescriber
	activity      serviceports.AgentActivityPublisher
	watchtower    serviceports.WatchtowerProjector
	schedules     *DefinitionSchedules
}

func NewActivities(p ActivitiesParams) *Activities {
	logger := p.Logger.Named("agent-activities")
	scorer := agentscoring.New(
		agentscoring.WithToolPolicies(toolPolicies(p.QueryTools, p.ActionTools)),
	)

	return &Activities{
		logger:        logger,
		definitions:   p.Definitions,
		controls:      p.Controls,
		runRepo:       p.RunRepo,
		proposalRepo:  p.ProposalRepo,
		runs:          p.Runs,
		runtime:       p.Runtime,
		steps:         p.Steps,
		trajectory:    p.Trajectory,
		contexts:      p.Contexts,
		recorder:      p.Recorder,
		notifier:      p.Notifier,
		plans:         p.Plans,
		evaluations:   p.Evaluations,
		evalCases:     p.EvalCases,
		caseService:   p.CaseService,
		users:         p.Users,
		scorer:        scorer,
		decisions:     p.Decisions,
		conversations: p.Conversations,
		subjects:      p.Subjects,
		activity:      p.Activity,
		watchtower:    p.Watchtower,
		schedules:     p.Schedules,
	}
}

func (a *Activities) PrepareRunActivity(
	ctx context.Context,
	payload *AgentRunPayload,
) (*PrepareRunResult, error) {
	tenant := payload.tenantInfo()

	definition, err := a.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         payload.DefinitionID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"agent definition unavailable", "DefinitionUnavailable", err,
		)
	}
	if !definition.Enabled {
		return nil, temporal.NewNonRetryableApplicationError(
			"agent definition is disabled", "DefinitionDisabled", nil,
		)
	}

	control, err := a.controls.GetOrCreate(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("load agent control: %w", err)
	}

	subject, err := a.subjects.Describe(ctx, tenant, payload.SubjectType, payload.SubjectID)
	if err != nil {
		return nil, err
	}

	hash := hashSubject(definition, subject)
	if err = a.updateRun(ctx, tenant, payload.RunID, func(run *agent.AgentRun) {
		run.Status = agent.RunStatusDiagnosing
		run.InputContextHash = hash
		run.PromptVersion = promptVersion
		if run.StartedAt == 0 {
			run.StartedAt = timeutils.NowUnix()
		}
	}); err != nil {
		return nil, err
	}

	return &PrepareRunResult{
		Definition:             definition,
		Subject:                subject,
		ShadowMode:             definition.EffectiveShadow(control.ShadowMode),
		DecisionTimeoutSeconds: definition.DecisionTimeoutSeconds,
		RunTimeoutSeconds:      definition.RunTimeoutSeconds,
	}, nil
}

// trajectoryFor opens the durable event writer, when one is configured.
//
// Optional, like the metrics collector: an installation without it still runs
// its agents, it simply keeps no account of how. Every call through the writer
// is nil-safe.
func (a *Activities) trajectoryFor(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
) serviceports.AgentRunEventWriter {
	if a.trajectory == nil {
		return nil
	}

	return a.trajectory.Recorder(ctx, tenant, serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAgentRun,
		ID:   runID,
	})
}

// runRequest is what a background run asks the runtime: its subject, as the
// agent, unattended, with every write claimed in the run's ledger.
func (a *Activities) runRequest(
	ctx context.Context,
	payload *AgentRunPayload,
	definition *agentdefinition.Definition,
	subject *agentdefinition.RuntimeSubject,
) (*serviceports.RunRequest, error) {
	tenant := payload.tenantInfo()
	actor := agentActor(tenant)

	input := backgroundInput(payload, subject)
	runtimeContext, err := a.contexts.Build(ctx, &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      actor,
		Trigger:    payload.Trigger,
		Subject:    subject,
		Query: (&serviceports.ContextQuery{
			Actor:        actor,
			DefinitionID: definition.ID,
			RunID:        payload.RunID,
			Input:        input,
		}).Request(),
	})
	if err != nil {
		return nil, fmt.Errorf("build runtime context: %w", err)
	}

	return &serviceports.RunRequest{
		Definition: definition,
		Actor:      actor,
		Context:    runtimeContext,
		Input:      input,
		RunID:      payload.RunID,
		Unattended: true,
		// The ledger is what makes a retried write safe. Without it a second
		// attempt re-runs every write the first one made, and the tools do
		// not dedupe: RequiresIdempotencyKey is checked for presence and, bar
		// the two that forward it to an email provider, never looked up.
		Steps: a.steps,
		StepOwner: serviceports.RunStepOwner{
			Kind: serviceports.RunStepOwnerAgentRun,
			ID:   payload.RunID,
		},
		Attempt: int(activity.GetInfo(ctx).Attempt),
	}, nil
}

// RunAgentActivity is the whole loop in one activity, as a run was before the
// loop moved into workflow code. Only runs that started on that code call it.
func (a *Activities) RunAgentActivity(
	ctx context.Context,
	input *RunAgentInput,
) (*RunAgentResult, error) {
	payload := input.Payload
	tenant := payload.tenantInfo()

	req, err := a.runRequest(ctx, payload, input.Definition, input.Subject)
	if err != nil {
		return nil, err
	}

	activity.RecordHeartbeat(ctx, "running")

	// The writer rides a context cancellation cannot reach: the events worth
	// keeping are most often the ones emitted as something is going wrong.
	keep := context.WithoutCancel(ctx)
	trajectory := a.trajectoryFor(keep, tenant, payload.RunID)
	req.Emit = func(event serviceports.StreamEvent) {
		activity.RecordHeartbeat(ctx, "working")
		serviceports.RecordTrajectory(trajectory, keep, event)
	}

	outcome, err := a.runtime.Run(ctx, req)
	serviceports.FlushTrajectory(trajectory, keep)
	if err != nil {
		return nil, err
	}

	settled, err := a.settleRun(ctx, settleRunParams{
		Payload:    payload,
		Definition: input.Definition,
		Subject:    input.Subject,
		Outcome:    outcome,
	})
	if err != nil {
		return nil, err
	}

	return &RunAgentResult{
		Reply:            outcome.Reply,
		Model:            outcome.Model,
		ToolCallsUsed:    outcome.ToolCallsUsed,
		Exhausted:        outcome.Exhausted,
		ProposalsRaised:  settled.ProposalsRaised,
		PendingProposals: settled.PendingProposals,
	}, nil
}

// OpenRunActivity builds the run's turn, which reads permissions, the agent's
// memory and the ledger, so it happens here rather than in workflow code.
//
// A run in shadow mode is opened in simulation. Shadow mode means the agent's
// work is watched and not acted on, and an automatic write that ran anyway
// would be exactly the action shadow mode exists to withhold.
func (a *Activities) OpenRunActivity(
	ctx context.Context,
	input *OpenRunInput,
) (*OpenRunResult, error) {
	definition := input.Definition
	if input.Shadow && !definition.SimulationMode {
		simulated := *definition
		simulated.SimulationMode = true
		definition = &simulated
	}

	req, err := a.runRequest(ctx, input.Payload, definition, input.Subject)
	if err != nil {
		return nil, err
	}

	return &OpenRunResult{
		Run:  agentflow.NewRunContext(req, agentflow.PriorityBackground),
		Turn: a.runtime.OpenTurn(ctx, req).State(),
	}, nil
}

// FinishRunActivity files what a run did: its proposals, its summary, and the
// account of how it got there. It runs however the loop ended, so a write the
// run made before a failure is still on the record.
func (a *Activities) FinishRunActivity(
	ctx context.Context,
	input *FinishRunInput,
) (*FinishRunResult, error) {
	outcome := input.Run
	if outcome == nil {
		outcome = &serviceports.RunResult{}
	}

	settled, err := a.settleRun(ctx, settleRunParams{
		Payload:    input.Payload,
		Definition: input.Definition,
		Subject:    input.Subject,
		Outcome:    outcome,
		Failed:     input.Failure != nil,
	})
	if err != nil {
		return nil, err
	}

	// Last, so an attempt that fails before this is the one that writes it:
	// written earlier, a retry would write the run's account twice.
	a.recordTrajectory(ctx, input.Payload, input.Events)

	return settled, nil
}

// PendingProposalsActivity counts the run's proposals still waiting on a
// person.
func (a *Activities) PendingProposalsActivity(
	ctx context.Context,
	input *PendingProposalsInput,
) (int, error) {
	proposals, err := a.proposalRepo.ListByRun(ctx, repositories.ListAgentProposalsByRunRequest{
		RunID:      input.RunID,
		TenantInfo: input.TenantInfo,
	})
	if err != nil {
		return 0, fmt.Errorf("list the run's proposals: %w", err)
	}

	return countPending(proposals), nil
}

func countPending(proposals []*agent.AgentProposal) int {
	pending := 0
	for _, proposal := range proposals {
		if proposal.Status == agent.ProposalStatusPending {
			pending++
		}
	}

	return pending
}

func (a *Activities) recordTrajectory(
	ctx context.Context,
	payload *AgentRunPayload,
	events []temporaltype.StreamItem,
) {
	writer := a.trajectoryFor(ctx, payload.tenantInfo(), payload.RunID)
	for _, event := range events {
		serviceports.RecordTrajectory(writer, ctx, serviceports.StreamEvent{
			Event: event.Event,
			Data:  event.Data,
			At:    event.At,
		})
	}
	serviceports.FlushTrajectory(writer, ctx)
}

// settleRunParams groups what filing a finished run needs.
type settleRunParams struct {
	Payload    *AgentRunPayload
	Definition *agentdefinition.Definition
	Subject    *agentdefinition.RuntimeSubject
	Outcome    *serviceports.RunResult
	// Failed says the loop did not finish. What it proposed is still filed,
	// but the run is not left awaiting a decision it will never hear.
	Failed bool
}

// settleRun files a run's proposals and summary.
func (a *Activities) settleRun(ctx context.Context, p settleRunParams) (*FinishRunResult, error) {
	tenant := p.Payload.tenantInfo()

	run, err := a.runRepo.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         p.Payload.RunID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load agent run: %w", err)
	}

	// The step ledger keeps a retry from running a tool twice, but the
	// proposals those tools raised are written here, after the loop. An
	// attempt that got this far and then failed on the update below would
	// otherwise have its proposals recorded a second time, and the person
	// would be asked to approve the same change on two cards.
	recorded, err := a.recordProposals(ctx, recordProposalsParams{
		Actor:      agentActor(tenant),
		Definition: p.Definition,
		Run:        run,
		Actions:    p.Outcome.Actions,
		Subject:    p.Subject,
		TenantInfo: tenant,
		Taint:      p.Outcome.Taint,
	})
	if err != nil {
		return nil, err
	}

	pending := countPending(recorded.Proposals)

	run.ModelIdentifier = p.Outcome.Model
	run.Summary = stringutils.Ellipsize(strings.TrimSpace(p.Outcome.Reply), maxSummaryChars)
	run.RecordTaint(p.Outcome.Taint, timeutils.NowUnix())
	if fingerprint := p.Outcome.ServedFingerprint(); fingerprint != nil {
		run.Fingerprint = fingerprint
	}
	if pending > 0 && !p.Failed {
		run.Status = agent.RunStatusAwaitingDecision
	}
	if _, err = a.runRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("update agent run: %w", err)
	}
	a.announceRun(ctx, run)

	return &FinishRunResult{
		ProposalsRaised:  len(recorded.Proposals),
		PendingProposals: pending,
	}, nil
}

func (a *Activities) CompleteRunActivity(ctx context.Context, input *CompleteRunInput) error {
	var completed *agent.AgentRun
	err := a.updateRun(ctx, input.TenantInfo, input.RunID, func(run *agent.AgentRun) {
		run.Status = input.Status
		if input.Error != "" {
			run.ErrorMessage = stringutils.Ellipsize(input.Error, maxSummaryChars)
		}
		completedAt := timeutils.NowUnix()
		run.CompletedAt = &completedAt
		completed = run
	})
	if err != nil {
		return err
	}

	a.projectFailedRun(ctx, input.TenantInfo, completed)

	return nil
}

// projectFailedRun puts a run that could not finish on the watchtower. A
// run that ended any other way was either watched or uneventful, and the
// feed only carries what somebody has to look at.
func (a *Activities) projectFailedRun(
	ctx context.Context,
	tenant pagination.TenantInfo,
	run *agent.AgentRun,
) {
	if a.watchtower == nil || run == nil || run.Status != agent.RunStatusFailed {
		return
	}

	name := ""
	if run.AgentDefinitionID.IsNotNil() {
		definition, err := a.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
			ID:         run.AgentDefinitionID,
			TenantInfo: tenant,
		})
		if err != nil {
			a.logger.Warn("failed to name the agent behind a failed run", zap.Error(err))
		} else {
			name = definition.Name
		}
	}

	a.watchtower.Upsert(ctx, watchtowersources.DescribeFailedRun(run, name))
}

func (a *Activities) ExpireProposalsActivity(
	ctx context.Context,
	input *ExpireProposalsInput,
) error {
	if _, err := a.proposalRepo.ExpirePendingByRun(
		ctx,
		repositories.ExpireAgentProposalsByRunRequest{
			RunID:      input.RunID,
			TenantInfo: input.TenantInfo,
		},
	); err != nil {
		return fmt.Errorf("expire proposals: %w", err)
	}

	return a.updateRun(ctx, input.TenantInfo, input.RunID, func(run *agent.AgentRun) {
		run.Status = agent.RunStatusCompleted
		completedAt := timeutils.NowUnix()
		run.CompletedAt = &completedAt
	})
}

// StartScheduledRunActivity starts the run a definition's schedule fired for,
// when the definition may run now.
//
// The schedule decides when; this decides whether. The definition may have
// been disabled or turned into a chat agent since the schedule last heard, it
// may already have as many runs open as it is allowed, or it may be past its
// budget; each is a slot skipped, not a failure. The slot names the run's
// workflow, so the same slot cannot start two runs.
func (a *Activities) StartScheduledRunActivity(
	ctx context.Context,
	payload *ScheduledRunPayload,
) (*StartScheduledRunResult, error) {
	tenant := pagination.TenantInfo{OrgID: payload.OrganizationID, BuID: payload.BusinessUnitID}
	now := timeutils.NowUnix()

	definition, err := a.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         payload.DefinitionID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"agent definition unavailable", "DefinitionUnavailable", err,
		)
	}
	if !definition.Enabled || !definition.IsBackground() || !Scheduled(definition) {
		return &StartScheduledRunResult{Skipped: "not_runnable"}, nil
	}
	if definition.EndsAt != nil && *definition.EndsAt <= now {
		return &StartScheduledRunResult{Skipped: "ended"}, nil
	}

	open, err := a.runRepo.CountOpen(ctx, repositories.CountOpenAgentRunsRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
	})
	if err != nil {
		return nil, err
	}
	if open >= definition.MaxConcurrentRuns {
		return &StartScheduledRunResult{Skipped: "at_concurrency_limit"}, nil
	}

	// The next slot is kept on the definition for the screens that show it;
	// the schedule, not this, is what fires it.
	var next *int64
	if computed, cErr := definition.ComputeNextRun(now); cErr == nil &&
		(definition.EndsAt == nil || computed < *definition.EndsAt) {
		next = &computed
	}
	if _, err = a.definitions.MarkRun(ctx, repositories.MarkAgentDefinitionRunRequest{
		ID:                definition.ID,
		TenantInfo:        tenant,
		LastRunAt:         now,
		NextRunAt:         next,
		ExpectedNextRunAt: definition.NextRunAt,
	}); err != nil {
		return nil, fmt.Errorf("record the scheduled run: %w", err)
	}

	run, err := a.runs.StartForDefinition(ctx, &serviceports.StartAgentRunForDefinitionRequest{
		DefinitionID: definition.ID,
		Trigger:      definition.TriggerMode.RunTrigger(),
		Slot:         payload.Slot,
		TenantInfo:   tenant,
	}, agentActor(tenant))
	if err != nil {
		if errors.Is(err, serviceports.ErrAgentRunAlreadyOpen) {
			return &StartScheduledRunResult{Skipped: "slot_already_started"}, nil
		}
		if errortypes.IsBusinessError(err) {
			return &StartScheduledRunResult{Skipped: err.Error()}, nil
		}

		return nil, err
	}

	return &StartScheduledRunResult{Started: true, RunID: run.ID.String()}, nil
}

// ReconcileDefinitionSchedulesActivity makes every agent's schedule match the
// agent.
func (a *Activities) ReconcileDefinitionSchedulesActivity(
	ctx context.Context,
) (*ReconcileResult, error) {
	return a.schedules.Reconcile(ctx)
}

func (a *Activities) updateRun(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
	mutate func(run *agent.AgentRun),
) error {
	run, err := a.runRepo.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return fmt.Errorf("load agent run: %w", err)
	}

	mutate(run)

	if _, err = a.runRepo.Update(ctx, run); err != nil {
		return fmt.Errorf("update agent run: %w", err)
	}
	a.announceRun(ctx, run)

	return nil
}

// announceRun tells connected clients the run moved. The worker announces
// as the system, since no person is acting.
func (a *Activities) announceRun(ctx context.Context, run *agent.AgentRun) {
	if a.activity == nil {
		return
	}

	a.activity.RunChanged(ctx, run, serviceports.SystemAuditActor(), serviceports.ActivityUpdated)
}

func agentActor(tenant pagination.TenantInfo) *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeAgent,
		PrincipalID:    serviceports.AgentPrincipalID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
}

func backgroundInput(payload *AgentRunPayload, subject *agentdefinition.RuntimeSubject) string {
	return agentdefinition.BackgroundRunInput(payload.Trigger, payload.EventKind, subject)
}

func subjectEvidence(
	subject *agentdefinition.RuntimeSubject,
	runID pulid.ID,
) proposalrecorder.EvidenceFunc {
	return func(action serviceports.PendingAction, _ pulid.ID) []agent.EvidenceRef {
		evidence := []agent.EvidenceRef{{
			Type: "agent_run",
			ID:   runID.String(),
			Note: "proposed by " + action.ToolName,
		}}
		if subject != nil {
			evidence = append(evidence, agent.EvidenceRef{
				Type: strings.ToLower(string(subject.Type)),
				ID:   subject.ID,
				Note: subject.Label,
			})
		}

		return evidence
	}
}

func hashSubject(
	definition *agentdefinition.Definition,
	subject *agentdefinition.RuntimeSubject,
) string {
	encoded, err := sonic.Marshal(map[string]any{
		"definition": definition.ID,
		"version":    definition.Version,
		"subject":    subject,
	})
	if err != nil {
		encoded = []byte(fmt.Sprintf("%v", subject))
	}

	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// ExpireStaleProposalsActivity marks every pending proposal past its expiry,
// in every tenant. Distinct from ExpireProposalsActivity, which closes the
// proposals of one finished run.
func (a *Activities) ExpireStaleProposalsActivity(
	ctx context.Context,
	input *ExpireStaleProposalsInput,
) (*ExpireStaleProposalsResult, error) {
	expired, err := a.proposalRepo.ExpirePending(ctx, repositories.ExpireAgentProposalsRequest{
		Before: input.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("expire pending proposals: %w", err)
	}

	if expired > 0 {
		a.logger.Info("expired agent proposals past their decision window",
			zap.Int("expired", expired),
		)
	}

	// A plan outlives none of its steps: once they have expired, so has it.
	if a.plans != nil {
		expiredPlans, planErr := a.plans.ExpirePending(ctx, repositories.ExpireAgentPlansRequest{
			Before: input.Now,
		})
		if planErr != nil {
			return nil, fmt.Errorf("expire pending plans: %w", planErr)
		}
		if expiredPlans > 0 {
			a.logger.Info("expired agent plans past their decision window",
				zap.Int("expired", expiredPlans),
			)
		}
	}

	return &ExpireStaleProposalsResult{Expired: expired}, nil
}

// DeleteStaleAskThreadsActivity removes one batch of unkept quick questions
// older than the cut-off. It reports how many went, so the workflow knows
// whether to ask again.
func (a *Activities) DeleteStaleAskThreadsActivity(
	ctx context.Context,
	input *DeleteStaleAskThreadsInput,
) (*DeleteStaleAskThreadsResult, error) {
	if a.conversations == nil {
		return &DeleteStaleAskThreadsResult{}, nil
	}

	deleted, err := a.conversations.DeleteStaleThreads(ctx, repositories.DeleteStaleThreadsRequest{
		Origin: conversation.ThreadOriginAsk,
		Before: input.Before,
		Limit:  deleteStaleAskBatch,
	})
	if err != nil {
		return nil, fmt.Errorf("delete stale ask threads: %w", err)
	}
	if deleted > 0 {
		a.logger.Info("removed quick questions nobody kept", zap.Int("deleted", deleted))
	}

	drafts, err := a.conversations.DeleteStaleThreads(ctx, repositories.DeleteStaleThreadsRequest{
		Origin:          conversation.ThreadOriginFormula,
		Before:          input.Before,
		Limit:           deleteStaleAskBatch,
		SubjectlessOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("delete stale formula threads: %w", err)
	}
	if drafts > 0 {
		a.logger.Info("removed formula conversations about templates never saved",
			zap.Int("deleted", drafts))
	}

	return &DeleteStaleAskThreadsResult{Deleted: deleted + drafts}, nil
}

// RemindPendingProposalsActivity brings proposals that have waited past the
// cut-off back to the people who can decide them, once each. It runs in the
// same sweep as expiry so a proposal is reminded about before it expires, not
// after.
func (a *Activities) RemindPendingProposalsActivity(
	ctx context.Context,
	input *RemindPendingProposalsInput,
) (*RemindPendingProposalsResult, error) {
	if a.notifier == nil {
		return &RemindPendingProposalsResult{}, nil
	}

	reminded, err := a.notifier.RemindPending(ctx, serviceports.RemindPendingProposalsRequest{
		Now:       input.Now,
		OlderThan: time.Duration(input.OlderThanSeconds) * time.Second,
		Limit:     reminderBatchLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("remind pending proposals: %w", err)
	}

	if reminded > 0 {
		a.logger.Info("reminded deciders of agent proposals still pending",
			zap.Int("reminded", reminded),
		)
	}

	return &RemindPendingProposalsResult{Reminded: reminded}, nil
}

// recordProposalsParams groups what filing a run's proposed writes needs.
type recordProposalsParams struct {
	Actor      *serviceports.RequestActor
	Definition *agentdefinition.Definition
	Run        *agent.AgentRun
	Actions    []serviceports.PendingAction
	Subject    *agentdefinition.RuntimeSubject
	TenantInfo pagination.TenantInfo
	Taint      *agent.RunTaint
}

// recordProposals files the run's proposed writes, once.
//
// A run records its proposals in one batch at the end, so a run that already
// has any is one whose earlier attempt got here. Re-recording them would put a
// second identical card in front of whoever has to decide, which is the same
// duplicate the step ledger prevents one layer down.
func (a *Activities) recordProposals(
	ctx context.Context,
	p recordProposalsParams,
) (*proposalrecorder.RecordResult, error) {
	existing, err := a.proposalRepo.ListByRun(ctx, repositories.ListAgentProposalsByRunRequest{
		RunID:      p.Run.ID,
		TenantInfo: p.TenantInfo,
	})
	if err != nil {
		return nil, fmt.Errorf("read the proposals this run already raised: %w", err)
	}
	if len(existing) > 0 {
		a.logger.Info("this run had already recorded its proposals; they are not recorded again",
			zap.String("run", p.Run.ID.String()),
			zap.Int("proposals", len(existing)),
		)

		return &proposalrecorder.RecordResult{Run: p.Run, Proposals: existing}, nil
	}

	recorded, err := a.recorder.Record(ctx, &proposalrecorder.RecordRequest{
		Actor:      p.Actor,
		Definition: p.Definition,
		Run:        p.Run,
		Actions:    p.Actions,
		Evidence:   subjectEvidence(p.Subject, p.Run.ID),
		Taint:      p.Taint,
	})
	if err != nil {
		return nil, fmt.Errorf("record proposals: %w", err)
	}

	return recorded, nil
}
