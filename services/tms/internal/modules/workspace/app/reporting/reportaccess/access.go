package reportaccess

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

var (
	errNoEngine  = errors.New("report access: no permission engine")
	errNoCatalog = errors.New("report access: no catalog")
)

type Decision uint8

const (
	Available Decision = iota
	NeedsKindRead
	WithheldByCeiling
	NeedsFieldGroup
	NeedsExport
	OwnScopeUnsupported
)

func (d Decision) String() string {
	switch d {
	case Available:
		return "available"
	case NeedsKindRead:
		return "needs_kind_read"
	case WithheldByCeiling:
		return "withheld_by_ceiling"
	case NeedsFieldGroup:
		return "needs_field_group"
	case NeedsExport:
		return "needs_export"
	case OwnScopeUnsupported:
		return "own_scope_unsupported"
	default:
		return "unknown"
	}
}

func DecisionOf(reason string) Decision {
	for d := Available; d <= OwnScopeUnsupported; d++ {
		if d.String() == reason {
			return d
		}
	}
	return NeedsKindRead
}

type Use struct {
	Decision    Decision
	Kind        permission.Resource
	Sensitivity permission.FieldSensitivity
	Reference   bool
	Group       protectedfield.Group
	HoldsGroup  bool
}

func (u Use) Available() bool { return u.Decision == Available }

func (u Use) policy() analytics.Access {
	if u.Decision != Available {
		return analytics.Denied(u.Decision.String())
	}
	return analytics.Allowed().WithSensitivity(sensitivityOf(u.Sensitivity))
}

func sensitivityOf(s permission.FieldSensitivity) table.Sensitivity {
	switch s {
	case permission.SensitivityInternal:
		return table.Internal
	case permission.SensitivityRestricted:
		return table.Restricted
	case permission.SensitivityConfidential:
		return table.Confidential
	case permission.SensitivityPublic:
		return table.Public
	default:
		return table.Public
	}
}

type Mode uint8

const (
	ModeRead Mode = iota
	ModeExport
)

type Params struct {
	Catalog  *analytics.Catalog
	Engine   permissionports.PermissionEngine
	Registry *permissioncatalog.Registry
	Views    permissionports.ReferenceViews
	Fields   permissionports.FieldAccess
}

type Resolver struct {
	catalog  *analytics.Catalog
	engine   permissionports.PermissionEngine
	registry *permissioncatalog.Registry
	views    permissionports.ReferenceViews
	fields   permissionports.FieldAccess
	kinds    map[*analytics.Entity]kind
	prefixes map[*analytics.Entity]string
}

func New(p Params) (*Resolver, error) {
	if p.Catalog == nil {
		return nil, errNoCatalog
	}
	if p.Registry == nil {
		p.Registry = permissioncatalog.NewRegistry()
	}
	r := &Resolver{
		catalog:  p.Catalog,
		engine:   p.Engine,
		registry: p.Registry,
		views:    p.Views,
		fields:   p.Fields,
		kinds:    make(map[*analytics.Entity]kind, len(entityKinds)),
		prefixes: make(map[*analytics.Entity]string),
	}
	for _, e := range p.Catalog.Entities() {
		k, ok := entityKinds[e.Key]
		if !ok {
			return nil, fmt.Errorf("report access: entity %q has no kind", e.Key)
		}
		if !p.Registry.HasResource(k.resource.String()) {
			return nil, fmt.Errorf(
				"report access: entity %q names unregistered resource %q",
				e.Key,
				k.resource,
			)
		}
		if k.ownership != "" && e.Table.Column(k.ownership) < 0 {
			return nil, fmt.Errorf(
				"report access: entity %q has no ownership column %q",
				e.Key,
				k.ownership,
			)
		}
		r.kinds[e] = k
	}
	for e, k := range r.kinds {
		if k.withinRecord {
			r.prefixes[e] = r.recordEdge(e, k.resource)
		}
	}
	return r, nil
}

func (r *Resolver) Catalog() *analytics.Catalog { return r.catalog }

func (r *Resolver) recordEdge(e *analytics.Entity, resource permission.Resource) string {
	for root, k := range r.kinds {
		if k.withinRecord || k.resource != resource {
			continue
		}
		for _, g := range root.Edges {
			if g.Target == e && !g.Pick {
				return g.Name
			}
		}
	}
	return ""
}

func (r *Resolver) Resource(e *analytics.Entity) permission.Resource {
	return r.kinds[e].resource
}

func (r *Resolver) RootEntity(resource permission.Resource) (*analytics.Entity, bool) {
	for _, e := range r.catalog.Entities() {
		if k, ok := r.kinds[e]; ok && k.resource == resource && !k.withinRecord {
			return e, true
		}
	}
	return nil, false
}

func (r *Resolver) OwnScopeSupported(e *analytics.Entity) bool {
	return r.kinds[e].ownership != ""
}

func (r *Resolver) Resources() []permission.Resource {
	out := make([]permission.Resource, 0, len(r.kinds))
	for _, k := range r.kinds {
		if !slices.Contains(out, k.resource) {
			out = append(out, k.resource)
		}
	}
	slices.Sort(out)
	return out
}

