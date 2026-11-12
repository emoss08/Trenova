package agentjobs

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type AgentRunWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env       *testsuite.TestWorkflowEnvironment
	runtime   *agentruntime.Service
	payload   *agentcontract.AgentRunPayload
	completed []CompleteRunInput
	expired   []ExpireProposalsInput
	opened    *OpenRunInput
	finished  *FinishRunInput
}

func (s *AgentRunWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.completed = nil
	s.expired = nil
	s.opened = nil
	s.finished = nil
	s.runtime = assistanttest.Runtime(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  &agenttest.ScriptedCompletion{},
		QueryTools:  &agenttest.StubQueryRegistry{},
		ActionTools: &agenttest.StubActionRegistry{},
		Permissions: &agenttest.StubPermissions{},
	})
	s.payload = &agentcontract.AgentRunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		RunID:        pulid.MustNew("ar_"),
		DefinitionID: pulid.MustNew("agd_"),
		Trigger:      agent.RunTriggerEvent,
		SubjectType:  agent.SubjectBillingQueueItem,
		SubjectID:    pulid.MustNew("bqi_"),
	}

	s.env.RegisterActivity(&Activities{})
	s.env.RegisterActivity(&agentflow.Activities{})
	s.env.RegisterWorkflowWithOptions(NewWorkflows(s.runtime).AgentRunWorkflow,
		workflow.RegisterOptions{Name: agentcontract.AgentRunWorkflowName})

	var a *Activities
	s.env.OnActivity(a.CompleteRunActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *CompleteRunInput) error {
			s.completed = append(s.completed, *input)
			return nil
		}).Maybe()
	s.env.OnActivity(a.ExpireProposalsActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *ExpireProposalsInput) error {
			s.expired = append(s.expired, *input)
			return nil
		}).Maybe()
}

func (s *AgentRunWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *AgentRunWorkflowTestSuite) definition() *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		ID:              s.payload.DefinitionID,
		OrganizationID:  s.payload.OrganizationID,
		BusinessUnitID:  s.payload.BusinessUnitID,
		Name:            "Billing exceptions",
		Instructions:    "Clear the billing queue.",
		AutonomyCeiling: agent.TierPropose,
		Enabled:         true,
	}
	definition.ApplyDefaults()

	return definition
}

func (s *AgentRunWorkflowTestSuite) prepared(shadow bool) *PrepareRunResult {
	return &PrepareRunResult{
		Definition:        s.definition(),
		ShadowMode:        shadow,
		RunTimeoutSeconds: 120,
	}
}

func (s *AgentRunWorkflowTestSuite) stubPrepare(result *PrepareRunResult) {
	var a *Activities
	s.env.OnActivity(a.PrepareRunActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, payload *agentcontract.AgentRunPayload) (*PrepareRunResult, error) {
			s.Equal(s.payload.RunID, payload.RunID)
			return result, nil
		}).
		Once()
}

// stubOpen opens the run as a real runtime turn, so the loop in workflow code
// has something to drive.
func (s *AgentRunWorkflowTestSuite) stubOpen() {
	var a *Activities
	s.env.OnActivity(a.OpenRunActivity, mock.Anything, mock.Anything).
		Return(func(ctx context.Context, input *OpenRunInput) (*OpenRunResult, error) {
			s.opened = input
			req := &agentruntimeports.RunRequest{
				Definition: input.Definition,
				Actor:      agentActor(input.Payload.TenantInfo()),
				Input:      "A scheduled run.",
				RunID:      input.Payload.RunID,
				Unattended: true,
			}

			return &OpenRunResult{
				Run:  agentflow.NewRunContext(req, agentflowcontract.PriorityBackground),
				Turn: s.runtime.OpenTurn(ctx, req).State(),
			}, nil
		}).Once()
}

func (s *AgentRunWorkflowTestSuite) stubAnswer(text string) {
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *agentflow.ModelCallInput) (*modelstream.Reply, error) {
			s.False(in.Stream, "nobody watches a background run, so nothing is streamed")
			return &modelstream.Reply{Completion: &llmports.ChatCompletionResult{
				Text:            text,
				ModelIdentifier: "test-model",
			}}, nil
		},
	).Once()
}

func (s *AgentRunWorkflowTestSuite) stubFinish(pending int) {
	var a *Activities
	s.env.OnActivity(a.FinishRunActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *FinishRunInput) (*FinishRunResult, error) {
			s.finished = input
			return &FinishRunResult{ProposalsRaised: pending, PendingProposals: pending}, nil
		}).Once()
}

func (s *AgentRunWorkflowTestSuite) run() {
	s.env.ExecuteWorkflow(agentcontract.AgentRunWorkflowName, s.payload)
	s.True(s.env.IsWorkflowCompleted())
}

func (s *AgentRunWorkflowTestSuite) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: s.payload.OrganizationID,
		BuID:  s.payload.BusinessUnitID,
	}
}

func (s *AgentRunWorkflowTestSuite) TestRunsTheLoopAndFilesWhatItDid() {
	s.stubPrepare(s.prepared(false))
	s.stubOpen()
	s.stubAnswer("Nothing needed attention.")
	s.stubFinish(0)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Require().NotNil(s.finished)
	s.Nil(s.finished.Failure)
	s.Require().NotNil(s.finished.Run)
	s.Equal("Nothing needed attention.", s.finished.Run.Reply)
	s.Require().Len(s.completed, 1)
	s.Equal(agent.RunStatusCompleted, s.completed[0].Status)
}

