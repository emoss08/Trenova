//go:build replayrecord

// Records the AssistantTurnWorkflow histories that replay_test.go replays.
//
// It runs the real workflow on a real Temporal server, with stand-in
// activities registered under the real activity names. History records only an
// activity's name, input and result, so the histories are exactly what
// production writes for each path through the workflow. Re-record only when the
// histories are deliberately being replaced, never to make a failing replay
// pass: a replay failure means turns like these would break.
//
//	temporal server start-dev &
//	go test -tags replayrecord -run TestRecordAssistantTurnHistories ./internal/core/temporaljobs/assistantjobs/
package assistantjobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/temporalproto"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type recordedTurn struct {
	name string
	// drained says a reader took the last event, so the turn closes at once
	// and tells nobody afterwards.
	drained bool
}

func TestRecordAssistantTurnHistories(t *testing.T) {
	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		address = "127.0.0.1:7233"
	}

	c, err := client.Dial(client.Options{HostPort: address})
	require.NoError(t, err)
	defer c.Close()

	turns := []recordedTurn{
		{name: "answered-with-a-tool-read", drained: true},
		{name: "answered-nobody-watching"},
	}

	out := filepath.Join("testdata", "replay")
	require.NoError(t, os.MkdirAll(out, 0o755))

	for _, rec := range turns {
		t.Run(rec.name, func(t *testing.T) {
			runtime := replayRuntime()
			queue := "replay-record-turn-" + rec.name
			w := worker.New(c, queue, worker.Options{})
			w.RegisterWorkflowWithOptions(
				NewWorkflows(runtime).AssistantTurnWorkflow,
				workflow.RegisterOptions{Name: AssistantTurnWorkflowName},
			)
			payload := replayPayload()
			registerTurnStandIns(t, w, runtime, payload)
			require.NoError(t, w.Start())
			defer w.Stop()

			run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
				ID:        fmt.Sprintf("replay-record/turn/%s/%d", rec.name, time.Now().UnixNano()),
				TaskQueue: queue,
			}, AssistantTurnWorkflowName, payload)
			require.NoError(t, err)

			if rec.drained {
				require.NoError(t, c.SignalWorkflow(t.Context(), run.GetID(), run.GetRunID(),
					temporaltype.SignalStreamDrained, nil))
			}
			var result AssistantTurnResult
			require.NoError(t, run.Get(t.Context(), &result))
			require.Equal(t, string(conversation.AssistantTurnStatusCompleted), result.Status)

			writeTurnHistory(t, c, run, filepath.Join(out, rec.name+".json"))
		})
	}
}

func registerTurnStandIns(
	t *testing.T,
	w worker.Worker,
	runtime *agentruntime.Service,
	payload *AssistantTurnPayload,
) {
	t.Helper()

	plan := replayPlan(t, runtime, payload)
	w.RegisterActivityWithOptions(
		func(context.Context, *AssistantTurnPayload) (*assistantservice.TurnPlan, error) {
			return plan, nil
		},
		activity.RegisterOptions{Name: "PrepareTurnActivity"},
	)

	replies := []*agentruntime.ModelReply{
		{Completion: &serviceports.ChatCompletionResult{
			ToolCalls: []serviceports.ToolCall{{
				ID:        "call_recorded_1",
				Name:      replayToolName,
				Arguments: map[string]any{"proNumber": "12345"},
			}},
			ModelIdentifier: "recorded-model",
			InputTokens:     1200,
			OutputTokens:    40,
		}},
		{Completion: &serviceports.ChatCompletionResult{
			Text:            "Load 12345 is in Memphis.",
			ModelIdentifier: "recorded-model",
			InputTokens:     1300,
			OutputTokens:    12,
		}},
	}
	calls := 0
	w.RegisterActivityWithOptions(
		func(context.Context, *agentflow.ModelCallInput) (*agentruntime.ModelReply, error) {
			reply := replies[min(calls, len(replies)-1)]
			calls++

			return reply, nil
		},
		activity.RegisterOptions{Name: "ModelCallActivity"},
	)

	w.RegisterActivityWithOptions(
		func(_ context.Context, in *agentflow.ToolInput) (*agentflow.ToolResult, error) {
			return &agentflow.ToolResult{Outcome: agentruntime.ToolOutcome{
				Content: "<tool_result name=\"" + in.Call.Call.Name +
					"\">{\"status\":\"InTransit\",\"city\":\"Memphis\"}</tool_result>",
				Summary: "Load 12345",
			}}, nil
		},
		activity.RegisterOptions{Name: replayToolName},
	)

	w.RegisterActivityWithOptions(
		func(_ context.Context, in *FinishTurnInput) (*TurnEnding, error) {
			return &TurnEnding{
				Result: AssistantTurnResult{
					Status: string(conversation.AssistantTurnStatusCompleted),
				},
				Event: temporaltype.StreamItem{
					Event: serviceports.AssistantEventDone,
					Data:  map[string]any{"turnId": in.Payload.TurnID.String()},
				},
			}, nil
		},
		activity.RegisterOptions{Name: "FinishTurnActivity"},
	)

	w.RegisterActivityWithOptions(func(context.Context, *NotifyUnseenTurnInput) error {
		return nil
	}, activity.RegisterOptions{Name: "NotifyUnseenTurnActivity"})

	w.RegisterActivityWithOptions(func(context.Context, *AssistantTurnPayload, string) error {
		return nil
	}, activity.RegisterOptions{Name: "CloseTurnActivity"})
}

func writeTurnHistory(t *testing.T, c client.Client, run client.WorkflowRun, path string) {
	t.Helper()

	history := &historypb.History{}
	iter := c.GetWorkflowHistory(t.Context(), run.GetID(), run.GetRunID(), false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		event, err := iter.Next()
		require.NoError(t, err)
		history.Events = append(history.Events, event)
	}

	encoded, err := temporalproto.CustomJSONMarshalOptions{Indent: "  "}.Marshal(history)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(encoded, '\n'), 0o644))
}
