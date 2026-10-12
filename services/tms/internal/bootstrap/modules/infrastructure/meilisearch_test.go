package infrastructure

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
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
		fx.Invoke(func(repo customerports.SearchRepository) {
			require.NotNil(t, repo)
		}),
	)

	require.NoError(t, app.Err())
}
