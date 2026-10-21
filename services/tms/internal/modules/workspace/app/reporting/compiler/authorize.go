package compiler

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

type authzResult struct {
	scopes map[string]permission.DataScope
}

// entityUse is what a report reads of one entity: the fields it shows,
// filters or sorts by, and the edges it follows out of it. A joined entity
// whose uses are all in its kind's reference view is read through the record
// that links to it and needs no read of its own (reference-views.md).
type entityUse struct {
	entity   *reportcatalog.Entity
	refs     []*resolvedRef
	linked   []string
	kindRead bool
}

func (u *entityUse) use(key string, class reportcatalog.FieldClass) {
	if class != reportcatalog.ClassReference {
		u.kindRead = true
		return
	}
	if !slices.Contains(u.linked, key) {
		u.linked = append(u.linked, key)
	}
}

type permissionDetails struct {
	engine  permissionports.PermissionEngine
	req     *reportports.ReportCompileRequest
	details map[permission.Resource]*permissionports.ResourcePermissionDetail
}

func (p *permissionDetails) get(
	ctx context.Context,
	resource permission.Resource,
) (*permissionports.ResourcePermissionDetail, error) {
	if detail, ok := p.details[resource]; ok {
		return detail, nil
	}
	detail, err := p.engine.GetResourcePermissions(
		ctx, p.req.Tenant.UserID, p.req.Tenant.OrgID, resource.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("resolve permissions for %q: %w", resource, err)
	}
	p.details[resource] = detail
	return detail, nil
}

func (p *permissionDetails) reads(ctx context.Context, resource permission.Resource) (bool, error) {
	detail, err := p.get(ctx, resource)
	if err != nil {
		return false, err
	}
	return hasOperation(detail, permission.OpRead), nil
}

func (c *Compiler) authorize(
	ctx context.Context,
	req *reportports.ReportCompileRequest,
	v *validatedDef,
	requireExport bool,
) (*authzResult, error) {
	result := &authzResult{
		scopes: make(map[string]permission.DataScope, len(v.entityKeys)),
	}
	details := &permissionDetails{
		engine: c.permissionEngine,
		req:    req,
		details: make(
			map[permission.Resource]*permissionports.ResourcePermissionDetail,
			len(v.entityKeys),
		),
	}
	actor := permissionports.UserActor(req.Tenant)

	var linked permissionports.LinkedFields
	if c.referenceViews != nil {
		linked = c.referenceViews.Person(actor, details.reads)
	}

	uses := c.entityUses(v)
	var denied []string

	for _, entityKey := range v.entityKeys {
		entity, ok := c.catalog.Entity(entityKey)
		if !ok {
			return nil, fmt.Errorf("catalog entity %q vanished during compilation", entityKey)
		}
		use := uses[entityKey]
		if use == nil {
			use = &entityUse{entity: entity, kindRead: true}
		}

		detail, err := details.get(ctx, entity.Resource)
		if err != nil {
			return nil, err
		}
		reads := hasOperation(detail, permission.OpRead)

		switch {
		case use.kindRead && !reads:
			denied = append(denied, fmt.Sprintf(
				"you do not have read access to %s", entity.PluralLabel,
			))
			continue
		case use.kindRead && requireExport && !hasOperation(detail, permission.OpExport):
			denied = append(denied, fmt.Sprintf(
				"you do not have export access to %s", entity.PluralLabel,
			))
			continue
		case !reads:
			denied = append(denied, c.checkLinkedAccess(ctx, linked, use)...)
			continue
		}

		result.scopes[entityKey] = detail.DataScope

		for _, ref := range use.refs {
			if fieldErr := c.checkFieldAccess(detail, ref); fieldErr != "" {
				denied = append(denied, fieldErr)
			}
		}
	}

	denied = append(denied, c.checkProtectedFields(ctx, actor, v)...)

	if len(denied) > 0 {
		return nil, errortypes.NewAuthorizationError(
			"This report references data you do not have access to: {0}",
			strings.Join(dedupeStrings(denied), "; "),
		)
	}

	return result, nil
}

