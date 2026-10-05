package cloud_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/cloud"
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/controlplane"
	"github.com/emoss08/trenova/internal/cloud/planbilling"
	"github.com/emoss08/trenova/internal/core/services/editioninfo"
	"github.com/emoss08/trenova/internal/core/services/entitlementservice"
	"github.com/emoss08/trenova/internal/core/services/platformbillingservice"
	"github.com/emoss08/trenova/internal/core/services/usageservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func platformConfig(mode config.PlatformMode, controlPlane bool) *config.Config {
	cfg := &config.Config{Platform: config.PlatformConfig{Mode: mode}}
	cloudconfig.Attach(cfg, &cloudconfig.Settings{
		ControlPlane: cloudconfig.PlatformControlPlaneConfig{Enabled: controlPlane},
	})

	return cfg
}

func TestSelectEntitlementProvider(t *testing.T) {
	t.Parallel()

	local := &entitlementservice.LocalEntitlementProvider{}
	remote := &controlplane.CloudEntitlementProvider{}

	tests := []struct {
		name         string
		mode         config.PlatformMode
		controlPlane bool
		wantRemote   bool
	}{
		{name: "control plane on a self-hosted instance", mode: config.PlatformModeSelfHosted, controlPlane: true, wantRemote: true},
		{name: "cloud without the control plane", mode: config.PlatformModeCloud},
		{name: "self-hosted", mode: config.PlatformModeSelfHosted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := cloud.SelectEntitlementProvider(cloud.EntitlementProviderParams{
				Default:      local,
				Config:       platformConfig(tt.mode, tt.controlPlane),
				ControlPlane: remote,
			})
			if tt.wantRemote {
				require.Same(t, remote, provider)
				return
			}
			require.Same(t, local, provider)
		})
	}
}

func TestSelectUsageProvider(t *testing.T) {
	t.Parallel()

	noop := usageservice.NewNoopUsageProvider()
	localPlan := &planbilling.LocalPlanUsageProvider{}
	remote := &controlplane.CloudUsageProvider{}
	params := func(mode config.PlatformMode, controlPlane bool) cloud.UsageProviderParams {
		return cloud.UsageProviderParams{
			Default:      noop,
			Config:       platformConfig(mode, controlPlane),
			LocalPlan:    localPlan,
			ControlPlane: remote,
		}
	}

	require.Same(t, noop, cloud.SelectUsageProvider(params(config.PlatformModeSelfHosted, false)))
	require.Same(t, localPlan, cloud.SelectUsageProvider(params(config.PlatformModeCloud, false)))
	require.Same(t, remote, cloud.SelectUsageProvider(params(config.PlatformModeCloud, true)))
}

func TestSelectBillingProvider(t *testing.T) {
	t.Parallel()

	local := &platformbillingservice.LocalBillingProvider{}
	localPlan := &planbilling.LocalPlanBillingProvider{}
	remote := &controlplane.CloudBillingProvider{}
	params := func(mode config.PlatformMode, controlPlane bool) cloud.BillingProviderParams {
		return cloud.BillingProviderParams{
			Default:      local,
			Config:       platformConfig(mode, controlPlane),
			LocalPlan:    localPlan,
			ControlPlane: remote,
		}
	}

	require.Same(t, local, cloud.SelectBillingProvider(params(config.PlatformModeSelfHosted, false)))
	require.Same(t, localPlan, cloud.SelectBillingProvider(params(config.PlatformModeCloud, false)))
	require.Same(t, remote, cloud.SelectBillingProvider(params(config.PlatformModeDevelopment, true)))
}

func TestSelectEditionInfo(t *testing.T) {
	t.Parallel()

	selfHosted := cloud.SelectEditionInfo(cloud.EditionInfoParams{
		Default: editioninfo.SelfHosted(),
		Config:  platformConfig(config.PlatformModeSelfHosted, false),
	})
	assert.Equal(t, cloud.EditionName, selfHosted.Name())
	assert.False(t, selfHosted.SharedTenancy())

	hosted := cloud.SelectEditionInfo(cloud.EditionInfoParams{
		Default: editioninfo.SelfHosted(),
		Config:  platformConfig(config.PlatformModeCloud, false),
	})
	assert.True(t, hosted.SharedTenancy())

	controlled := cloud.SelectEditionInfo(cloud.EditionInfoParams{
		Default: editioninfo.SelfHosted(),
		Config:  platformConfig(config.PlatformModeSelfHosted, true),
	})
	assert.True(t, controlled.SharedTenancy())
}
