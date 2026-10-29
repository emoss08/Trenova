package carrierintel

import (
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/carrierokconnector"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/fmcsaconnector"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/outboundlimit"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/carrierintelligenceconfig"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"go.uber.org/fx"
)

var Module = fx.Module("carrierintel",
	fx.Provide(
		fx.Annotate(outboundlimit.New, fx.ResultTags(`name:"carrierIntelLimiter"`)),
		fx.Annotate(newCarrierOK, fx.ResultTags(`group:"carrierIntelConnectors"`)),
		fx.Annotate(newFMCSA, fx.ResultTags(`group:"carrierIntelConnectors"`)),
	),
)

func newCarrierOK(
	appConfig *appconfig.Config,
	carrierIntelligenceConfig *carrierintelligenceconfig.Config,
) carrierintelports.CarrierIntelConnector {
	return carrierokconnector.New(appConfig, carrierIntelligenceConfig)
}

func newFMCSA(
	appConfig *appconfig.Config,
	carrierIntelligenceConfig *carrierintelligenceconfig.Config,
) carrierintelports.CarrierIntelConnector {
	return fmcsaconnector.New(appConfig, carrierIntelligenceConfig)
}
