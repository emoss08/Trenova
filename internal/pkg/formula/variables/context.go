package variables

import (
	"fmt"
	"reflect"

	"github.com/emoss08/trenova/internal/pkg/formula/errors"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
)

// * DefaultVariableContext implements VariableContext using the schema system
type DefaultVariableContext struct {
	entity   any
	resolver *schema.DefaultDataResolver
	metadata map[string]any
}

// * NewDefaultContext creates a new variable context
func NewDefaultContext(entity any, resolver *schema.DefaultDataResolver) *DefaultVariableContext {
	return &DefaultVariableContext{
		entity:   entity,
		resolver: resolver,
		metadata: make(map[string]any),
	}
}

// * NewDefaultContextWithMetadata creates a new context with metadata
func NewDefaultContextWithMetadata(entity any, resolver *schema.DefaultDataResolver, metadata map[string]any) *DefaultVariableContext {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return &DefaultVariableContext{
		entity:   entity,
		resolver: resolver,
		metadata: metadata,
	}
}

// GetEntity returns the primary entity
func (c *DefaultVariableContext) GetEntity() any {
	return c.entity
}

// GetField retrieves a field value by path
func (c *DefaultVariableContext) GetField(path string) (any, error) {
	if c.resolver == nil {
		entityType := "unknown"
		if c.entity != nil {
			entityType = reflect.TypeOf(c.entity).String()
		}
		return nil, errors.NewResolveError(path, entityType, fmt.Errorf("no resolver configured"))
	}
	
	fieldSource := &schema.FieldSource{
		Path: path,
	}
	
	return c.resolver.ResolveField(c.entity, fieldSource)
}

// GetComputed retrieves a computed value by function name
func (c *DefaultVariableContext) GetComputed(function string) (any, error) {
	if c.resolver == nil {
		entityType := "unknown"
		if c.entity != nil {
			entityType = reflect.TypeOf(c.entity).String()
		}
		return nil, errors.NewComputeError(function, entityType, fmt.Errorf("no resolver configured"))
	}
	
	fieldSource := &schema.FieldSource{
		Computed: true,
		Function: function,
	}
	
	return c.resolver.ResolveField(c.entity, fieldSource)
}

// GetMetadata returns context metadata
func (c *DefaultVariableContext) GetMetadata() map[string]any {
	return c.metadata
}

// * SetMetadata adds metadata to the context
func (c *DefaultVariableContext) SetMetadata(key string, value any) {
	c.metadata[key] = value
}

// * WithEntity returns a new context with a different entity
func (c *DefaultVariableContext) WithEntity(entity any) *DefaultVariableContext {
	return &DefaultVariableContext{
		entity:   entity,
		resolver: c.resolver,
		metadata: c.metadata,
	}
}

// * WithResolver returns a new context with a different resolver
func (c *DefaultVariableContext) WithResolver(resolver *schema.DefaultDataResolver) *DefaultVariableContext {
	return &DefaultVariableContext{
		entity:   c.entity,
		resolver: resolver,
		metadata: c.metadata,
	}
}