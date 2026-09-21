package agentjobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type AgentRunWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env       *testsuite.TestWorkflowEnvironment
	payload   *AgentRunPayload
	completed []CompleteRunInput
	expired   []ExpireProposalsInput
}

func (s *AgentRunWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.completed = nil
	s.expired = nil
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

func (s *AgentRunWorkflowTestSuite) prepared(shadow bool, decisionTimeout int) *PrepareRunResult {
	return &PrepareRunResult{
		Definition: &agentdefinition.Definition{
			ID:             s.payload.DefinitionID,
			OrganizationID: s.payload.OrganizationID,
			BusinessUnitID: s.payload.BusinessUnitID,
			Name:           "Billing exceptions",
		},
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

func (s *AgentRunWorkflowTestSuite) stubRun(pending int) {
	var a *Activities
	s.env.OnActivity(a.RunAgentActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *RunAgentInput) (*RunAgentResult, error) {
			s.Equal(s.payload.RunID, input.Payload.RunID)
			s.NotNil(input.Definition)
			return &RunAgentResult{
				Reply:            "done",
				ProposalsRaised:  pending,
				PendingProposals: pending,
			}, nil
		}).Once()
}

func (s *AgentRunWorkflowTestSuite) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: s.payload.OrganizationID,
		BuID:  s.payload.BusinessUnitID,
	}
}

func (s *AgentRunWorkflowTestSuite) TestShadowRunCompletesAsShadow() {
	s.stubPrepare(s.prepared(true, 600))
	s.stubRun(2)

	s.env.ExecuteWorkflow(AgentRunWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Require().Len(s.completed, 1)
	s.Equal(agent.RunStatusShadowCompleted, s.completed[0].Status)
	s.Equal(s.payload.RunID, s.completed[0].RunID)
	s.Equal(s.tenant(), s.completed[0].TenantInfo)
	s.Empty(s.expired)
}

func (s *AgentRunWorkflowTestSuite) TestNoProposalsCompletesImmediately() {
	s.stubPrepare(s.prepared(false, 600))
	s.stubRun(0)

	s.env.ExecuteWorkflow(AgentRunWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Require().Len(s.completed, 1)
	s.Equal(agent.RunStatusCompleted, s.completed[0].Status)
}

func (s *AgentRunWorkflowTestSuite) TestPendingProposalsExpireAfterTimeout() {
	s.stubPrepare(s.prepared(false, 600))
	s.stubRun(1)

	s.env.ExecuteWorkflow(AgentRunWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Require().Len(s.expired, 1)
	s.Equal(s.payload.RunID, s.expired[0].RunID)
	s.Equal(s.tenant(), s.expired[0].TenantInfo)
	s.Empty(s.completed, "an expired run is not also marked completed")
}

func (s *AgentRunWorkflowTestSuite) TestDecisionSignalCompletesRun() {
	s.stubPrepare(s.prepared(false, 600))
	s.stubRun(1)

	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(AgentDecisionSignalName, DecisionSignal{
			ProposalID:      pulid.MustNew("agp_"),
			Decision:        agent.DecisionAccepted,
			DecidedByUserID: pulid.MustNew("usr_"),
		})
	}, time.Minute)

	s.env.ExecuteWorkflow(AgentRunWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
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

	s.env.ExecuteWorkflow(AgentRunWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
	s.Require().Len(s.completed, 1)
	s.Equal(agent.RunStatusFailed, s.completed[0].Status)
	s.Contains(s.completed[0].Error, "definition disabled")
}

func (s *AgentRunWorkflowTestSuite) TestRunFailureMarksRunFailed() {
	s.stubPrepare(s.prepared(false, 600))
	var a *Activities
	s.env.OnActivity(a.RunAgentActivity, mock.Anything, mock.Anything).
		Return(nil, errors.New("provider unavailable"))

	s.env.ExecuteWorkflow(AgentRunWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
	s.Require().NotEmpty(s.completed)
	s.Equal(agent.RunStatusFailed, s.completed[len(s.completed)-1].Status)
}

func TestAgentRunWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(AgentRunWorkflowTestSuite))
}

type AgentSweepWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *AgentSweepWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *AgentSweepWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func dueFixture(n int) []DueDefinition {
	due := make([]DueDefinition, 0, n)
	for i := 0; i < n; i++ {
		due = append(due, DueDefinition{
			DefinitionID:   pulid.MustNew("agd_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			NextRunAt:      1_700_000_000,
		})
	}

	return due
}

func (s *AgentSweepWorkflowTestSuite) TestStartsEachDueDefinitionIndependently() {
	var a *Activities
	due := dueFixture(3)
	started := make(map[pulid.ID]struct{}, 3)

	s.env.OnActivity(a.ListDueDefinitionsActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *ListDueDefinitionsInput) (*ListDueDefinitionsResult, error) {
			s.Equal(sweepDueLimit, input.Limit)
			s.Positive(input.Now)
			return &ListDueDefinitionsResult{Due: due}, nil
		}).Once()
	s.env.OnActivity(a.StartDueRunActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, item *DueDefinition) (*StartDueRunResult, error) {
			started[item.DefinitionID] = struct{}{}
			switch item.DefinitionID {
			case due[0].DefinitionID:
				return &StartDueRunResult{Started: true, RunID: "ar_1"}, nil
			case due[1].DefinitionID:
				return &StartDueRunResult{Started: false, Skipped: "concurrency"}, nil
			default:
				return nil, errors.New("temporal unavailable")
			}
		})

	s.env.ExecuteWorkflow(AgentSweepWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *SweepResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(3, result.Found)
	s.Equal(1, result.Started)
	s.Equal(1, result.Skipped)
	s.Equal(1, result.Failed)
	s.Len(started, 3, "one failure must not stop the other definitions from starting")
}

func (s *AgentSweepWorkflowTestSuite) TestNothingDueIsANoop() {
	var a *Activities
	s.env.OnActivity(a.ListDueDefinitionsActivity, mock.Anything, mock.Anything).
		Return(&ListDueDefinitionsResult{Due: []DueDefinition{}}, nil).Once()

	s.env.ExecuteWorkflow(AgentSweepWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *SweepResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(0, result.Found)
	s.Equal(0, result.Started)
}

func TestAgentSweepWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(AgentSweepWorkflowTestSuite))
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
