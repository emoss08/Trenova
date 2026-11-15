package gqlext

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/selfscoped"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/fx"
)

const (
	fieldGuardExtensionName = "ProtectedFieldGuard"
	protectedDirective      = "protected"
	protectedGroupArgument  = "group"
	kindDirective           = "kind"
	kindResourceArgument    = "resource"
	authorizesDirective     = "authorizes"
	rootQuery               = "Query"
	rootMutation            = "Mutation"
	rootSubscription        = "Subscription"
	withheldExtension       = "withheldByAccess"
	withheldByPlanExtension = "withheldByPlan"
)

type ProtectedFieldMap map[string]map[string]protectedfield.Group

func (m ProtectedFieldMap) Group(object, field string) (protectedfield.Group, bool) {
	fields, ok := m[object]
	if !ok {
		return "", false
	}
	group, ok := fields[field]
	return group, ok
}

func ProtectedFields(schema *ast.Schema) (ProtectedFieldMap, error) {
	fields := make(ProtectedFieldMap)
	if schema == nil {
		return fields, nil
	}

	for name, def := range schema.Types {
		for _, field := range def.Fields {
			directive := field.Directives.ForName(protectedDirective)
			if directive == nil {
				continue
			}
			if def.Kind != ast.Object && def.Kind != ast.Interface {
				return nil, fmt.Errorf(
					"@%s on %s.%s: only object and interface fields can be protected",
					protectedDirective, name, field.Name,
				)
			}
			if field.Type.NonNull {
				return nil, fmt.Errorf(
					"@%s on %s.%s: a protected field must be nullable, because null is what "+
						"a reader without the grant receives",
					protectedDirective, name, field.Name,
				)
			}
			argument := directive.Arguments.ForName(protectedGroupArgument)
			if argument == nil || argument.Value == nil {
				return nil, fmt.Errorf(
					"@%s on %s.%s names no group", protectedDirective, name, field.Name,
				)
			}
			group := protectedfield.Group(argument.Value.Raw)
			if !group.IsValid() {
				return nil, fmt.Errorf(
					"@%s on %s.%s names unknown group %q",
					protectedDirective, name, field.Name, argument.Value.Raw,
				)
			}
			if fields[name] == nil {
				fields[name] = make(map[string]protectedfield.Group, 2)
			}
			fields[name][field.Name] = group
		}
	}

	return fields, nil
}

type FieldGuardParams struct {
	fx.In

	PermissionEngine permissionports.PermissionEngine
	Plan             *onplan.Lookup
}

type FieldGuardExtension struct {
	engine    permissionports.PermissionEngine
	fields    ProtectedFieldMap
	registry  *permissioncatalog.Registry
	views     *refview.Views
	resources map[string]permission.Resource
	wrappers  map[string]struct{}
	// returns maps every type that holds records of a kind, a connection or
	// an edge of them included, to that kind.
	returns map[string]permission.Resource
}

// recordIDField is how a read names its rows. It is never withheld: the
// resolver or row-level security has already decided the row may be read,
// and an id carries none of the row's content.
const recordIDField = "id"

var _ interface {
	graphql.HandlerExtension
	graphql.OperationInterceptor
	graphql.ResponseInterceptor
	graphql.FieldInterceptor
} = (*FieldGuardExtension)(nil)

func NewFieldGuardExtension(p FieldGuardParams) *FieldGuardExtension {
	registry := fieldsensitivity.DefaultRegistry()
	return &FieldGuardExtension{
		engine:   p.PermissionEngine,
		registry: registry,
		views:    refview.New(registry, p.Plan),
	}
}

func (*FieldGuardExtension) ExtensionName() string {
	return fieldGuardExtensionName
}

func (e *FieldGuardExtension) Validate(schema graphql.ExecutableSchema) error {
	fields, err := ProtectedFields(schema.Schema())
	if err != nil {
		return err
	}
	resources, err := KindTypes(schema.Schema(), e.registry)
	if err != nil {
		return err
	}
	e.fields = fields
	e.resources = resources
	e.wrappers = WrapperTypes(schema.Schema())
	e.returns = RecordTypes(schema.Schema(), resources)
	return nil
}

