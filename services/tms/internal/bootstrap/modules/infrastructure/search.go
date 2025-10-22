package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch"
	"go.uber.org/fx"
)

var SearchModule = fx.Module("search",
	fx.Provide(
		meilisearch.NewConnection,
		fx.Annotate(
			meilisearch.NewEngine,
		),
	),
)
