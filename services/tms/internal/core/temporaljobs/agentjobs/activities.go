package agentjobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/pkg/pagination"
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

	Logger       *zap.Logger
	Definitions  repositories.AgentDefinitionRepository
	Controls     repositories.AgentControlRepository
	RunRepo      repositories.AgentRunRepository
	ProposalRepo repositories.AgentProposalRepository
	Runs         serviceports.AgentRunService
	Runtime      serviceports.AgentRuntime
	Contexts     serviceports.RuntimeContextBuilder
	Recorder     *proposalrecorder.Service
	BillingQueue serviceports.BillingQueueService
	Shipment     serviceports.ShipmentService
	Console      repositories.DispatchConsoleRepository `optional:"true"`
	Notifier     serviceports.AgentProposalNotifier     `optional:"true"`
	Content      serviceports.DocumentContentService    `optional:"true"`
	Plans        repositories.AgentPlanRepository       `optional:"true"`
}

type Activities struct {
	logger       *zap.Logger
	definitions  repositories.AgentDefinitionRepository
	controls     repositories.AgentControlRepository
	runRepo      repositories.AgentRunRepository
	proposalRepo repositories.AgentProposalRepository
	runs         serviceports.AgentRunService
	runtime      serviceports.AgentRuntime
	contexts     serviceports.RuntimeContextBuilder
	recorder     *proposalrecorder.Service
	notifier     serviceports.AgentProposalNotifier
	plans        repositories.AgentPlanRepository
	subjects     *SubjectContext
}

func NewActivities(p ActivitiesParams) *Activities {
	logger := p.Logger.Named("agent-activities")

	return &Activities{
		logger:       logger,
		definitions:  p.Definitions,
		controls:     p.Controls,
		runRepo:      p.RunRepo,
		proposalRepo: p.ProposalRepo,
		runs:         p.Runs,
		runtime:      p.Runtime,
		contexts:     p.Contexts,
		recorder:     p.Recorder,
		notifier:     p.Notifier,
		plans:        p.Plans,
		subjects: &SubjectContext{
			content:      p.Content,
			billingQueue: p.BillingQueue,
			shipments:    p.Shipment,
			console:      p.Console,
			logger:       logger,
		},
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

func (a *Activities) RunAgentActivity(
	ctx context.Context,
	input *RunAgentInput,
) (*RunAgentResult, error) {
	payload := input.Payload
	tenant := payload.tenantInfo()
	actor := agentActor(tenant)
	definition := input.Definition

	runtimeContext, err := a.contexts.Build(ctx, &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      actor,
		Trigger:    payload.Trigger,
		Subject:    input.Subject,
	})
	if err != nil {
		return nil, fmt.Errorf("build runtime context: %w", err)
	}

	activity.RecordHeartbeat(ctx, "running")

	outcome, err := a.runtime.Run(ctx, &serviceports.RunRequest{
		Definition: definition,
		Actor:      actor,
		Context:    runtimeContext,
		Input:      backgroundInput(payload, input.Subject),
		RunID:      payload.RunID,
		Unattended: true,
		Emit: func(serviceports.StreamEvent) {
			activity.RecordHeartbeat(ctx, "working")
		},
	})
	if err != nil {
		return nil, err
	}

	run, err := a.runRepo.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         payload.RunID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load agent run: %w", err)
	}

	recorded, err := a.recorder.Record(ctx, &proposalrecorder.RecordRequest{
		Actor:      actor,
		Definition: definition,
		Run:        run,
		Actions:    outcome.Actions,
		Evidence:   subjectEvidence(input.Subject, payload.RunID),
	})
	if err != nil {
		return nil, fmt.Errorf("record proposals: %w", err)
	}

	pending := 0
	for _, proposal := range recorded.Proposals {
		if proposal.Status == agent.ProposalStatusPending {
			pending++
		}
	}

	run.ModelIdentifier = outcome.Model
	run.Summary = stringutils.Ellipsize(strings.TrimSpace(outcome.Reply), maxSummaryChars)
	if pending > 0 {
		run.Status = agent.RunStatusAwaitingDecision
	}
	if _, err = a.runRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("update agent run: %w", err)
	}

	return &RunAgentResult{
		Reply:            outcome.Reply,
		Model:            outcome.Model,
		ToolCallsUsed:    outcome.ToolCallsUsed,
		Exhausted:        outcome.Exhausted,
		ProposalsRaised:  len(recorded.Proposals),
		PendingProposals: pending,
	}, nil
}