// RecordTypes maps every type that holds records of a kind to the kind: a
// type of a kind, and a connection or an edge whose nodes are of one.
func RecordTypes(
	schema *ast.Schema,
	kinds map[string]permission.Resource,
) map[string]permission.Resource {
	out := make(map[string]permission.Resource, len(kinds))
	for name, kind := range kinds {
		out[name] = kind
	}
	if schema == nil {
		return out
	}
	for name, def := range schema.Types {
		if kind, ok := wrappedKind(schema, kinds, def, 0); ok {
			out[name] = kind
		}
	}

	return out
}

func wrappedKind(
	schema *ast.Schema,
	kinds map[string]permission.Resource,
	def *ast.Definition,
	depth int,
) (permission.Resource, bool) {
	if def == nil || depth > 2 || !wrapperType(def) {
		return "", false
	}
	field := def.Fields.ForName("node")
	if field == nil {
		field = def.Fields.ForName("edges")
	}
	if field == nil {
		return "", false
	}
	inner := field.Type.Name()
	if kind, ok := kinds[inner]; ok {
		return kind, true
	}

	return wrappedKind(schema, kinds, schema.Types[inner], depth+1)
}

func WrapperTypes(schema *ast.Schema) map[string]struct{} {
	out := make(map[string]struct{})
	if schema == nil {
		return out
	}
	for name, def := range schema.Types {
		if wrapperType(def) {
			out[name] = struct{}{}
		}
	}

	return out
}

// ResourceTypes maps each object type named for a registered resource
// (Customer is customer, BillingQueueItem is billing_queue_item) to it. A
// type with no resource of its own belongs to the record that carries it.
func ResourceTypes(
	schema *ast.Schema,
	registry *permissioncatalog.Registry,
) map[string]permission.Resource {
	out := make(map[string]permission.Resource)
	if schema == nil || registry == nil {
		return out
	}
	for name, def := range schema.Types {
		if def.Kind != ast.Object || def.BuiltIn {
			continue
		}
		resource := textcase.ConvertCamelToSnake(name)
		if registry.HasResource(resource) {
			out[name] = permission.Resource(resource)
		}
	}

	return out
}

// KindTypes is ResourceTypes plus every type that declares the kind it shows
// with @kind(resource:), such as the shipment page's projection of its
// customer, so a projection is gated as the record it projects.
func KindTypes(
	schema *ast.Schema,
	registry *permissioncatalog.Registry,
) (map[string]permission.Resource, error) {
	out := ResourceTypes(schema, registry)
	if schema == nil {
		return out, nil
	}
	for name, def := range schema.Types {
		directive := def.Directives.ForName(kindDirective)
		if directive == nil {
			continue
		}
		argument := directive.Arguments.ForName(kindResourceArgument)
		if def.Kind != ast.Object || argument == nil || argument.Value == nil {
			return nil, fmt.Errorf("@%s on %s must name a resource on an object type",
				kindDirective, name)
		}
		if !registry.HasResource(argument.Value.Raw) {
			return nil, fmt.Errorf("@%s on %s names unknown resource %q",
				kindDirective, name, argument.Value.Raw)
		}
		if own, named := out[name]; named && own.String() != argument.Value.Raw {
			return nil, fmt.Errorf("@%s on %s names %q, but the type is the %q resource",
				kindDirective, name, argument.Value.Raw, own)
		}
		out[name] = permission.Resource(argument.Value.Raw)
	}

	return out, nil
}

