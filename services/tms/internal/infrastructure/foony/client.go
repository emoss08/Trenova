package foony

import (
	"fmt"

	realtime "github.com/Foony-Limited/realtime-go"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Config *config.Config
}

func New(p Params) (*realtime.Rest, error) {
	cfg := p.Config.GetFoonyConfig()

	client, err := realtime.NewRest(realtime.RestOptions{Key: cfg.APIKey})
	if err != nil {
		return nil, fmt.Errorf("create Foony REST client: %w", err)
	}

	return client, nil
}
