package graphql

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
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/fx"
)

const (
	fieldGuardExtensionName = "ProtectedFieldGuard"
	protectedDirective      = "protected"
	protectedGroupArgument  = "group"
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
	engine permissionports.PermissionEngine
	fields ProtectedFieldMap
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationInterceptor
	graphql.FieldInterceptor
} = (*FieldGuardExtension)(nil)

func NewFieldGuardExtension(p FieldGuardParams) *FieldGuardExtension {
	return &FieldGuardExtension{engine: p.PermissionEngine}
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
	return nil
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
	group, protected := e.fields.Group(fc.Object, fc.Field.Name)
	if !protected {
		return next(ctx)
	}
	if e.visible(ctx).Has(group) {
		return next(ctx)
	}
	return nil, nil
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
