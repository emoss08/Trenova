package infrastructure

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"go.uber.org/fx"
)

var CacheModule = fx.Module("cache",
	fx.Provide(redis.NewClient),
	fx.Provide(redis.NewScriptLoader),
	fx.Invoke(func(lc fx.Lifecycle, scriptLoader *redis.ScriptLoader) (*redis.ScriptLoader, error) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return scriptLoader.LoadScripts(ctx)
			},
			OnStop: func(context.Context) error {
				return scriptLoader.UnloadScripts()
			},
		})

		return scriptLoader, nil
	}),
	fx.Invoke(func(lc fx.Lifecycle, client *redis.Client) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				// Perform Redis health check on startup
				if err := client.HealthCheck(ctx); err != nil {
					return err
				}
				return nil
			},
			OnStop: func(context.Context) error {
				return client.Close()
			},
		})
	}),
)
