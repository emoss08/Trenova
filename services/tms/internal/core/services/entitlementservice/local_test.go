package entitlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCatalog struct {
	features []platformcatalog.Feature
}

func (c fakeCatalog) GetFeature(key platformcatalog.FeatureKey) (platformcatalog.Feature, bool) {
	for _, feature := range c.features {
		if feature.Key == key {
			return feature, true
		}
	}

	return platformcatalog.Feature{}, false
}

func (c fakeCatalog) ListFeatures() []platformcatalog.Feature {
	return c.features
}

func (c fakeCatalog) ListMeters() []platformcatalog.Meter {
	return nil
}

func withCatalog() *LocalEntitlementProvider {
	return NewLocalEntitlementProvider(LocalEntitlementProviderParams{
		Catalog: fakeCatalog{features: []platformcatalog.Feature{
			{Key: platformcatalog.FeatureCoreTMS},
			{Key: platformcatalog.FeatureDispatch},
		}},
	})
}

func TestLocalEntitlementProvider_CheckFeature(t *testing.T) {
	t.Parallel()

	result, err := withCatalog().CheckFeature(t.Context(), &services.FeatureCheckRequest{
		FeatureKey: platformcatalog.FeatureCoreTMS,
	})

	require.NoError(t, err)
	require.True(t, result.Allowed)
	require.Equal(t, reasonCommunityMode, result.Reason)
	require.NotZero(t, result.CheckedAt)
}

func TestLocalEntitlementProvider_CheckFeatureUnknown(t *testing.T) {
	t.Parallel()

	result, err := withCatalog().CheckFeature(t.Context(), &services.FeatureCheckRequest{
		FeatureKey: platformcatalog.FeatureKey("unknown"),
	})

	require.NoError(t, err)
	require.False(t, result.Allowed)
	require.Equal(t, reasonFeatureNotFound, result.Reason)
}

func TestLocalEntitlementProvider_WithoutACatalogAllowsEverything(t *testing.T) {
	t.Parallel()

	provider := NewLocalEntitlementProvider(LocalEntitlementProviderParams{})

	result, err := provider.CheckFeature(t.Context(), &services.FeatureCheckRequest{
		FeatureKey: platformcatalog.FeatureKey("anything"),
		CheckedAt:  42,
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, reasonCommunityMode, result.Reason)
	assert.Equal(t, int64(42), result.CheckedAt)

	list, err := provider.ListEntitlements(t.Context(), &services.EntitlementsRequest{})
	require.NoError(t, err)
	assert.Empty(t, list.Features)
	assert.NotZero(t, list.CheckedAt)
}

func TestLocalEntitlementProvider_ListsEveryCatalogFeatureAsAllowed(t *testing.T) {
	t.Parallel()

	list, err := withCatalog().ListEntitlements(t.Context(), &services.EntitlementsRequest{})
	require.NoError(t, err)
	require.Len(t, list.Features, 2)
	for _, feature := range list.Features {
		assert.True(t, feature.Allowed)
		assert.Equal(t, reasonCommunityMode, feature.Reason)
	}
}
