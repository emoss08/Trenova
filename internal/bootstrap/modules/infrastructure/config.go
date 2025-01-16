package infrastructure

import (
	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/pkg/config"
	"go.uber.org/fx"
)

var ConfigModule = fx.Module("config", fx.Provide(
	config.NewManager,
	provideConfig,
),
)

// provideConfig loads the config and validates it.
func provideConfig(m *config.Manager) (*config.Config, error) {
	cfg, err := m.Load()
	if err != nil {
		return nil, eris.Wrap(err, "failed to load config")
	}

	// Validate the configuration
	if err = m.Validate(); err != nil {
		return nil, eris.Wrap(err, "invalid config")
	}

	return cfg, nil
}
