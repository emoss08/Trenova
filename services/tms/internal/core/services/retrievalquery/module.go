package retrievalquery

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module("retrieval-query",
	fx.Provide(
		New,
		NewCatalogIndex,
		asQueryVectorizer,
		asCatalogVectorIndex,
	),
)

func asQueryVectorizer(s *Service) serviceports.QueryVectorizer { return s }

func asCatalogVectorIndex(c *CatalogIndex) serviceports.CatalogVectorIndex { return c }