// Authorizes reports whether a field declares, with @authorizes(resource:),
// that its resolver checks read on a kind: on a root field, the records of
// that kind it returns inside a card or overview are its own; on a field of a
// record, the resolver returns a record of that kind only to a caller who
// reads it, so the record is shown whole.
func Authorizes(def *ast.FieldDefinition, kind permission.Resource) bool {
	if def == nil {
		return false
	}
	for _, directive := range def.Directives {
		if directive.Name != authorizesDirective {
			continue
		}
		if argument := directive.Arguments.ForName(kindResourceArgument); argument != nil &&
			argument.Value != nil && argument.Value.Raw == kind.String() {
			return true
		}
	}

	return false
}

type fieldVisibilityKey struct{}

// fieldVisibility is one operation's memo of what its caller may see, and
// the paths a person's read withheld.
type fieldVisibility struct {
	once    sync.Once
	visible protectedfield.GroupSet

	accessOnce sync.Once
	access     *refview.PersonAccess

	viewsOnce sync.Once
	views     *refview.Views

	mu       sync.Mutex
	withheld map[string]struct{}
	byPlan   map[gqlctx.WithheldByPlan]struct{}
}

func (e *FieldGuardExtension) InterceptOperation(
	ctx context.Context,
	next graphql.OperationHandler,
) graphql.ResponseHandler {
	return next(context.WithValue(ctx, fieldVisibilityKey{}, &fieldVisibility{
		withheld: make(map[string]struct{}, 4),
		byPlan:   make(map[gqlctx.WithheldByPlan]struct{}),
	}))
}

// InterceptResponse names every path a person's read withheld, without list
// indices, so a client can tell a field it may not see from one that is
// empty: what their access withheld in withheldByAccess, and what the
// organization's plan does not include in withheldByPlan, with its feature.
func (e *FieldGuardExtension) InterceptResponse(
	ctx context.Context,
	next graphql.ResponseHandler,
) *graphql.Response {
	resp := next(ctx)
	holder, ok := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility)
	if resp == nil || !ok {
		return resp
	}
	withheld := holder.withheldPaths()
	byPlan := holder.withheldByPlan()
	if len(withheld) == 0 && len(byPlan) == 0 {
		return resp
	}
	if resp.Extensions == nil {
		resp.Extensions = make(map[string]any, 2)
	}
	if len(withheld) > 0 {
		resp.Extensions[withheldExtension] = withheld
	}
	if len(byPlan) > 0 {
		resp.Extensions[withheldByPlanExtension] = byPlan
	}

	return resp
}

func (v *fieldVisibility) withhold(path ast.Path) {
	name := gqlctx.WithheldName(path)
	v.mu.Lock()
	v.withheld[name] = struct{}{}
	v.mu.Unlock()
}

func (v *fieldVisibility) withholdByPlan(path ast.Path, feature platformcatalog.FeatureKey) {
	entry := gqlctx.WithheldByPlan{Path: gqlctx.WithheldName(path), Feature: feature}
	v.mu.Lock()
	v.byPlan[entry] = struct{}{}
	v.mu.Unlock()
}

func (v *fieldVisibility) withheldByPlan() []gqlctx.WithheldByPlan {
	v.mu.Lock()
	defer v.mu.Unlock()

	return gqlctx.SortedByPlan(v.byPlan)
}

func (v *fieldVisibility) withheldPaths() []string {
	v.mu.Lock()
	defer v.mu.Unlock()

	out := make([]string, 0, len(v.withheld))
	for name := range v.withheld {
		out = append(out, name)
	}
	slices.Sort(out)

	return out
}

