package invoiceadjustmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRefuseSelfApproval(t *testing.T) {
	t.Parallel()

	submitter := pulid.MustNew("usr_")
	tests := []struct {
		name             string
		approver         pulid.ID
		approvalRequired bool
		separate         bool
		refused          bool
		readsControl     bool
	}{
		{
			name:             "submitter approving with a separate approver required",
			approver:         submitter,
			approvalRequired: true,
			separate:         true,
			refused:          true,
			readsControl:     true,
		},
		{
			name:             "submitter approving where one person may do both",
			approver:         submitter,
			approvalRequired: true,
			separate:         false,
			readsControl:     true,
		},
		{
			name:             "another person approving",
			approver:         pulid.MustNew("usr_"),
			approvalRequired: true,
			separate:         true,
		},
		{
			name:             "submitter retrying one that never needed approval",
			approver:         submitter,
			approvalRequired: false,
			separate:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tenantInfo := pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			}
			adjustment := &invoiceadjustment.InvoiceAdjustment{
				ID:               pulid.MustNew("iadj_"),
				OrganizationID:   tenantInfo.OrgID,
				BusinessUnitID:   tenantInfo.BuID,
				Status:           invoiceadjustment.StatusPendingApproval,
				ApprovalRequired: tt.approvalRequired,
				SubmittedByID:    submitter,
			}
			repo := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentRepository(t)
			repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(adjustment, nil).Once()
			controls := tenantportsmock.NewMockInvoiceAdjustmentControlRepository(t)
			if tt.readsControl {
				controls.EXPECT().
					GetByOrgID(mock.Anything, tenantInfo.OrgID).
					Return(&invoiceadjustmentcontrol.InvoiceAdjustmentControl{RequireSeparateApprover: tt.separate}, nil).
					Once()
			}
			svc := &Service{repo: repo, adjustmentCtrlRepo: controls}

			err := svc.refuseSelfApproval(
				t.Context(),
				&invoiceadjustmentports.ApproveInvoiceAdjustmentRequest{
					AdjustmentID: adjustment.ID,
					TenantInfo:   tenantInfo,
				},
				testutil.NewSessionActor(tt.approver, tenantInfo.OrgID, tenantInfo.BuID),
			)

			if tt.refused {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "someone else must approve it")
				return
			}
			require.NoError(t, err)
		})
	}
}