// Shadow mode means the agent is watched and not acted on. The run is opened
// in simulation, so an automatic write is previewed rather than made.
func (s *AgentRunWorkflowTestSuite) TestShadowRunIsOpenedInSimulationAndCompletesAsShadow() {
	s.stubPrepare(s.prepared(true))
	s.stubOpen()
	s.stubAnswer("Proposed two changes.")
	s.stubFinish(2)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Require().NotNil(s.opened)
	s.True(s.opened.Shadow)
	s.Require().Len(s.completed, 1)
	s.Equal(agent.RunStatusShadowCompleted, s.completed[0].Status)
	s.Equal(s.payload.RunID, s.completed[0].RunID)
	s.Equal(s.tenant(), s.completed[0].TenantInfo)
	s.Empty(s.expired)
}

// A run's proposals wait on decisions as records, not inside the workflow:
// it ends once they are filed, leaving the run waiting, so a run that raised
// proposals holds no workflow open for its decision window. The decision that
// leaves nothing pending, or the expiry sweep, closes it (agentrunclose).
func (s *AgentRunWorkflowTestSuite) TestPendingProposalsEndTheWorkflowWithTheRunWaiting() {
	s.stubPrepare(s.prepared(false))
	s.stubOpen()
	s.stubAnswer("Proposed two changes.")
	s.stubFinish(2)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Require().NotNil(s.finished)
	s.Empty(s.completed, "the run stays waiting on its decisions")
	s.Empty(s.expired, "nothing expires when the workflow ends")
}

func (s *AgentRunWorkflowTestSuite) TestPrepareFailureMarksRunFailed() {
	var a *Activities
	s.env.OnActivity(a.PrepareRunActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"definition disabled", "AgentDefinitionUnavailable", nil,
		)).Once()

	s.run()

	s.Error(s.env.GetWorkflowError())
	s.Require().Len(s.completed, 1)
	s.Equal(agent.RunStatusFailed, s.completed[0].Status)
	s.Contains(s.completed[0].Error, "definition disabled")
}

// A model that fails for good fails the run, and what the run did before it
// is still filed: a write it made is on the record, and a proposal it raised
// is not left waiting on a run that will never hear its decision.
func (s *AgentRunWorkflowTestSuite) TestModelFailureFilesWhatRanAndFailsTheRun() {
	s.stubPrepare(s.prepared(false))
	s.stubOpen()
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"provider rejected the request", modelcall.ErrTypeModelRejected, nil,
		)).Once()
	s.stubFinish(1)

	s.run()

	s.Error(s.env.GetWorkflowError())
	s.Require().NotNil(s.finished)
	s.Require().NotNil(s.finished.Failure)
	s.Require().Len(s.expired, 1)
	s.Require().NotEmpty(s.completed)
	s.Equal(agent.RunStatusFailed, s.completed[len(s.completed)-1].Status)
}

func TestAgentRunWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(AgentRunWorkflowTestSuite))
}

type AgentScheduledRunWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *AgentScheduledRunWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.RegisterActivity(&Activities{})
}

func (s *AgentScheduledRunWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

// A schedule's firing starts the run for its slot. The slot names the run's
// workflow, so the same slot cannot start two runs.
func (s *AgentScheduledRunWorkflowTestSuite) TestStartsTheRunForTheSlot() {
	var a *Activities
	payload := &agentcontract.ScheduledRunPayload{
		DefinitionID:   pulid.MustNew("agd_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	var slot int64
	s.env.OnActivity(a.StartScheduledRunActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, fired *agentcontract.ScheduledRunPayload) (*StartScheduledRunResult, error) {
			s.Equal(payload.DefinitionID, fired.DefinitionID)
			slot = fired.Slot
			return &StartScheduledRunResult{Started: true, RunID: "ar_1"}, nil
		}).
		Once()

	s.env.ExecuteWorkflow(AgentScheduledRunWorkflow, payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Positive(slot, "the slot is filled in from the time the schedule fired")

	var result *StartScheduledRunResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.True(result.Started)
}

func TestAgentScheduledRunWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(AgentScheduledRunWorkflowTestSuite))
}

type AssistantRetentionWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *AssistantRetentionWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

// Each sweep removes a batch at a time and keeps going while batches come
// back full, so one quiet night clears a backlog without one long
// transaction. Formula drafts go after a month quiet; palette question
// records after two months, past any allowance that reads them.
func (s *AssistantRetentionWorkflowTestSuite) TestSweepsEachRecordInBatchesUntilShort() {
	var a *Activities
	draftCalls := 0
	var draftCutoffs []int64
	var questionCutoff int64

	s.env.OnActivity(a.DeleteStaleFormulaDraftsActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *RetentionInput) (*RetentionBatch, error) {
			draftCalls++
			draftCutoffs = append(draftCutoffs, input.Before)
			if draftCalls == 1 {
				return &RetentionBatch{Deleted: deleteStaleDraftBatch}, nil
			}
			return &RetentionBatch{Deleted: 3}, nil
		})
	s.env.OnActivity(a.DeleteStalePaletteQuestionsActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *RetentionInput) (*RetentionBatch, error) {
			questionCutoff = input.Before
			return &RetentionBatch{Deleted: 7}, nil
		})

	s.env.ExecuteWorkflow(AssistantRetentionWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *RetentionResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(deleteStaleDraftBatch+3, result.FormulaDrafts)
	s.Equal(7, result.PaletteQuestions)
	s.Equal(2, draftCalls)
	s.Require().Len(draftCutoffs, 2)
	s.Equal(draftCutoffs[0], draftCutoffs[1], "one cut-off for the whole sweep")
	s.Less(draftCutoffs[0], time.Now().Add(-29*24*time.Hour).Unix())
	s.Less(questionCutoff, time.Now().Add(-61*24*time.Hour).Unix())
}

func TestAssistantRetentionWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(AssistantRetentionWorkflowTestSuite))
}
