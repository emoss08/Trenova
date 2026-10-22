package compiler

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

type authzResult struct {
	scopes map[string]permission.DataScope
}

// entityUse is what a report reads of one entity: the fields it shows,
// filters or sorts by, and the edges it follows out of it. rows marks an
// entity whose own rows the query reads, the root and every entity a pick
// edge reads its way through, which needs its kind's read whatever is used
// of it. Everything else is decided field by field by reportaccess, the rule
// the builder's catalog is annotated by.
type entityUse struct {
	entity *reportcatalog.Entity
	refs   []*resolvedRef
	edges  []*reportcatalog.Edge
	rows   bool
}

func (u *entityUse) addEdge(edge *reportcatalog.Edge) {
	if !slices.Contains(u.edges, edge) {
		u.edges = append(u.edges, edge)
	}
}

// kindRead reports whether the report uses the entity beyond its reference
// view, which a run exports as the kind's own data.
func (u *entityUse) kindRead() bool {
	if u.rows {
		return true
	}
	for _, ref := range u.refs {
		if ref.field.Class != reportcatalog.ClassReference {
			return true
		}
	}
	for _, edge := range u.edges {
		if edge.Class != reportcatalog.ClassReference {
			return true
		}
	}
	return false
}

func (c *Compiler) authorize(
	ctx context.Context,
	req *reportports.ReportCompileRequest,
	v *validatedDef,
	requireExport bool,
) (*authzResult, error) {
	uses := c.entityUses(v)

	resources := make([]permission.Resource, 0, len(v.entityKeys))
	for _, entityKey := range v.entityKeys {
		if entity, ok := c.catalog.Entity(entityKey); ok {
			resources = append(resources, entity.Resource)
		}
	}
	access, err := c.access.For(ctx, req.Tenant, resources)
	if err != nil {
		return nil, err
	}

	result := &authzResult{
		scopes: make(map[string]permission.DataScope, len(v.entityKeys)),
	}
	var denied []string

	for _, entityKey := range v.entityKeys {
		entity, ok := c.catalog.Entity(entityKey)
		if !ok {
			return nil, fmt.Errorf("catalog entity %q vanished during compilation", entityKey)
		}
		use := uses[entityKey]
		if use == nil {
			use = &entityUse{entity: entity, rows: true}
		}

		kind, kindErr := access.Kind(ctx, entity.Resource)
		if kindErr != nil {
			return nil, kindErr
		}

		switch {
		case use.rows && !kind.Reads:
			denied = append(denied, c.access.KindRead(entity))
			continue
		case kind.Reads && requireExport && !kind.Exports && use.kindRead():
			denied = append(denied, c.access.KindExport(entity))
			continue
		}
		if kind.Reads {
			result.scopes[entityKey] = kind.Scope
		}

		reasons, useErr := c.deniedUses(ctx, access, use)
		if useErr != nil {
			return nil, useErr
		}
		denied = append(denied, reasons...)
	}

	if len(denied) > 0 {
		return nil, errortypes.NewAuthorizationError(
			"This report references data you do not have access to: {0}",
			strings.Join(dedupeStrings(denied), "; "),
		)
	}

	return result, nil
}

// deniedUses decides each edge and field a report uses of one entity.
func (c *Compiler) deniedUses(
	ctx context.Context,
	access *reportaccess.Access,
	use *entityUse,
) ([]string, error) {
	var denied []string
	for _, edge := range use.edges {
		decided, err := access.Edge(ctx, use.entity, edge)
		if err != nil {
			return nil, err
		}
		if reason := c.access.Reason(use.entity, edge.Label, decided); reason != "" {
			denied = append(denied, reason)
		}
	}
	for _, ref := range use.refs {
		decided, err := access.Field(ctx, use.entity, ref.field)
		if err != nil {
			return nil, err
		}
		if reason := c.access.Reason(use.entity, ref.field.Label, decided); reason != "" {
			denied = append(denied, reason)
		}
	}
	return denied, nil
}

// entityUses gathers, per entity, the fields a report uses of it and the edges
// it follows out of it. The root and every entity a pick edge reads its way
// through read their kind's rows, not a view.
func (c *Compiler) entityUses(v *validatedDef) map[string]*entityUse {
	uses := make(map[string]*entityUse, len(v.entityKeys))
	useOf := func(entity *reportcatalog.Entity) *entityUse {
		use, ok := uses[entity.Key]
		if !ok {
			use = &entityUse{entity: entity}
			uses[entity.Key] = use
		}
		return use
	}

	useOf(v.entity).rows = true

	for _, ref := range sortedRefs(v.refs) {
		for _, step := range ref.path.Steps {
			if source, ok := c.catalog.Entity(step.Edge.Source); ok {
				useOf(source).addEdge(step.Edge)
			}
			useOf(step.Entity)
			for _, via := range c.access.PickVia(step.Edge) {
				useOf(via).rows = true
			}
		}
		terminal := useOf(ref.entity)
		if !slices.Contains(terminal.refs, ref) {
			terminal.refs = append(terminal.refs, ref)
		}
	}

	return uses
}

func sortedRefs(refs map[string]*resolvedRef) []*resolvedRef {
	keys := make([]string, 0, len(refs))
	for key := range refs {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	sorted := make([]*resolvedRef, 0, len(keys))
	for _, key := range keys {
		sorted = append(sorted, refs[key])
	}
	return sorted
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
