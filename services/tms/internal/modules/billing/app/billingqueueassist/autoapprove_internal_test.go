package billingqueueassist

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAutoApprove_WaitingOnNovaIsOffWithoutItsReviewOnThePlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		gate   *plantest.Gate
		reads  int
		reason string
	}{
		{
			name:   "off the plan, nothing is approved on a stale verdict",
			gate:   plantest.Without(platformcatalog.FeatureAgentAutomation),
			reason: "Auto-approve is off",
		},
		{
			name:   "on the plan, the item is checked",
			gate:   plantest.AllFeatures(),
			reads:  1,
			reason: "Not ready",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tenantInfo := pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			}
			controls := tenantportsmock.NewMockBillingControlRepository(t)
			controls.EXPECT().GetByOrgID(mock.Anything, tenantInfo.OrgID).
				Return(&billingcontrol.BillingControl{
					QueueAutoApproveMode: billingcontrol.QueueAutoApproveModeWhenAgentAgrees,
				}, nil).
				Once()
			queue := billingqueueportsmock.NewMockBillingQueueService(t)
			queue.EXPECT().GetByID(mock.Anything, mock.Anything).
				Return(&billingqueue.BillingQueueItem{}, nil).
				Maybe()
			s := &Service{l: zap.NewNop(), controls: controls, queue: queue, features: tt.gate}

			outcome, err := s.AutoApprove(t.Context(), tenantInfo, pulid.MustNew("bqi_"))

			require.NoError(t, err)
			assert.False(t, outcome.Approved)
			assert.Equal(t, tt.reason, outcome.Reason)
			queue.AssertNumberOfCalls(t, "GetByID", tt.reads)
		})
	}
}
