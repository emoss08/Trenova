package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"go.uber.org/fx"
)

var MeilisearchClientModule = fx.Module("meilisearch-client",
	fx.Provide(
		fx.Annotate(
			meilisearch.New,
			fx.As(new(customerports.SearchRepository)),
		),
	),
)