func (e *FieldGuardExtension) InterceptField(
	ctx context.Context,
	next graphql.Resolver,
) (any, error) {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || fc.Field.Field == nil {
		return next(ctx)
	}
	ceiling, agentRead := gqlctx.FieldCeilingFrom(ctx)
	views := e.requestViews(ctx)
	nonNull := fc.Field.Definition != nil && fc.Field.Definition.Type.NonNull
	if withheld, err := e.returnsOffPlan(ctx, views, fc, ceiling, agentRead, nonNull); withheld {
		return nil, err
	}
	record, isRecord := e.record(fc)
	decided := isRecord && fc.Field.Name != recordIDField
	var decision refview.Decision
	if decided {
		var access refview.Access
		if agentRead {
			access = refview.AgentAccess{Ceilings: ceiling}
		} else {
			access = e.personAccess(ctx)
		}
		var denial permissionports.PlanDenial
		decision, denial = e.decide(ctx, views, access, record, fc.Field.Name)
		if decision == refview.WithheldPlan {
			return e.withholdByPlan(ctx, next, ceiling, agentRead, nonNull, denial)
		}
	}
	if group, protected := e.fields.Group(fc.Object, fc.Field.Name); protected &&
		!e.visible(ctx).Has(group) {
		e.recordWithheld(ctx, ceiling, agentRead, fc.Path(), false)
		return nil, nil //nolint:nilnil // a withheld field is answered null
	}
	if !decided || decision.Visible() {
		return next(ctx)
	}

	if agentRead {
		ceiling.Withhold(fc.Path(), nonNull)
		if nonNull {
			return next(ctx)
		}
		return nil, nil //nolint:nilnil // a withheld field is answered null
	}
	if nonNull {
		return nil, errortypes.NewAuthorizationError(
			"{0} of a linked {1} needs read access to {1}; the operation must declare "+
				"@requiresRead for it",
			fc.Field.Name, record.kind,
		)
	}
	e.recordWithheld(ctx, ceiling, false, fc.Path(), false)

	return nil, nil //nolint:nilnil // a withheld field is answered null
}

func (e *FieldGuardExtension) recordWithheld(
	ctx context.Context,
	ceiling *gqlctx.FieldCeiling,
	agentRead bool,
	path ast.Path,
	strip bool,
) {
	if agentRead {
		ceiling.Withhold(path, strip)
		return
	}
	if holder, ok := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility); ok {
		holder.withhold(path)
	}
}

// returnsOffPlan withholds a field that returns records of a kind the
// organization's plan does not include, before its resolver runs: a root
// field is refused, as is a non-null one, and any other is answered null.
// The kind is the returned type's own, even where it is part of its
// carrier.
func (e *FieldGuardExtension) returnsOffPlan(
	ctx context.Context,
	views *refview.Views,
	fc *graphql.FieldContext,
	ceiling *gqlctx.FieldCeiling,
	agentRead bool,
	nonNull bool,
) (bool, error) {
	if fc.Field.Definition == nil {
		return false, nil
	}
	kind, ok := e.returns[fc.Field.Definition.Type.Name()]
	if !ok {
		return false, nil
	}
	denial, off, err := views.Plan().Resource(ctx, kind.String())
	if err != nil {
		return true, err
	}
	if !off {
		return false, nil
	}
	if isRoot(fc.Object) || nonNull {
		return true, denial.Restriction()
	}
	e.recordWithheldByPlan(ctx, ceiling, agentRead, fc.Path(), denial.Feature, false)

	return true, nil
}

// withholdByPlan answers a field the plan does not include: null when it
// may be, and otherwise refused for a person, or resolved and marked for the
// agent read to strip.
func (e *FieldGuardExtension) withholdByPlan(
	ctx context.Context,
	next graphql.Resolver,
	ceiling *gqlctx.FieldCeiling,
	agentRead bool,
	nonNull bool,
	denial permissionports.PlanDenial,
) (any, error) {
	fc := graphql.GetFieldContext(ctx)
	if nonNull && !agentRead {
		return nil, denial.Restriction()
	}
	e.recordWithheldByPlan(ctx, ceiling, agentRead, fc.Path(), denial.Feature, nonNull)
	if nonNull {
		return next(ctx)
	}

	return nil, nil //nolint:nilnil // a withheld field is answered null
}

