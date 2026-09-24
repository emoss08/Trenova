//go:build replayrecord

package agentqualityjobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/services/agentqualityservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
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

type recordedSuite struct {
	name        string
	stopAtCheck int
	judge       bool
}

func TestRecordQualityHistories(t *testing.T) {
	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		address = "127.0.0.1:7233"
	}

	c, err := client.Dial(client.Options{HostPort: address})
	require.NoError(t, err)
	defer c.Close()

	require.NoError(t, os.MkdirAll(replayDir, 0o755))

	for _, rec := range []recordedSuite{
		{name: "suite-completed-with-judge", judge: true},
		{name: "suite-budget-stopped", stopAtCheck: 2},
	} {
		t.Run(rec.name, func(t *testing.T) {
			queue := "replay-record-quality-" + rec.name
			w := worker.New(c, queue, worker.Options{})
			w.RegisterWorkflowWithOptions(AgentSuiteRunWorkflow,
				workflow.RegisterOptions{Name: AgentSuiteRunWorkflowName})
			w.RegisterWorkflowWithOptions(
				func(workflow.Context, *agentjobs.AgentEvaluationPayload) error { return nil },
				workflow.RegisterOptions{Name: agentjobs.AgentEvaluationWorkflowName},
			)
			registerQualityStandIns(w, rec)
			require.NoError(t, w.Start())
			defer w.Stop()

			payload := &SuiteRunPayload{
				OrganizationID:    pulid.MustNew("org_"),
				BusinessUnitID:    pulid.MustNew("bu_"),
				SuiteRunID:        agentquality.NewSuiteRunID(),
				AgentDefinitionID: pulid.MustNew("agd_"),
				SampleSeed:        17,
				Settings: agentquality.SuiteSettings{
					MaxCasesPerAgent: 3, JudgeEnabled: rec.judge, JudgeSampleRate: 0.5,
				},
				DayStart:   1_800_000_000,
				MonthStart: 1_799_000_000,
			}

			run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
				ID:        fmt.Sprintf("replay-record/%s/%d", rec.name, time.Now().UnixNano()),
				TaskQueue: queue,
			}, AgentSuiteRunWorkflowName, payload)
			require.NoError(t, err)
			require.NoError(t, run.Get(t.Context(), nil))

			writeQualityHistory(t, c, run, filepath.Join(replayDir, rec.name+".json"))
		})
	}
}

func registerQualityStandIns(w worker.Worker, rec recordedSuite) {
	cases := []agentqualityservice.SuiteCase{
		{EvaluationID: pulid.MustNew(agent.EvaluationIDPrefix), Ordinal: 1},
		{EvaluationID: pulid.MustNew(agent.EvaluationIDPrefix), Ordinal: 2},
		{EvaluationID: pulid.MustNew(agent.EvaluationIDPrefix), Ordinal: 3},
	}
	listed := false
	w.RegisterActivityWithOptions(func(
		context.Context,
		*agentqualityservice.ListSuiteCasesRequest,
	) ([]agentqualityservice.SuiteCase, error) {
		if listed {
			return []agentqualityservice.SuiteCase{}, nil
		}
		listed = true

		return cases, nil
	}, activity.RegisterOptions{Name: "ListSuiteCasesActivity"})

	checks := 0
	w.RegisterActivityWithOptions(func(
		context.Context,
		*agentqualityservice.CheckBudgetRequest,
	) (*agentquality.BudgetDecision, error) {
		checks++
		if rec.stopAtCheck > 0 && checks >= rec.stopAtCheck {
			return &agentquality.BudgetDecision{Stop: true, Reason: "Recorded budget stop."}, nil
		}

		return &agentquality.BudgetDecision{}, nil
	}, activity.RegisterOptions{Name: "CheckEvalBudgetActivity"})

	w.RegisterActivityWithOptions(func(
		context.Context,
		*agentqualityservice.ScoreSuiteRequest,
	) (*agentqualityservice.ScoredSuite, error) {
		return &agentqualityservice.ScoredSuite{
			Scored:     len(cases),
			JudgeCases: []pulid.ID{cases[0].EvaluationID},
		}, nil
	}, activity.RegisterOptions{Name: "ScoreSuiteActivity"})

	w.RegisterActivityWithOptions(func(
		context.Context,
		*agentqualityservice.JudgeCaseRequest,
	) (*agentqualityservice.JudgedCase, error) {
		return &agentqualityservice.JudgedCase{Judged: true}, nil
	}, activity.RegisterOptions{Name: "JudgeCaseActivity"})

	w.RegisterActivityWithOptions(func(
		_ context.Context,
		req *agentqualityservice.FinalizeSuiteRequest,
	) (*agentqualityservice.FinalizedSuite, error) {
		status := agentquality.SuiteRunStatusCompleted
		if req.StopReason != "" {
			status = agentquality.SuiteRunStatusBudgetStopped
		}

		return &agentqualityservice.FinalizedSuite{Status: status}, nil
	}, activity.RegisterOptions{Name: "FinalizeSuiteActivity"})

	w.RegisterActivityWithOptions(func(
		context.Context,
		*agentqualityservice.FailSuiteRequest,
	) error {
		return nil
	}, activity.RegisterOptions{Name: "FailSuiteActivity"})
}

func writeQualityHistory(t *testing.T, c client.Client, run client.WorkflowRun, path string) {
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
