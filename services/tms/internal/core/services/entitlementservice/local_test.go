package entitlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/require"
)

func TestLocalEntitlementProvider_CheckFeature(t *testing.T) {
	t.Parallel()

	registry, err := platformcatalog.NewRegistry(platformcatalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{platformcatalog.NewStaticProvider()},
	})
	require.NoError(t, err)

	provider := NewLocalEntitlementProvider(LocalEntitlementProviderParams{Registry: registry})

	result, err := provider.CheckFeature(context.Background(), &services.FeatureCheckRequest{
		FeatureKey: platformcatalog.FeatureCoreTMS,
	})

	require.NoError(t, err)
	require.True(t, result.Allowed)
	require.Equal(t, "community_mode", result.Reason)
}

func TestLocalEntitlementProvider_CheckFeatureUnknown(t *testing.T) {
	t.Parallel()

	registry, err := platformcatalog.NewRegistry(platformcatalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{platformcatalog.NewStaticProvider()},
	})
	require.NoError(t, err)

	provider := NewLocalEntitlementProvider(LocalEntitlementProviderParams{Registry: registry})

	result, err := provider.CheckFeature(context.Background(), &services.FeatureCheckRequest{
		FeatureKey: platformcatalog.FeatureKey("unknown"),
	})

	require.NoError(t, err)
	require.False(t, result.Allowed)
	require.Equal(t, "feature_not_found", result.Reason)
}
