package agentflow

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func TestRunStampsEachEventWithTheWorkflowsClock(t *testing.T) {
	t.Parallel()

	lookup := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Result:   map[string]any{"status": "InTransit"},
	}
	h := newHarness(t, harnessParams{query: []agentports.AgentQueryTool{lookup}})
	h.replies(
		toolReply("shipment_get", map[string]any{"proNumber": "12345"}),
		textReply("It is in transit."),
	)
	started := time.Date(2026, time.March, 2, 9, 30, 0, 0, time.UTC)
	h.env.SetStartTime(started)

	result := h.run(t, runContext("shipment_get"))

	require.NotEmpty(t, result.Outcome.Events)
	for _, event := range result.Outcome.Events {
		assert.GreaterOrEqual(t, event.At, started.Unix(),
			"%s carries the instant it happened, read from the workflow's clock", event.Event)
		assert.Less(t, event.At, started.Add(time.Hour).Unix())
	}
}

func openDelegateEnv(t *testing.T, opener DelegateOpener) *testsuite.TestActivityEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	rt := agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  &agenttest.ScriptedCompletion{},
		QueryTools:  &agenttest.StubQueryRegistry{},
		ActionTools: &agenttest.StubActionRegistry{},
		Permissions: &agenttest.StubPermissions{},
	})
	env.RegisterActivity(NewActivities(ActivitiesParams{
		Logger:    zap.NewNop(),
		Runtime:   rt,
		Delegates: opener,
	}))

	return env
}

func delegateInput(t *testing.T) *OpenDelegateInput {
	t.Helper()

	rc := runContext()
	rc.Definition.ID = pulid.MustNew("agdef_")
	rc.StepOwner = agentports.RunStepOwner{
		Kind: agentports.RunStepOwnerAssistantTurn,
		ID:   pulid.MustNew("atrn_"),
	}
	delegate := delegateDefinition()

	return &OpenDelegateInput{
		Run: rc,
		Call: agentruntime.DelegateCall{
			Call:     agentports.ToolCall{ID: "call_task_7", Name: "delegate_task"},
			Delegate: agentdefinition.RuntimeDelegate{ID: delegate.ID, Name: delegate.Name},
			Task:     "Find load 12345.",
		},
	}
}

func TestOpenDelegateActivity_TracesTheOpeningAndLinksToTheDelegatesTrace(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	in := delegateInput(t)
	env := openDelegateEnv(t, &scriptedOpener{
		runtime: agentruntime.New(agentruntime.Params{
			Logger:      zap.NewNop(),
			Completion:  &agenttest.ScriptedCompletion{},
			QueryTools:  &agenttest.StubQueryRegistry{},
			ActionTools: &agenttest.StubActionRegistry{},
			Permissions: &agenttest.StubPermissions{},
		}),
		definition: delegateDefinition(),
	})
	var a *Activities
	_, err := env.ExecuteActivity(a.OpenDelegateActivity, in)
	require.NoError(t, err)

	turn := aitrace.ForRun(in.Run.StepOwner, nil)
	span := aitracetest.One(t, turn.TraceID, aitrace.SpanDelegateOpen)
	assert.Equal(t, turn.RootSpanID, span.Parent.SpanID())
	require.Len(t, span.Links, 1)
	delegate := aitrace.ForDelegate(in.Run.StepOwner.ID, "call_task_7")
	assert.Equal(t, delegate.TraceID, span.Links[0].SpanContext.TraceID())
	assert.Equal(t, delegate.RootSpanID, span.Links[0].SpanContext.SpanID())
	assert.Contains(t, span.Attributes, aitrace.AIDelegateCallID.String("call_task_7"))
	assert.Contains(t, span.Attributes, aitrace.AIDelegateAgentName.String("Shipment Desk"))
}

func TestOpenDelegateActivity_ADeclinedTaskSaysWhy(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	in := delegateInput(t)
	env := openDelegateEnv(t, &scriptedOpener{declined: "Shipment Desk is turned off."})
	var a *Activities
	_, err := env.ExecuteActivity(a.OpenDelegateActivity, in)
	require.Error(t, err)

	turn := aitrace.ForRun(in.Run.StepOwner, nil)
	span := aitracetest.One(t, turn.TraceID, aitrace.SpanDelegateOpen)
	assert.Equal(t, codes.Error, span.Status.Code)
	assert.Contains(t, span.Attributes,
		aitrace.AIDelegateDeclined.String("Shipment Desk is turned off."))
}

func TestModelCallActivity_TellsTheCompletionWhichAttemptItIs(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	rt := agentruntime.New(agentruntime.Params{
		Logger: zap.NewNop(),
		Completion: &agenttest.ScriptedCompletion{
			Turns: []*agentports.ChatCompletionResult{textReply("Done.")},
		},
		QueryTools:  &agenttest.StubQueryRegistry{},
		ActionTools: &agenttest.StubActionRegistry{},
		Permissions: &agenttest.StubPermissions{},
	})
	env.RegisterActivity(NewActivities(ActivitiesParams{Logger: zap.NewNop(), Runtime: rt}))

	runID := pulid.MustNew("ar_")
	req := &agentports.ChatCompletionRequest{Attribution: aiusageports.AIUsageAttribution{
		OwnerKind: agentports.RunStepOwnerAgentRun,
		OwnerID:   runID,
		RunID:     runID,
	}}
	var a *Activities
	_, err := env.ExecuteActivity(a.ModelCallActivity, &ModelCallInput{Request: req})
	require.NoError(t, err)

	anchor := aitrace.ForAttribution(&req.Attribution)
	span := aitracetest.One(t, anchor.TraceID, aitrace.SpanCompletion)
	assert.Contains(t, span.Attributes, aitrace.AIActivityAttempt.Int(1))
	assert.Contains(t, span.Attributes, aitrace.AIStream.Bool(false))
}
