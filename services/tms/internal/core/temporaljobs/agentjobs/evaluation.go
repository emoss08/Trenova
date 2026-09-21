package agentjobs

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// ReplayRunActivity replays the evaluation's source run against its agent as
// it is now and stores what the replay would have done beside what the
// original did.
//
// The replay runs with simulation forced on, so an automatic write is
// previewed and a proposal is only collected, never recorded for anyone to
// decide. Nothing it does reaches a record, and nothing it does reaches the
// activity feed: the evaluation row is the only thing it writes.
func (a *Activities) ReplayRunActivity(
	ctx context.Context,
	payload *AgentEvaluationPayload,
) (*ReplayRunResult, error) {
	tenant := payload.tenantInfo()

	evaluation, err := a.evaluations.GetByID(ctx, repositories.GetAgentEvaluationByIDRequest{
		ID:         payload.EvaluationID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"agent evaluation unavailable", "EvaluationUnavailable", err,
		)
	}
	if evaluation.Status.Terminal() {
		return &ReplayRunResult{Model: evaluation.Model, ToolCallsUsed: evaluation.ToolCallsUsed}, nil
	}

	source, err := a.runRepo.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         evaluation.SourceRunID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"source run unavailable", "SourceRunUnavailable", err,
		)
	}

	definition, err := a.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         evaluation.AgentDefinitionID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"agent definition unavailable", "DefinitionUnavailable", err,
		)
	}

	originals, err := a.originalProposals(ctx, tenant, source.ID)
	if err != nil {
		return nil, err
	}

	replay, err := a.replayInput(ctx, tenant, source, definition)
	if err != nil {
		return nil, err
	}

	startedAt := timeutils.NowUnix()
	evaluation.Status = agent.EvaluationStatusRunning
	evaluation.StartedAt = &startedAt
	evaluation.Input = replay.input
	evaluation.DefinitionVersion = definition.Version
	evaluation.PromptVersion = promptVersion
	evaluation.OriginalProposals = len(originals)
	if evaluation, err = a.evaluations.Update(ctx, evaluation); err != nil {
		return nil, fmt.Errorf("mark evaluation running: %w", err)
	}

	activity.RecordHeartbeat(ctx, "replaying")

	// The replay's agent is the current one with simulation forced on. The
	// copy keeps the stored definition as it is: an evaluation must not
	// switch the live agent into simulation.
	replayed := *definition
	replayed.SimulationMode = true

	actor := agentActor(tenant)
	outcome, err := a.runtime.Run(ctx, &serviceports.RunRequest{
		Definition: &replayed,
		Actor:      actor,
		Context:    replay.context,
		History:    replay.history,
		Input:      replay.input,
		RunID:      evaluation.ID,
		Unattended: true,
		Emit: func(serviceports.StreamEvent) {
			activity.RecordHeartbeat(ctx, "replaying")
		},
	})
	if err != nil {
		return nil, err
	}

	actions := make([]agent.ReplayAction, 0, len(outcome.Actions))
	for _, action := range outcome.Actions {
		actions = append(actions, agent.ReplayAction{
			ToolName:   action.ToolName,
			Arguments:  action.Arguments,
			Rationale:  action.Rationale,
			Tier:       action.Tier,
			Simulation: action.Simulation,
		})
	}

	completedAt := timeutils.NowUnix()
	evaluation.Status = agent.EvaluationStatusCompleted
	evaluation.CompletedAt = &completedAt
	evaluation.Model = outcome.Model
	evaluation.ProviderID = outcome.ProviderID
	evaluation.Reply = stringutils.Ellipsize(strings.TrimSpace(outcome.Reply), maxSummaryChars)
	evaluation.Actions = actions
	evaluation.Comparison = agent.CompareReplay(originals, actions)
	evaluation.ToolCallsUsed = outcome.ToolCallsUsed
	if _, err = a.evaluations.Update(ctx, evaluation); err != nil {
		return nil, fmt.Errorf("store evaluation outcome: %w", err)
	}

	return &ReplayRunResult{
		Model:         outcome.Model,
		ToolCallsUsed: outcome.ToolCallsUsed,
		Actions:       len(actions),
	}, nil
}

// FailEvaluationActivity records why a replay did not finish, so the row
// never sits at Running with nothing to say.
func (a *Activities) FailEvaluationActivity(ctx context.Context, input *FailEvaluationInput) error {
	evaluation, err := a.evaluations.GetByID(ctx, repositories.GetAgentEvaluationByIDRequest{
		ID:         input.EvaluationID,
		TenantInfo: input.TenantInfo,
	})
	if err != nil {
		return err
	}
	if evaluation.Status.Terminal() {
		return nil
	}

	completedAt := timeutils.NowUnix()
	evaluation.Status = agent.EvaluationStatusFailed
	evaluation.ErrorMessage = stringutils.Ellipsize(input.Error, maxSummaryChars)
	evaluation.CompletedAt = &completedAt
	_, err = a.evaluations.Update(ctx, evaluation)

	return err
}

