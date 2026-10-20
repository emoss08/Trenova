package gqlext

import (
	"context"
	"fmt"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/fx"
)

const (
	fieldGuardExtensionName = "ProtectedFieldGuard"
	protectedDirective      = "protected"
	protectedGroupArgument  = "group"
	rootQuery               = "Query"
	rootMutation            = "Mutation"
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
	resources map[string]permission.Resource
}

// reducedTypes are the records an agent read shows only by id and name when
// another record carries them, as the agent tools always have: the person
// who created a customer is named, not described.
var reducedTypes = map[string]map[string]struct{}{
	"User":         reducedFields,
	"Organization": reducedFields,
	"BusinessUnit": reducedFields,
}

var reducedFields = map[string]struct{}{"id": {}, "name": {}}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationInterceptor
	graphql.FieldInterceptor
} = (*FieldGuardExtension)(nil)

func NewFieldGuardExtension(p FieldGuardParams) *FieldGuardExtension {
	return &FieldGuardExtension{
		engine:   p.PermissionEngine,
		registry: fieldsensitivity.DefaultRegistry(),
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
	e.fields = fields
	e.resources = ResourceTypes(schema.Schema(), e.registry)
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

type fieldVisibilityKey struct{}

type fieldVisibility struct {
	once    sync.Once
	visible permission.FieldGroupSet
}

func (e *FieldGuardExtension) InterceptOperation(
	ctx context.Context,
	next graphql.OperationHandler,
) graphql.ResponseHandler {
	if len(e.fields) == 0 {
		return next(ctx)
	}
	return next(context.WithValue(ctx, fieldVisibilityKey{}, &fieldVisibility{}))
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
		if agentRead {
			ceiling.Withhold(fc.Path(), false)
		}
		return nil, nil //nolint:nilnil // a withheld field is answered null
	}
	if agentRead && e.withholds(ctx, ceiling, fc) {
		nonNull := fc.Field.Definition != nil && fc.Field.Definition.Type.NonNull
		ceiling.Withhold(fc.Path(), nonNull)
		if !nonNull {
			return nil, nil //nolint:nilnil // a withheld field is answered null
		}
	}
	return next(ctx)
}

// withholds reports whether an agent read's ceiling hides a field: a nested
// user, organization or business unit beyond its id and name, or a field
// the registry ranks above the ceiling on the resource that carries it.
func (e *FieldGuardExtension) withholds(
	ctx context.Context,
	ceiling *gqlctx.FieldCeiling,
	fc *graphql.FieldContext,
) bool {
	if kept, reduced := reducedTypes[fc.Object]; reduced && nested(fc) {
		_, keep := kept[fc.Field.Name]
		return !keep
	}
	resource, ok := e.resourceOf(fc)
	if !ok {
		return false
	}

	return !fieldsensitivity.VisibleAt(
		e.registry.GetFieldSensitivity(resource.String(), fc.Field.Name),
		ceiling.For(ctx, resource),
	)
}

func (e *FieldGuardExtension) resourceOf(fc *graphql.FieldContext) (permission.Resource, bool) {
	for f := fc; f != nil; f = f.Parent {
		if resource, ok := e.resources[f.Object]; ok {
			return resource, true
		}
	}

	return "", false
}

func nested(fc *graphql.FieldContext) bool {
	for f := fc.Parent; f != nil; f = f.Parent {
		if f.Object != "" && f.Object != rootQuery && f.Object != rootMutation {
			return true
		}
	}

	return false
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
