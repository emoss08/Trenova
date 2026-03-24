package infrastructure

import (
	"testing"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func TestMeilisearchClientModuleProvidesSearchRepository(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Search: config.SearchConfig{
			Enabled: false,
		},
	}

	app := fx.New(
		fx.Supply(cfg),
		fx.Supply(zap.NewNop()),
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		MeilisearchClientModule,
		fx.Invoke(func(repo repoports.SearchRepository) {
			require.NotNil(t, repo)
		}),
	)

	require.NoError(t, app.Err())
}
