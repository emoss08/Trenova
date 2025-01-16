package infrastructure

import (
	"github.com/trenova-app/transport/internal/infrastructure/search/meilisearch"
	"go.uber.org/fx"
)

var SearchModule = fx.Module("search", fx.Provide(meilisearch.NewClient))
