//revive:disable-next-line:var-naming
package api

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/entitlementservice"
	"github.com/emoss08/trenova/internal/core/services/platformbillingservice"
	"github.com/emoss08/trenova/internal/core/services/usageservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/controlplane"
	"go.uber.org/fx"
)

type PlatformProviderSelectorParams struct {
	fx.In

	Config           *config.Config
	LocalEntitlement *entitlementservice.LocalEntitlementProvider
	LocalBilling     *platformbillingservice.LocalBillingProvider
	NoopUsage        *usageservice.NoopUsageProvider
	CloudEntitlement *controlplane.CloudEntitlementProvider
	CloudBilling     *controlplane.CloudBillingProvider
	CloudUsage       *controlplane.CloudUsageProvider
}

func SelectEntitlementProvider(
	p PlatformProviderSelectorParams,
) (services.EntitlementProvider, error) {
	if p.Config.Platform.ControlPlane.Enabled {
		return p.CloudEntitlement, nil
	}

	return p.LocalEntitlement, nil
}

func SelectBillingProvider(p PlatformProviderSelectorParams) (services.BillingProvider, error) {
	if p.Config.Platform.ControlPlane.Enabled {
		return p.CloudBilling, nil
	}

	return p.LocalBilling, nil
}

func SelectUsageProvider(p PlatformProviderSelectorParams) (services.UsageProvider, error) {
	if p.Config.Platform.ControlPlane.Enabled {
		return p.CloudUsage, nil
	}

	return p.NoopUsage, nil
}
