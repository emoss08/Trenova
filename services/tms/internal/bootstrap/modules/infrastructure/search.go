package infrastructure

import (
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch/providers"
	"go.uber.org/fx"
)

var SearchModule = fx.Module("search",
	fx.Provide(
		meilisearch.NewConnection,
		fx.Annotate(
			meilisearch.NewEngine,
			fx.As(new(ports.SearchEngine)),
		),
		providers.NewSearchHelper,
	),
)
