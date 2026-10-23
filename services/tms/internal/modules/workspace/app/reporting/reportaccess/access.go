package reportaccess

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

var errNoEngine = errors.New("report access: no permission engine")

// Decision is whether a caller may use one catalog field or edge in a report,
// and when not, what they lack.
type Decision uint8

const (
	Available Decision = iota
	// NeedsKindRead is a field or edge outside its kind's reference view, of
	// a kind the caller cannot read.
	NeedsKindRead
	// WithheldByCeiling is a field ranked above the caller's ceiling.
	WithheldByCeiling
	// NeedsFieldGroup is a protected field whose group the caller lacks.
	NeedsFieldGroup
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
	default:
		return "unknown"
	}
}

// Use is the decision on one field or edge, with what the builder shows
// beside it.
type Use struct {
	Decision Decision
	// Kind is the resource whose read is missing when Decision is
	// NeedsKindRead: the field's own kind, or, for an edge that picks one row
	// of a relationship, the kind it reads through.
	Kind        permission.Resource
	Sensitivity permission.FieldSensitivity
	// Reference is whether the field or edge is in its kind's reference view,
	// so a report can use it through a join without the kind's read.
	Reference bool
	// Group is the protected field group, empty for an unprotected field.
	Group permission.FieldGroup
	// HoldsGroup is whether the caller holds Group.
	HoldsGroup bool
}

func (u Use) Available() bool { return u.Decision == Available }

// Params are what the decision is made from. Without Views a record reached
// through another needs its kind's read; without Fields every protected
// field is refused.
type Params struct {
	Catalog  *reportcatalog.Catalog
	Engine   permissionports.PermissionEngine
	Registry *permission.Registry
	Views    permissionports.ReferenceViews
	Fields   permissionports.FieldAccess
}

// Resolver decides which catalog fields and edges a caller may use. The
// report compiler, the catalog the builder reads and the agent's dataset
// tools all ask it, so what the builder offers is what the compiler accepts.
type Resolver struct {
	catalog  *reportcatalog.Catalog
	engine   permissionports.PermissionEngine
	registry *permission.Registry
	views    permissionports.ReferenceViews
	fields   permissionports.FieldAccess
}

func New(p Params) *Resolver {
	if p.Catalog == nil {
		p.Catalog = &reportcatalog.Default
	}
	if p.Registry == nil {
		p.Registry = permission.NewRegistry()
	}
	return &Resolver{
		catalog:  p.Catalog,
		engine:   p.Engine,
		registry: p.Registry,
		views:    p.Views,
		fields:   p.Fields,
	}
}

