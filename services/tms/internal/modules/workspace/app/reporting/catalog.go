package reporting

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

// Catalog is the report catalog as the caller may use it, every entity,
// field and edge decided by the rule the compiler applies to a report, from
// one read of the caller's permissions.
type Catalog struct {
	Version  string
	Entities []reportaccess.Entity
}

func (s *Service) Catalog(ctx context.Context, req *Request) (*Catalog, error) {
	access, err := s.access.ForCatalog(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	entities, err := access.Catalog(ctx)
	if err != nil {
		return nil, err
	}

	return &Catalog{Version: reportcatalog.Default.Version, Entities: entities}, nil
}

// CatalogAccess is the resolver the catalog and the compiler decide by, for
// a surface that explains a decision in its own words.
func (s *Service) CatalogAccess() *reportaccess.Resolver {
	if s == nil {
		return nil
	}
	return s.access
}
