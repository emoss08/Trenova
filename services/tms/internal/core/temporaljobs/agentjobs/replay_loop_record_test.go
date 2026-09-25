//go:build replayrecord

// Records the AgentRunWorkflow histories of runs driven in workflow code that
// replay_loop_test.go replays, with stand-in activities registered under the
// real activity names. Re-record only when the histories are deliberately
// being replaced, never to make a failing replay pass.
//
//	temporal server start-dev &
//	go test -tags replayrecord -run TestRecordAgentLoopHistories ./internal/core/temporaljobs/agentjobs/
package agentjobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type recordedLoop struct {
	name    string
	replies []*agentruntime.ModelReply
	pending int
	drive   func(t *testing.T, c client.Client, run client.WorkflowRun) (leaveOpen bool)
}

func TestRecordAgentLoopHistories(t *testing.T) {
	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		address = "127.0.0.1:7233"
	}

	c, err := client.Dial(client.Options{HostPort: address})
	require.NoError(t, err)
	defer c.Close()

	read := &agentruntime.ModelReply{Completion: &serviceports.ChatCompletionResult{
		ToolCalls: []serviceports.ToolCall{{
			ID:        "call_recorded_read",
			Name:      loopReplayRead,
			Arguments: map[string]any{"proNumber": "12345"},
		}},
		ModelIdentifier: "recorded-model",
		InputTokens:     900,
		OutputTokens:    30,
	}}
	write := &agentruntime.ModelReply{Completion: &serviceports.ChatCompletionResult{
		ToolCalls: []serviceports.ToolCall{{
			ID:        "call_recorded_write",
			Name:      loopReplayWrite,
			Arguments: map[string]any{"moveId": "smv_recorded", "workerId": "wrk_recorded"},
		}},
		ModelIdentifier: "recorded-model",
		InputTokens:     1000,
		OutputTokens:    35,
	}}
	answer := &agentruntime.ModelReply{Completion: &serviceports.ChatCompletionResult{
		Text:            "Proposed assigning the move.",
		ModelIdentifier: "recorded-model",
		InputTokens:     1100,
		OutputTokens:    10,
	}}

	loops := []recordedLoop{
		{
			name:    "loop-completed-nothing-pending",
			replies: []*agentruntime.ModelReply{read, answer},
			drive: func(t *testing.T, _ client.Client, run client.WorkflowRun) bool {
				require.NoError(t, run.Get(t.Context(), nil))
				return false
			},
		},
		{
			name:    "loop-waiting-on-a-decision",
			replies: []*agentruntime.ModelReply{read, write, answer},
			pending: 1,
			drive: func(t *testing.T, c client.Client, run client.WorkflowRun) bool {
				waitForTimer(t, c, run)
				return true
			},
		},
	}

	out := filepath.Join("testdata", "replay-loop")
	require.NoError(t, os.MkdirAll(out, 0o755))

	for _, rec := range loops {
		t.Run(rec.name, func(t *testing.T) {
			runtime := loopReplayRuntime()
			queue := "replay-record-" + rec.name
			w := worker.New(c, queue, worker.Options{})
			w.RegisterWorkflowWithOptions(
				NewWorkflows(runtime).AgentRunWorkflow,
				workflow.RegisterOptions{Name: AgentRunWorkflowName},
			)
			registerLoopStandIns(t, w, runtime, rec)
			require.NoError(t, w.Start())
			defer w.Stop()

			payload := &AgentRunPayload{
				BasePayload: temporaltype.BasePayload{
					OrganizationID: pulid.MustNew("org_"),
					BusinessUnitID: pulid.MustNew("bu_"),
				},
				RunID:        pulid.MustNew("ar_"),
				DefinitionID: pulid.MustNew("agdef_"),
				Trigger:      agent.RunTriggerEvent,
				SubjectType:  agent.SubjectShipment,
				SubjectID:    pulid.MustNew("shp_"),
			}

			run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
				ID:        fmt.Sprintf("replay-record/%s/%d", rec.name, time.Now().UnixNano()),
				TaskQueue: queue,
			}, AgentRunWorkflowName, payload)
			require.NoError(t, err)

			leaveOpen := rec.drive(t, c, run)
			writeHistory(t, c, run, filepath.Join(out, rec.name+".json"))

			if leaveOpen {
				_ = c.TerminateWorkflow(
					context.WithoutCancel(t.Context()),
					run.GetID(),
					run.GetRunID(),
					"recorded",
				)
			}
		})
	}
}

