package modules

import (
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func warmQueryCaches(log *zap.Logger) {
	entities := domainregistry.RegisterEntities()

	querybuilder.WarmFieldCache(entities...)

	searchableEntities := make([]domaintypes.PostgresSearchable, 0)
	for _, e := range entities {
		if se, ok := e.(domaintypes.PostgresSearchable); ok {
			searchableEntities = append(searchableEntities, se)
		}
	}

	if len(searchableEntities) > 0 {
		querybuilder.WarmFieldConfigCache(searchableEntities...)
	}

	log.Info("Query caches warmed",
		zap.Int("totalEntities", len(entities)),
		zap.Int("searchableEntities", len(searchableEntities)),
	)
}

var QueryCacheModule = fx.Module("query-cache-warming",
	fx.Invoke(warmQueryCaches),
)
