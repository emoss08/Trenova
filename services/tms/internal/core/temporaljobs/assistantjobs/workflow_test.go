package assistantjobs

import (
	"errors"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type AssistantTurnWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env     *testsuite.TestWorkflowEnvironment
	payload *AssistantTurnPayload
}

func (s *AssistantTurnWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.payload = &AssistantTurnPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		TurnID:   pulid.MustNew("atrn_"),
		ThreadID: pulid.MustNew("athr_"),
		Content:  "How many loads are late?",
		Actor: serviceports.RequestActor{
			PrincipalType: serviceports.PrincipalTypeUser,
			UserID:        pulid.MustNew("usr_"),
		},
	}
}

func (s *AssistantTurnWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *AssistantTurnWorkflowTestSuite) TestAnswersTheQuestion() {
	var a *Activities
	s.env.OnActivity(a.AssistantTurnActivity, mock.Anything, mock.Anything).
		Return(&AssistantTurnResult{Status: "Completed"}, nil).Once()

	s.env.ExecuteWorkflow(AssistantTurnWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result AssistantTurnResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal("Completed", result.Status)
}

// A refusal is an answer. Retrying one would ask the model to decline the same
// question twice and charge for the privilege.
func (s *AssistantTurnWorkflowTestSuite) TestDoesNotRetryARefusal() {
	var a *Activities
	s.env.OnActivity(a.AssistantTurnActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"out of scope", errTypeRefused, nil,
		)).Once()

	s.env.ExecuteWorkflow(AssistantTurnWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}

// A turn that arrives without the person who asked for it must fail rather
// than run as something nobody granted.
func (s *AssistantTurnWorkflowTestSuite) TestDoesNotRetryAMissingActor() {
	var a *Activities
	s.env.OnActivity(a.AssistantTurnActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"no actor", errTypeNoActor, nil,
		)).Once()

	s.env.ExecuteWorkflow(AssistantTurnWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}

// A fault before the turn began is worth one more go: nothing was said, and
// nothing was charged for.
func (s *AssistantTurnWorkflowTestSuite) TestRetriesATransientFaultOnce() {
	var a *Activities
	s.env.OnActivity(a.AssistantTurnActivity, mock.Anything, mock.Anything).
		Return(nil, errors.New("connection refused")).Once()
	s.env.OnActivity(a.AssistantTurnActivity, mock.Anything, mock.Anything).
		Return(&AssistantTurnResult{Status: "Completed"}, nil).Once()

	s.env.ExecuteWorkflow(AssistantTurnWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *AssistantTurnWorkflowTestSuite) TestRunsOnTheChatQueue() {
	definitions := RegisterWorkflows()
	s.Require().Len(definitions, 1)
	s.Equal(
		temporaltype.TaskQueueAgentChat.String(),
		definitions[0].TaskQueue,
		"a person watching a reply must not queue behind a scheduled run",
	)
}

func TestAssistantTurnWorkflowSuite(t *testing.T) {
	suite.Run(t, new(AssistantTurnWorkflowTestSuite))
}
