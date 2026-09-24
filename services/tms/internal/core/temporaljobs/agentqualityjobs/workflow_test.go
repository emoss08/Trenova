package agentqualityjobs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/services/agentqualityservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type suiteHarness struct {
	env        *testsuite.TestWorkflowEnvironment
	mu         sync.Mutex
	activities []string
	replayed   []pulid.ID
	finalized  *agentqualityservice.FinalizeSuiteRequest
	failed     bool
}

func (h *suiteHarness) record(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activities = append(h.activities, info.ActivityType.Name)
}

func (h *suiteHarness) count(name string) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	total := 0
	for _, activityName := range h.activities {
		if activityName == name {
			total++
		}
	}

	return total
}

func settings(judge bool) agentquality.SuiteSettings {
	return agentquality.SuiteSettings{
		MaxCasesPerAgent:    50,
		NightlyBudgetUSD:    decimal.NewFromInt(5),
		MonthlyBudgetUSD:    decimal.NewFromInt(50),
		JudgeEnabled:        judge,
		JudgeSampleRate:     0.2,
		RegressionThreshold: 0.1,
		MinCases:            10,
		ForceRerunDays:      7,
	}
}

func suiteCases(count int, from int) []agentqualityservice.SuiteCase {
	cases := make([]agentqualityservice.SuiteCase, 0, count)
	for idx := range count {
		cases = append(cases, agentqualityservice.SuiteCase{
			EvaluationID: pulid.MustNew(agent.EvaluationIDPrefix),
			Ordinal:      from + idx + 1,
			Status:       agent.EvaluationStatusPending,
		})
	}

	return cases
}

type suiteScript struct {
	pages       [][]agentqualityservice.SuiteCase
	stopAtCheck int
	judgeCases  []pulid.ID
}

func newSuiteHarness(t *testing.T, script suiteScript) *suiteHarness {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	h := &suiteHarness{env: env}

	env.RegisterWorkflowWithOptions(AgentSuiteRunWorkflow,
		workflow.RegisterOptions{Name: AgentSuiteRunWorkflowName})
	env.RegisterWorkflowWithOptions(
		func(_ workflow.Context, payload *agentjobs.AgentEvaluationPayload) error {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.replayed = append(h.replayed, payload.EvaluationID)

			return nil
		},
		workflow.RegisterOptions{Name: agentjobs.AgentEvaluationWorkflowName},
	)
	env.RegisterActivity(&Activities{})
	env.SetOnActivityStartedListener(h.record)

	var a *Activities
	page := 0
	env.OnActivity(a.ListSuiteCasesActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			_ *agentqualityservice.ListSuiteCasesRequest,
		) ([]agentqualityservice.SuiteCase, error) {
			if page >= len(script.pages) {
				return []agentqualityservice.SuiteCase{}, nil
			}
			page++

			return script.pages[page-1], nil
		})

	checks := 0
	env.OnActivity(a.CheckEvalBudgetActivity, mock.Anything, mock.Anything).
		Return(func(
			context.Context,
			*agentqualityservice.CheckBudgetRequest,
		) (*agentquality.BudgetDecision, error) {
			checks++
			if script.stopAtCheck > 0 && checks >= script.stopAtCheck {
				return &agentquality.BudgetDecision{
					Stop:   true,
					Reason: "Tonight's budget is spent.",
				}, nil
			}

			return &agentquality.BudgetDecision{}, nil
		})
	env.OnActivity(a.ScoreSuiteActivity, mock.Anything, mock.Anything).
		Return(&agentqualityservice.ScoredSuite{Scored: 3, JudgeCases: script.judgeCases}, nil).
		Maybe()
	env.OnActivity(a.JudgeCaseActivity, mock.Anything, mock.Anything).
		Return(&agentqualityservice.JudgedCase{Judged: true}, nil).Maybe()
	env.OnActivity(a.FinalizeSuiteActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			req *agentqualityservice.FinalizeSuiteRequest,
		) (*agentqualityservice.FinalizedSuite, error) {
			h.finalized = req
			status := agentquality.SuiteRunStatusCompleted
			if req.StopReason != "" {
				status = agentquality.SuiteRunStatusBudgetStopped
			}

			return &agentqualityservice.FinalizedSuite{Status: status}, nil
		}).Maybe()
	env.OnActivity(a.FailSuiteActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *agentqualityservice.FailSuiteRequest) error {
			h.failed = true
			return nil
		}).Maybe()

	return h
}

