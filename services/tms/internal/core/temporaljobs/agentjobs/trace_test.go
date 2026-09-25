package agentjobs

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

func TestFinishRun_EmitsTheRunsRootAtItsAnchorFromItsRealStart(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	runID := pulid.MustNew("ar_")
	definition := loopReplayDefinition(pulid.MustNew("agdef_"))
	definition.Version = 11
	payload := &AgentRunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		RunID:   runID,
		Trigger: agent.RunTriggerEvent,
		Origin:  "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}

	emitRunRoot(t.Context(), &FinishRunInput{
		Payload:    payload,
		Definition: definition,
		Run: &serviceports.RunResult{
			ToolCallsUsed: 4,
			Usage:         &serviceports.RunUsage{ModelCalls: 3, InputTokens: 900, OutputTokens: 60},
		},
	}, &agent.AgentRun{CreatedAt: 1_790_000_100, StartedAt: 1_790_000_105},
		&FinishRunResult{PendingProposals: 2})

	anchor := aitrace.AnchorFor(aitrace.AnchorAgentRun, runID.String())
	root := aitracetest.One(t, anchor.TraceID, "invoke_agent Recorded desk")
	assert.Equal(t, anchor.RootSpanID, root.SpanContext.SpanID())
	assert.True(t, root.StartTime.Equal(time.Unix(1_790_000_100, 0)))
	for _, want := range []attribute.KeyValue{
		aitrace.GenAIAgentID.String(definition.ID.String()),
		aitrace.AIAgentVersion.Int64(11),
		aitrace.AIRunID.String(runID.String()),
		aitrace.AIOwnerKind.String(string(serviceports.RunStepOwnerAgentRun)),
		aitrace.AITrigger.String(string(agent.RunTriggerEvent)),
		aitrace.AIStatus.String(string(agent.RunStatusAwaitingDecision)),
		aitrace.AITokensInput.Int64(900),
		aitrace.AIToolCalls.Int(4),
		aitrace.AIPurpose.String(aitrace.PurposeLive),
	} {
		assert.Contains(t, root.Attributes, want)
	}
	require.Len(t, root.Links, 1)
	assert.Equal(t, "00f067aa0ba902b7", root.Links[0].SpanContext.SpanID().String())
}

func TestFinishRun_AFailedRunsRootSaysHowItFailed(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	runID := pulid.MustNew("ar_")
	emitRunRoot(t.Context(), &FinishRunInput{
		Payload:    &AgentRunPayload{RunID: runID, Trigger: agent.RunTriggerScheduled},
		Definition: loopReplayDefinition(pulid.MustNew("agdef_")),
		Failure:    &modelcall.Failure{TimedOut: true},
	}, &agent.AgentRun{}, &FinishRunResult{})

	anchor := aitrace.AnchorFor(aitrace.AnchorAgentRun, runID.String())
	root := aitracetest.One(t, anchor.TraceID, "invoke_agent Recorded desk")
	assert.Equal(t, codes.Error, root.Status.Code)
	assert.Contains(t, root.Attributes, aitrace.ErrorType.String("timeout"))
	assert.Contains(t, root.Attributes, aitrace.AIStatus.String(string(agent.RunStatusFailed)))
}

func TestFinishReplay_AnEvaluationIsItsOwnTraceMarkedAsOne(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	started := int64(1_790_000_200)
	evaluation := &agent.Evaluation{
		ID:                pulid.MustNew(agent.EvaluationIDPrefix),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		DefinitionVersion: 6,
		Trigger:           agent.RunTriggerEvent,
		StartedAt:         &started,
	}
	emitEvaluationRoot(t.Context(), evaluation, &serviceports.RunResult{ToolCallsUsed: 2},
		agent.EvaluationStatusCompleted, "")

	anchor := aitrace.AnchorFor(aitrace.AnchorEvaluation, evaluation.ID.String())
	assert.Equal(t, anchor, aitrace.ForAttribution(&serviceports.AIUsageAttribution{
		RunID: evaluation.ID,
	}), "the replay's model calls are traced under the same anchor")
	root := aitracetest.One(t, anchor.TraceID, aitrace.OperationInvokeAgent)
	assert.Contains(t, root.Attributes, aitrace.AIPurpose.String(aitrace.PurposeEvaluation))
	assert.Contains(t, root.Attributes, aitrace.AISimulation.Bool(true))
	assert.Contains(t, root.Attributes, aitrace.AIAgentVersion.Int64(6))
	assert.True(t, root.StartTime.Equal(time.Unix(started, 0)))
}

type expiringProposals struct {
	repositories.AgentProposalRepository

	expired int
}

func (p *expiringProposals) ExpirePendingByRun(
	context.Context,
	repositories.ExpireAgentProposalsByRunRequest,
) (int, error) {
	return p.expired, nil
}

type expiringRuns struct {
	repositories.AgentRunRepository
}

func (expiringRuns) GetByID(
	_ context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return &agent.AgentRun{ID: req.ID}, nil
}

func (expiringRuns) Update(_ context.Context, run *agent.AgentRun) (*agent.AgentRun, error) {
	return run, nil
}

func TestExpireProposals_TracesTheExpiryAndLinksItToTheRun(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	a := &Activities{
		logger:       zap.NewNop(),
		proposalRepo: &expiringProposals{expired: 2},
		runRepo:      expiringRuns{},
	}
	runID := pulid.MustNew("ar_")
	ctx, sweep := otel.Tracer("agentjobs-test").Start(t.Context(), "RunActivity:ExpireProposalsActivity")
	require.NoError(t, a.ExpireProposalsActivity(ctx, &ExpireProposalsInput{
		RunID:      runID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}))
	sweep.End()

	span := aitracetest.One(t, sweep.SpanContext().TraceID(), "trenova.ai.proposal.expire")
	assert.Equal(t, sweep.SpanContext().SpanID(), span.Parent.SpanID())
	assert.Contains(t, span.Attributes, aitrace.AIExpired.Int(2))
	assert.Contains(t, span.Attributes, aitrace.AIRunID.String(runID.String()))
	require.Len(t, span.Links, 1)
	run := aitrace.AnchorFor(aitrace.AnchorAgentRun, runID.String())
	assert.Equal(t, run.RootSpanID, span.Links[0].SpanContext.SpanID())
}
