package foony

import (
	realtime "github.com/Foony-Limited/realtime-go"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

func New(p Params) *realtime.Rest {
	cfg := p.Config.GetFoonyConfig()
	log := p.Logger.Named("infrastructure.foony")

	client, err := realtime.NewRest(realtime.RestOptions{Key: cfg.APIKey})
	if err != nil {
		log.Error("failed to create Foony REST client", zap.Error(err))
		return nil
	}

	return client
}
