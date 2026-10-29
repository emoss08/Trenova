package agentruntime

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type versionSequence struct {
	mu       sync.Mutex
	versions []int64
	reads    int
}

func (v *versionSequence) Version(
	context.Context,
	pagination.TenantInfo,
	agenttoolports.ToolTarget,
) (int64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	version := v.versions[min(v.reads, len(v.versions)-1)]
	v.reads++

	return version, nil
}

type tracedDispatch struct {
	rt         *Service
	req        *agentruntimeports.RunRequest
	ledger     *runtimetest.KeyedLedger
	shipmentID pulid.ID
	call       toolcall.Call
}

func newTracedDispatch(t *testing.T, tier, ceiling agent.AutonomyTier) *tracedDispatch {
	t.Helper()
	aitracetest.Install()

	tool := &targetedStubTool{actionTool("shipment_place_hold", tier, nil)}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agenttoolports.AgentTool{tool},
	}, nil)
	rewire(rt, func(p *Params) { p.Versions = &versionSequence{versions: []int64{7, 8}} })

	definition := runtimetest.Definition("shipment_place_hold")
	definition.ID = pulid.MustNew("agdef_")
	definition.Version = 3
	definition.AutonomyCeiling = ceiling
	ledger := runtimetest.NewKeyedLedger()
	shipmentID := pulid.MustNew("shp_")

	return &tracedDispatch{
		rt: rt,
		req: &agentruntimeports.RunRequest{
			Definition: definition,
			Actor:      runtimetest.Actor(),
			Input:      "Hold the shipment",
			Steps:      ledger,
			StepOwner: agentruntimeports.RunStepOwner{
				Kind: agentruntimeports.RunStepOwnerAssistantTurn,
				ID:   pulid.MustNew("atrn_"),
			},
			Attempt: 1,
		},
		ledger:     ledger,
		shipmentID: shipmentID,
		call: toolcall.Call{
			Call: llmports.ToolCall{
				ID:        "call_hold",
				Name:      "shipment_place_hold",
				Arguments: map[string]any{"shipmentId": shipmentID.String()},
			},
		},
	}
}

func (d *tracedDispatch) anchor() aitrace.Anchor {
	return aitrace.ForRun(d.req.StepOwner, d.req.Delegation)
}

func (d *tracedDispatch) settled(t *testing.T) agentruntimeports.RunStep {
	t.Helper()

	require.Len(t, d.ledger.Steps, 1)
	for _, step := range d.ledger.Steps {
		return step
	}

	return agentruntimeports.RunStep{}
}

func stringAttr(t *testing.T, span tracetest.SpanStub, key attribute.Key) string {
	t.Helper()

	value, ok := aitracetest.Attr(span, key)
	require.True(t, ok, "span %q has no %s", span.Name, key)

	return value.AsString()
}

