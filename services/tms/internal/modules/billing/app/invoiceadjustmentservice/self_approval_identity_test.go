package invoiceadjustmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRefuseSelfApprovalRefusesAnActorWithNoUserWhenSeparationIsRequired(t *testing.T) {
	t.Parallel()

	for name, actor := range map[string]*permissionports.RequestActor{
		"no actor": nil,
		"api key": {
			PrincipalType: permissionports.PrincipalTypeAPIKey,
			APIKeyID:      pulid.MustNew("ak_"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := approveAs(t, actor, true, true)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "needs a signed-in user to approve it")
		})
	}
}

func TestRefuseSelfApprovalLetsAnActorWithNoUserThroughWhenNothingNeedsSeparation(t *testing.T) {
	t.Parallel()

	require.NoError(t, approveAs(t, nil, true, false))
	require.NoError(t, approveAs(t, nil, false, true))
}

func approveAs(
	t *testing.T,
	actor *permissionports.RequestActor,
	approvalRequired, separate bool,
) error {
	t.Helper()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	adjustment := &invoiceadjustment.InvoiceAdjustment{
		ID:               pulid.MustNew("iadj_"),
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		Status:           invoiceadjustment.StatusPendingApproval,
		ApprovalRequired: approvalRequired,
		SubmittedByID:    pulid.MustNew("usr_"),
	}
	repo := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(adjustment, nil).Once()
	controls := tenantportsmock.NewMockInvoiceAdjustmentControlRepository(t)
	controls.EXPECT().
		GetByOrgID(mock.Anything, tenantInfo.OrgID).
		Return(&invoiceadjustmentcontrol.InvoiceAdjustmentControl{
			RequireSeparateApprover: separate,
		}, nil).
		Maybe()
	svc := &Service{repo: repo, adjustmentCtrlRepo: controls}

	return svc.refuseSelfApproval(
		t.Context(),
		&invoiceadjustmentports.ApproveInvoiceAdjustmentRequest{
			AdjustmentID: adjustment.ID,
			TenantInfo:   tenantInfo,
		},
		actor,
	)
}
