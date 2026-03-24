package resolver

import (
	goErrors "errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/emoss08/trenova/pkg/formulatypes"
)

var (
	ErrNilSource         = goErrors.New("nil source")
	ErrResolveComputed   = goErrors.New("use ResolveComputed for computed fields")
	ErrNoPathOrField     = goErrors.New("no path or field specified")
	ErrTransformNotFound = goErrors.New("transform not found")
	ErrComputedNotFound  = goErrors.New("computed function not found")
	ErrNilValue          = goErrors.New("value is nil")
)

type ComputedFunc func(entity any) (any, error)

type Resolver struct {
	mu          sync.RWMutex
	computedFns map[string]ComputedFunc
}

func NewResolver() *Resolver {
	return &Resolver{
		computedFns: make(map[string]ComputedFunc),
	}
}

func (r *Resolver) RegisterComputed(name string, fn ComputedFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.computedFns[name] = fn
}

func (r *Resolver) GetComputed(name string) (ComputedFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.computedFns[name]
	return fn, ok
}

func (r *Resolver) ResolveField(entity any, source *formulatypes.FieldSource) (any, error) {
	if source == nil {
		return nil, errors.NewResolveError("", "unknown", ErrNilSource)
	}

	if source.Computed {
		return nil, errors.NewResolveError(
			source.Path,
			"computed",
			ErrResolveComputed,
		)
	}

	path := source.Path
	if path == "" {
		path = source.Field
	}

	if path == "" {
		return nil, errors.NewResolveError("", "unknown", ErrNoPathOrField)
	}

	value, err := r.resolvePath(entity, path)
	if err != nil {
		if source.Nullable && goErrors.Is(err, ErrNilValue) {
			return nil, nil //nolint:nilnil // nil is valid for nullable fields
		}
		return nil, errors.NewResolveError(path, reflect.TypeOf(entity).String(), err)
	}

	if source.Transform != "" {
		transformFn, ok := GetTransform(source.Transform)
		if !ok {
			return nil, errors.NewTransformError(
				source.Transform,
				"unknown",
				value,
				ErrTransformNotFound,
			)
		}
		transformed, transErr := transformFn(value)
		if transErr != nil {
			return nil, errors.NewTransformError(source.Transform, "any", value, transErr)
		}
		return transformed, nil
	}

	return value, nil
}

func (r *Resolver) ResolveComputed(
	entity any,
	functionName string,
) (any, error) {
	fn, ok := r.GetComputed(functionName)
	if !ok {
		return nil, errors.NewComputeError(
			functionName,
			reflect.TypeOf(entity).String(),
			ErrComputedNotFound,
		)
	}

	result, err := fn(entity)
	if err != nil {
		return nil, errors.NewComputeError(functionName, reflect.TypeOf(entity).String(), err)
	}

	return result, nil
}

func (r *Resolver) resolvePath(entity any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := reflect.ValueOf(entity)

	for _, part := range parts {
		if !current.IsValid() {
			return nil, fmt.Errorf("invalid value at path segment: %s", part)
		}

		if current.Kind() == reflect.Pointer {
			if current.IsNil() {
				return nil, ErrNilValue
			}
			current = current.Elem()
		}

		if current.Kind() != reflect.Struct {
			return nil, fmt.Errorf(
				"expected struct at path segment %s, got %s",
				part,
				current.Kind(),
			)
		}

		field := current.FieldByName(part)
		if !field.IsValid() {
			field = r.findFieldByJSONTag(current, part)
			if !field.IsValid() {
				return nil, fmt.Errorf("field %s not found", part)
			}
		}

		current = field
	}

	if !current.IsValid() {
		return nil, ErrNilValue
	}

	if current.Kind() == reflect.Pointer && current.IsNil() {
		return nil, ErrNilValue
	}

	return current.Interface(), nil
}

func (r *Resolver) findFieldByJSONTag(v reflect.Value, tagName string) reflect.Value {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]
		if name == tagName {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

func (r *Resolver) ResolveAllFields(
	entity any,
	fieldSources map[string]*formulatypes.FieldSource,
) (map[string]any, error) {
	result := make(map[string]any)

	for fieldName, source := range fieldSources {
		var value any
		var err error

		if source.Computed {
			value, err = r.ResolveComputed(entity, source.Function)
		} else {
			value, err = r.ResolveField(entity, source)
		}

		if err != nil {
			if !source.Nullable {
				return nil, err
			}
			value = nil
		}

		formulatypes.SetNestedValue(result, fieldName, value)
	}

	return result, nil
}