// For is one caller's access for the length of one request, read from one
// batch of their permissions on the resources given. A resource asked about
// later is read on its own and remembered.
func (r *Resolver) For(
	ctx context.Context,
	tenant pagination.TenantInfo,
	resources []permission.Resource,
) (*Access, error) {
	if r.engine == nil {
		return nil, errNoEngine
	}

	names := make([]string, 0, len(resources))
	for _, resource := range resources {
		if !slices.Contains(names, resource.String()) {
			names = append(names, resource.String())
		}
	}
	batch, err := r.engine.GetResourcePermissionsBatch(ctx, tenant.UserID, tenant.OrgID, names)
	if err != nil {
		return nil, fmt.Errorf("resolve report permissions: %w", err)
	}

	access := &Access{
		resolver: r,
		tenant:   tenant,
		actor:    permissionports.UserActor(tenant),
		details: make(
			map[permission.Resource]*permissionports.ResourcePermissionDetail,
			len(names),
		),
	}
	for _, name := range names {
		access.details[permission.Resource(name)] = batch[name]
	}
	access.groups = sync.OnceValue(func() permission.FieldGroupSet {
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

// ForCatalog is For on every resource the catalog reports on.
func (r *Resolver) ForCatalog(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*Access, error) {
	resources := make([]permission.Resource, 0, len(r.catalog.Entities))
	for i := range r.catalog.Entities {
		resources = append(resources, r.catalog.Entities[i].Resource)
	}
	return r.For(ctx, tenant, resources)
}

// Access is one caller's access to the report catalog for one request. It is
// safe for concurrent use.
type Access struct {
	resolver *Resolver
	tenant   pagination.TenantInfo
	actor    *permissionports.RequestActor
	linked   permissionports.LinkedFields
	groups   func() permission.FieldGroupSet

	mu      sync.Mutex
	details map[permission.Resource]*permissionports.ResourcePermissionDetail
}

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

// Kind is the caller's grant on one resource.
type Kind struct {
	Reads   bool
	Exports bool
	Scope   permission.DataScope
}

func (a *Access) Kind(ctx context.Context, resource permission.Resource) (Kind, error) {
	detail, err := a.detail(ctx, resource)
	if err != nil {
		return Kind{}, err
	}
	if detail == nil {
		return Kind{}, nil
	}
	return Kind{
		Reads:   hasOperation(detail, permission.OpRead),
		Exports: hasOperation(detail, permission.OpExport),
		Scope:   detail.DataScope,
	}, nil
}

// Field decides one field of an entity, wherever the report reaches it: on
// a kind the caller reads, under their ceiling; on one they cannot read,
// only through its reference view, under the ceiling refview applies; and,
// either way, only with its protected group.
func (a *Access) Field(
	ctx context.Context,
	entity *reportcatalog.Entity,
	field *reportcatalog.Field,
) (Use, error) {
	use := Use{
		Sensitivity: a.resolver.registry.GetFieldSensitivity(entity.Resource.String(), field.Key),
		Reference:   field.Class == reportcatalog.ClassReference,
		Group:       field.Protected,
	}
	if field.Protected != "" {
		use.HoldsGroup = a.groups().Has(field.Protected)
	}

	decision, err := a.decide(ctx, entity, field.Key, use.Reference, &use.Sensitivity)
	if err != nil {
		return Use{}, err
	}
	use.Decision = decision
	if decision == NeedsKindRead {
		use.Kind = entity.Resource
	}
	if decision == Available && field.Protected != "" && !use.HoldsGroup {
		use.Decision = NeedsFieldGroup
	}

	return use, nil
}

// Edge decides following one edge out of an entity. An edge is a field of
// its source: free on a kind the caller reads, and otherwise only through
// the source's reference view. An edge that picks one row of a relationship
// reads that relationship's rows, so it needs read on each kind it passes.
func (a *Access) Edge(
	ctx context.Context,
	entity *reportcatalog.Entity,
	edge *reportcatalog.Edge,
) (Use, error) {
	use := Use{
		Sensitivity: a.resolver.registry.GetFieldSensitivity(entity.Resource.String(), edge.Name),
		Reference:   edge.Class == reportcatalog.ClassReference,
	}

	decision, err := a.decide(ctx, entity, edge.Name, use.Reference, nil)
	if err != nil {
		return Use{}, err
	}
	use.Decision = decision
	if decision == NeedsKindRead {
		use.Kind = entity.Resource
		return use, nil
	}
	if decision != Available {
		return use, nil
	}

	kind, err := a.pickVia(ctx, edge)
	if err != nil {
		return Use{}, err
	}
	if kind != "" {
		use.Decision = NeedsKindRead
		use.Kind = kind
	}

	return use, nil
}

// PickVia is the kinds an edge that picks one row of a relationship reads its
// way through, each of which needs the caller's read.
func (r *Resolver) PickVia(edge *reportcatalog.Edge) []*reportcatalog.Entity {
	var via []*reportcatalog.Entity
	r.collectPickVia(edge, &via)
	return via
}

func (r *Resolver) collectPickVia(edge *reportcatalog.Edge, via *[]*reportcatalog.Entity) {
	if edge == nil || edge.Pick == nil {
		return
	}
	_, path, err := r.catalog.ResolvePath(edge.Source, edge.Pick.Via)
	if err != nil {
		return
	}
	for _, step := range path.Steps {
		*via = append(*via, step.Entity)
		r.collectPickVia(step.Edge, via)
	}
}

func (a *Access) pickVia(
	ctx context.Context,
	edge *reportcatalog.Edge,
) (permission.Resource, error) {
	for _, entity := range a.resolver.PickVia(edge) {
		reads, err := a.readsErr(ctx, entity.Resource)
		if err != nil {
			return "", err
		}
		if !reads {
			return entity.Resource, nil
		}
	}
	return "", nil
}

// decide is the one rule for a key of an entity. sensitivity, when given, is
// the key's tier on a kind the caller reads, held to their role's reach; an
// edge carries none.
func (a *Access) decide(
	ctx context.Context,
	entity *reportcatalog.Entity,
	key string,
	reference bool,
	sensitivity *permission.FieldSensitivity,
) (Decision, error) {
	detail, err := a.detail(ctx, entity.Resource)
	if err != nil {
		return NeedsKindRead, err
	}

	if hasOperation(detail, permission.OpRead) {
		if sensitivity != nil && !detail.MaxSensitivity.CanAccess(*sensitivity) {
			return WithheldByCeiling, nil
		}
		return Available, nil
	}

	if !reference || a.linked == nil {
		return NeedsKindRead, nil
	}

	switch a.linked.Decide(ctx, entity.Resource, key) {
	case permissionports.LinkedFieldVisible:
		return Available, nil
	case permissionports.LinkedFieldWithheldSensitivity:
		return WithheldByCeiling, nil
	case permissionports.LinkedFieldWithheldKind:
		return NeedsKindRead, nil
	default:
		return NeedsKindRead, nil
	}
}

func hasOperation(detail *permissionports.ResourcePermissionDetail, op permission.Operation) bool {
	return detail != nil && slices.Contains(detail.Operations, op)
}
