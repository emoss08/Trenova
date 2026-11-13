package agentruntime

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requestText(req *llmports.ChatCompletionRequest) string {
	var b strings.Builder
	b.WriteString(req.System)
	for idx := range req.Messages {
		b.WriteByte('\n')
		b.WriteString(req.Messages[idx].Content)
	}

	return b.String()
}

// A replay of a case runs the agent's own loop. What the model is told and
// what it reads must be the situation the person decided in: the date they
// asked on, and the tool results the original was given, with the records as
// they are now left unread.
func TestRun_AReplayReadsWhatItsOriginalWasGivenAsOfWhenItWasAsked(t *testing.T) {
	t.Parallel()

	asked := time.Date(2026, time.March, 3, 9, 30, 0, 0, time.UTC).Unix()
	eta := asked + 2*24*60*60
	live := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Resource: permission.ResourceShipment,
		Result:   map[string]any{"proNumber": "S-100", "status": "Delivered"},
	}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("shipment_get", map[string]any{"id": "shp_1"}),
		runtimetest.TextTurn("S-100 is in transit and due in two days."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{live}}, nil, nil)
	recording := (&agentquality.EvalCase{ToolFixtures: []agentquality.ToolFixture{{
		Tool: "shipment_get",
		Args: map[string]any{"id": "shp_1"},
		Result: map[string]any{
			"proNumber": "S-100",
			"status":    "InTransit",
			"eta":       timeutils.DescribeUnixInstantIn(eta, asked, "UTC"),
		},
	}}}).Recording()
	require.NotNil(t, recording)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_get"),
		Actor:      runtimetest.Actor(),
		Input:      "Where is S-100?",
		Context:    agentturn.RuntimeContext{Now: asked, Timezone: "UTC"},
		Recorded:   recording,
	})
	require.NoError(t, err)

	assert.Zero(t, live.Calls, "the records as they are now are not read")
	assert.Equal(t, "S-100 is in transit and due in two days.", result.Reply)

	require.Len(t, completion.Requests, 2)
	first := requestText(completion.Requests[0])
	assert.Contains(t, first, "2026-03-03 Tuesday",
		"the model is told it is the day the person asked")
	second := requestText(completion.Requests[1])
	assert.Contains(t, second, "InTransit")
	assert.NotContains(t, second, "Delivered",
		"what happened to the shipment afterwards is not part of the case")
	assert.Contains(t, second, "(in 2 days)",
		"two days after the day the model was told it is")
}

// The same turn with nothing recorded reads the records, and is told the
// clock it was given.
func TestRun_ATurnWithNothingRecordedReadsTheRecords(t *testing.T) {
	t.Parallel()

	live := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Resource: permission.ResourceShipment,
		Result:   map[string]any{"proNumber": "S-100", "status": "Delivered"},
	}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("shipment_get", map[string]any{"id": "shp_1"}),
		runtimetest.TextTurn("S-100 was delivered."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{live}}, nil, nil)

	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_get"),
		Actor:      runtimetest.Actor(),
		Input:      "Where is S-100?",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, live.Calls)
	require.Len(t, completion.Requests, 2)
	assert.Contains(t, requestText(completion.Requests[1]), "Delivered")
}
