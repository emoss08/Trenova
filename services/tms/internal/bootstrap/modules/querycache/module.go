package querycache

import (
	"reflect"

	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("querycache",
	fx.Invoke(WarmFieldCaches),
)

type WarmCacheParams struct {
	fx.In
	Logger *zap.Logger
}

// WarmFieldCaches pre-populates field caches at application startup
// This avoids reflection during request processing
func WarmFieldCaches(p WarmCacheParams) error {
	logger := p.Logger.With(zap.String("component", "querycache"))
	logger.Info("🔥 Warming field caches for query builder")

	entities := domainregistry.RegisterEntities()

	// Extract type names for logging
	var entityTypes []string
	for _, entity := range entities {
		t := reflect.TypeOf(entity)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		entityTypes = append(entityTypes, t.String())
	}

	querybuilder.WarmFieldCache(entities...)

	logger.Info("🔥 Field caches warmed successfully",
		zap.Int("entities_cached", len(entities)),
		zap.Strings("entity_types", entityTypes),
	)

	return nil
}
