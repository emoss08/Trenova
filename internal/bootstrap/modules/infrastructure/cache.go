package infrastructure

import (
	"github.com/trenova-app/transport/internal/infrastructure/cache/redis"
	"go.uber.org/fx"
)

var CacheModule = fx.Module("cache", fx.Provide(redis.NewClient))
