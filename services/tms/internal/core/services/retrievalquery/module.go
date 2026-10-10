package retrievalquery

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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

func asQueryVectorizer(s *Service) agentports.QueryVectorizer { return s }

func asCatalogVectorIndex(c *CatalogIndex) agentports.CatalogVectorIndex { return c }
