package reportaccess

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func (a *Access) kindPlan(resource permission.Resource) *permissionports.PlanDenial {
	detail := a.cached(resource)
	if detail == nil {
		return nil
	}
	return detail.PlanDenial
}

func (r *Resolver) declaresFeature(resource permission.Resource, key string) bool {
	_, declared := r.registry.FieldFeatureOf(resource.String(), key)
	return declared
}

func (a *Access) fieldPlan(
	ctx context.Context,
	resource permission.Resource,
	key string,
) (Decision, *permissionports.PlanDenial, bool) {
	if denial := a.kindPlan(resource); denial != nil {
		return NotOnPlan, denial, true
	}
	if a.linked == nil || !a.resolver.declaresFeature(resource, key) {
		return Available, nil, false
	}
	denial, off, err := a.linked.PlanDenial(ctx, resource, key)
	switch {
	case err != nil:
		return NeedsKindRead, nil, true
	case off:
		return NotOnPlan, &denial, true
	default:
		return Available, nil, false
	}
}

func (a *Access) edgePlan(ctx context.Context, g *analytics.Edge) (Use, bool) {
	r := a.resolver
	from := r.kinds[g.From].resource
	if decision, denial, withheld := a.fieldPlan(ctx, from, g.Name); withheld {
		return Use{Decision: decision, Kind: from, Plan: denial}, true
	}
	target := r.kinds[g.Target].resource
	if denial := a.kindPlan(target); denial != nil {
		return Use{Decision: NotOnPlan, Kind: target, Plan: denial}, true
	}
	for _, via := range r.PickVia(g) {
		resource := r.kinds[via].resource
		if denial := a.kindPlan(resource); denial != nil {
			return Use{Decision: NotOnPlan, Kind: resource, Plan: denial}, true
		}
	}
	return Use{}, false
}

func (a *Access) linkedPlan(
	ctx context.Context,
	resource permission.Resource,
	key string,
) (Decision, *permissionports.PlanDenial) {
	if a.linked == nil {
		return NeedsKindRead, nil
	}
	denial, off, err := a.linked.PlanDenial(ctx, resource, key)
	if err != nil || !off {
		return NeedsKindRead, nil
	}
	return NotOnPlan, &denial
}

type Reach struct {
	root   *analytics.Entity
	edges  []*analytics.Edge
	fields []*analytics.Field
}

func (r *Resolver) ReachOf(def *ir.Definition) (*Reach, error) {
	resolved, err := ir.Resolve(def, r.catalog, ir.Limits{})
	if err != nil {
		return nil, fmt.Errorf("resolve report definition: %w", err)
	}
	root, ok := r.catalog.Entity(def.Entity)
	if !ok {
		return nil, fmt.Errorf("report access: unknown entity %q", def.Entity)
	}

	reach := &Reach{root: root}
	for _, path := range slices.Sorted(maps.Keys(resolved.Fields)) {
		via, field, found := r.catalog.Resolve(def.Entity, path)
		if !found {
			continue
		}
		for _, edge := range via {
			if !slices.Contains(reach.edges, edge) {
				reach.edges = append(reach.edges, edge)
			}
		}
		if !slices.Contains(reach.fields, field) {
			reach.fields = append(reach.fields, field)
		}
	}
	return reach, nil
}

func (a *Access) OffPlan(ctx context.Context, reach *Reach) *permissionports.PlanDenial {
	r := a.resolver
	if denial := a.kindPlan(r.kinds[reach.root].resource); denial != nil {
		return denial
	}
	for _, edge := range reach.edges {
		if use, withheld := a.edgePlan(ctx, edge); withheld && use.Decision == NotOnPlan {
			return use.Plan
		}
	}
	for _, field := range reach.fields {
		resource := r.kinds[field.Entity].resource
		key := APIKey(field.Entity.Table.Columns[field.Column].Name)
		if decision, denial, _ := a.fieldPlan(ctx, resource, key); decision == NotOnPlan {
			return denial
		}
	}
	return nil
}

func (a *Access) MissingOperation(reach *Reach) (Use, bool) {
	r := a.resolver
	entities := make([]*analytics.Entity, 0, 1+len(reach.edges)+len(reach.fields))
	entities = append(entities, reach.root)
	for _, edge := range reach.edges {
		entities = append(entities, edge.Target)
	}
	for _, field := range reach.fields {
		entities = append(entities, field.Entity)
	}
	for _, e := range entities {
		if k := r.kinds[e]; a.lacksRequired(k) {
			return k.operationUse(), true
		}
	}
	return Use{}, false
}
