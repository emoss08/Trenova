package temporaljobs

import (
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type TemporalClientParams struct {
	fx.In

	Config *config.Manager
	Logger *logger.Logger
}

func NewTemporalClient(p TemporalClientParams) client.Client {
	cfg := p.Config.Temporal()
	log := p.Logger.With().
		Str("component", "temporal-client").
		Logger()

	c, err := client.Dial(client.Options{
		HostPort: cfg.HostPort,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial temporal client")
	}

	log.Info().Msg("temporal client dialed successfully")

	return c
}