func TestDispatchStep_TracesAProposalAndStampsItWithTheSpan(t *testing.T) {
	t.Parallel()

	d := newTracedDispatch(t, agent.TierPropose, agent.TierPropose)
	outcome := d.rt.DispatchStep(t.Context(), d.req, &d.call)

	require.NotNil(t, outcome.Action)
	action := outcome.Action
	span := aitracetest.One(t, d.anchor().TraceID, "execute_tool shipment_place_hold")
	assert.Equal(t, d.anchor().RootSpanID, span.Parent.SpanID())

	step := d.settled(t)
	assert.Equal(t, step.Key, stringAttr(t, span, aitrace.AIStepKey))
	assert.Equal(t, aitrace.StepStateFresh, stringAttr(t, span, aitrace.AIStepState))
	assert.Equal(t, aitrace.OutcomeProposed, stringAttr(t, span, aitrace.AIOutcome))
	assert.Equal(t, string(agent.TierPropose), stringAttr(t, span, aitrace.AITier))
	assert.Equal(
		t,
		string(agenttooltrust.TierSourcePolicyDefault),
		stringAttr(t, span, aitrace.AITierSource),
	)
	assert.Equal(t, "call_hold", stringAttr(t, span, aitrace.GenAIToolCallID))
	assert.Equal(
		t,
		string(agenttooltrust.ToolEffectChange),
		stringAttr(t, span, aitrace.AIToolEffect),
	)
	assert.Contains(t, span.Attributes, aitrace.AIHeldBy.StringSlice(action.HeldBy))
	assert.Contains(t, span.Attributes, aitrace.AITainted.Bool(false))
	assert.Equal(t, codes.Unset, span.Status.Code)

	assert.True(t, action.ProposalID.IsNotNil())
	assert.Equal(t, "ap_", action.ProposalID.Prefix())
	assert.Equal(t, action.ProposalID.String(), stringAttr(t, span, aitrace.AIProposalID))
	assert.Equal(t, span.SpanContext.TraceID().String(), action.TraceID)
	assert.Equal(t, span.SpanContext.SpanID().String(), action.SpanID)
	assert.Equal(t, agenttooltrust.TierSourcePolicyDefault, action.TierSource)
	assert.Equal(t, step.Key, action.StepKey)
	require.NotNil(t, action.Target)
	assert.Equal(t, int64(7), action.Target.Version)

	assert.Equal(t, action.TraceID, step.TraceID, "the step names the span it ran in")
	assert.Equal(t, action.SpanID, step.SpanID)
	assert.Equal(t, d.req.Definition.ID, step.DefinitionID)
	require.NotNil(t, step.DefinitionVersion)
	assert.Equal(t, int64(3), *step.DefinitionVersion)
	assert.Empty(t, step.DelegateCallID)
	assert.Equal(t, aitrace.OutcomeProposed, step.Outcome.Verdict)
	assert.Empty(t, step.Outcome.Reason)
}

func TestDispatchStep_AReplayedCallKeepsItsProposalAndSaysSo(t *testing.T) {
	t.Parallel()

	d := newTracedDispatch(t, agent.TierPropose, agent.TierPropose)
	first := d.rt.DispatchStep(t.Context(), d.req, &d.call)
	d.req.Attempt = 2
	second := d.rt.DispatchStep(t.Context(), d.req, &d.call)

	require.NotNil(t, first.Action)
	require.NotNil(t, second.Action)
	assert.Equal(t, first.Action.ProposalID, second.Action.ProposalID,
		"a retry hands back the proposal the first attempt minted, not a new one")

	spans := aitracetest.Named(d.anchor().TraceID, "execute_tool shipment_place_hold")
	require.Len(t, spans, 2)
	assert.Equal(t, aitrace.StepStateFresh, stringAttr(t, spans[0], aitrace.AIStepState))
	assert.Equal(t, aitrace.StepStateReplayed, stringAttr(t, spans[1], aitrace.AIStepState))
	assert.Equal(t, aitrace.OutcomeProposed, stringAttr(t, spans[1], aitrace.AIOutcome))
	assert.Equal(t, first.Action.ProposalID.String(), stringAttr(t, spans[1], aitrace.AIProposalID))
}

func TestDispatchStep_RecordsWhyACallWasRefused(t *testing.T) {
	t.Parallel()

	d := newTracedDispatch(t, agent.TierPropose, agent.TierPropose)
	rewire(d.rt, func(p *Params) {
		p.Permissions = &stubPermissions{Denied: map[string]bool{
			permission.ResourceShipmentMove.String() + ":" + string(permission.OpUpdate): true,
		}}
	})

	outcome := d.rt.DispatchStep(t.Context(), d.req, &d.call)

	assert.True(t, outcome.Failed)
	assert.Nil(t, outcome.Action)
	step := d.settled(t)
	assert.Equal(t, aitrace.OutcomeDenied, step.Outcome.Verdict)
	assert.Equal(t, "lacks update access to shipment_move", step.Outcome.Reason)

	span := aitracetest.One(t, d.anchor().TraceID, "execute_tool shipment_place_hold")
	assert.Equal(t, codes.Error, span.Status.Code)
	assert.Equal(t, aitrace.OutcomeDenied, stringAttr(t, span, aitrace.ErrorType))
	assert.Equal(t, aitrace.OutcomeDenied, stringAttr(t, span, aitrace.AIOutcome))
}

