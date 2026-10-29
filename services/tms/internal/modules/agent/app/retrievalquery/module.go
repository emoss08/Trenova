package retrievalquery

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
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

func asQueryVectorizer(s *Service) retrievalports.QueryVectorizer { return s }

func asCatalogVectorIndex(c *CatalogIndex) retrievalports.CatalogVectorIndex { return c }