func (e *FieldGuardExtension) recordWithheldByPlan(
	ctx context.Context,
	ceiling *gqlctx.FieldCeiling,
	agentRead bool,
	path ast.Path,
	feature platformcatalog.FeatureKey,
	strip bool,
) {
	if agentRead {
		ceiling.WithholdByPlan(path, feature, strip)
		return
	}
	if holder, ok := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility); ok {
		holder.withholdByPlan(path, feature)
	}
}

// recordContext is the record a field belongs to: its kind, and how it was
// reached. A part of another record (a shipment's stop) follows the record
// it is part of, all of it, with no reference view of its own; own is the
// part's own kind, which the plan is asked about as well.
type recordContext struct {
	kind  permission.Resource
	own   permission.Resource
	reach refview.Reach
	part  bool
	// carried is a root record that a mutation's payload carries rather than
	// returns, which is linked for a caller who cannot read its kind.
	carried bool
	// through is the record's own field a deeper field is reached through,
	// such as a location's state for the state's abbreviation.
	through string
}

// decide is refview's decision on a field of a record, and the feature the
// plan withholds it for. A field reached through one of the record's own
// fields (a state under a location's address) is withheld by the plan when
// either field belongs to a feature it leaves out.
func (e *FieldGuardExtension) decide(
	ctx context.Context,
	views *refview.Views,
	access refview.Access,
	record recordContext,
	field string,
) (refview.Decision, permissionports.PlanDenial) {
	if record.carried && record.reach == refview.Root && !access.Reads(ctx, record.kind) {
		record.reach = refview.Nested
	}
	own := record.kind
	if record.part {
		own = record.own
	}
	if record.through != "" {
		denial, off, err := views.PlanDenial(ctx, own, record.through)
		switch {
		case err != nil:
			return refview.WithheldKind, permissionports.PlanDenial{}
		case off:
			return refview.WithheldPlan, denial
		}
	}
	if record.part {
		decision := views.DecidePart(ctx, access, record.own, record.kind, field, record.reach)
		if decision != refview.WithheldPlan {
			return decision, permissionports.PlanDenial{}
		}
		denial, _, _ := views.PartDenial(ctx, record.own, record.kind, field)
		return decision, denial
	}
	if record.through != "" && views.InView(record.kind, record.through) {
		field = record.through
	}
	decision := views.Decide(ctx, access, record.kind, field, record.reach)
	if decision != refview.WithheldPlan {
		return decision, permissionports.PlanDenial{}
	}
	denial, _, _ := views.PlanDenial(ctx, record.kind, field)

	return decision, denial
}

// record finds the record a field belongs to: the nearest enclosing type
// that is a kind. A field of a type with no kind of its own (a worker's
// profile, a shipment's assignment) belongs to the record that carries it.
func (e *FieldGuardExtension) record(fc *graphql.FieldContext) (recordContext, bool) {
	for f := fc; f != nil; f = f.Parent {
		kind, ok := e.resources[f.Object]
		if !ok {
			continue
		}
		record := e.reachOf(f.Parent, kind)
		if f != fc {
			record.through = f.Field.Name
		}
		return record, true
	}

	return recordContext{}, false
}

// reachOf is how the record a field returned was reached: from its root
// resolver, through a connection, its edges or a mutation's payload; under
// a self-scoped root; through a field whose resolver checks read on its kind
// (@authorizes on a field of a record); or carried by another record. A
// record a mutation's payload carries is marked carried: the mutation
// authorized what it wrote, which need not be that kind (a training record
// carries its worker, an assignment its customer), so it is the root record
// only for a caller who reads its kind or when the mutation declares
// @authorizes for it.
func (e *FieldGuardExtension) reachOf(
	returned *graphql.FieldContext,
	kind permission.Resource,
) recordContext {
	if via := returningField(returned); via != nil && !isRoot(via.Object) &&
		Authorizes(via.Field.Definition, kind) {
		return recordContext{kind: kind, reach: refview.Root}
	}
	nested, throughRecord, carried := false, false, false
	for f := returned; f != nil; f = f.Parent {
		if isRoot(f.Object) {
			if selfscoped.Root(f.Object, f.Field.Name) {
				return recordContext{kind: kind, reach: refview.SelfScoped}
			}
			if !throughRecord && Authorizes(f.Field.Definition, kind) {
				nested, carried = false, false
			}
			break
		}
		if _, record := e.resources[f.Object]; record && !returnedByRoot(f) {
			throughRecord = true
		}
		if nested {
			continue
		}
		switch {
		case e.payload(f):
			carried = true
		case !e.wrapper(f):
			nested = true
		}
	}
	if !nested {
		return recordContext{
			kind:    kind,
			reach:   refview.Root,
			carried: carried,
		}
	}
	if carrier, ok := e.record(returned); ok && e.views.PartOf(kind, carrier.kind) {
		carrier.part = carrier.part || carrier.kind != kind
		carrier.own = kind
		carrier.through = ""
		return carrier
	}

	return recordContext{kind: kind, reach: refview.Nested}
}

