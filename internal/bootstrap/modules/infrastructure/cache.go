package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"go.uber.org/fx"
)

var CacheModule = fx.Module("cache", fx.Provide(redis.NewClient))