func (a *Activities) CompleteRunActivity(ctx context.Context, input *CompleteRunInput) error {
	return a.updateRun(ctx, input.TenantInfo, input.RunID, func(run *agent.AgentRun) {
		run.Status = input.Status
		if input.Error != "" {
			run.ErrorMessage = stringutils.Ellipsize(input.Error, maxSummaryChars)
		}
		completedAt := timeutils.NowUnix()
		run.CompletedAt = &completedAt
	})
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

func (a *Activities) ListDueDefinitionsActivity(
	ctx context.Context,
	input *ListDueDefinitionsInput,
) (*ListDueDefinitionsResult, error) {
	definitions, err := a.definitions.ListDueAcrossTenants(ctx, repositories.ListDueAcrossTenantsRequest{
		Now:   input.Now,
		Limit: input.Limit,
	})
	if err != nil {
		return nil, err
	}

	due := make([]DueDefinition, 0, len(definitions))
	for _, definition := range definitions {
		if definition.NextRunAt == nil {
			continue
		}
		due = append(due, DueDefinition{
			DefinitionID:   definition.ID,
			OrganizationID: definition.OrganizationID,
			BusinessUnitID: definition.BusinessUnitID,
			NextRunAt:      *definition.NextRunAt,
		})
	}

	return &ListDueDefinitionsResult{Due: due}, nil
}

func (a *Activities) StartDueRunActivity(
	ctx context.Context,
	due *DueDefinition,
) (*StartDueRunResult, error) {
	tenant := pagination.TenantInfo{OrgID: due.OrganizationID, BuID: due.BusinessUnitID}
	now := timeutils.NowUnix()

	definition, err := a.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         due.DefinitionID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	if !definition.Enabled || !definition.IsBackground() {
		return &StartDueRunResult{Skipped: "not_runnable"}, nil
	}

	open, err := a.runRepo.CountOpen(ctx, repositories.CountOpenAgentRunsRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
	})
	if err != nil {
		return nil, err
	}
	if open >= definition.MaxConcurrentRuns {
		return &StartDueRunResult{Skipped: "at_concurrency_limit"}, nil
	}

	var next *int64
	ended := definition.EndsAt != nil && *definition.EndsAt <= now
	if !ended {
		computed, cErr := definition.ComputeNextRun(now)
		if cErr != nil {
			return nil, fmt.Errorf("compute next run: %w", cErr)
		}
		if definition.EndsAt == nil || computed < *definition.EndsAt {
			next = &computed
		}
	}

	expected := due.NextRunAt
	claimed, err := a.definitions.MarkRun(ctx, repositories.MarkAgentDefinitionRunRequest{
		ID:                definition.ID,
		TenantInfo:        tenant,
		LastRunAt:         now,
		NextRunAt:         next,
		ExpectedNextRunAt: &expected,
	})
	if err != nil {
		return nil, err
	}
	if !claimed {
		return &StartDueRunResult{Skipped: "slot_already_claimed"}, nil
	}

	if ended {
		definition.Enabled = false
		definition.NextRunAt = nil
		if _, uErr := a.definitions.Update(ctx, definition); uErr != nil {
			a.logger.Warn("agent sweep: could not disable an ended definition",
				zap.String("definitionId", definition.ID.String()),
				zap.Error(uErr),
			)
		}

		return &StartDueRunResult{Skipped: "ended"}, nil
	}

	run, err := a.runs.StartForDefinition(ctx, &serviceports.StartAgentRunForDefinitionRequest{
		DefinitionID: definition.ID,
		Trigger:      definition.TriggerMode.RunTrigger(),
		Slot:         due.NextRunAt,
		TenantInfo:   tenant,
	}, agentActor(tenant))
	if err != nil {
		return nil, err
	}

	return &StartDueRunResult{Started: true, RunID: run.ID.String()}, nil
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

	return nil
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
	var builder strings.Builder
	switch payload.Trigger {
	case agent.RunTriggerEvent:
		builder.WriteString("An event started this run")
		if payload.EventKind != "" {
			builder.WriteString(": ")
			builder.WriteString(string(payload.EventKind))
		}
		builder.WriteString(".")
	case agent.RunTriggerScheduled:
		builder.WriteString("This is a scheduled run.")
	case agent.RunTriggerContinuous:
		builder.WriteString("This is one pass of a continuous run.")
	default:
		builder.WriteString("A person started this run.")
	}
	if subject != nil {
		builder.WriteString(" It concerns ")
		builder.WriteString(subject.Label)
		builder.WriteString(" (")
		builder.WriteString(string(subject.Type))
		builder.WriteString(" ")
		builder.WriteString(subject.ID)
		builder.WriteString("), described in the runtime context.")
	}
	builder.WriteString(" Follow your instructions: look up what you need, act through your tools " +
		"where you are allowed to, propose what needs a person, and finish with a short report of " +
		"what you found and did.")

	return builder.String()
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

func hashSubject(definition *agentdefinition.Definition, subject *agentdefinition.RuntimeSubject) string {
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
