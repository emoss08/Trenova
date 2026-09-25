package accountingsyncjobs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type WorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}

func (s *WorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *WorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func refreshPayload() *RefreshReferencePayload {
	return &RefreshReferencePayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ConnectionID:   pulid.MustNew("acctc_"),
	}
}

type finishRecorder struct {
	mu       sync.Mutex
	failures []string
}

func (r *finishRecorder) record(_ context.Context, _ *RefreshReferencePayload, failure string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failures = append(r.failures, failure)
	return nil
}

func (s *WorkflowTestSuite) expectMarks(recorder *finishRecorder) {
	var a *Activities
	s.env.OnActivity(a.MarkAccountingReferenceRefreshStartedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	s.env.OnActivity(a.MarkAccountingReferenceRefreshFinishedActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(recorder.record).Once()
}

func (s *WorkflowTestSuite) TestRefreshPullsEveryKindThenRescoresAndAsksTheModel() {
	var a *Activities
	recorder := &finishRecorder{}
	s.expectMarks(recorder)

	pulled := make([]accountingsync.ReferenceKind, 0)
	var mu sync.Mutex
	s.env.OnActivity(a.PullAccountingReferenceActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, _ *RefreshReferencePayload, kind accountingsync.ReferenceKind) (*ReferenceKindResult, error) {
			mu.Lock()
			defer mu.Unlock()
			pulled = append(pulled, kind)
			return &ReferenceKindResult{Kind: string(kind), Fetched: 2}, nil
		},
	)
	needs := []pulid.ID{pulid.MustNew("acctm_"), pulid.MustNew("acctm_")}
	s.env.OnActivity(a.RescoreAccountingMappingsActivity, mock.Anything, mock.Anything).
		Return(&RescoreReferenceResult{Targets: 20, Created: 20, Updated: 9, Proposed: 7, NeedsModel: needs}, nil).
		Once()
	s.env.OnActivity(a.AccountingMappingModelPassActivity, mock.Anything, mock.Anything, needs).
		Return(2, nil).Once()

	s.env.ExecuteWorkflow(RefreshAccountingReferenceWorkflow, refreshPayload())

	s.True(s.env.IsWorkflowCompleted())
	s.Require().NoError(s.env.GetWorkflowError())
	var result RefreshReferenceResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(accountingsync.AllReferenceKinds(), pulled)
	s.Len(result.Kinds, len(accountingsync.AllReferenceKinds()))
	s.Equal(9, result.Proposed)
	s.Equal(2, result.ModelProposed)
	s.Equal([]string{""}, recorder.failures)
}

func (s *WorkflowTestSuite) TestRefreshSkipsTheModelWhenNothingNeedsIt() {
	var a *Activities
	recorder := &finishRecorder{}
	s.expectMarks(recorder)
	s.env.OnActivity(a.PullAccountingReferenceActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(&ReferenceKindResult{}, nil)
	s.env.OnActivity(a.RescoreAccountingMappingsActivity, mock.Anything, mock.Anything).
		Return(&RescoreReferenceResult{Targets: 3}, nil).Once()

	s.env.ExecuteWorkflow(RefreshAccountingReferenceWorkflow, refreshPayload())

	s.Require().NoError(s.env.GetWorkflowError())
	s.Equal([]string{""}, recorder.failures)
}

func (s *WorkflowTestSuite) TestRefreshRecordsAFailedPull() {
	var a *Activities
	recorder := &finishRecorder{}
	s.expectMarks(recorder)
	s.env.OnActivity(a.PullAccountingReferenceActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"QuickBooks Online is not connected", "BusinessError", errors.New("not connected"),
		))

	s.env.ExecuteWorkflow(RefreshAccountingReferenceWorkflow, refreshPayload())

	s.True(s.env.IsWorkflowCompleted())
	s.Require().Error(s.env.GetWorkflowError())
	s.Equal([]string{"QuickBooks Online is not connected"}, recorder.failures)
}

func (s *WorkflowTestSuite) TestRefreshKeepsDeterministicProposalsWhenTheModelFails() {
	var a *Activities
	recorder := &finishRecorder{}
	s.expectMarks(recorder)
	s.env.OnActivity(a.PullAccountingReferenceActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(&ReferenceKindResult{}, nil)
	s.env.OnActivity(a.RescoreAccountingMappingsActivity, mock.Anything, mock.Anything).
		Return(&RescoreReferenceResult{Proposed: 4, NeedsModel: []pulid.ID{pulid.MustNew("acctm_")}}, nil).Once()
	s.env.OnActivity(a.AccountingMappingModelPassActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(0, temporal.NewNonRetryableApplicationError("model rejected", "ModelRejected", nil))

	s.env.ExecuteWorkflow(RefreshAccountingReferenceWorkflow, refreshPayload())

	s.Require().NoError(s.env.GetWorkflowError())
	var result RefreshReferenceResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(4, result.Proposed)
	s.Equal([]string{""}, recorder.failures)
}

func connections(n int) []RefreshReferencePayload {
	out := make([]RefreshReferencePayload, 0, n)
	for range n {
		out = append(out, *refreshPayload())
	}
	return out
}

func (s *WorkflowTestSuite) TestSweepStartsARefreshPerConnection() {
	var a *Activities
	page := connections(3)
	s.env.OnActivity(a.ListAccountingReferenceConnectionsActivity, mock.Anything, pulid.Nil).
		Return(&ReferenceSweepPage{Connections: page, LastID: page[2].ConnectionID}, nil).Once()

	started := make([]pulid.ID, 0, len(page))
	var mu sync.Mutex
	s.env.OnWorkflow(RefreshAccountingReferenceWorkflow, mock.Anything, mock.Anything).Return(
		func(_ workflow.Context, payload *RefreshReferencePayload) (*RefreshReferenceResult, error) {
			mu.Lock()
			defer mu.Unlock()
			started = append(started, payload.ConnectionID)
			return &RefreshReferenceResult{}, nil
		},
	)

	s.env.ExecuteWorkflow(RefreshAllAccountingReferenceWorkflow, &ReferenceSweepPayload{})

	s.Require().NoError(s.env.GetWorkflowError())
	var result ReferenceSweepResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(3, result.Started)
	s.ElementsMatch([]pulid.ID{page[0].ConnectionID, page[1].ConnectionID, page[2].ConnectionID}, started)
}

func (s *WorkflowTestSuite) TestSweepContinuesAsNewWhenThereIsAnotherPage() {
	var a *Activities
	page := connections(2)
	s.env.OnActivity(a.ListAccountingReferenceConnectionsActivity, mock.Anything, pulid.Nil).
		Return(&ReferenceSweepPage{Connections: page, LastID: page[1].ConnectionID, More: true}, nil).Once()
	s.env.OnWorkflow(RefreshAccountingReferenceWorkflow, mock.Anything, mock.Anything).
		Return(&RefreshReferenceResult{}, nil)

	s.env.ExecuteWorkflow(RefreshAllAccountingReferenceWorkflow, &ReferenceSweepPayload{})

	err := s.env.GetWorkflowError()
	s.Require().Error(err)
	s.True(workflow.IsContinueAsNewError(err))
}

func TestReferenceWorkflowIDIsPerConnection(t *testing.T) {
	t.Parallel()
	id := pulid.MustNew("acctc_")
	if got := ReferenceWorkflowID(id); got != "accounting-reference:"+id.String() {
		t.Fatalf("unexpected workflow id %q", got)
	}
}
