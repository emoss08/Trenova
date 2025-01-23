package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/search/meilisearch"
	"go.uber.org/fx"
)

var SearchModule = fx.Module("search", fx.Provide(meilisearch.NewClient))
