package platformbillingservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCatalog struct{}

func (fakeCatalog) GetFeature(platformcatalog.FeatureKey) (platformcatalog.Feature, bool) {
	return platformcatalog.Feature{}, false
}

func (fakeCatalog) ListFeatures() []platformcatalog.Feature {
	return []platformcatalog.Feature{{Key: platformcatalog.FeatureCoreTMS}}
}

func (fakeCatalog) ListMeters() []platformcatalog.Meter {
	return []platformcatalog.Meter{{Key: platformcatalog.MeterAPIRequests, Unit: "request"}}
}

func TestLocalBillingProvider_WithoutACatalogIsAnActiveCommunityPlan(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	summary, err := NewLocalBillingProvider(LocalBillingProviderParams{}).GetBillingSummary(
		t.Context(),
		&services.BillingSummaryRequest{OrganizationID: orgID, BusinessUnitID: buID, CheckedAt: 7},
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

func TestLocalBillingProvider_ReportsEveryCatalogFeatureAndMeter(t *testing.T) {
	t.Parallel()

	summary, err := NewLocalBillingProvider(LocalBillingProviderParams{Catalog: fakeCatalog{}}).
		GetBillingSummary(t.Context(), &services.BillingSummaryRequest{})
	require.NoError(t, err)

	require.Len(t, summary.Features, 1)
	assert.True(t, summary.Features[0].Allowed)
	require.Len(t, summary.Usage, 1)
	assert.Equal(t, "request", summary.Usage[0].Unit)
	assert.NotZero(t, summary.CheckedAt)
}
