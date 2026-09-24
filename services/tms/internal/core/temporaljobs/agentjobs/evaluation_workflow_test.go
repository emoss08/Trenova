package agentjobs

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type evaluationRun struct {
	activities []string
	finished   *FinishReplayInput
	failed     bool
	err        error
}

func runEvaluationWorkflow(
	t *testing.T,
	open func(context.Context) (*OpenReplayResult, error),
) *evaluationRun {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	runtime := agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  &agentruntimetest.ScriptedCompletion{},
		QueryTools:  &agentruntimetest.StubQueryRegistry{},
		ActionTools: &agentruntimetest.StubActionRegistry{},
		Permissions: &agentruntimetest.StubPermissions{},
	})

	env.RegisterActivity(&Activities{})
	env.RegisterActivity(&agentflow.Activities{})
	env.RegisterWorkflowWithOptions(NewWorkflows(runtime).AgentEvaluationWorkflow,
		workflow.RegisterOptions{Name: AgentEvaluationWorkflowName})

	result := &evaluationRun{}
	var mu sync.Mutex
	record := func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
		mu.Lock()
		defer mu.Unlock()
		result.activities = append(result.activities, info.ActivityType.Name)
	}
	env.SetOnActivityStartedListener(record)

	var a *Activities
	var fa *agentflow.Activities
	env.OnActivity(a.OpenReplayActivity, mock.Anything, mock.Anything).
		Return(func(ctx context.Context, _ *AgentEvaluationPayload) (*OpenReplayResult, error) {
			opened, err := open(ctx)
			if err != nil || opened.Done {
				return opened, err
			}
			req := &serviceports.RunRequest{
				Definition: replayDefinition(),
				Actor: &serviceports.RequestActor{
					PrincipalType: serviceports.PrincipalTypeUser,
					PrincipalID:   pulid.MustNew("usr_"),
					UserID:        pulid.MustNew("usr_"),
				},
				Input:        "Rate S-100",
				RunID:        pulid.MustNew(agent.EvaluationIDPrefix),
				Unattended:   true,
				UsagePurpose: serviceports.AIUsagePurposeEvaluation,
			}
			opened.Run = agentflow.NewRunContext(req, agentflow.PriorityEvaluation)
			opened.Turn = runtime.OpenTurn(ctx, req).State()

			return opened, nil
		}).Once()
	env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).
		Return(&agentruntime.ModelReply{Completion: &serviceports.ChatCompletionResult{
			Text:            "The rate stays at 1,350.",
			ModelIdentifier: "test-model",
		}}, nil).Maybe()
	env.OnActivity(a.FinishReplayActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *FinishReplayInput) (*ReplayRunResult, error) {
			result.finished = input
			return &ReplayRunResult{Model: "test-model"}, nil
		}).Maybe()
	env.OnActivity(a.FailEvaluationActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *FailEvaluationInput) error {
			result.failed = true
			return nil
		}).Maybe()

	env.ExecuteWorkflow(AgentEvaluationWorkflowName, &AgentEvaluationPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		EvaluationID: pulid.MustNew(agent.EvaluationIDPrefix),
	})
	require.True(t, env.IsWorkflowCompleted())
	result.err = env.GetWorkflowError()

	return result
}

func replayDefinition() *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		ID:              pulid.MustNew("agd_"),
		Name:            "Billing desk",
		Instructions:    "Clear the billing queue.",
		AutonomyCeiling: agent.TierPropose,
		Enabled:         true,
		SimulationMode:  true,
	}
	definition.ApplyDefaults()

	return definition
}

func TestAgentEvaluationWorkflow_ACaseReplayIssuesTheSameCommandsAsARunReplay(t *testing.T) {
	t.Parallel()

	runReplay := runEvaluationWorkflow(t, func(context.Context) (*OpenReplayResult, error) {
		return &OpenReplayResult{Originals: []agent.OriginalProposal{{
			ID:       pulid.MustNew("aprop_"),
			ToolName: "update_rate",
			Params:   map[string]any{"rate": float64(1200)},
			CorrectedParams: map[string]any{
				"rate": float64(1350),
			},
			Decision: agent.DecisionModified,
		}}}, nil
	})
	caseReplay := runEvaluationWorkflow(t, func(context.Context) (*OpenReplayResult, error) {
		return &OpenReplayResult{Originals: []agent.OriginalProposal{{
			ToolName: "update_rate",
			Params:   map[string]any{"rate": float64(1350)},
			Decision: agent.DecisionAccepted,
		}}}, nil
	})

	require.NoError(t, runReplay.err)
	require.NoError(t, caseReplay.err)
	assert.Equal(t,
		[]string{"OpenReplayActivity", "ModelCallActivity", "FinishReplayActivity"},
		runReplay.activities,
	)
	assert.Equal(t, runReplay.activities, caseReplay.activities,
		"which source an evaluation replays is decided inside its activities")
	require.NotNil(t, runReplay.finished)
	require.Len(t, runReplay.finished.Originals, 1)
	assert.Equal(t, float64(1350), runReplay.finished.Originals[0].CorrectedParams["rate"],
		"the corrected parameters travel from open to finish")
}

func TestAgentEvaluationWorkflow_ASkippedReplayEndsAfterOpening(t *testing.T) {
	t.Parallel()

	skipped := runEvaluationWorkflow(t, func(context.Context) (*OpenReplayResult, error) {
		return &OpenReplayResult{Done: true}, nil
	})

	require.NoError(t, skipped.err)
	assert.Equal(t, []string{"OpenReplayActivity"}, skipped.activities)
	assert.False(
		t,
		skipped.failed,
		"a skip is recorded by the activity, not failed by the workflow",
	)
	assert.Nil(t, skipped.finished)
}
