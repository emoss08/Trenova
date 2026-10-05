package cloud

import (
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/controlplane"
	"github.com/emoss08/trenova/internal/cloud/planbilling"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/editioninfo"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

const EditionName = "cloud"

type EntitlementProviderParams struct {
	fx.In

	Default      services.EntitlementProvider
	Config       *config.Config
	ControlPlane *controlplane.CloudEntitlementProvider
}

func SelectEntitlementProvider(p EntitlementProviderParams) services.EntitlementProvider {
	if cloudconfig.From(p.Config).ControlPlane.Enabled {
		return p.ControlPlane
	}

	return p.Default
}

type BillingProviderParams struct {
	fx.In

	Default      services.BillingProvider
	Config       *config.Config
	LocalPlan    *planbilling.LocalPlanBillingProvider
	ControlPlane *controlplane.CloudBillingProvider
}

func SelectBillingProvider(p BillingProviderParams) services.BillingProvider {
	switch {
	case cloudconfig.From(p.Config).ControlPlane.Enabled:
		return p.ControlPlane
	case p.Config.Platform.IsCloud():
		return p.LocalPlan
	default:
		return p.Default
	}
}

type UsageProviderParams struct {
	fx.In

	Default      services.UsageProvider
	Config       *config.Config
	LocalPlan    *planbilling.LocalPlanUsageProvider
	ControlPlane *controlplane.CloudUsageProvider
}

func SelectUsageProvider(p UsageProviderParams) services.UsageProvider {
	switch {
	case cloudconfig.From(p.Config).ControlPlane.Enabled:
		return p.ControlPlane
	case p.Config.Platform.IsCloud():
		return p.LocalPlan
	default:
		return p.Default
	}
}

type EditionInfoParams struct {
	fx.In

	Default services.EditionInfo
	Config  *config.Config
}

func SelectEditionInfo(p EditionInfoParams) services.EditionInfo {
	return editioninfo.NewStatic(
		EditionName,
		p.Config.Platform.IsCloud() || cloudconfig.From(p.Config).ControlPlane.Enabled,
	)
}
