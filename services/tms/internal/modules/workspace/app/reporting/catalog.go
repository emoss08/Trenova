package reporting

import (
	"context"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
)

type Catalog struct {
	Version        string
	Rules          analytics.RuleSet
	Entities       []CatalogEntity
	Kinds          *reportaccess.Resolver
	WithheldByPlan []PlanWithheld
}

type PlanWithheld struct {
	Entity      string
	EntityLabel string
	Name        string
	Label       string
	Edge        bool
	Denial      permissionports.PlanDenial
}

func (c *Catalog) WithheldEntity(key string) (PlanWithheld, bool) {
	for i := range c.WithheldByPlan {
		if c.WithheldByPlan[i].Entity == key && c.WithheldByPlan[i].Name == "" {
			return c.WithheldByPlan[i], true
		}
	}
	return PlanWithheld{}, false
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
	return CatalogFor(ctx, s.access, req)
}

func CatalogFor(
	ctx context.Context,
	kinds *reportaccess.Resolver,
	req *Request,
) (*Catalog, error) {
	access, err := kinds.For(ctx, req.TenantInfo, reportaccess.ModeRead)
	if err != nil {
		return nil, err
	}
	access.LimitFields(req.FieldCeiling)

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
		if access.LacksOperation(entity) {
			continue
		}
		if use := access.EntityUse(entity); use.Decision == reportaccess.NotOnPlan {
			out.WithheldByPlan = append(out.WithheldByPlan, PlanWithheld{
				Entity:      ev.Key,
				EntityLabel: ev.Plural,
				Label:       ev.Plural,
				Denial:      *use.Plan,
			})
			continue
		}
		out.Entities = append(out.Entities, out.decide(ctx, kinds, access, entity, ev))
	}
	return out, nil
}

func (c *Catalog) decide(
	ctx context.Context,
	kinds *reportaccess.Resolver,
	access *reportaccess.Access,
	entity *analytics.Entity,
	ev analytics.EntityView,
) CatalogEntity {
	resource := kinds.Resource(entity)
	kind := access.Kind(resource)
	decided := CatalogEntity{
		Resource:          resource,
		Readable:          kind.Reads,
		Exportable:        kind.Reads && kind.Exports,
		OwnScopeSupported: kinds.OwnScopeSupported(entity),
		Fields:            make([]reportaccess.Use, 0, len(entity.Fields)),
		Edges:             make([]reportaccess.Use, 0, len(entity.Edges)),
	}

	fields := make([]analytics.FieldView, 0, len(ev.Fields))
	for i, f := range entity.Fields {
		use := access.FieldUse(ctx, f)
		if use.Decision == reportaccess.NotOnPlan {
			c.withhold(&ev, ev.Fields[i].Key, ev.Fields[i].Label, false, use)
			continue
		}
		fields = append(fields, ev.Fields[i])
		decided.Fields = append(decided.Fields, use)
	}

	edges := make([]analytics.EdgeView, 0, len(ev.Edges))
	for i, g := range entity.Edges {
		use := access.EdgeUse(ctx, g)
		if use.Decision == reportaccess.NotOnPlan {
			c.withhold(&ev, ev.Edges[i].Name, ev.Edges[i].Label, true, use)
			continue
		}
		edges = append(edges, ev.Edges[i])
		decided.Edges = append(decided.Edges, use)
	}

	ev.Fields, ev.Edges = fields, edges
	decided.View = ev
	return decided
}

func (c *Catalog) withhold(
	ev *analytics.EntityView,
	name, label string,
	edge bool,
	use reportaccess.Use,
) {
	c.WithheldByPlan = append(c.WithheldByPlan, PlanWithheld{
		Entity:      ev.Key,
		EntityLabel: ev.Label,
		Name:        name,
		Label:       label,
		Edge:        edge,
		Denial:      *use.Plan,
	})
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