func (r *Resolver) For(
	ctx context.Context,
	tenant pagination.TenantInfo,
	mode Mode,
) (*Access, error) {
	if r.engine == nil {
		return nil, errNoEngine
	}

	resources := r.Resources()
	names := make([]string, 0, len(resources))
	for _, resource := range resources {
		names = append(names, resource.String())
	}
	batch, err := r.engine.GetResourcePermissionsBatch(ctx, tenant.UserID, tenant.OrgID, names)
	if err != nil {
		return nil, fmt.Errorf("resolve report permissions: %w", err)
	}

	access := &Access{
		resolver: r,
		tenant:   tenant,
		actor:    permissionports.UserActor(tenant),
		export:   mode == ModeExport,
		details: make(
			map[permission.Resource]*permissionports.ResourcePermissionDetail,
			len(names),
		),
	}
	for _, name := range names {
		access.details[permission.Resource(name)] = batch[name]
	}
	access.groups = sync.OnceValue(func() protectedfield.GroupSet {
		if r.fields == nil {
			return 0
		}
		return r.fields.Visible(ctx, access.actor)
	})
	if r.views != nil {
		access.linked = r.views.Person(access.actor, permissionports.PersonGrants{
			Reads:  access.readsErr,
			Detail: access.detail,
		})
	}

	return access, nil
}

type Denial struct {
	Entity *analytics.Entity
	Label  string
	Use    Use
}

type Access struct {
	resolver *Resolver
	tenant   pagination.TenantInfo
	actor    *permissionports.RequestActor
	linked   permissionports.LinkedFields
	groups   func() protectedfield.GroupSet
	export   bool

	mu      sync.Mutex
	details map[permission.Resource]*permissionports.ResourcePermissionDetail
	denials []Denial
}

var _ analytics.Policy = (*Access)(nil)

func (a *Access) detail(
	ctx context.Context,
	resource permission.Resource,
) (*permissionports.ResourcePermissionDetail, error) {
	a.mu.Lock()
	detail, ok := a.details[resource]
	a.mu.Unlock()
	if ok {
		return detail, nil
	}

	detail, err := a.resolver.engine.GetResourcePermissions(
		ctx, a.tenant.UserID, a.tenant.OrgID, resource.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("resolve permissions for %q: %w", resource, err)
	}

	a.mu.Lock()
	a.details[resource] = detail
	a.mu.Unlock()

	return detail, nil
}

func (a *Access) readsErr(ctx context.Context, resource permission.Resource) (bool, error) {
	detail, err := a.detail(ctx, resource)
	if err != nil {
		return false, err
	}
	return hasOperation(detail, permission.OpRead), nil
}

func (a *Access) cached(resource permission.Resource) *permissionports.ResourcePermissionDetail {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.details[resource]
}

type Kind struct {
	Reads   bool
	Exports bool
	Scope   permission.DataScope
}

func (a *Access) Kind(resource permission.Resource) Kind {
	detail := a.cached(resource)
	if detail == nil {
		return Kind{}
	}
	return Kind{
		Reads:   hasOperation(detail, permission.OpRead),
		Exports: hasOperation(detail, permission.OpExport),
		Scope:   detail.DataScope,
	}
}

func (a *Access) record(e *analytics.Entity, label string, use Use) {
	if use.Available() {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, d := range a.denials {
		if d.Entity == e && d.Label == label && d.Use.Decision == use.Decision {
			return
		}
	}
	a.denials = append(a.denials, Denial{Entity: e, Label: label, Use: use})
}

func (a *Access) Denials() []Denial {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.denials)
}

func (a *Access) EntityUse(e *analytics.Entity) Use {
	k := a.resolver.kinds[e]
	kind := a.Kind(k.resource)
	switch {
	case !kind.Reads:
		return Use{Decision: NeedsKindRead, Kind: k.resource}
	case a.export && !kind.Exports:
		return Use{Decision: NeedsExport, Kind: k.resource}
	case kind.Scope == permission.DataScopeOwn && k.ownership == "":
		return Use{Decision: OwnScopeUnsupported, Kind: k.resource}
	default:
		return Use{}
	}
}

func (a *Access) Mode() Mode {
	if a.export {
		return ModeExport
	}
	return ModeRead
}

func (a *Access) Entity(_ context.Context, e *analytics.Entity) analytics.Access {
	use := a.EntityUse(e)
	a.record(e, "", use)
	return use.policy()
}

func (a *Access) FieldUse(ctx context.Context, f *analytics.Field) Use {
	r := a.resolver
	k := r.kinds[f.Entity]
	key := APIKey(f.Entity.Table.Columns[f.Column].Name)
	use := Use{
		Sensitivity: r.registry.GetFieldSensitivity(k.resource.String(), key),
		Reference:   !k.withinRecord && r.registry.InReferenceView(k.resource.String(), key),
	}
	if group, ok := r.Protection(f); ok {
		use.Group = group
		use.HoldsGroup = a.groups().Has(group)
	}

	use.Decision = a.decide(ctx, k, key, use.Reference, &use.Sensitivity)
	switch {
	case use.Decision == NeedsKindRead:
		use.Kind = k.resource
	case use.Decision == Available && use.Group != "" && !use.HoldsGroup:
		use.Decision = NeedsFieldGroup
	case use.Decision == Available && a.needsExport(k, use.Reference):
		use.Decision = NeedsExport
		use.Kind = k.resource
	}
	return use
}

