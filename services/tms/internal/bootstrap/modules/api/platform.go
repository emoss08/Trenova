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
	LocalPlanBilling *platformbillingservice.LocalPlanBillingProvider
	NoopUsage        *usageservice.NoopUsageProvider
	LocalPlanUsage   *usageservice.LocalPlanUsageProvider
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
	switch {
	case p.Config.Platform.ControlPlane.Enabled:
		return p.CloudBilling, nil
	case p.Config.Platform.IsCloud():
		return p.LocalPlanBilling, nil
	default:
		return p.LocalBilling, nil
	}
}

func SelectUsageProvider(p PlatformProviderSelectorParams) (services.UsageProvider, error) {
	switch {
	case p.Config.Platform.ControlPlane.Enabled:
		return p.CloudUsage, nil
	case p.Config.Platform.IsCloud():
		return p.LocalPlanUsage, nil
	default:
		return p.NoopUsage, nil
	}
}
