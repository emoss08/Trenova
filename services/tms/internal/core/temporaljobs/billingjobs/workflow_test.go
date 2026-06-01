package billingjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type BillingWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *BillingWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *BillingWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *BillingWorkflowTestSuite) TestSendInvoiceEmailWorkflow() {
	invoiceID := pulid.MustNew("inv_")
	payload := &SendInvoiceEmailPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			UserID:         pulid.MustNew("usr_"),
			Timestamp:      timeutils.NowUnix(),
		},
		InvoiceID: invoiceID,
		BaseURL:   "https://example.test",
	}
	expected := &SendInvoiceEmailResult{
		InvoiceID:   invoiceID,
		SendStatus:  "Sent",
		Attempts:    2,
		CompletedAt: timeutils.NowUnix(),
	}

	var a *Activities
	s.env.OnActivity(a.SendInvoiceEmailActivity, mock.Anything, payload).Return(expected, nil)

	s.env.ExecuteWorkflow(SendInvoiceEmailWorkflow, payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *SendInvoiceEmailResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(invoiceID, result.InvoiceID)
	s.Equal(2, result.Attempts)
	s.Equal("Sent", result.SendStatus)
}

func (s *BillingWorkflowTestSuite) TestGenerateInvoicePDFWorkflow() {
	invoiceID := pulid.MustNew("inv_")
	sessionID := pulid.MustNew("dus_")
	documentID := pulid.MustNew("doc_")
	payload := &GenerateInvoicePDFPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			UserID:         pulid.MustNew("usr_"),
			Timestamp:      timeutils.NowUnix(),
		},
		InvoiceID: invoiceID,
	}
	prepared := &PrepareInvoicePDFUploadResult{
		InvoiceID: invoiceID,
		SessionID: sessionID,
	}
	finalized := &documentuploadjobs.FinalizeUploadResult{
		SessionID:  sessionID,
		DocumentID: &documentID,
		Status:     "Available",
	}
	expected := &GenerateInvoicePDFResult{
		InvoiceID:   invoiceID,
		DocumentID:  documentID,
		CompletedAt: timeutils.NowUnix(),
	}

	var a *Activities
	s.env.OnActivity(a.PrepareInvoicePDFUploadActivity, mock.Anything, payload).
		Return(prepared, nil)
	s.env.OnWorkflow(
		documentuploadjobs.FinalizeDocumentUploadWorkflow,
		mock.Anything,
		mock.MatchedBy(func(childPayload *documentuploadjobs.FinalizeUploadPayload) bool {
			return childPayload.SessionID == sessionID
		}),
	).Return(finalized, nil)
	s.env.OnActivity(
		a.CompleteInvoicePDFGenerationActivity,
		mock.Anything,
		payload,
		documentID,
	).Return(expected, nil)

	s.env.ExecuteWorkflow(GenerateInvoicePDFWorkflow, payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *GenerateInvoicePDFResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(invoiceID, result.InvoiceID)
	s.Equal(documentID, result.DocumentID)
}

func TestBillingWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(BillingWorkflowTestSuite))
}
