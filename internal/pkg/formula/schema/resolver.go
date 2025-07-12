package schema

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/emoss08/trenova/internal/pkg/formula/conversion"
	"github.com/emoss08/trenova/internal/pkg/formula/errors"
)

// * DataResolver is responsible for fetching and transforming data based on schemas
type DataResolver interface {
	// * ResolveEntity fetches an entity by ID with the specified preloads
	ResolveEntity(ctx context.Context, schema *SchemaDefinition, entityID string) (any, error)

	// * ResolveField extracts a field value from an entity
	ResolveField(entity any, fieldSource *FieldSource) (any, error)

	// * ResolveComputed calculates a computed field value
	ResolveComputed(entity any, fieldSource *FieldSource) (any, error)
}

// * TransformFunc is a function that transforms a value
type TransformFunc func(value any) (any, error)

// * DefaultDataResolver provides a default implementation of DataResolver
type DefaultDataResolver struct {
	transforms map[string]TransformFunc
	computers  map[string]ComputeFunc
}

// * ComputeFunc is a function that computes a value from an entity
type ComputeFunc func(entity any) (any, error)

// * NewDefaultDataResolver creates a new default resolver with standard transforms
func NewDefaultDataResolver() *DefaultDataResolver {
	resolver := &DefaultDataResolver{
		transforms: make(map[string]TransformFunc),
		computers:  make(map[string]ComputeFunc),
	}

	// * Register standard transforms
	resolver.RegisterTransform(TransformDecimalToFloat64, transformNumericToFloat64)
	resolver.RegisterTransform(TransformInt64ToFloat64, transformNumericToFloat64)
	resolver.RegisterTransform(TransformInt16ToFloat64, transformNumericToFloat64)

	return resolver
}

// * RegisterTransform registers a new transform function
func (r *DefaultDataResolver) RegisterTransform(name string, fn TransformFunc) {
	r.transforms[name] = fn
}

// * RegisterComputer registers a new compute function
func (r *DefaultDataResolver) RegisterComputer(name string, fn ComputeFunc) {
	r.computers[name] = fn
}

// * ResolveField extracts and transforms a field value
func (r *DefaultDataResolver) ResolveField(entity any, fieldSource *FieldSource) (any, error) {
	if fieldSource.Computed {
		return r.ResolveComputed(entity, fieldSource)
	}

	// * Extract value using reflection
	value, err := r.extractFieldValue(entity, fieldSource.Path)
	if err != nil {
		entityType := "unknown"
		if entity != nil {
			entityType = reflect.TypeOf(entity).String()
		}
		return nil, errors.NewResolveError(fieldSource.Path, entityType, err)
	}

	// * Apply transform if specified
	if fieldSource.Transform != "" {
		transform, exists := r.transforms[fieldSource.Transform]
		if !exists {
			return nil, fmt.Errorf("transform not found: %s", fieldSource.Transform)
		}
		transformedValue, err := transform(value)
		if err != nil {
			sourceType := "unknown"
			if value != nil {
				sourceType = reflect.TypeOf(value).String()
			}
			return nil, errors.NewTransformError(sourceType, fieldSource.Transform, value, err)
		}
		return transformedValue, nil
	}

	return value, nil
}

// * ResolveComputed calculates a computed field value
func (r *DefaultDataResolver) ResolveComputed(entity any, fieldSource *FieldSource) (any, error) {
	computer, exists := r.computers[fieldSource.Function]
	if !exists {
		return nil, fmt.Errorf("compute function not found: %s", fieldSource.Function)
	}

	result, err := computer(entity)
	if err != nil {
		entityType := "unknown"
		if entity != nil {
			entityType = reflect.TypeOf(entity).String()
		}
		return nil, errors.NewComputeError(fieldSource.Function, entityType, err)
	}
	return result, nil
}

// * extractFieldValue uses reflection to extract a value from a struct or map
func (r *DefaultDataResolver) extractFieldValue(entity any, path string) (any, error) {
	// * Handle nil
	if entity == nil {
		return nil, nil
	}

	// * Handle maps specially
	if m, ok := entity.(map[string]any); ok {
		return r.extractFromMap(m, path)
	}

	// * Get reflect value for structs
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	// * Handle nested paths (e.g., "Customer.Name")
	parts := strings.Split(path, ".")
	current := v

	for i, part := range parts {
		// * Get the field
		field := current.FieldByName(part)
		if !field.IsValid() {
			return nil, fmt.Errorf("field not found: %s in path %s", part, path)
		}

		// * Handle pointers
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				return nil, nil
			}
			field = field.Elem()
		}

		// * If this is the last part, return the value
		if i == len(parts)-1 {
			return field.Interface(), nil
		}

		// * Otherwise, continue navigating
		current = field
	}

	return current.Interface(), nil
}

// * extractFromMap extracts a value from a map using a path
func (r *DefaultDataResolver) extractFromMap(m map[string]any, path string) (any, error) {
	// Handle nested paths (e.g., "Customer.Name")
	parts := strings.Split(path, ".")
	current := any(m)

	for i, part := range parts {
		// Try to get the value as a map
		if currentMap, ok := current.(map[string]any); ok {
			value, exists := currentMap[part]
			
			// If the exact part doesn't exist, try camelCase version
			if !exists {
				camelCasePart := strings.ToLower(part[:1]) + part[1:]
				if value, exists = currentMap[camelCasePart]; !exists {
					return nil, fmt.Errorf("field not found: %s in path %s", part, path)
				}
			}

			// If this is the last part, return the value
			if i == len(parts)-1 {
				return value, nil
			}

			// Otherwise, continue navigating
			current = value
		} else {
			return nil, fmt.Errorf("cannot navigate path %s: %s is not a map", path, part)
		}
	}

	return current, nil
}

// * Standard transform functions
func transformNumericToFloat64(value any) (any, error) {
	f, _ := conversion.ToFloat64(value)
	return f, nil
}