func suitePayload(judge bool) *SuiteRunPayload {
	return &SuiteRunPayload{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		SuiteRunID:        agentquality.NewSuiteRunID(),
		AgentDefinitionID: pulid.MustNew("agd_"),
		SampleSeed:        11,
		Settings:          settings(judge),
		DayStart:          1_800_000_000,
		MonthStart:        1_799_000_000,
	}
}

func TestAgentSuiteRunWorkflow_ReplaysEachCaseThenScoresJudgesAndFinalizes(t *testing.T) {
	t.Parallel()

	cases := suiteCases(3, 0)
	h := newSuiteHarness(t, suiteScript{
		pages:      [][]agentqualityservice.SuiteCase{cases},
		judgeCases: []pulid.ID{cases[1].EvaluationID},
	})

	h.env.ExecuteWorkflow(AgentSuiteRunWorkflowName, suitePayload(true))
	require.True(t, h.env.IsWorkflowCompleted())
	require.NoError(t, h.env.GetWorkflowError())

	var result SuiteRunResult
	require.NoError(t, h.env.GetWorkflowResult(&result))
	assert.Equal(t, 3, result.Replayed)
	assert.Equal(t, 1, result.Judged)
	assert.Equal(t, agentquality.SuiteRunStatusCompleted, result.Status)

	assert.Equal(t, []pulid.ID{
		cases[0].EvaluationID, cases[1].EvaluationID, cases[2].EvaluationID,
	}, h.replayed, "cases are asked in order, each once")
	assert.Equal(t, 4, h.count("CheckEvalBudgetActivity"), "one check per case and per judgement")
	assert.Equal(t, 1, h.count("ScoreSuiteActivity"))
	assert.Equal(t, 1, h.count("JudgeCaseActivity"))
	require.NotNil(t, h.finalized)
	assert.Empty(t, h.finalized.StopReason)
	assert.False(t, h.failed)
}

func TestAgentSuiteRunWorkflow_SkipsCasesAlreadyReplayed(t *testing.T) {
	t.Parallel()

	cases := suiteCases(2, 0)
	cases[0].Status = agent.EvaluationStatusCompleted
	h := newSuiteHarness(t, suiteScript{pages: [][]agentqualityservice.SuiteCase{cases}})

	h.env.ExecuteWorkflow(AgentSuiteRunWorkflowName, suitePayload(false))
	require.NoError(t, h.env.GetWorkflowError())
	assert.Equal(t, []pulid.ID{cases[1].EvaluationID}, h.replayed)
	assert.Zero(t, h.count("JudgeCaseActivity"), "judging is off")
}

func TestAgentSuiteRunWorkflow_StopsWhenTheBudgetRunsOut(t *testing.T) {
	t.Parallel()

	cases := suiteCases(5, 0)
	h := newSuiteHarness(t, suiteScript{
		pages:       [][]agentqualityservice.SuiteCase{cases},
		stopAtCheck: 3,
		judgeCases:  []pulid.ID{cases[0].EvaluationID},
	})

	h.env.ExecuteWorkflow(AgentSuiteRunWorkflowName, suitePayload(true))
	require.True(t, h.env.IsWorkflowCompleted())
	require.NoError(t, h.env.GetWorkflowError())

	assert.Equal(t, []pulid.ID{cases[0].EvaluationID, cases[1].EvaluationID}, h.replayed,
		"no case is asked once the budget is spent")
	assert.Zero(t, h.count("JudgeCaseActivity"), "a stopped run spends nothing on a judge")
	require.NotNil(t, h.finalized)
	assert.Equal(t, "Tonight's budget is spent.", h.finalized.StopReason)

	var result SuiteRunResult
	require.NoError(t, h.env.GetWorkflowResult(&result))
	assert.Equal(t, agentquality.SuiteRunStatusBudgetStopped, result.Status)
}

