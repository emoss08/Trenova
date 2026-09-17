package carrierintel

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/carrierokconnector"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/fmcsaconnector"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/outboundlimit"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

var Module = fx.Module("carrierintel",
	fx.Provide(
		fx.Annotate(outboundlimit.New, fx.ResultTags(`name:"carrierIntelLimiter"`)),
		fx.Annotate(newCarrierOK, fx.ResultTags(`group:"carrierIntelConnectors"`)),
		fx.Annotate(newFMCSA, fx.ResultTags(`group:"carrierIntelConnectors"`)),
	),
)

func newCarrierOK(cfg *config.Config) services.CarrierIntelConnector {
	return carrierokconnector.New(cfg)
}

func newFMCSA(cfg *config.Config) services.CarrierIntelConnector {
	return fmcsaconnector.New(cfg)
}
