package googlemaps

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"googlemaps.github.io/maps"
)

type ClientParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

func NewClient(p ClientParams) *maps.Client {
	client, err := maps.NewClient(maps.WithAPIKey(p.Config.Google.APIKey))
	if err != nil {
		p.Logger.Error("failed to create Google Maps client", zap.Error(err))
		return nil
	}

	return client
}
