package api

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/entitlementservice"
	"github.com/emoss08/trenova/internal/core/services/usageservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/controlplane"
	"github.com/stretchr/testify/require"
)

func TestSelectEntitlementProvider_UsesControlPlaneWhenEnabled(t *testing.T) {
	t.Parallel()

	local := &entitlementservice.LocalEntitlementProvider{}
	cloud := &controlplane.CloudEntitlementProvider{}

	provider, err := SelectEntitlementProvider(PlatformProviderSelectorParams{
		Config: &config.Config{
			Platform: config.PlatformConfig{
				Mode: config.PlatformModeSelfHosted,
				ControlPlane: config.PlatformControlPlaneConfig{
					Enabled: true,
				},
			},
		},
		LocalEntitlement: local,
		CloudEntitlement: cloud,
	})

	require.NoError(t, err)
	require.Same(t, cloud, provider)
}

func TestSelectUsageProvider_UsesNoopWhenControlPlaneDisabled(t *testing.T) {
	t.Parallel()

	noop := &usageservice.NoopUsageProvider{}
	cloud := &controlplane.CloudUsageProvider{}

	provider, err := SelectUsageProvider(PlatformProviderSelectorParams{
		Config: &config.Config{
			Platform: config.PlatformConfig{
				Mode: config.PlatformModeCloud,
				ControlPlane: config.PlatformControlPlaneConfig{
					Enabled: false,
				},
			},
		},
		NoopUsage:  noop,
		CloudUsage: cloud,
	})

	require.NoError(t, err)
	require.Same(t, noop, provider)
}
