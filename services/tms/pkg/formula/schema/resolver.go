package schema

import (
	"context"
	goErrors "errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/emoss08/trenova/pkg/formula/conversion"
	"github.com/emoss08/trenova/pkg/formula/errors"
)

const (
	UnknownType = "unknown"
)

type DataResolver interface {
	ResolveEntity(ctx context.Context, schema *Definition, entityID string) (any, error)
	ResolveField(entity any, fieldSource *FieldSource) (any, error)
	ResolveComputed(entity any, fieldSource *FieldSource) (any, error)
}

type TransformFunc func(value any) (any, error)

type DefaultDataResolver struct {
	transforms map[string]TransformFunc
	computers  map[string]ComputeFunc
}

type ComputeFunc func(entity any) (any, error)

func NewDefaultDataResolver() *DefaultDataResolver {
	resolver := &DefaultDataResolver{
		transforms: make(map[string]TransformFunc),
		computers:  make(map[string]ComputeFunc),
	}

	resolver.RegisterTransform(TransformDecimalToFloat64, transformNumericToFloat64)
	resolver.RegisterTransform(TransformInt64ToFloat64, transformNumericToFloat64)
	resolver.RegisterTransform(TransformInt16ToFloat64, transformNumericToFloat64)

	return resolver
}

func (r *DefaultDataResolver) RegisterTransform(name string, fn TransformFunc) {
	r.transforms[name] = fn
}

func (r *DefaultDataResolver) RegisterComputer(name string, fn ComputeFunc) {
	r.computers[name] = fn
}

func (r *DefaultDataResolver) ResolveField(entity any, fieldSource *FieldSource) (any, error) {
	if fieldSource.Computed {
		return r.ResolveComputed(entity, fieldSource)
	}

	value, err := r.extractFieldValue(entity, fieldSource.Path)
	if err != nil {
		entityType := UnknownType
		if entity != nil {
			entityType = reflect.TypeOf(entity).String()
		}
		return nil, errors.NewResolveError(fieldSource.Path, entityType, err)
	}

	if fieldSource.Transform != "" {
		transform, exists := r.transforms[fieldSource.Transform]
		if !exists {
			return nil, fmt.Errorf("transform not found: %s", fieldSource.Transform)
		}
		transformedValue, transformErr := transform(value)
		if transformErr != nil {
			sourceType := UnknownType
			if value != nil {
				sourceType = reflect.TypeOf(value).String()
			}
			return nil, errors.NewTransformError(
				sourceType,
				fieldSource.Transform,
				value,
				transformErr,
			)
		}

		return transformedValue, nil
	}

	return value, nil
}

func (r *DefaultDataResolver) ResolveComputed(entity any, fieldSource *FieldSource) (any, error) {
	computer, exists := r.computers[fieldSource.Function]
	if !exists {
		return nil, fmt.Errorf("compute function not found: %s", fieldSource.Function)
	}

	result, err := computer(entity)
	if err != nil {
		entityType := UnknownType
		if entity != nil {
			entityType = reflect.TypeOf(entity).String()
		}
		return nil, errors.NewComputeError(fieldSource.Function, entityType, err)
	}
	return result, nil
}

func (r *DefaultDataResolver) extractFieldValue(entity any, path string) (any, error) {
	if entity == nil {
		return nil, goErrors.New("entity is nil")
	}

	if m, ok := entity.(map[string]any); ok {
		return r.extractFromMap(m, path)
	}

	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, goErrors.New("entity is nil")
		}
		v = v.Elem()
	}

	parts := strings.Split(path, ".")
	current := v

	for i, part := range parts {
		field := current.FieldByName(part)
		if !field.IsValid() {
			return nil, fmt.Errorf("field not found: %s in path %s", part, path)
		}

		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				return nil, goErrors.New("field is nil")
			}
			field = field.Elem()
		}

		if i == len(parts)-1 {
			return field.Interface(), nil
		}

		current = field
	}

	return current.Interface(), nil
}

func (r *DefaultDataResolver) extractFromMap(m map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := any(m)

	for i, part := range parts {
		if currentMap, ok := current.(map[string]any); ok { //nolint:nestif // this is fine
			value, exists := currentMap[part]

			if !exists {
				camelCasePart := strings.ToLower(part[:1]) + part[1:]
				if value, exists = currentMap[camelCasePart]; !exists {
					return nil, fmt.Errorf("field not found: %s in path %s", part, path)
				}
			}

			if i == len(parts)-1 {
				return value, nil
			}

			current = value
		} else {
			return nil, fmt.Errorf("cannot navigate path %s: %s is not a map", path, part)
		}
	}

	return current, nil
}

func transformNumericToFloat64(value any) (any, error) {
	f, _ := conversion.ToFloat64(value)
	return f, nil
}
