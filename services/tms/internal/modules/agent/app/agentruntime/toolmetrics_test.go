package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func toolOutcomeCounts(t *testing.T, registry *prometheus.Registry) map[[2]string]float64 {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)

	counts := make(map[[2]string]float64)
	for _, family := range families {
		if family.GetName() != "trenova_assistant_tool_outcomes_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			var key [2]string
			for _, label := range metric.GetLabel() {
				switch label.GetName() {
				case "tool":
					key[0] = label.GetValue()
				case "verdict":
					key[1] = label.GetValue()
				}
			}
			counts[key] = metric.GetCounter().GetValue()
		}
	}

	return counts
}

func countedRuntime(
	completion *scriptedCompletion,
	query *stubQueryRegistry,
) (*Service, *prometheus.Registry) {
	registry := prometheus.NewRegistry()
	rt := newRuntime(completion, query, &stubActionRegistry{}, nil)
	rt.metrics = aimetrics.NewAssistant(registry, zap.NewNop(), true)

	return rt, registry
}

// Every call the loop answers is counted once, by the tool it named and what
// became of it, refusals included; a name the model invented is counted under
// one label rather than its own.
func TestRun_CountsEveryToolCallByToolAndVerdict(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		callsTurn(
			llmports.ToolCall{ID: "invented", Name: "delete_everything"},
			llmports.ToolCall{
				ID:        "ran",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			},
			llmports.ToolCall{
				ID:             "cut_off",
				Name:           "shipment_get",
				Arguments:      map[string]any{},
				ArgumentsError: "unexpected end of JSON input",
			},
		),
		textTurn("Done."),
	}}
	rt, registry := countedRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{queryTool("shipment_get", map[string]any{}, nil)},
	})

	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition("shipment_get"),
		Actor:      testActor(),
		Input:      "Look it up.",
	})
	require.NoError(t, err)

	counts := toolOutcomeCounts(t, registry)
	assert.InDelta(t, 1, counts[[2]string{"shipment_get", aitrace.OutcomeRan}], 0)
	assert.InDelta(t, 1, counts[[2]string{"shipment_get", aitrace.OutcomeInvalid}], 0)
	assert.InDelta(t, 1, counts[[2]string{unregisteredToolLabel, aitrace.OutcomeInvalid}], 0)
	_, named := counts[[2]string{"delete_everything", aitrace.OutcomeInvalid}]
	assert.False(t, named, "an invented name never becomes a label")
}

type replayingEffects struct {
	*localEffects
}

func (*replayingEffects) Replaying() bool { return true }

// Workflow code re-runs the loop over its history when a worker rebuilds an
// execution; a call counted on the first pass is not counted again.
func TestDrive_DoesNotCountCallsWhileReplaying(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn("shipment_get", map[string]any{"id": "shp_1"}),
		textTurn("Done."),
	}}
	rt, registry := countedRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{queryTool("shipment_get", map[string]any{}, nil)},
	})
	req := &agentruntimeports.RunRequest{
		Definition: testDefinition("shipment_get"),
		Actor:      testActor(),
		Input:      "Look it up.",
	}
	fx := &replayingEffects{localEffects: &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(kernelports.StreamEvent) {},
	}}

	_, err := rt.Drive(rt.OpenTurn(t.Context(), req), fx)
	require.NoError(t, err)

	assert.Empty(t, toolOutcomeCounts(t, registry))
}

func TestRestoreTurn_KeepsTheProviderKindItsOutcomesAreLabelledWith(t *testing.T) {
	t.Parallel()

	rt, _ := countedRuntime(&scriptedCompletion{}, &stubQueryRegistry{})
	req := &agentruntimeports.RunRequest{
		Definition: testDefinition("shipment_get"),
		Actor:      testActor(),
		Input:      "Look it up.",
	}
	turn := rt.OpenTurn(t.Context(), req)
	turn.providerKind = aiprovider.KindAnthropicMessages

	restored := rt.RestoreTurn(req, turn.State())
	assert.Equal(t, aiprovider.KindAnthropicMessages, restored.providerKind)
}
