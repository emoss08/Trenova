package billingcontrolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestQueueAutoApproveRule_RefusesWaitingOnNovaWithoutItsReview(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mode    billingcontrol.QueueAutoApproveMode
		gate    *plantest.Gate
		refused bool
	}{
		{
			name:    "without agent automation",
			mode:    billingcontrol.QueueAutoApproveModeWhenAgentAgrees,
			gate:    plantest.Without(platformcatalog.FeatureAgentAutomation),
			refused: true,
		},
		{
			name:    "without the Desk",
			mode:    billingcontrol.QueueAutoApproveModeWhenAgentAgrees,
			gate:    plantest.Without(platformcatalog.FeatureAIDesk),
			refused: true,
		},
		{
			name: "with Nova's review on the plan",
			mode: billingcontrol.QueueAutoApproveModeWhenAgentAgrees,
			gate: plantest.AllFeatures(),
		},
		{
			name: "clear checks need no review",
			mode: billingcontrol.QueueAutoApproveModeWhenChecksClear,
			gate: plantest.Without(
				platformcatalog.FeatureAgentAutomation,
				platformcatalog.FeatureAIDesk,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := validBillingControl()
			entity.QueueAutoApproveMode = tt.mode

			result := validationtest.RunRule(
				t,
				createQueueAutoApproveRule(tt.gate),
				entity,
				validationtest.Update(),
			)

			if !tt.refused {
				result.Exactly()
				return
			}
			result.Exactly(validationtest.At("queueAutoApproveMode", errortypes.ErrInvalid))
			assert.Equal(
				t,
				[]string{
					"Your plan does not include Nova's review of the billing queue, so items cannot wait for Nova to agree",
				},
				result.Messages("queueAutoApproveMode"),
			)
		})
	}
}

func TestGet_ReadsTheApprovalModeAutomationUses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		gate *plantest.Gate
		want billingcontrol.QueueAutoApproveMode
	}{
		{
			name: "Nova's review on the plan",
			gate: plantest.AllFeatures(),
			want: billingcontrol.QueueAutoApproveModeWhenAgentAgrees,
		},
		{
			name: "Nova's review off the plan",
			gate: plantest.Without(platformcatalog.FeatureAgentAutomation),
			want: billingcontrol.QueueAutoApproveModeOff,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stored := validBillingControl()
			stored.QueueAutoApproveMode = billingcontrol.QueueAutoApproveModeWhenAgentAgrees
			repo := tenantportsmock.NewMockBillingControlRepository(t)
			repo.EXPECT().
				GetByOrgID(mock.Anything, stored.OrganizationID).
				Return(stored, nil).
				Once()
			svc := New(Params{Logger: zap.NewNop(), Repo: repo, Features: tt.gate})

			got, err := svc.Get(t.Context(), tenantports.GetBillingControlRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: stored.OrganizationID,
					BuID:  stored.BusinessUnitID,
				},
			})

			require.NoError(t, err)
			assert.Equal(t, tt.want, got.QueueAutoApproveMode)
		})
	}
}