func TestAgentSuiteRunWorkflow_ContinuesAsNewEveryHundredCases(t *testing.T) {
	t.Parallel()

	first := suiteCases(casesPerExecution, 0)
	h := newSuiteHarness(t, suiteScript{
		pages: [][]agentqualityservice.SuiteCase{first, suiteCases(20, casesPerExecution)},
	})

	payload := suitePayload(false)
	h.env.ExecuteWorkflow(AgentSuiteRunWorkflowName, payload)
	require.True(t, h.env.IsWorkflowCompleted())

	err := h.env.GetWorkflowError()
	require.Error(t, err)
	var continued *workflow.ContinueAsNewError
	require.True(t, errors.As(err, &continued), "the run continues as new, got %v", err)

	var next SuiteRunPayload
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(continued.Input, &next))
	assert.Equal(t, casesPerExecution, next.Cursor, "the cursor is the last case walked past")
	assert.Equal(t, casesPerExecution, next.Replayed)
	assert.Equal(t, payload.SuiteRunID, next.SuiteRunID)
	assert.Len(t, h.replayed, casesPerExecution)
	assert.Zero(t, h.count("FinalizeSuiteActivity"), "only the last execution finalizes")
}

func TestAgentSuiteRunWorkflow_RecordsWhyItFailed(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(AgentSuiteRunWorkflow,
		workflow.RegisterOptions{Name: AgentSuiteRunWorkflowName})
	env.RegisterActivity(&Activities{})

	var a *Activities
	failed := false
	env.OnActivity(a.ListSuiteCasesActivity, mock.Anything, mock.Anything).
		Return(nil, errors.New("database unavailable"))
	env.OnActivity(a.FailSuiteActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *agentqualityservice.FailSuiteRequest) error {
			failed = true
			return nil
		})

	env.ExecuteWorkflow(AgentSuiteRunWorkflowName, suitePayload(false))
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	assert.True(t, failed, "a run that cannot go on is closed as failed, never left Running")
}

type sweepHarness struct {
	env      *testsuite.TestWorkflowEnvironment
	opened   []pulid.ID
	skipped  []pulid.ID
	children []*SuiteRunPayload
	sweepKey []string
}

func plannedAgent(skip bool) agentqualityservice.PlannedAgent {
	fingerprint := &agent.Fingerprint{DefinitionVersion: 3, PromptHash: "p", ToolSpecHash: "t"}

	return agentqualityservice.PlannedAgent{
		AgentDefinitionID: pulid.MustNew("agd_"),
		Name:              "Billing desk",
		Fingerprint:       fingerprint,
		FingerprintHash:   fingerprint.Hash(),
		SuiteRevision:     "rev",
		ActiveCases:       12,
		Skip:              skip,
	}
}

func newSweepHarness(t *testing.T, plan *agentqualityservice.SweepPlan) *sweepHarness {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	h := &sweepHarness{env: env}

	env.RegisterWorkflowWithOptions(AgentQualitySweepWorkflow,
		workflow.RegisterOptions{Name: AgentQualitySweepWorkflowName})
	env.RegisterWorkflowWithOptions(
		func(_ workflow.Context, payload *SuiteRunPayload) (*SuiteRunResult, error) {
			h.children = append(h.children, payload)

			return &SuiteRunResult{Status: agentquality.SuiteRunStatusCompleted}, nil
		},
		workflow.RegisterOptions{Name: AgentSuiteRunWorkflowName},
	)
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.PlanSweepActivity, mock.Anything, mock.Anything).Return(plan, nil)
	env.OnActivity(a.RecordSkippedSuiteActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			req *agentqualityservice.RecordSkippedRequest,
		) (*agentqualityservice.OpenedSuite, error) {
			h.skipped = append(h.skipped, req.Agent.AgentDefinitionID)
			h.sweepKey = append(h.sweepKey, req.SweepKey)

			return &agentqualityservice.OpenedSuite{Status: agentquality.SuiteRunStatusSkipped}, nil
		}).Maybe()
	env.OnActivity(a.OpenSuiteActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			req *agentqualityservice.OpenSuiteRequest,
		) (*agentqualityservice.OpenedSuite, error) {
			h.opened = append(h.opened, req.Agent.AgentDefinitionID)
			h.sweepKey = append(h.sweepKey, req.SweepKey)
			id := agentquality.NewSuiteRunID()

			return &agentqualityservice.OpenedSuite{
				SuiteRunID: id,
				Status:     agentquality.SuiteRunStatusRunning,
				CasesTotal: 10,
				SampleSeed: 5,
				WorkflowID: agentquality.SuiteRunWorkflowID(id),
			}, nil
		}).Maybe()

	return h
}

