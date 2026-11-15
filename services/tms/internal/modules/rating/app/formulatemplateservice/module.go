package formulatemplateservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/fx"
)

func asPricingTemplatePlan(s *Service) permissionports.PricingTemplatePlan {
	return s
}

var Module = fx.Module("formulatemplateservice",
	fx.Provide(New, asPricingTemplatePlan),
)