func TestDispatchStep_WrapsAnAutomaticWriteAndItsVersions(t *testing.T) {
	t.Parallel()

	d := newTracedDispatch(t, agent.TierAutoExecute, agent.TierAutoExecute)
	outcome := d.rt.DispatchStep(t.Context(), d.req, &d.call)

	require.NotNil(t, outcome.Action)
	action := outcome.Action
	assert.True(t, action.Executed)
	require.NotNil(t, action.Target)
	assert.Equal(t, int64(7), action.Target.Version, "the version before the write")
	require.NotNil(t, action.ExecutedVersion)
	assert.Equal(t, int64(8), *action.ExecutedVersion, "the version the write left")
	assert.NotZero(t, action.ExecutedAt, "when the write ran, not when it was filed")

	tool := aitracetest.One(t, d.anchor().TraceID, "execute_tool shipment_place_hold")
	assert.Equal(t, aitrace.OutcomeRan, stringAttr(t, tool, aitrace.AIOutcome))
	write := aitracetest.One(t, d.anchor().TraceID, "trenova.ai.write shipment")
	assert.Equal(t, tool.SpanContext.SpanID(), write.Parent.SpanID())
	for _, want := range []attribute.KeyValue{
		aitrace.AIEntityType.String("shipment"),
		aitrace.AIEntityID.String(d.shipmentID.String()),
		aitrace.AIVersionBefore.Int64(7),
		aitrace.AIVersionAfter.Int64(8),
		aitrace.AIProposalID.String(action.ProposalID.String()),
		aitrace.AISimulated.Bool(false),
	} {
		assert.Contains(t, write.Attributes, want)
	}
	assert.Equal(t, aitrace.OutcomeRan, d.settled(t).Outcome.Verdict)
}

func TestDispatchStep_ADelegatesCallIsTracedUnderItsTaskAndLinked(t *testing.T) {
	t.Parallel()

	d := newTracedDispatch(t, agent.TierPropose, agent.TierPropose)
	d.req.Delegation = &agentruntimeports.Delegation{CallID: "call_task_1", StepScope: "scope-1"}
	turn := aitrace.ForRun(d.req.StepOwner, nil)
	ctx, activity := otel.Tracer("agentruntime-test").
		Start(aitrace.ContextWithAnchor(t.Context(), turn), "RunActivity:shipment_place_hold")

	outcome := d.rt.DispatchStep(ctx, d.req, &d.call)
	activity.End()

	require.NotNil(t, outcome.Action)
	delegate := aitrace.ForDelegate(d.req.StepOwner.ID, "call_task_1")
	span := aitracetest.One(t, delegate.TraceID, "execute_tool shipment_place_hold")
	assert.Equal(t, delegate.RootSpanID, span.Parent.SpanID())
	require.Len(t, span.Links, 1)
	assert.Equal(t, activity.SpanContext().SpanID(), span.Links[0].SpanContext.SpanID(),
		"the delegate's call links back to the activity that ran it in the turn's trace")
	assert.Equal(t, "call_task_1", stringAttr(t, span, aitrace.AIDelegateCallID))

	step := d.settled(t)
	assert.Equal(t, "call_task_1", step.DelegateCallID)
	assert.Equal(t, delegate.TraceID.String(), step.TraceID)
	assert.Equal(t, delegate.TraceID.String(), outcome.Action.TraceID)
}

func TestCompletionRequest_AttributesTheTurnItsTaskAndTheAgentsVersion(t *testing.T) {
	t.Parallel()

	d := newTracedDispatch(t, agent.TierPropose, agent.TierPropose)
	d.req.Delegation = &agentruntimeports.Delegation{CallID: "call_task_2"}
	d.req.ThreadID = pulid.MustNew("athr_")

	attribution := d.rt.OpenTurn(t.Context(), d.req).completionRequest().Attribution

	assert.Equal(t, agentruntimeports.RunStepOwnerAssistantTurn, attribution.OwnerKind)
	assert.Equal(t, d.req.StepOwner.ID, attribution.OwnerID)
	assert.Equal(t, "call_task_2", attribution.DelegateCallID)
	require.NotNil(t, attribution.DefinitionVersion)
	assert.Equal(t, int64(3), *attribution.DefinitionVersion)
	assert.True(t, attribution.RunID.IsNil(), "a chat turn has no run until it is saved")
	assert.Equal(t, d.req.ThreadID, attribution.ThreadID)
}

