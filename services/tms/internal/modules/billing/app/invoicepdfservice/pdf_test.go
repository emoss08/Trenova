package invoicepdfservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs/billingcontract"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func TestGeneratePDFStartsWorkflow(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	workflowStarter := kernelportsmock.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true).Once()
	workflowStarter.EXPECT().
		StartWorkflow(
			mock.Anything,
			mock.MatchedBy(func(options client.StartWorkflowOptions) bool {
				return strings.HasPrefix(options.ID, "invoice-pdf-generate-"+invoiceID.String()) &&
					options.TaskQueue == temporaltype.TaskQueueBilling.String()
			}),
			billingcontract.GenerateInvoicePDFWorkflowName,
			mock.MatchedBy(func(args []any) bool {
				require.Len(t, args, 1)
				payload, ok := args[0].(*billingcontract.GenerateInvoicePDFPayload)
				return ok &&
					payload.InvoiceID == invoiceID &&
					payload.BaseURL == "https://billing.example.test" &&
					payload.OrganizationID == orgID &&
					payload.BusinessUnitID == buID &&
					payload.UserID == userID
			}),
		).
		Return(invoiceservicetest.WorkflowRun{ID: "wf-invoice-pdf", RunID: "run-1"}, nil).
		Once()

	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: invoiceID, TenantInfo: tenantInfo}).
		Return(&invoice.Invoice{ID: invoiceID, Status: invoice.StatusPosted}, nil).
		Once()

	svc := &Service{
		l:               zap.NewNop(),
		repo:            invoiceRepo,
		workflowStarter: workflowStarter,
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{Logger: zap.NewNop()},
		),
	}

	result, err := svc.GeneratePDF(
		t.Context(),
		&invoiceports.InvoicePreviewRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo,
			BaseURL:    "https://billing.example.test",
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, invoiceID, result.InvoiceID)
	assert.Equal(t, "wf-invoice-pdf", result.WorkflowID)
	assert.Equal(t, "run-1", result.WorkflowRunID)
	assert.Equal(t, "Queued", result.Status)
}

func TestGeneratePDFWorkflowDisabled(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	workflowStarter := kernelportsmock.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(false).Once()

	svc := &Service{
		l:               zap.NewNop(),
		workflowStarter: workflowStarter,
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{Logger: zap.NewNop()},
		),
	}

	result, err := svc.GeneratePDF(
		t.Context(),
		&invoiceports.InvoicePreviewRequest{InvoiceID: invoiceID, TenantInfo: tenantInfo},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.Error(t, err)
	require.Nil(t, result)
	assert.ErrorIs(t, err, kernelports.ErrWorkflowStarterDisabled)
}