func TestAgentQualitySweepWorkflow_RunsChangedAgentsAndRecordsTheRest(t *testing.T) {
	t.Parallel()

	changed := plannedAgent(false)
	unchanged := plannedAgent(true)
	plan := &agentqualityservice.SweepPlan{
		Enabled:    true,
		DayStart:   1_800_000_000,
		MonthStart: 1_799_000_000,
		Settings:   settings(false),
		Agents:     []agentqualityservice.PlannedAgent{changed, unchanged},
	}
	h := newSweepHarness(t, plan)

	orgID := pulid.MustNew("org_")
	h.env.ExecuteWorkflow(AgentQualitySweepWorkflowName, &SweepPayload{
		OrganizationID: orgID,
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	require.True(t, h.env.IsWorkflowCompleted())
	require.NoError(t, h.env.GetWorkflowError())

	var result SweepResult
	require.NoError(t, h.env.GetWorkflowResult(&result))
	assert.Equal(t, SweepResult{Planned: 2, Ran: 1, Skipped: 1}, result)
	assert.Equal(t, []pulid.ID{changed.AgentDefinitionID}, h.opened)
	assert.Equal(t, []pulid.ID{unchanged.AgentDefinitionID}, h.skipped)
	require.Len(t, h.children, 1)
	assert.Equal(t, plan.DayStart, h.children[0].DayStart, "budget windows come from the plan")
	assert.Equal(t, orgID, h.children[0].OrganizationID)
	for _, key := range h.sweepKey {
		assert.Contains(t, key, "/agd_", "each agent's run is keyed on the sweep and the agent")
	}
}

func TestAgentQualitySweepWorkflow_DoesNothingWhenTurnedOff(t *testing.T) {
	t.Parallel()

	h := newSweepHarness(t, &agentqualityservice.SweepPlan{
		Enabled: false,
		Agents:  []agentqualityservice.PlannedAgent{plannedAgent(false)},
	})

	h.env.ExecuteWorkflow(AgentQualitySweepWorkflowName, &SweepPayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	require.NoError(t, h.env.GetWorkflowError())
	assert.Empty(t, h.opened)
	assert.Empty(t, h.children)
}

func TestAgentQualitySweepWorkflow_ContinuesAsNewWithItsPlan(t *testing.T) {
	t.Parallel()

	agents := make([]agentqualityservice.PlannedAgent, 0, agentsPerExecution+5)
	for range agentsPerExecution + 5 {
		agents = append(agents, plannedAgent(true))
	}
	h := newSweepHarness(t, &agentqualityservice.SweepPlan{Enabled: true, Agents: agents})

	h.env.ExecuteWorkflow(AgentQualitySweepWorkflowName, &SweepPayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	require.True(t, h.env.IsWorkflowCompleted())

	var continued *workflow.ContinueAsNewError
	require.True(t, errors.As(h.env.GetWorkflowError(), &continued))

	var next SweepPayload
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(continued.Input, &next))
	assert.Equal(t, agentsPerExecution, next.Cursor)
	require.NotNil(t, next.Plan, "the plan is carried, not made again")
	assert.Len(t, next.Plan.Agents, agentsPerExecution+5)
	require.NotNil(t, next.Result)
	assert.Equal(t, agentsPerExecution, next.Result.Skipped)
}