// entityUses gathers, per entity, the fields a report uses of it and the edges
// it follows out of it. The root and every entity a pick edge reads its way
// through need their kind's read: the query reads their rows, not a view.
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

	useOf(v.entity).kindRead = true

	for _, ref := range sortedRefs(v.refs) {
		for _, step := range ref.path.Steps {
			if source, ok := c.catalog.Entity(step.Edge.Source); ok {
				useOf(source).use(step.Edge.Name, step.Edge.Class)
			}
			useOf(step.Entity)
			c.markPickVia(useOf, step.Edge)
		}
		terminal := useOf(ref.entity)
		terminal.refs = append(terminal.refs, ref)
		terminal.use(ref.field.Key, ref.field.Class)
	}

	return uses
}

func (c *Compiler) markPickVia(
	useOf func(*reportcatalog.Entity) *entityUse,
	edge *reportcatalog.Edge,
) {
	if edge == nil || edge.Pick == nil {
		return
	}
	_, via, err := c.catalog.ResolvePath(edge.Source, edge.Pick.Via)
	if err != nil {
		return
	}
	for _, step := range via.Steps {
		useOf(step.Entity).kindRead = true
		c.markPickVia(useOf, step.Edge)
	}
}

// checkLinkedAccess decides, through refview, the reference fields and edges
// a report uses of an entity whose kind the caller cannot read.
func (c *Compiler) checkLinkedAccess(
	ctx context.Context,
	linked permissionports.LinkedFields,
	use *entityUse,
) []string {
	entity := use.entity
	if linked == nil {
		return []string{fmt.Sprintf("you do not have read access to %s", entity.PluralLabel)}
	}

	var denied []string
	for _, key := range use.linked {
		switch linked.Decide(ctx, entity.Resource, key) {
		case permissionports.LinkedFieldVisible:
		case permissionports.LinkedFieldWithheldKind:
			denied = append(denied, fmt.Sprintf(
				"you do not have read access to %s", entity.PluralLabel,
			))
		case permissionports.LinkedFieldWithheldSensitivity:
			denied = append(denied, fmt.Sprintf(
				"field %q on %s requires %s access",
				keyLabel(entity, key), entity.Label,
				c.permissionRegistry.GetFieldSensitivity(entity.Resource.String(), key),
			))
		}
	}

	return denied
}

// checkProtectedFields refuses a field in a protected group the caller does
// not hold, as the GraphQL field guard withholds it.
func (c *Compiler) checkProtectedFields(
	ctx context.Context,
	actor *permissionports.RequestActor,
	v *validatedDef,
) []string {
	var (
		denied  []string
		visible permission.FieldGroupSet
		asked   bool
	)
	for _, ref := range sortedRefs(v.refs) {
		group := ref.field.Protected
		if group == "" {
			continue
		}
		if !asked {
			if c.fieldAccess != nil {
				visible = c.fieldAccess.Visible(ctx, actor)
			}
			asked = true
		}
		if visible.Has(group) {
			continue
		}
		denied = append(denied, fmt.Sprintf(
			"field %q on %s is a protected field and needs %s access",
			ref.field.Label, ref.entity.Label, group.Resource(),
		))
	}

	return denied
}

func (c *Compiler) checkFieldAccess(
	detail *permissionports.ResourcePermissionDetail,
	ref *resolvedRef,
) string {
	sensitivity := c.permissionRegistry.GetFieldSensitivity(
		ref.entity.Resource.String(), ref.field.Key,
	)
	if !detail.MaxSensitivity.CanAccess(sensitivity) {
		return fmt.Sprintf(
			"field %q on %s requires %s access",
			ref.field.Label, ref.entity.Label, sensitivity,
		)
	}

	if len(detail.AccessibleFields) > 0 && !containsField(detail.AccessibleFields, ref.field.Key) {
		return fmt.Sprintf(
			"field %q on %s is not in your accessible field set",
			ref.field.Label, ref.entity.Label,
		)
	}

	return ""
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

func keyLabel(entity *reportcatalog.Entity, key string) string {
	if field, ok := entity.Field(key); ok {
		return field.Label
	}
	if edge, ok := entity.Edge(key); ok {
		return edge.Label
	}
	return key
}

func hasOperation(detail *permissionports.ResourcePermissionDetail, op permission.Operation) bool {
	for _, allowed := range detail.Operations {
		if allowed == op {
			return true
		}
	}
	return false
}

func containsField(fields []string, field string) bool {
	for _, f := range fields {
		if f == field {
			return true
		}
	}
	return false
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