func TestStreamCompletion_OpensTheCompletionSpanUnderTheTurn(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	rt := newRuntime(&scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.TextTurn("Load 12345 is in Memphis."),
	}}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	turnID := pulid.MustNew("atrn_")
	req := &llmports.ChatCompletionRequest{
		Attribution: aiusageports.AIUsageAttribution{
			OwnerKind: agentruntimeports.RunStepOwnerAssistantTurn,
			OwnerID:   turnID,
			Feature:   "AgentTurn",
		},
	}
	ctx := aitrace.WithCallOrigin(t.Context(), aitrace.CallOrigin{ActivityAttempt: 2, Stream: true})

	reply, err := rt.streamer.StreamCompletion(ctx, req, func(kernelports.StreamEvent) {})
	require.NoError(t, err)
	assert.False(t, reply.Looped)

	anchor := aitrace.ForAttribution(&req.Attribution)
	span := aitracetest.One(t, anchor.TraceID, aitrace.SpanCompletion)
	assert.Equal(t, anchor.RootSpanID, span.Parent.SpanID())
	for _, want := range []attribute.KeyValue{
		aitrace.AIActivityAttempt.Int(2),
		aitrace.AIStream.Bool(true),
		aitrace.AILooped.Bool(false),
		aitrace.AIFeature.String("AgentTurn"),
		aitrace.AIAttempts.Int(0),
	} {
		assert.Contains(t, span.Attributes, want)
	}
}

func TestRun_TotalsWhatEveryModelCallUsed(t *testing.T) {
	t.Parallel()

	first := decimal.RequireFromString("0.010")
	second := decimal.RequireFromString("0.025")
	read := runtimetest.ToolTurn("shipment_get", map[string]any{"proNumber": "12345"})
	read.InputTokens, read.OutputTokens, read.CostUSD = 1000, 40, &first
	answer := runtimetest.TextTurn("It is in Memphis.")
	answer.InputTokens, answer.OutputTokens, answer.CostUSD = 1200, 12, &second

	rt := newRuntime(&scriptedCompletion{Turns: []*llmports.ChatCompletionResult{read, answer}},
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool("shipment_get", map[string]any{"city": "Memphis"}, nil),
		}}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_get"),
		Actor:      runtimetest.Actor(),
		Input:      "Where is 12345?",
	})
	require.NoError(t, err)

	require.NotNil(t, result.Usage)
	assert.Equal(t, 2, result.Usage.ModelCalls)
	assert.Equal(t, int64(2200), result.Usage.InputTokens)
	assert.Equal(t, int64(52), result.Usage.OutputTokens)
	require.NotNil(t, result.Usage.CostUSD)
	assert.True(t, decimal.RequireFromString("0.035").Equal(*result.Usage.CostUSD))
}

func TestRunUsage_AddsWithoutSharingItsCost(t *testing.T) {
	t.Parallel()

	cost := decimal.RequireFromString("0.5")
	var total *agentruntimeports.RunUsage
	total = total.Add(&agentruntimeports.RunUsage{ModelCalls: 1, InputTokens: 10, CostUSD: &cost})
	total = total.Add(&agentruntimeports.RunUsage{ModelCalls: 2, OutputTokens: 5})
	total = total.Add(nil)
	cost = decimal.RequireFromString("9")

	assert.Equal(t, 3, total.ModelCalls)
	assert.Equal(t, int64(10), total.InputTokens)
	assert.Equal(t, int64(5), total.OutputTokens)
	assert.True(t, decimal.RequireFromString("0.5").Equal(*total.CostUSD))
	assert.Nil(t, (*agentruntimeports.RunUsage)(nil).Clone())
}
