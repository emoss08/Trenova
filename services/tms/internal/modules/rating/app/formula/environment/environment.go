package environment

import (
	goErrors "errors"
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

var ErrSchemaNotFound = goErrors.New("schema not found")

type Params struct {
	fx.In

	Registry *schema.Registry
	Resolver *resolver.Resolver
}

type Builder struct {
	registry *schema.Registry
	resolver *resolver.Resolver
}

func NewBuilder(p Params) *Builder {
	return &Builder{registry: p.Registry, resolver: p.Resolver}
}

type Built struct {
	Env      map[string]any
	Failures map[string]error
	Sources  Sources
}

func (b *Builder) definition(schemaID string) (*formulatypes.Definition, error) {
	definition, ok := b.registry.Get(schemaID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSchemaNotFound, schemaID)
	}
	return definition, nil
}

func (b *Builder) Build(
	entity any,
	schemaID string,
	provided map[string]any,
	variables map[string]any,
) (*Built, error) {
	definition, err := b.definition(schemaID)
	if err != nil {
		return nil, err
	}

	env := make(map[string]any, len(definition.Properties)+len(provided)+len(variables))
	failures := make(map[string]error)
	computedEntity := resolver.Memoize(entity)

	for fieldPath, source := range definition.FieldSources {
		if source.Provided {
			formulatypes.SetNestedValue(env, fieldPath, nil)
			continue
		}

		value, resolveErr := guardResolve(entity, fieldPath, func() (any, error) {
			if source.Computed {
				return b.resolver.ResolveComputed(computedEntity, source.Function)
			}
			return b.resolver.ResolveField(entity, source)
		})
		switch {
		case resolveErr != nil:
			value = nil
			if !source.Nullable || isPanic(resolveErr) {
				failures[fieldPath] = resolveErr
			}
		case value == nil && !source.Nullable:
			if property, ok := propertyAt(definition.Properties, fieldPath); ok {
				value = zeroValue(&property)
			}
		}

		formulatypes.SetNestedValue(env, fieldPath, value)
	}

	b.clearAbsentRelations(entity, definition, env)

	sources := ForSchema(definition)
	if err = Merge(env, provided, sources, formulatypes.ValueSourceProvided); err != nil {
		return nil, err
	}
	if err = Merge(env, variables, sources, formulatypes.ValueSourceInput); err != nil {
		return nil, err
	}

	return &Built{Env: env, Failures: failures, Sources: sources}, nil
}

func (b *Builder) Zero(
	schemaID string,
	provided map[string]any,
	variables map[string]any,
) (*Built, error) {
	definition, err := b.definition(schemaID)
	if err != nil {
		return nil, err
	}

	env := zeroProperties(definition.Properties)
	sources := make(Sources, len(provided)+len(variables))
	if err = Merge(env, provided, sources, formulatypes.ValueSourceProvided); err != nil {
		return nil, err
	}
	if err = Merge(env, variables, sources, formulatypes.ValueSourceSample); err != nil {
		return nil, err
	}

	return &Built{Env: env, Failures: map[string]error{}, Sources: sources}, nil
}

func (b *Builder) clearAbsentRelations(
	entity any,
	definition *formulatypes.Definition,
	env map[string]any,
) {
	objects := NullableObjects(definition.Properties, "")
	for path := range objects {
		relation := objects[path].Source.Relation
		if relation == "" {
			continue
		}
		value, err := guardResolve(entity, path, func() (any, error) {
			return b.resolver.ResolveField(entity, &formulatypes.FieldSource{
				Path:     relation,
				Nullable: true,
			})
		})
		if err == nil && value == nil {
			setPath(env, path, nil)
		}
	}
}

func (b *Builder) FieldRoots(schemaID string) map[string]struct{} {
	definition, ok := b.registry.Get(schemaID)
	if !ok {
		return nil
	}

	roots := make(map[string]struct{}, len(definition.Properties))
	for name := range definition.Properties {
		roots[name] = struct{}{}
	}
	return roots
}

