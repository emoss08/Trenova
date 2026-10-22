package gqlext

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/selfscoped"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
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
)

type ProtectedFieldMap map[string]map[string]permission.FieldGroup

func (m ProtectedFieldMap) Group(object, field string) (permission.FieldGroup, bool) {
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
			group := permission.FieldGroup(argument.Value.Raw)
			if !group.IsValid() {
				return nil, fmt.Errorf(
					"@%s on %s.%s names unknown group %q",
					protectedDirective, name, field.Name, argument.Value.Raw,
				)
			}
			if fields[name] == nil {
				fields[name] = make(map[string]permission.FieldGroup, 2)
			}
			fields[name][field.Name] = group
		}
	}

	return fields, nil
}

type FieldGuardParams struct {
	fx.In

	PermissionEngine permissionports.PermissionEngine
}

type FieldGuardExtension struct {
	engine    permissionports.PermissionEngine
	fields    ProtectedFieldMap
	registry  *permission.Registry
	views     *refview.Views
	resources map[string]permission.Resource
	scoped    []string
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
		views:    refview.New(registry),
		scoped:   registry.DatabaseScopedResources(),
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
	return nil
}

// ResourceTypes maps each object type named for a registered resource
// (Customer is customer, BillingQueueItem is billing_queue_item) to it. A
// type with no resource of its own belongs to the record that carries it.
func ResourceTypes(
	schema *ast.Schema,
	registry *permission.Registry,
) map[string]permission.Resource {
	out := make(map[string]permission.Resource)
	if schema == nil || registry == nil {
		return out
	}
	for name, def := range schema.Types {
		if def.Kind != ast.Object || def.BuiltIn {
			continue
		}
		resource := stringutils.ConvertCamelToSnake(name)
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
	registry *permission.Registry,
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

// Authorizes reports whether a root field declares, with
// @authorizes(resource:), that its resolver checks read on a kind, so the
// records of that kind it returns inside a card or overview are its own.
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
	visible permission.FieldGroupSet

	accessOnce sync.Once
	access     *refview.PersonAccess

	mu       sync.Mutex
	withheld map[string]struct{}
}

func (e *FieldGuardExtension) InterceptOperation(
	ctx context.Context,
	next graphql.OperationHandler,
) graphql.ResponseHandler {
	return next(context.WithValue(ctx, fieldVisibilityKey{}, &fieldVisibility{
		withheld: make(map[string]struct{}, 4),
	}))
}

// InterceptResponse names every path a person's read withheld, without list
// indices, so a client can tell a field it may not see from one that is
// empty.
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
	if len(withheld) == 0 {
		return resp
	}
	if resp.Extensions == nil {
		resp.Extensions = make(map[string]any, 1)
	}
	resp.Extensions[withheldExtension] = withheld

	return resp
}

func (v *fieldVisibility) withhold(path ast.Path) {
	name := gqlctx.WithheldName(path)
	v.mu.Lock()
	v.withheld[name] = struct{}{}
	v.mu.Unlock()
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
	if group, protected := e.fields.Group(fc.Object, fc.Field.Name); protected &&
		!e.visible(ctx).Has(group) {
		e.recordWithheld(ctx, ceiling, agentRead, fc.Path(), false)
		return nil, nil //nolint:nilnil // a withheld field is answered null
	}
	record, ok := e.record(fc)
	if !ok || fc.Field.Name == recordIDField {
		return next(ctx)
	}
	var access refview.Access
	if agentRead {
		access = refview.AgentAccess{Ceilings: ceiling}
	} else {
		access = e.personAccess(ctx)
	}
	decision := e.decide(ctx, access, record, fc.Field.Name)
	if decision.Visible() {
		return next(ctx)
	}

	nonNull := fc.Field.Definition != nil && fc.Field.Definition.Type.NonNull
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

// recordContext is the record a field belongs to: its kind, and how it was
// reached. A part of another record (a shipment's stop) follows the record
// it is part of, all of it, with no reference view of its own.
type recordContext struct {
	kind  permission.Resource
	reach refview.Reach
	part  bool
	// carried is a root record of a database-scoped kind that a mutation's
	// payload carries rather than returns, which is linked for a caller who
	// cannot read its kind.
	carried bool
	// through is the record's own field a deeper field is reached through,
	// such as a location's state for the state's abbreviation.
	through string
}

func (e *FieldGuardExtension) decide(
	ctx context.Context,
	access refview.Access,
	record recordContext,
	field string,
) refview.Decision {
	if record.carried && record.reach == refview.Root && !access.Reads(ctx, record.kind) {
		record.reach = refview.Nested
	}
	if record.part && record.reach == refview.Nested && !access.Reads(ctx, record.kind) {
		return refview.WithheldKind
	}
	if record.through != "" && !record.part && e.views.InView(record.kind, record.through) {
		field = record.through
	}

	return e.views.Decide(ctx, access, record.kind, field, record.reach)
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
// a self-scoped root; or carried by another record. A shipment, order,
// invoice or worker a mutation's payload carries is marked carried: the
// mutation authorized what it wrote, which need not be that kind (a training
// record carries its worker), and row-level security does not hide a row of
// those kinds the caller cannot read, so it is the root record only for a
// caller who reads its kind or when the mutation declares @authorizes for it.
func (e *FieldGuardExtension) reachOf(
	returned *graphql.FieldContext,
	kind permission.Resource,
) recordContext {
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
		if _, record := e.resources[f.Object]; record {
			throughRecord = true
		}
		if nested {
			continue
		}
		switch {
		case e.payload(f):
			carried = true
		case !wrapper(f):
			nested = true
		}
	}
	if !nested {
		return recordContext{
			kind:    kind,
			reach:   refview.Root,
			carried: carried && slices.Contains(e.scoped, kind.String()),
		}
	}
	if carrier, ok := e.record(returned); ok && e.views.PartOf(kind, carrier.kind) {
		carrier.part = carrier.part || carrier.kind != kind
		carrier.through = ""
		return carrier
	}

	return recordContext{kind: kind, reach: refview.Nested}
}

// wrapper reports whether a field's parent type only carries what its root
// field returned: a connection or an edge.
func wrapper(fc *graphql.FieldContext) bool {
	return strings.HasSuffix(fc.Object, "Connection") || strings.HasSuffix(fc.Object, "Edge")
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
			if e.engine == nil {
				return false, nil
			}
			return base.CheckPermission(ctx, e.engine, authCtx, kind, permission.OpRead)
		},
	)
}

func (e *FieldGuardExtension) visible(ctx context.Context) permission.FieldGroupSet {
	holder, ok := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility)
	if !ok {
		return e.resolveVisible(ctx)
	}
	holder.once.Do(func() {
		holder.visible = e.resolveVisible(ctx)
	})
	return holder.visible
}

func (e *FieldGuardExtension) resolveVisible(ctx context.Context) permission.FieldGroupSet {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil || e.engine == nil {
		return 0
	}
	return fieldsensitivity.VisibleGroups(
		ctx,
		func(ctx context.Context, resource permission.Resource) (bool, error) {
			return base.CheckPermission(ctx, e.engine, authCtx, resource, permission.OpRead)
		},
	)
}
