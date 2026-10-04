package api

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/entitlementservice"
	"github.com/emoss08/trenova/internal/core/services/platformbillingservice"
	"github.com/emoss08/trenova/internal/core/services/usageservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/controlplane"
	"github.com/stretchr/testify/require"
)

func selectorParams(mode config.PlatformMode, controlPlane bool) PlatformProviderSelectorParams {
	return PlatformProviderSelectorParams{
		Config: &config.Config{
			Platform: config.PlatformConfig{
				Mode:         mode,
				ControlPlane: config.PlatformControlPlaneConfig{Enabled: controlPlane},
			},
		},
		LocalEntitlement: &entitlementservice.LocalEntitlementProvider{},
		LocalBilling:     &platformbillingservice.LocalBillingProvider{},
		LocalPlanBilling: &platformbillingservice.LocalPlanBillingProvider{},
		NoopUsage:        &usageservice.NoopUsageProvider{},
		LocalPlanUsage:   &usageservice.LocalPlanUsageProvider{},
		CloudEntitlement: &controlplane.CloudEntitlementProvider{},
		CloudBilling:     &controlplane.CloudBillingProvider{},
		CloudUsage:       &controlplane.CloudUsageProvider{},
	}
}

func TestSelectEntitlementProvider_UsesControlPlaneWhenEnabled(t *testing.T) {
	t.Parallel()

	p := selectorParams(config.PlatformModeSelfHosted, true)
	provider, err := SelectEntitlementProvider(p)

	require.NoError(t, err)
	require.Same(t, p.CloudEntitlement, provider)
}

func TestSelectEntitlementProvider_StaysLocalInCloudModeWithoutControlPlane(t *testing.T) {
	t.Parallel()

	p := selectorParams(config.PlatformModeCloud, false)
	provider, err := SelectEntitlementProvider(p)

	require.NoError(t, err)
	require.Same(t, p.LocalEntitlement, provider)
}

func TestSelectUsageProvider(t *testing.T) {
	t.Parallel()

	selfHosted := selectorParams(config.PlatformModeSelfHosted, false)
	provider, err := SelectUsageProvider(selfHosted)
	require.NoError(t, err)
	require.Same(t, selfHosted.NoopUsage, provider)

	cloud := selectorParams(config.PlatformModeCloud, false)
	provider, err = SelectUsageProvider(cloud)
	require.NoError(t, err)
	require.Same(t, cloud.LocalPlanUsage, provider)

	controlPlane := selectorParams(config.PlatformModeCloud, true)
	provider, err = SelectUsageProvider(controlPlane)
	require.NoError(t, err)
	require.Same(t, controlPlane.CloudUsage, provider)
}

func TestSelectBillingProvider(t *testing.T) {
	t.Parallel()

	selfHosted := selectorParams(config.PlatformModeSelfHosted, false)
	provider, err := SelectBillingProvider(selfHosted)
	require.NoError(t, err)
	require.Same(t, selfHosted.LocalBilling, provider)

	cloud := selectorParams(config.PlatformModeCloud, false)
	provider, err = SelectBillingProvider(cloud)
	require.NoError(t, err)
	require.Same(t, cloud.LocalPlanBilling, provider)

	controlPlane := selectorParams(config.PlatformModeDevelopment, true)
	provider, err = SelectBillingProvider(controlPlane)
	require.NoError(t, err)
	require.Same(t, controlPlane.CloudBilling, provider)
}