func (b *Builder) RequiredPreloads(schemaID string) []string {
	definition, ok := b.registry.Get(schemaID)
	if !ok {
		return nil
	}

	preloadSet := make(map[string]struct{}, len(definition.DataSource.Preloads))
	addRelation := func(path string) {
		if path != "" {
			preloadSet[path] = struct{}{}
		}
	}

	for _, preload := range definition.DataSource.Preloads {
		addRelation(preload)
	}

	for _, source := range definition.FieldSources {
		for _, preload := range source.Preload {
			addRelation(preload)
		}
		for _, required := range source.Requires {
			if !isSchemaField(definition, required) {
				addRelation(required)
			}
		}
		if index := strings.LastIndex(source.Path, "."); index > 0 {
			addRelation(source.Path[:index])
		}
	}

	return slices.Sorted(maps.Keys(preloadSet))
}

func isSchemaField(definition *formulatypes.Definition, name string) bool {
	if _, ok := definition.FieldSources[name]; ok {
		return true
	}
	_, ok := definition.Properties[name]
	return ok
}

func zeroProperties(properties map[string]formulatypes.Property) map[string]any {
	env := make(map[string]any, len(properties))
	for name := range properties {
		property := properties[name]
		env[name] = zeroValue(&property)
	}
	return env
}

func zeroValue(property *formulatypes.Property) any {
	if property.IsDateTime() {
		return time.Time{}
	}
	switch fieldType, _ := TypeOf(property); fieldType {
	case "object":
		if property.Properties == nil {
			return map[string]any{}
		}
		return zeroProperties(property.Properties)
	case "array":
		if property.Items == nil {
			return []any{}
		}
		return []any{zeroValue(property.Items)}
	case "boolean":
		return false
	case "string":
		return ""
	case "integer":
		return int64(0)
	default:
		return decimal.Zero
	}
}

func TypeOf(property *formulatypes.Property) (string, bool) {
	nullable := property.Source.Nullable
	switch typed := property.Type.(type) {
	case string:
		return typed, nullable
	case []string:
		return firstType(typed, nullable)
	case []any:
		names := make([]string, 0, len(typed))
		for _, entry := range typed {
			if name, ok := entry.(string); ok {
				names = append(names, name)
			}
		}
		return firstType(names, nullable)
	default:
		return "", nullable
	}
}

func firstType(names []string, nullable bool) (string, bool) {
	resolved := ""
	for _, name := range names {
		if name == "null" {
			nullable = true
			continue
		}
		if resolved == "" && name != "" {
			resolved = name
		}
	}
	return resolved, nullable
}

func NullableObjects(
	properties map[string]formulatypes.Property,
	prefix string,
) map[string]formulatypes.Property {
	objects := make(map[string]formulatypes.Property)

	for name := range properties {
		prop := properties[name]
		if len(prop.Properties) == 0 {
			continue
		}

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		if _, nullable := TypeOf(&prop); nullable {
			objects[path] = prop
		}
		maps.Copy(objects, NullableObjects(prop.Properties, path))
	}

	return objects
}

func propertyAt(
	properties map[string]formulatypes.Property,
	path string,
) (formulatypes.Property, bool) {
	head, rest, nested := strings.Cut(path, ".")
	property, ok := properties[head]
	if !ok || !nested {
		return property, ok
	}
	return propertyAt(property.Properties, rest)
}

func ValueAt(env map[string]any, path string) any {
	head, rest, nested := strings.Cut(path, ".")
	value := env[head]
	if !nested {
		return value
	}
	child, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return ValueAt(child, rest)
}

func setPath(env map[string]any, path string, value any) {
	segments := strings.Split(path, ".")
	current := env
	for _, segment := range segments[:len(segments)-1] {
		nested, ok := current[segment].(map[string]any)
		if !ok {
			return
		}
		current = nested
	}
	current[segments[len(segments)-1]] = value
}

func guardResolve(
	entity any,
	fieldPath string,
	resolve func() (any, error),
) (value any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			value = nil
			err = errors.NewResolveError(
				fieldPath,
				fmt.Sprintf("%T", resolver.Unwrap(entity)),
				errors.NewPanicError(recovered, debug.Stack()),
			)
		}
	}()
	return resolve()
}

func isPanic(err error) bool {
	var panicErr *errors.PanicError
	return goErrors.As(err, &panicErr)
}
