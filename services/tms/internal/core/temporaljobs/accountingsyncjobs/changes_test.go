package accountingsyncjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type ChangesWorkflowSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func TestChangesWorkflowSuite(t *testing.T) {
	suite.Run(t, new(ChangesWorkflowSuite))
}

func (s *ChangesWorkflowSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *ChangesWorkflowSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func changesPayload() *ChangesPayload {
	return &ChangesPayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ConnectionID:   pulid.MustNew("acctc_"),
	}
}

func (s *ChangesWorkflowSuite) TestReadsEveryPageThenEvaluatesAfterTheSettleTime() {
	var a *Activities
	var readAt, evaluatedAt time.Time
	s.env.OnActivity(a.PollAccountingChangesActivity, mock.Anything, mock.Anything).
		Return(&ChangesReadResult{Recorded: 2, More: true}, nil).Once()
	s.env.OnActivity(a.PollAccountingChangesActivity, mock.Anything, mock.Anything).
		Return(&ChangesReadResult{Recorded: 1}, nil).
		Once().Run(func(mock.Arguments) { readAt = s.env.Now() })
	s.env.OnActivity(a.EvaluateAccountingInboundActivity, mock.Anything, mock.Anything).
		Return(&ChangesEvaluationResult{Evaluated: 3, Proposed: 2, Applied: 1, More: true}, nil).
		Once().Run(func(mock.Arguments) { evaluatedAt = s.env.Now() })
	s.env.OnActivity(a.EvaluateAccountingInboundActivity, mock.Anything, mock.Anything).
		Return(&ChangesEvaluationResult{}, nil).Once()

	s.env.ExecuteWorkflow(PollAccountingChangesWorkflow, changesPayload())

	s.Require().True(s.env.IsWorkflowCompleted())
	s.Require().NoError(s.env.GetWorkflowError())
	var result ChangesRunResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(2, result.Reads)
	s.Equal(3, result.Recorded)
	s.Equal(2, result.Proposed)
	s.Equal(1, result.Applied)
	s.GreaterOrEqual(evaluatedAt.Sub(readAt), accountingsync.InboundEvaluationSettle,
		"a change is evaluated only once it has settled")
}

func (s *ChangesWorkflowSuite) TestStopsWhenTheConnectionHolds() {
	var a *Activities
	s.env.OnActivity(a.PollAccountingChangesActivity, mock.Anything, mock.Anything).
		Return(&ChangesReadResult{Held: true}, nil).Once()

	s.env.ExecuteWorkflow(PollAccountingChangesWorkflow, changesPayload())

	s.Require().True(s.env.IsWorkflowCompleted())
	var result ChangesRunResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.True(result.Held)
	s.Zero(result.Evaluated)
}

func (s *ChangesWorkflowSuite) TestASignalWhileReadingReadsAgain() {
	var a *Activities
	s.env.OnActivity(a.PollAccountingChangesActivity, mock.Anything, mock.Anything).
		Return(&ChangesReadResult{}, nil).Twice()
	s.env.OnActivity(a.EvaluateAccountingInboundActivity, mock.Anything, mock.Anything).
		Return(&ChangesEvaluationResult{}, nil).Twice()
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(ChangesSignalName, ChangesSignal{})
	}, changesIdleWait/2)

	s.env.ExecuteWorkflow(PollAccountingChangesWorkflow, changesPayload())

	s.Require().True(s.env.IsWorkflowCompleted())
	var result ChangesRunResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(2, result.Reads, "a webhook that arrives during a run costs one more read")
}

func TestChangesWorkflowIDIsPerConnection(t *testing.T) {
	t.Parallel()
	id := pulid.MustNew("acctc_")
	assert.Equal(t, "accounting-changes:"+id.String(), ChangesWorkflowID(id))
}

func TestChangesWorkflowsAreRegistered(t *testing.T) {
	t.Parallel()
	names := map[string]bool{}
	for _, def := range RegisterWorkflows() {
		names[def.Name] = true
	}
	require.True(t, names[PollAccountingChangesWorkflowName])
	require.True(t, names[KickAccountingChangesWorkflowName])
}
