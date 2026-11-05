package entitlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalEntitlementProvider_CheckFeature(t *testing.T) {
	t.Parallel()

	result, err := NewLocalEntitlementProvider().CheckFeature(
		t.Context(),
		&platformcatalogports.FeatureCheckRequest{FeatureKey: platformcatalog.FeatureCoreTMS},
	)

	require.NoError(t, err)
	require.True(t, result.Allowed)
	require.Equal(t, reasonCommunityMode, result.Reason)
	require.NotZero(t, result.CheckedAt)
}

func TestLocalEntitlementProvider_AllowsEverything(t *testing.T) {
	t.Parallel()

	provider := NewLocalEntitlementProvider()

	result, err := provider.CheckFeature(t.Context(), &platformcatalogports.FeatureCheckRequest{
		FeatureKey: platformcatalog.FeatureKey("anything"),
		CheckedAt:  42,
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, reasonCommunityMode, result.Reason)
	assert.Equal(t, int64(42), result.CheckedAt)

	list, err := provider.ListEntitlements(t.Context(), &platformcatalogports.EntitlementsRequest{})
	require.NoError(t, err)
	assert.Empty(t, list.Features)
	assert.NotZero(t, list.CheckedAt)
}
