package infrastructure

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch"
	"go.uber.org/fx"
)

var MeilisearchClientModule = fx.Module("meilisearch-client",
	fx.Provide(
		fx.Annotate(
			meilisearch.New,
			fx.As(new(repositories.SearchRepository)),
		),
	),
)