// originalProposals reads what the source run proposed and the latest
// decision on each, in the shape the comparison judges.
func (a *Activities) originalProposals(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
) ([]agent.OriginalProposal, error) {
	proposals, err := a.proposalRepo.ListByRun(ctx, repositories.ListAgentProposalsByRunRequest{
		RunID:      runID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("list source proposals: %w", err)
	}

	ids := make([]pulid.ID, 0, len(proposals))
	for _, proposal := range proposals {
		ids = append(ids, proposal.ID)
	}
	decisions, err := a.decisions.ListByProposals(
		ctx,
		repositories.ListAgentDecisionsByProposalsRequest{ProposalIDs: ids, TenantInfo: tenant},
	)
	if err != nil {
		return nil, fmt.Errorf("list source decisions: %w", err)
	}

	// Newest first, so the first decision seen for a proposal is its latest.
	latest := make(map[pulid.ID]agent.DecisionType, len(decisions))
	for _, decision := range decisions {
		if decision.ProposalID == nil {
			continue
		}
		if _, seen := latest[*decision.ProposalID]; !seen {
			latest[*decision.ProposalID] = decision.Decision
		}
	}

	originals := make([]agent.OriginalProposal, 0, len(proposals))
	for _, proposal := range proposals {
		originals = append(originals, agent.OriginalProposal{
			ID:       proposal.ID,
			ToolName: proposal.ToolName,
			Params:   proposal.ToolParams,
			Status:   proposal.Status,
			Decision: latest[proposal.ID],
		})
	}

	return originals, nil
}

type replayInput struct {
	input   string
	history []conversation.Message
	context agentdefinition.RuntimeContext
}

// replayInput rebuilds what the source run was asked. A background run is
// asked about its subject, described as it is now; a chat run is asked the
// person's message that produced it, with the conversation before it as
// history. Both are as faithful as a replay can be without freezing the
// world: the subject may have moved on, and the comparison reads that as a
// change in the agent, which is the one thing the reader has to weigh.
func (a *Activities) replayInput(
	ctx context.Context,
	tenant pagination.TenantInfo,
	source *agent.AgentRun,
	definition *agentdefinition.Definition,
) (*replayInput, error) {
	actor := agentActor(tenant)

	if source.SubjectType == agent.SubjectAssistantThread {
		return a.chatReplayInput(ctx, tenant, source, definition, actor)
	}

	subject, err := a.subjects.Describe(ctx, tenant, source.SubjectType, source.SubjectID)
	if err != nil {
		return nil, fmt.Errorf("describe subject: %w", err)
	}

	runtimeContext, err := a.contexts.Build(ctx, &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      actor,
		Trigger:    source.Trigger,
		Subject:    subject,
	})
	if err != nil {
		return nil, fmt.Errorf("build runtime context: %w", err)
	}

	return &replayInput{
		input:   backgroundInput(&AgentRunPayload{Trigger: source.Trigger}, subject),
		context: runtimeContext,
	}, nil
}

func (a *Activities) chatReplayInput(
	ctx context.Context,
	tenant pagination.TenantInfo,
	source *agent.AgentRun,
	definition *agentdefinition.Definition,
	actor *serviceports.RequestActor,
) (*replayInput, error) {
	if a.conversations == nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"conversation history unavailable", "HistoryUnavailable", nil,
		)
	}

	messages, err := a.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:   source.SubjectID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("list thread messages: %w", err)
	}

	// The turn that produced the run is the last user message written before
	// the run was opened. The run is opened as the turn's proposals are
	// recorded, after the message was saved, so the order holds.
	turn := -1
	for i, message := range messages {
		if message.Role != conversation.RoleUser || message.CreatedAt > source.CreatedAt {
			continue
		}
		turn = i
	}
	if turn < 0 {
		return nil, temporal.NewNonRetryableApplicationError(
			"the conversation no longer holds the message that produced this run",
			"TurnUnavailable", nil,
		)
	}

	runtimeContext, err := a.contexts.Build(ctx, &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      actor,
		Trigger:    agent.RunTriggerChat,
		Page:       messages[turn].PageContext,
	})
	if err != nil {
		return nil, fmt.Errorf("build runtime context: %w", err)
	}

	return &replayInput{
		input:   messages[turn].Content,
		history: messages[:turn],
		context: runtimeContext,
	}, nil
}
