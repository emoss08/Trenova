/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package infrastructure

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"go.uber.org/fx"
)

var CacheModule = fx.Module("cache",
	fx.Provide(redis.NewClient),
	fx.Provide(redis.NewScriptLoader),
	fx.Provide(redis.NewCacheAdapter),
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
