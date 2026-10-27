package reporting

import (
	"context"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Catalog struct {
	Version  string
	Rules    analytics.RuleSet
	Entities []CatalogEntity
	Kinds    *reportaccess.Resolver
}

type CatalogEntity struct {
	View              analytics.EntityView
	Resource          permission.Resource
	Readable          bool
	Exportable        bool
	OwnScopeSupported bool
	Fields            []reportaccess.Use
	Edges             []reportaccess.Use
}

func (e *CatalogEntity) Usable() bool {
	if e.Readable {
		return true
	}
	for _, use := range e.Fields {
		if use.Available() {
			return true
		}
	}
	return false
}

func (s *Service) Catalog(ctx context.Context, req *Request) (*Catalog, error) {
	return CatalogFor(ctx, s.access, req.TenantInfo)
}

func CatalogFor(
	ctx context.Context,
	kinds *reportaccess.Resolver,
	tenant pagination.TenantInfo,
) (*Catalog, error) {
	access, err := kinds.For(ctx, tenant, reportaccess.ModeRead)
	if err != nil {
		return nil, err
	}

	catalog := kinds.Catalog()
	view := catalog.For(ctx, access)
	out := &Catalog{
		Version:  view.Version,
		Rules:    catalog.Rules(),
		Entities: make([]CatalogEntity, 0, len(view.Entities)),
		Kinds:    kinds,
	}
	for i := range view.Entities {
		ev := view.Entities[i]
		entity, ok := catalog.Entity(ev.Key)
		if !ok {
			continue
		}
		resource := kinds.Resource(entity)
		kind := access.Kind(resource)
		decided := CatalogEntity{
			View:              ev,
			Resource:          resource,
			Readable:          kind.Reads,
			Exportable:        kind.Reads && kind.Exports,
			OwnScopeSupported: kinds.OwnScopeSupported(entity),
			Fields:            make([]reportaccess.Use, 0, len(entity.Fields)),
			Edges:             make([]reportaccess.Use, 0, len(entity.Edges)),
		}
		for _, f := range entity.Fields {
			decided.Fields = append(decided.Fields, access.FieldUse(ctx, f))
		}
		for _, g := range entity.Edges {
			decided.Edges = append(decided.Edges, access.EdgeUse(ctx, g))
		}
		out.Entities = append(out.Entities, decided)
	}
	return out, nil
}

func (s *Service) CatalogAccess() *reportaccess.Resolver {
	if s == nil {
		return nil
	}
	return s.access
}

func (s *Service) CatalogVersion() string {
	return s.engine.Catalog().Version()
}
