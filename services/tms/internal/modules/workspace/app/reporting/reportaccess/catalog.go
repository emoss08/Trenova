package reportaccess

import (
	"context"

	"github.com/emoss08/trenova/pkg/reportcatalog"
)

// Entity is one catalog entity as one caller may use it. Fields and Edges
// run parallel to the entity's own.
type Entity struct {
	Entity *reportcatalog.Entity
	// Readable is whether the caller reads the kind: a report can start from
	// it and use every field their ceiling and groups allow. An entity they
	// cannot read is reached only through a join, and only its available
	// reference fields can be used.
	Readable bool
	// Exportable is whether a run may export the kind's own fields.
	Exportable bool
	Fields     []Use
	Edges      []Use
}

// Usable reports whether a report can use anything of the entity: it is
// readable, or one of its fields is available through its reference view.
func (e *Entity) Usable() bool {
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

// Catalog decides every entity, field and edge of the resolver's catalog
// for the caller, as the compiler will decide a report that uses them.
func (a *Access) Catalog(ctx context.Context) ([]Entity, error) {
	catalog := a.resolver.catalog
	entities := make([]Entity, 0, len(catalog.Entities))
	for i := range catalog.Entities {
		entity, err := a.Entity(ctx, &catalog.Entities[i])
		if err != nil {
			return nil, err
		}
		entities = append(entities, entity)
	}
	return entities, nil
}

// Entity decides one catalog entity for the caller.
func (a *Access) Entity(ctx context.Context, entity *reportcatalog.Entity) (Entity, error) {
	kind, err := a.Kind(ctx, entity.Resource)
	if err != nil {
		return Entity{}, err
	}

	out := Entity{
		Entity:     entity,
		Readable:   kind.Reads,
		Exportable: kind.Reads && kind.Exports,
		Fields:     make([]Use, 0, len(entity.Fields)),
		Edges:      make([]Use, 0, len(entity.Edges)),
	}
	for i := range entity.Fields {
		use, fieldErr := a.Field(ctx, entity, &entity.Fields[i])
		if fieldErr != nil {
			return Entity{}, fieldErr
		}
		out.Fields = append(out.Fields, use)
	}
	for i := range entity.Edges {
		use, edgeErr := a.Edge(ctx, entity, &entity.Edges[i])
		if edgeErr != nil {
			return Entity{}, edgeErr
		}
		out.Edges = append(out.Edges, use)
	}

	return out, nil
}
