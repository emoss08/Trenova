package invoiceadjustmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentengine"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPreviewDecisionRefusesAnAdjustmentNotPendingApproval(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	rejected := &invoiceadjustment.InvoiceAdjustment{
		ID:     pulid.MustNew("iadj_"),
		Status: invoiceadjustment.StatusRejected,
	}
	repo := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceadjustmentports.GetInvoiceAdjustmentRequest{ID: rejected.ID, TenantInfo: tenantInfo}).
		Return(rejected, nil).
		Once()
	documents := documentportsmock.NewMockDocumentRepository(t)
	documents.EXPECT().GetByResourceID(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	svc := &Service{engine: invoiceadjustmentengine.New(invoiceadjustmentengine.Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		DocumentRepo: documents,
	})}

	_, err := svc.PreviewDecision(
		t.Context(),
		&invoiceadjustmentports.InvoiceAdjustmentDecisionRequest{
			AdjustmentID: rejected.ID,
			Approve:      true,
			TenantInfo:   tenantInfo,
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "awaiting approval or whose execution failed")
}
