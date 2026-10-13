package billingjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs/billingcontract"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type AutoPostWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env     *testsuite.TestWorkflowEnvironment
	payload *billingcontract.AutoPostInvoicePayload
}

func (s *AutoPostWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.payload = &billingcontract.AutoPostInvoicePayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			UserID:         pulid.MustNew("usr_"),
			Timestamp:      timeutils.NowUnix(),
		},
		InvoiceID: pulid.MustNew("inv_"),
	}
}

func (s *AutoPostWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *AutoPostWorkflowTestSuite) TestPostedInvoiceReportsNothing() {
	var a *Activities
	s.env.OnActivity(a.AutoPostInvoiceActivity, mock.Anything, s.payload).
		Return(&AutoPostInvoiceResult{InvoiceID: s.payload.InvoiceID}, nil).
		Once()

	s.env.ExecuteWorkflow(AutoPostInvoiceWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *AutoPostWorkflowTestSuite) TestRefusedPostIsNotRetriedAndIsReported() {
	refusal := temporaltype.ToDomainRefusal(
		errortypes.NewBusinessError("The fiscal period for this invoice date is closed"),
	)

	var a *Activities
	s.env.OnActivity(a.AutoPostInvoiceActivity, mock.Anything, s.payload).
		Return(nil, refusal).
		Once()
	s.env.OnActivity(a.ReportAutoPostFailureActivity, mock.Anything,
		&billingcontract.AutoPostFailurePayload{
			AutoPostInvoicePayload: *s.payload,
			Reason:                 "The fiscal period for this invoice date is closed",
		}).
		Return(nil).
		Once()

	s.env.ExecuteWorkflow(AutoPostInvoiceWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}

func (s *AutoPostWorkflowTestSuite) TestOutageIsRetriedBeforeItIsReported() {
	var a *Activities
	s.env.OnActivity(a.AutoPostInvoiceActivity, mock.Anything, s.payload).
		Return(nil, errors.New("connection reset by peer")).
		Times(int(autoPostInvoiceRetryPolicy.MaximumAttempts))
	s.env.OnActivity(a.ReportAutoPostFailureActivity, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	s.env.ExecuteWorkflow(AutoPostInvoiceWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}

func (s *AutoPostWorkflowTestSuite) TestAnUndeliveredReportDoesNotChangeTheOutcome() {
	var a *Activities
	s.env.OnActivity(a.AutoPostInvoiceActivity, mock.Anything, s.payload).
		Return(nil, temporaltype.ToDomainRefusal(errortypes.NewNotFoundError("Invoice not found"))).
		Once()
	s.env.OnActivity(a.ReportAutoPostFailureActivity, mock.Anything, mock.Anything).
		Return(errors.New("notifications unavailable"))

	s.env.ExecuteWorkflow(AutoPostInvoiceWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "Invoice not found")
}

func (s *AutoPostWorkflowTestSuite) TestAnExecutionFromBeforeTheChangeReportsNothing() {
	s.env.OnGetVersion(changeReportAutoPostFailure, workflow.DefaultVersion, 1).
		Return(workflow.DefaultVersion)

	var a *Activities
	s.env.OnActivity(a.AutoPostInvoiceActivity, mock.Anything, s.payload).
		Return(nil, temporaltype.ToDomainRefusal(errortypes.NewBusinessError("closed"))).
		Once()

	s.env.ExecuteWorkflow(AutoPostInvoiceWorkflow, s.payload)

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}

func TestAutoPostWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(AutoPostWorkflowTestSuite))
}