func (a *Access) Field(ctx context.Context, f *analytics.Field, _ analytics.Path) analytics.Access {
	use := a.FieldUse(ctx, f)
	a.record(f.Entity, a.resolver.catalog.Label(ctx, f.LabelKey, f.Label), use)
	return use.policy()
}

func (a *Access) EdgeUse(ctx context.Context, g *analytics.Edge) Use {
	r := a.resolver
	k := r.kinds[g.From]
	use := Use{
		Sensitivity: r.registry.GetFieldSensitivity(k.resource.String(), g.Name),
		Reference:   !k.withinRecord && r.registry.InReferenceView(k.resource.String(), g.Name),
	}

	use.Decision = a.decide(ctx, k, g.Name, use.Reference, nil)
	if use.Decision == NeedsKindRead {
		use.Kind = k.resource
		return use
	}
	if use.Decision != Available {
		return use
	}
	if g.Pick {
		for _, via := range r.PickVia(g) {
			resource := r.kinds[via].resource
			if !a.Kind(resource).Reads {
				return Use{Decision: NeedsKindRead, Kind: resource, Reference: use.Reference}
			}
		}
	}
	target := r.kinds[g.Target]
	if a.Kind(target.resource).Scope == permission.DataScopeOwn && target.ownership == "" {
		return Use{Decision: OwnScopeUnsupported, Kind: target.resource, Reference: use.Reference}
	}
	if a.needsExport(k, use.Reference) {
		return Use{Decision: NeedsExport, Kind: k.resource, Reference: use.Reference}
	}
	return use
}

func (a *Access) Edge(ctx context.Context, g *analytics.Edge, _ analytics.Path) analytics.Access {
	use := a.EdgeUse(ctx, g)
	a.record(g.From, a.resolver.catalog.Label(ctx, g.LabelKey, g.Label), use)
	return use.policy()
}

func (a *Access) Rows(_ context.Context, e *analytics.Entity) []keel.Pred {
	k := a.resolver.kinds[e]
	if k.ownership == "" || a.Kind(k.resource).Scope != permission.DataScopeOwn {
		return nil
	}
	return []keel.Pred{keel.DynFrom(e.Table).Path(k.ownership).Eq(a.tenant.UserID.String())}
}

func (a *Access) needsExport(k kind, reference bool) bool {
	if !a.export || reference {
		return false
	}
	kind := a.Kind(k.resource)
	return kind.Reads && !kind.Exports
}

func (a *Access) decide(
	ctx context.Context,
	k kind,
	key string,
	reference bool,
	sensitivity *permission.FieldSensitivity,
) Decision {
	detail := a.cached(k.resource)
	if hasOperation(detail, permission.OpRead) {
		if sensitivity != nil && !detail.MaxSensitivity.CanAccess(*sensitivity) {
			return WithheldByCeiling
		}
		return Available
	}

	if !reference || a.linked == nil {
		return NeedsKindRead
	}

	switch a.linked.Decide(ctx, k.resource, key) {
	case permissionports.LinkedFieldVisible:
		return Available
	case permissionports.LinkedFieldWithheldSensitivity:
		return WithheldByCeiling
	case permissionports.LinkedFieldWithheldKind:
		return NeedsKindRead
	default:
		return NeedsKindRead
	}
}

func (r *Resolver) Protection(f *analytics.Field) (protectedfield.Group, bool) {
	key := APIKey(f.Entity.Table.Columns[f.Column].Name)
	return protectedfield.GroupAt(r.kinds[f.Entity].resource, r.steps(f.Entity, key), nil)
}

func (r *Resolver) steps(e *analytics.Entity, key string) []string {
	if prefix := r.prefixes[e]; prefix != "" {
		return []string{prefix, key}
	}
	return []string{key}
}

func (r *Resolver) PickVia(g *analytics.Edge) []*analytics.Entity {
	if !g.Pick {
		return nil
	}
	rel := &g.From.Table.Relations[g.Relation]
	t := g.From.Table
	var via []*analytics.Entity
	for _, ri := range rel.Path {
		if ri < 0 || int(ri) >= len(t.Relations) {
			return via
		}
		t = t.Relations[ri].Target
		if e, ok := r.catalog.EntityOf(t); ok {
			via = append(via, e)
		}
	}
	return via
}

func APIKey(column string) string {
	parts := strings.Split(column, "_")
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(p)
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	return b.String()
}

func hasOperation(detail *permissionports.ResourcePermissionDetail, op permission.Operation) bool {
	return detail != nil && slices.Contains(detail.Operations, op)
}
