package agentjobs

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
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
	payload   *AgentRunPayload
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
	s.runtime = agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  &agentruntimetest.ScriptedCompletion{},
		QueryTools:  &agentruntimetest.StubQueryRegistry{},
		ActionTools: &agentruntimetest.StubActionRegistry{},
		Permissions: &agentruntimetest.StubPermissions{},
	})
	s.payload = &AgentRunPayload{
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
		workflow.RegisterOptions{Name: AgentRunWorkflowName})

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

func (s *AgentRunWorkflowTestSuite) prepared(shadow bool, decisionTimeout int) *PrepareRunResult {
	return &PrepareRunResult{
		Definition:             s.definition(),
		ShadowMode:             shadow,
		DecisionTimeoutSeconds: decisionTimeout,
		RunTimeoutSeconds:      120,
	}
}

func (s *AgentRunWorkflowTestSuite) stubPrepare(result *PrepareRunResult) {
	var a *Activities
	s.env.OnActivity(a.PrepareRunActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, payload *AgentRunPayload) (*PrepareRunResult, error) {
			s.Equal(s.payload.RunID, payload.RunID)
			return result, nil
		}).Once()
}

// stubOpen opens the run as a real runtime turn, so the loop in workflow code
// has something to drive.
func (s *AgentRunWorkflowTestSuite) stubOpen() {
	var a *Activities
	s.env.OnActivity(a.OpenRunActivity, mock.Anything, mock.Anything).
		Return(func(ctx context.Context, input *OpenRunInput) (*OpenRunResult, error) {
			s.opened = input
			req := &serviceports.RunRequest{
				Definition: input.Definition,
				Actor:      agentActor(input.Payload.tenantInfo()),
				Input:      "A scheduled run.",
				RunID:      input.Payload.RunID,
				Unattended: true,
			}

			return &OpenRunResult{
				Run:  agentflow.NewRunContext(req, agentflow.PriorityBackground),
				Turn: s.runtime.OpenTurn(ctx, req).State(),
			}, nil
		}).Once()
}

func (s *AgentRunWorkflowTestSuite) stubAnswer(text string) {
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *agentflow.ModelCallInput) (*agentruntime.ModelReply, error) {
			s.False(in.Stream, "nobody watches a background run, so nothing is streamed")
			return &agentruntime.ModelReply{Completion: &serviceports.ChatCompletionResult{
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
	s.env.ExecuteWorkflow(AgentRunWorkflowName, s.payload)
	s.True(s.env.IsWorkflowCompleted())
}

func (s *AgentRunWorkflowTestSuite) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: s.payload.OrganizationID,
		BuID:  s.payload.BusinessUnitID,
	}
}

func (s *AgentRunWorkflowTestSuite) decide() {
	s.env.SignalWorkflow(AgentDecisionSignalName, DecisionSignal{
		ProposalID:      pulid.MustNew("agp_"),
		Decision:        agent.DecisionAccepted,
		DecidedByUserID: pulid.MustNew("usr_"),
	})
}

func (s *AgentRunWorkflowTestSuite) TestRunsTheLoopAndFilesWhatItDid() {
	s.stubPrepare(s.prepared(false, 600))
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
	s.stubPrepare(s.prepared(true, 600))
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

func (s *AgentRunWorkflowTestSuite) TestPendingProposalsExpireAfterTimeout() {
	s.stubPrepare(s.prepared(false, 600))
	s.stubOpen()
	s.stubAnswer("Proposed a change.")
	s.stubFinish(1)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Require().Len(s.expired, 1)
	s.Equal(s.payload.RunID, s.expired[0].RunID)
	s.Equal(s.tenant(), s.expired[0].TenantInfo)
	s.Empty(s.completed, "an expired run is not also marked completed")
}

// The first decision used to end the run, and every other proposal it raised
// was left pending on a run that had stopped listening. The run now waits
// until none is pending.
func (s *AgentRunWorkflowTestSuite) TestWaitsForEveryDecision() {
	s.stubPrepare(s.prepared(false, 600))
	s.stubOpen()
	s.stubAnswer("Proposed two changes.")
	s.stubFinish(2)

	var a *Activities
	counts := []int{1, 0}
	s.env.OnActivity(a.PendingProposalsActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *PendingProposalsInput) (int, error) {
			next := counts[0]
			counts = counts[1:]
			return next, nil
		}).Twice()

	s.env.RegisterDelayedCallback(s.decide, time.Minute)
	s.env.RegisterDelayedCallback(s.decide, 2*time.Minute)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Empty(counts, "each decision recounts what is left")
	s.Require().Len(s.completed, 1)
	s.Equal(agent.RunStatusCompleted, s.completed[0].Status)
	s.Empty(s.expired)
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
	s.stubPrepare(s.prepared(false, 600))
	s.stubOpen()
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"provider rejected the request", agentflow.ErrTypeModelRejected, nil,
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
	payload := &ScheduledRunPayload{
		DefinitionID:   pulid.MustNew("agd_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	var slot int64
	s.env.OnActivity(a.StartScheduledRunActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, fired *ScheduledRunPayload) (*StartScheduledRunResult, error) {
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

type DeleteStaleAskThreadsWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *DeleteStaleAskThreadsWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

// The sweep removes a batch at a time and keeps going while batches come
// back full, so one quiet night clears a backlog without one long
// transaction. The cut-off is a month before the workflow's own clock.
func (s *DeleteStaleAskThreadsWorkflowTestSuite) TestSweepsInBatchesUntilShort() {
	var a *Activities
	calls := 0
	var cutoffs []int64

	s.env.OnActivity(a.DeleteStaleAskThreadsActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *DeleteStaleAskThreadsInput) (*DeleteStaleAskThreadsResult, error) {
			calls++
			cutoffs = append(cutoffs, input.Before)
			if calls == 1 {
				return &DeleteStaleAskThreadsResult{Deleted: deleteStaleAskBatch}, nil
			}
			return &DeleteStaleAskThreadsResult{Deleted: 3}, nil
		})

	s.env.ExecuteWorkflow(DeleteStaleAskThreadsWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *DeleteStaleAskThreadsResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(deleteStaleAskBatch+3, result.Deleted)
	s.Equal(2, calls)
	s.Require().Len(cutoffs, 2)
	s.Equal(cutoffs[0], cutoffs[1], "one cut-off for the whole sweep")
	s.Less(cutoffs[0], time.Now().Add(-29*24*time.Hour).Unix())
}

func TestDeleteStaleAskThreadsWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteStaleAskThreadsWorkflowTestSuite))
}
