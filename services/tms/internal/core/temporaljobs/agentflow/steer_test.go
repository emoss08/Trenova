package agentflow

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
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

	lookup := &agentruntimetest.StubQueryTool{
		ToolName: "get_shipment",
		Result:   map[string]any{"status": "InTransit"},
	}
	h := newHarness(t, harnessParams{query: []serviceports.AgentQueryTool{lookup}})

	var asked []*serviceports.ChatCompletionRequest
	h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *ModelCallInput) (*agentruntime.ModelReply, error) {
			asked = append(asked, in.Request)
			if len(asked) == 1 {
				h.env.SignalWorkflow(SteerSignal, agentruntime.Steer{
					ID:      pulid.ID("aqm_01J9STEERSTEERSTEERSTEER00"),
					Content: "Actually the carrier is Werner, not JB Hunt.",
					Mentions: []agent.EntityRef{
						{Type: "carrier", ID: "car_01J9WERNERWERNERWERNER000", Label: "Werner"},
					},
				})
				return &agentruntime.ModelReply{
					Completion: toolReply("get_shipment", map[string]any{"shipmentId": "shp_1"}),
				}, nil
			}

			return &agentruntime.ModelReply{Completion: textReply("Switched to Werner.")}, nil
		}).Times(2)

	result := h.run(t, runContext("get_shipment"))

	require.Empty(t, result.Err)
	require.Len(t, asked, 2)
	last := asked[1].Messages[len(asked[1].Messages)-1]
	assert.Equal(t, serviceports.RoleUser, last.Role,
		"the steer is read after the step's results and before the next model call")
	assert.Contains(t, last.Content, "Actually the carrier is Werner, not JB Hunt.")
	assert.Contains(t, last.Content, "car_01J9WERNERWERNERWERNER000")

	assert.Equal(t, []pulid.ID{"aqm_01J9STEERSTEERSTEERSTEER00"}, result.Outcome.Steered,
		"the queued message the turn read is handed back, so the save clears it")
	saved := steeredMessages(result.Outcome)
	require.Len(t, saved, 1)
	assert.Equal(t, "Actually the carrier is Werner, not JB Hunt.", saved[0].Content,
		"the conversation keeps the person's words, not the prompt around them")
	assert.Contains(t, eventNames(result.Outcome), serviceports.AssistantEventSteered)
}

func TestRunReadsTheSameSteerOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessParams{})
	steer := agentruntime.Steer{ID: pulid.ID("aqm_01J9ONCEONCEONCEONCEONCE0"), Content: "Use the later slot."}
	h.env.RegisterDelayedCallback(func() {
		h.env.SignalWorkflow(SteerSignal, steer)
		h.env.SignalWorkflow(SteerSignal, steer)
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
