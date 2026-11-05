package platformbillingservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalBillingProvider_IsAnActiveCommunityPlan(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	summary, err := NewLocalBillingProvider().GetBillingSummary(
		t.Context(),
		&platformcatalogports.BillingSummaryRequest{OrganizationID: orgID, BusinessUnitID: buID, CheckedAt: 7},
	)
	require.NoError(t, err)

	assert.True(t, summary.Active)
	assert.Equal(t, communityReason, summary.Reason)
	assert.Equal(t, orgID, summary.OrganizationID)
	assert.Equal(t, buID, summary.BusinessUnitID)
	assert.Equal(t, communityPlanID, summary.Plan.Key)
	assert.Equal(t, activeStatus, summary.Subscription.Status)
	assert.Empty(t, summary.Features)
	assert.Empty(t, summary.Usage)
	assert.Equal(t, int64(7), summary.CheckedAt)
}

func TestLocalBillingProvider_StampsTheCheckTime(t *testing.T) {
	t.Parallel()

	summary, err := NewLocalBillingProvider().
		GetBillingSummary(t.Context(), &platformcatalogports.BillingSummaryRequest{})
	require.NoError(t, err)

	assert.NotZero(t, summary.CheckedAt)
}
