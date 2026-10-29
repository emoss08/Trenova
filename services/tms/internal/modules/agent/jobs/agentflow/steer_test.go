package agentflow

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func steeredMessages(outcome *Outcome) []conversation.Message {
	out := make([]conversation.Message, 0, 1)
	for _, message := range outcome.Result.Messages {
		if message.Kind == conversation.MessageKindSteer {
			out = append(out, message)
		}
	}

	return out
}

func TestRunReadsASteerAtItsNextStep(t *testing.T) {
	t.Parallel()

	lookup := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Result:   map[string]any{"status": "InTransit"},
	}
	h := newHarness(t, harnessParams{query: []agenttoolports.AgentQueryTool{lookup}})

	var asked []*llmports.ChatCompletionRequest
	h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *ModelCallInput) (*modelstream.Reply, error) {
			asked = append(asked, in.Request)
			if len(asked) == 1 {
				h.env.SignalWorkflow(agentflowcontract.SteerSignal, worldwatch.Steer{
					ID:      pulid.ID("aqm_01J9STEERSTEERSTEERSTEER00"),
					Content: "Actually the carrier is Werner, not JB Hunt.",
					Mentions: []agentpagecontext.EntityRef{
						{Type: "carrier", ID: "car_01J9WERNERWERNERWERNER000", Label: "Werner"},
					},
				})
				return &modelstream.Reply{
					Completion: toolReply("shipment_get", map[string]any{"shipmentId": "shp_1"}),
				}, nil
			}

			return &modelstream.Reply{Completion: textReply("Switched to Werner.")}, nil
		}).Times(2)

	result := h.run(t, runContext("shipment_get"))

	require.Empty(t, result.Err)
	require.Len(t, asked, 2)
	messages := asked[1].Messages
	words, notice := messages[len(messages)-2], messages[len(messages)-1]
	assert.Equal(t, llmports.RoleUser, words.Role,
		"the steer is read after the step's results and before the next model call")
	assert.Contains(t, words.Content, "Actually the carrier is Werner, not JB Hunt.")
	assert.Contains(t, words.Content, "car_01J9WERNERWERNERWERNER000")
	assert.Equal(t, llmports.RoleSystem, notice.Role,
		"what to make of the person's words comes from the system, not from them")

	assert.Equal(t, []pulid.ID{"aqm_01J9STEERSTEERSTEERSTEER00"}, result.Outcome.Steered,
		"the queued message the turn read is handed back, so the save clears it")
	saved := steeredMessages(result.Outcome)
	require.Len(t, saved, 1)
	assert.Equal(t, "Actually the carrier is Werner, not JB Hunt.", saved[0].Content,
		"the conversation keeps the person's words, not the prompt around them")
	assert.Contains(t, eventNames(result.Outcome), assistantstreamports.AssistantEventSteered)
}

func TestRunReadsTheSameSteerOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessParams{})
	steer := worldwatch.Steer{
		ID:      pulid.ID("aqm_01J9ONCEONCEONCEONCEONCE0"),
		Content: "Use the later slot.",
	}
	h.env.RegisterDelayedCallback(func() {
		h.env.SignalWorkflow(agentflowcontract.SteerSignal, steer)
		h.env.SignalWorkflow(agentflowcontract.SteerSignal, steer)
	}, 0)
	h.replies(textReply("Booked the later slot."))

	result := h.run(t, runContext())

	require.Empty(t, result.Err)
	assert.Equal(t, []pulid.ID{steer.ID}, result.Outcome.Steered)
	assert.Len(t, steeredMessages(result.Outcome), 1,
		"a steer delivered twice, as a retried signal can be, is read once")
}

func TestRunWithNothingToReadNeverAsksForTheChange(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessParams{})
	h.replies(textReply("Load 12345 is in Memphis."))

	result := h.run(t, runContext())

	require.Empty(t, result.Err)
	assert.Empty(t, result.Outcome.Steered)
	assert.Empty(t, steeredMessages(result.Outcome))
}
