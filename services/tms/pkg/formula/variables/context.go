package variables

import (
	"reflect"

	"github.com/emoss08/trenova/pkg/formula/errors"
	"github.com/emoss08/trenova/pkg/formula/schema"
)

type DefaultVariableContext struct {
	entity   any
	resolver *schema.DefaultDataResolver
	metadata map[string]any
}

func NewDefaultContext(entity any, resolver *schema.DefaultDataResolver) *DefaultVariableContext {
	return &DefaultVariableContext{
		entity:   entity,
		resolver: resolver,
		metadata: make(map[string]any),
	}
}

func NewDefaultContextWithMetadata(
	entity any,
	resolver *schema.DefaultDataResolver,
	metadata map[string]any,
) *DefaultVariableContext {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return &DefaultVariableContext{
		entity:   entity,
		resolver: resolver,
		metadata: metadata,
	}
}

func (c *DefaultVariableContext) GetEntity() any {
	return c.entity
}

func (c *DefaultVariableContext) GetField(path string) (any, error) {
	if c.resolver == nil {
		entityType := "unknown"
		if c.entity != nil {
			entityType = reflect.TypeOf(c.entity).String()
		}
		return nil, errors.NewResolveError(path, entityType, ErrNoResolverConfigured)
	}

	fieldSource := &schema.FieldSource{
		Path: path,
	}

	return c.resolver.ResolveField(c.entity, fieldSource)
}

func (c *DefaultVariableContext) GetComputed(function string) (any, error) {
	if c.resolver == nil {
		entityType := "unknown"
		if c.entity != nil {
			entityType = reflect.TypeOf(c.entity).String()
		}
		return nil, errors.NewComputeError(
			function,
			entityType,
			ErrNoResolverConfigured,
		)
	}

	fieldSource := &schema.FieldSource{
		Computed: true,
		Function: function,
	}

	return c.resolver.ResolveField(c.entity, fieldSource)
}

func (c *DefaultVariableContext) GetMetadata() map[string]any {
	return c.metadata
}

func (c *DefaultVariableContext) SetMetadata(key string, value any) {
	c.metadata[key] = value
}

func (c *DefaultVariableContext) WithEntity(entity any) *DefaultVariableContext {
	return &DefaultVariableContext{
		entity:   entity,
		resolver: c.resolver,
		metadata: c.metadata,
	}
}

func (c *DefaultVariableContext) WithResolver(
	resolver *schema.DefaultDataResolver,
) *DefaultVariableContext {
	return &DefaultVariableContext{
		entity:   c.entity,
		resolver: resolver,
		metadata: c.metadata,
	}
}
