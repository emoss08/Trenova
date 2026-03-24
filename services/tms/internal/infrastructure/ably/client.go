package ably

import (
	"github.com/ably/ably-go/ably"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

func New(p Params) *ably.REST {
	cfg := p.Config.GetAblyConfig()
	log := p.Logger.Named("infrastructure.ably")

	client, err := ably.NewREST(ably.WithKey(cfg.APIKey))
	if err != nil {
		log.Error("failed to create Ably REST client", zap.Error(err))
		return nil
	}

	return client
}
