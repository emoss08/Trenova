package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch/providers"
	"go.uber.org/fx"
)

var SearchModule = fx.Module("search",
	fx.Provide(
		meilisearch.NewConnection,
		meilisearch.NewClient,
		meilisearch.NewIndexer,
		meilisearch.NewSearcher,
		meilisearch.NewEngine,
		providers.NewSearchHelper,
	),
)
