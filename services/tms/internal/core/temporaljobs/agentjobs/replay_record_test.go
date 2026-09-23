//go:build replayrecord

// Records the AgentRunWorkflow histories that replay_test.go replays.
//
// It runs the real workflow on a real Temporal server, with stand-in
// activities registered under the real activity names. History records only an
// activity's name, input and result, so the histories are exactly what
// production writes for each path through the workflow. Re-record only when the
// histories are deliberately being replaced, never to make a failing replay
// pass: a replay failure means executions like these would break.
//
//	temporal server start-dev &
//	go test -tags replayrecord -run TestRecordAgentRunHistories ./internal/core/temporaljobs/agentjobs/
package agentjobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/temporalproto"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type recordedRun struct {
	name            string
	pending         int
	decisionTimeout int
	// drive moves the execution to the state being recorded and says whether
	// it should be left open, as a run waiting on a person is in production.
	drive func(t *testing.T, c client.Client, run client.WorkflowRun) (leaveOpen bool)
}

func TestRecordAgentRunHistories(t *testing.T) {
	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		address = "127.0.0.1:7233"
	}

	c, err := client.Dial(client.Options{HostPort: address})
	require.NoError(t, err)
	defer c.Close()

	runs := []recordedRun{
		{
			name: "completed-nothing-pending",
			drive: func(t *testing.T, _ client.Client, run client.WorkflowRun) bool {
				require.NoError(t, run.Get(t.Context(), nil))
				return false
			},
		},
		{
			// The state every in-flight run with proposals is in: the agent
			// has finished and the workflow is waiting on a person.
			name:            "waiting-on-a-decision",
			pending:         2,
			decisionTimeout: 86400,
			drive: func(t *testing.T, c client.Client, run client.WorkflowRun) bool {
				waitForTimer(t, c, run)
				return true
			},
		},
		{
			name:            "decided",
			pending:         1,
			decisionTimeout: 86400,
			drive: func(t *testing.T, c client.Client, run client.WorkflowRun) bool {
				waitForTimer(t, c, run)
				require.NoError(t, c.SignalWorkflow(t.Context(), run.GetID(), run.GetRunID(),
					AgentDecisionSignalName, DecisionSignal{
						ProposalID:      pulid.MustNew("ap_"),
						Decision:        agent.DecisionAccepted,
						DecidedByUserID: pulid.MustNew("usr_"),
					}))
				require.NoError(t, run.Get(t.Context(), nil))
				return false
			},
		},
		{
			name:            "decision-timed-out",
			pending:         1,
			decisionTimeout: 2,
			drive: func(t *testing.T, _ client.Client, run client.WorkflowRun) bool {
				require.NoError(t, run.Get(t.Context(), nil))
				return false
			},
		},
	}

	out := filepath.Join("testdata", "replay")
	require.NoError(t, os.MkdirAll(out, 0o755))

	for _, rec := range runs {
		t.Run(rec.name, func(t *testing.T) {
			queue := "replay-record-" + rec.name
			w := worker.New(c, queue, worker.Options{})
			w.RegisterWorkflowWithOptions(
				NewWorkflows(nil).AgentRunWorkflow,
				workflow.RegisterOptions{Name: AgentRunWorkflowName},
			)
			registerStandIns(w, rec)
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

func registerStandIns(w worker.Worker, rec recordedRun) {
	w.RegisterActivityWithOptions(
		func(_ context.Context, p *AgentRunPayload) (*PrepareRunResult, error) {
			return &PrepareRunResult{
				Definition: &agentdefinition.Definition{
					ID:   p.DefinitionID,
					Name: "Recorded desk",
				},
				DecisionTimeoutSeconds: rec.decisionTimeout,
				RunTimeoutSeconds:      600,
			}, nil
		},
		activity.RegisterOptions{Name: "PrepareRunActivity"},
	)

	w.RegisterActivityWithOptions(func(context.Context, *RunAgentInput) (*RunAgentResult, error) {
		return &RunAgentResult{
			Reply:            "Recorded reply.",
			Model:            "recorded-model",
			ToolCallsUsed:    3,
			ProposalsRaised:  rec.pending,
			PendingProposals: rec.pending,
		}, nil
	}, activity.RegisterOptions{Name: "RunAgentActivity"})

	w.RegisterActivityWithOptions(func(context.Context, *CompleteRunInput) error {
		return nil
	}, activity.RegisterOptions{Name: "CompleteRunActivity"})

	w.RegisterActivityWithOptions(func(context.Context, *ExpireProposalsInput) error {
		return nil
	}, activity.RegisterOptions{Name: "ExpireProposalsActivity"})
}

// waitForTimer returns once the workflow has started its decision timer, which
// is the point at which it is parked waiting on a person.
func waitForTimer(t *testing.T, c client.Client, run client.WorkflowRun) {
	t.Helper()

	require.Eventually(t, func() bool {
		iter := c.GetWorkflowHistory(t.Context(), run.GetID(), run.GetRunID(), false,
			enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		for iter.HasNext() {
			event, err := iter.Next()
			if err != nil {
				return false
			}
			if event.GetEventType() == enumspb.EVENT_TYPE_TIMER_STARTED {
				return true
			}
		}
		return false
	}, 30*time.Second, 100*time.Millisecond)
}

func writeHistory(t *testing.T, c client.Client, run client.WorkflowRun, path string) {
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
