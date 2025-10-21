package infrastructure

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"go.uber.org/fx"
)

var RedisModule = fx.Module("redis",
	fx.Provide(redis.NewConnection),
	fx.Invoke(func(lc fx.Lifecycle, conn *redis.Connection) error {
		sl := redis.NewScriptLoader(conn)
		conn.SetScriptLoader(sl)

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return sl.LoadScripts(ctx)
			},
			OnStop: func(context.Context) error {
				return sl.UnloadScripts()
			},
		})
		return nil
	}),
)
