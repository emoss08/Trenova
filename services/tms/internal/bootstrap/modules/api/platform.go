package api

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/entitlementservice"
	"github.com/emoss08/trenova/internal/core/services/usageservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/controlplane"
	"go.uber.org/fx"
)

type PlatformProviderSelectorParams struct {
	fx.In

	Config           *config.Config
	LocalEntitlement *entitlementservice.LocalEntitlementProvider
	NoopUsage        *usageservice.NoopUsageProvider
	CloudEntitlement *controlplane.CloudEntitlementProvider
	CloudUsage       *controlplane.CloudUsageProvider
}

func SelectEntitlementProvider(
	p PlatformProviderSelectorParams,
) (services.EntitlementProvider, error) {
	switch p.Config.Platform.GetMode() {
	case config.PlatformModeCommunity:
		return p.LocalEntitlement, nil
	case config.PlatformModeCloud, config.PlatformModeEnterprise:
		return p.CloudEntitlement, nil
	default:
		return nil, fmt.Errorf("unsupported platform mode %q", p.Config.Platform.GetMode())
	}
}

func SelectUsageProvider(p PlatformProviderSelectorParams) (services.UsageProvider, error) {
	switch p.Config.Platform.GetMode() {
	case config.PlatformModeCommunity:
		return p.NoopUsage, nil
	case config.PlatformModeCloud, config.PlatformModeEnterprise:
		return p.CloudUsage, nil
	default:
		return nil, fmt.Errorf("unsupported platform mode %q", p.Config.Platform.GetMode())
	}
}
