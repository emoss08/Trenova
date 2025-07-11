package schema

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/shopspring/decimal"
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
	resolver.RegisterTransform(TransformDecimalToFloat64, transformDecimalToFloat64)
	resolver.RegisterTransform(TransformInt64ToFloat64, transformInt64ToFloat64)
	resolver.RegisterTransform(TransformInt16ToFloat64, transformInt16ToFloat64)

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
		return nil, err
	}

	// * Apply transform if specified
	if fieldSource.Transform != "" {
		transform, exists := r.transforms[fieldSource.Transform]
		if !exists {
			return nil, fmt.Errorf("transform not found: %s", fieldSource.Transform)
		}
		return transform(value)
	}

	return value, nil
}

// * ResolveComputed calculates a computed field value
func (r *DefaultDataResolver) ResolveComputed(entity any, fieldSource *FieldSource) (any, error) {
	computer, exists := r.computers[fieldSource.Function]
	if !exists {
		return nil, fmt.Errorf("compute function not found: %s", fieldSource.Function)
	}

	return computer(entity)
}

// * extractFieldValue uses reflection to extract a value from a struct
func (r *DefaultDataResolver) extractFieldValue(entity any, path string) (any, error) {
	// * Handle nil
	if entity == nil {
		return nil, nil
	}

	// * Get reflect value
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

// * Standard transform functions
func transformDecimalToFloat64(value any) (any, error) {
	if value == nil {
		return 0.0, nil
	}

	switch v := value.(type) {
	case decimal.Decimal:
		f, _ := v.Float64()
		return f, nil
	case decimal.NullDecimal:
		if v.Valid {
			f, _ := v.Decimal.Float64()
			return f, nil
		}
		return 0.0, nil
	default:
		return nil, fmt.Errorf("cannot transform %T to float64", value)
	}
}

func transformInt64ToFloat64(value any) (any, error) {
	if value == nil {
		return 0.0, nil
	}

	switch v := value.(type) {
	case int64:
		return float64(v), nil
	case *int64:
		if v != nil {
			return float64(*v), nil
		}
		return 0.0, nil
	default:
		return nil, fmt.Errorf("cannot transform %T to float64", value)
	}
}

func transformInt16ToFloat64(value any) (any, error) {
	if value == nil {
		return 0.0, nil
	}

	switch v := value.(type) {
	case int16:
		return float64(v), nil
	case *int16:
		if v != nil {
			return float64(*v), nil
		}
		return 0.0, nil
	default:
		return nil, fmt.Errorf("cannot transform %T to float64", value)
	}
}