func registerLoopStandIns(
	t *testing.T,
	w worker.Worker,
	runtime *agentruntime.Service,
	rec recordedLoop,
) {
	t.Helper()

	w.RegisterActivityWithOptions(
		func(_ context.Context, p *AgentRunPayload) (*PrepareRunResult, error) {
			return &PrepareRunResult{
				Definition:             loopReplayDefinition(p.DefinitionID),
				DecisionTimeoutSeconds: 86400,
				RunTimeoutSeconds:      600,
			}, nil
		},
		activity.RegisterOptions{Name: "PrepareRunActivity"},
	)

	w.RegisterActivityWithOptions(
		func(ctx context.Context, in *OpenRunInput) (*OpenRunResult, error) {
			tenant := in.Payload.tenantInfo()
			req := &serviceports.RunRequest{
				Definition: in.Definition,
				Actor:      agentActor(tenant),
				Input:      "A shipment was created.",
				RunID:      in.Payload.RunID,
				Unattended: true,
				StepOwner: serviceports.RunStepOwner{
					Kind: serviceports.RunStepOwnerAgentRun,
					ID:   in.Payload.RunID,
				},
				Attempt: 1,
			}

			return &OpenRunResult{
				Run:  agentflow.NewRunContext(req, agentflow.PriorityBackground),
				Turn: runtime.OpenTurn(ctx, req).State(),
			}, nil
		},
		activity.RegisterOptions{Name: "OpenRunActivity"},
	)

	calls := 0
	w.RegisterActivityWithOptions(
		func(context.Context, *agentflow.ModelCallInput) (*agentruntime.ModelReply, error) {
			reply := rec.replies[min(calls, len(rec.replies)-1)]
			calls++

			return reply, nil
		},
		activity.RegisterOptions{Name: "ModelCallActivity"},
	)

	w.RegisterActivityWithOptions(
		func(_ context.Context, in *agentflow.ToolInput) (*agentflow.ToolResult, error) {
			return &agentflow.ToolResult{Outcome: agentruntime.ToolOutcome{
				Content: "<tool_result name=\"" + in.Call.Call.Name +
					"\">{\"status\":\"InTransit\"}</tool_result>",
			}}, nil
		},
		activity.RegisterOptions{Name: loopReplayRead},
	)

	w.RegisterActivityWithOptions(
		func(_ context.Context, in *agentflow.ToolInput) (*agentflow.ToolResult, error) {
			return &agentflow.ToolResult{Outcome: agentruntime.ToolOutcome{
				Content: "Recorded a proposal to run \"assign_move\". It is awaiting a " +
					"person's review at the Propose tier and has not run.",
				Action: &serviceports.PendingAction{
					ToolName:   in.Call.Call.Name,
					Arguments:  in.Call.Call.Arguments,
					Rationale:  "Cover the move.",
					Tier:       agent.TierPropose,
					ToolCallID: in.Call.Call.ID,
					Egress:     agent.EgressInternal,
					HeldBy:     []string{"tool_tier"},
				},
			}}, nil
		},
		activity.RegisterOptions{Name: loopReplayWrite},
	)

	w.RegisterActivityWithOptions(func(context.Context, *FinishRunInput) (*FinishRunResult, error) {
		return &FinishRunResult{ProposalsRaised: rec.pending, PendingProposals: rec.pending}, nil
	}, activity.RegisterOptions{Name: "FinishRunActivity"})

	w.RegisterActivityWithOptions(func(context.Context, *CompleteRunInput) error {
		return nil
	}, activity.RegisterOptions{Name: "CompleteRunActivity"})

	w.RegisterActivityWithOptions(func(context.Context, *ExpireProposalsInput) error {
		return nil
	}, activity.RegisterOptions{Name: "ExpireProposalsActivity"})

	w.RegisterActivityWithOptions(func(context.Context, *PendingProposalsInput) (int, error) {
		return rec.pending, nil
	}, activity.RegisterOptions{Name: "PendingProposalsActivity"})
}
