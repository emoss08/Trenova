// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package infrastructure

import (
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/rotisserie/eris"
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