// returningField is the field that returned a record: the field itself, or
// for an element of a list, the list's field above the element's index.
func returningField(fc *graphql.FieldContext) *graphql.FieldContext {
	for f := fc; f != nil; f = f.Parent {
		if f.Field.Field != nil {
			return f
		}
	}

	return nil
}

// returnedByRoot reports whether a field belongs to the type its root field
// returns: a card of a kind, whose records of a kind the root declares with
// @authorizes are the root's own, as they are when the card has no kind.
func returnedByRoot(fc *graphql.FieldContext) bool {
	return fc.Parent != nil && isRoot(fc.Parent.Object)
}

// wrapper reports whether a field's parent type only carries what its root
// field returned: a connection or an edge.
func (e *FieldGuardExtension) wrapper(fc *graphql.FieldContext) bool {
	_, ok := e.wrappers[fc.Object]
	return ok
}

// payload reports whether a field belongs to the type a mutation returns
// when that type is not itself a record of a kind.
func (e *FieldGuardExtension) payload(fc *graphql.FieldContext) bool {
	if fc.Parent == nil || fc.Parent.Object != rootMutation {
		return false
	}
	_, record := e.resources[fc.Object]

	return !record
}

func isRoot(object string) bool {
	return object == rootQuery || object == rootMutation || object == rootSubscription
}

// requestViews is the operation's views, which ask the plan once per
// feature for the whole operation.
func (e *FieldGuardExtension) requestViews(ctx context.Context) *refview.Views {
	holder, ok := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility)
	if !ok {
		return e.views.Request()
	}
	holder.viewsOnce.Do(func() {
		holder.views = e.views.Request()
	})
	return holder.views
}

func (e *FieldGuardExtension) personAccess(ctx context.Context) refview.Access {
	holder, ok := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility)
	if !ok {
		return e.newPersonAccess(ctx)
	}
	holder.accessOnce.Do(func() {
		holder.access = e.newPersonAccess(ctx)
	})
	return holder.access
}

func (e *FieldGuardExtension) newPersonAccess(ctx context.Context) *refview.PersonAccess {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return refview.NewPersonAccess(nil, nil, nil)
	}
	return refview.NewPersonAccess(
		e.engine,
		actorutil.FromAuthContext(authCtx),
		func(ctx context.Context, kind permission.Resource) (bool, error) {
			return resolvercore.CheckPermission(ctx, e.engine, authCtx, kind, permission.OpRead)
		},
	)
}

func (e *FieldGuardExtension) visible(ctx context.Context) protectedfield.GroupSet {
	holder, ok := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility)
	if !ok {
		return e.resolveVisible(ctx)
	}
	holder.once.Do(func() {
		holder.visible = e.resolveVisible(ctx)
	})
	return holder.visible
}

func (e *FieldGuardExtension) resolveVisible(ctx context.Context) protectedfield.GroupSet {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok {
		return 0
	}
	return resolvercore.VisibleFieldGroups(ctx, e.engine, authCtx)
}
