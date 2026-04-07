package engine

import (
	"fmt"
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"go.uber.org/fx"
)

type EnvironmentBuilderParams struct {
	fx.In

	Registry *schema.Registry
	Resolver *resolver.Resolver
}

type EnvironmentBuilder struct {
	registry *schema.Registry
	resolver *resolver.Resolver
}

func NewEnvironmentBuilder(p EnvironmentBuilderParams) *EnvironmentBuilder {
	return &EnvironmentBuilder{
		registry: p.Registry,
		resolver: p.Resolver,
	}
}

func (b *EnvironmentBuilder) Build(entity any, schemaID string) (map[string]any, error) {
	env := make(map[string]any)

	definition, ok := b.registry.Get(schemaID)
	if !ok {
		return b.buildFromEntity(entity)
	}

	for fieldPath, source := range definition.FieldSources {
		var value any
		var err error

		if source.Computed {
			value, err = b.resolver.ResolveComputed(entity, source.Function)
		} else {
			value, err = b.resolver.ResolveField(entity, source)
		}

		if err != nil {
			if source.Nullable {
				value = nil
			} else {
				continue
			}
		}

		formulatypes.SetNestedValue(env, fieldPath, value)
	}

	return env, nil
}

func (b *EnvironmentBuilder) BuildWithVariables(
	entity any,
	schemaID string,
	variables map[string]any,
) (map[string]any, error) {
	env, err := b.Build(entity, schemaID)
	if err != nil {
		return nil, err
	}

	mergeVariables(env, variables)

	return env, nil
}

func (b *EnvironmentBuilder) BuildValidationEnvironment(
	schemaID string,
	variables map[string]any,
) (map[string]any, error) {
	definition, ok := b.registry.Get(schemaID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSchemaNotFound, schemaID)
	}

	env := buildPropertyValidationEnvironment(definition.Properties)
	mergeVariables(env, variables)

	return env, nil
}

func (b *EnvironmentBuilder) buildFromEntity(entity any) (map[string]any, error) {
	env := make(map[string]any)

	computed := []string{
		"totalDistance",
		"totalStops",
		"hasHazmat",
		"requiresTemperatureControl",
		"temperatureDifferential",
		"totalWeight",
		"totalPieces",
		"totalLinearFeet",
		"baseRate",
		"freightChargeAmount",
		"otherChargeAmount",
		"currentTotalCharge",
	}

	for _, name := range computed {
		functionName := "compute" + strings.ToUpper(name[:1]) + name[1:]
		if value, err := b.resolver.ResolveComputed(entity, functionName); err == nil {
			env[name] = value
		}
	}

	return env, nil
}

func (b *EnvironmentBuilder) GetRequiredPreloads(schemaID string) []string {
	definition, ok := b.registry.Get(schemaID)
	if !ok {
		return nil
	}

	preloadSet := make(map[string]struct{})

	for _, preload := range definition.DataSource.Preloads {
		preloadSet[preload] = struct{}{}
	}

	for _, source := range definition.FieldSources {
		for _, preload := range source.Preload {
			preloadSet[preload] = struct{}{}
		}
	}

	preloads := make([]string, 0, len(preloadSet))
	for p := range preloadSet {
		preloads = append(preloads, p)
	}

	return preloads
}

func (b *EnvironmentBuilder) GetAvailableVariables(schemaID string) []*formulatypes.FieldSource {
	definition, ok := b.registry.Get(schemaID)
	if !ok {
		return nil
	}

	sources := make([]*formulatypes.FieldSource, 0, len(definition.FieldSources))
	for _, source := range definition.FieldSources {
		sources = append(sources, source)
	}

	return sources
}

func buildPropertyValidationEnvironment(
	properties map[string]formulatypes.Property,
) map[string]any {
	env := make(map[string]any, len(properties))

	for name, property := range properties {
		env[name] = defaultValueForProperty(property)
	}

	return env
}

func defaultValueForProperty(property formulatypes.Property) any {
	switch normalizePropertyType(property.Type) {
	case "object":
		if property.Properties == nil {
			return map[string]any{}
		}
		return buildPropertyValidationEnvironment(property.Properties)
	case "array":
		if property.Items == nil {
			return []any{}
		}
		return []any{defaultValueForProperty(*property.Items)}
	case "boolean":
		return false
	case "string":
		return ""
	case "integer":
		return int64(0)
	case "number":
		return 0.0
	default:
		return 0.0
	}
}

func normalizePropertyType(rawType any) string {
	switch value := rawType.(type) {
	case string:
		return value
	case []string:
		return firstNonNullType(value)
	case []any:
		candidates := make([]string, 0, len(value))
		for _, item := range value {
			if candidate, ok := item.(string); ok {
				candidates = append(candidates, candidate)
			}
		}
		return firstNonNullType(candidates)
	default:
		return ""
	}
}

func firstNonNullType(candidates []string) string {
	for _, candidate := range candidates {
		if candidate != "" && candidate != "null" {
			return candidate
		}
	}

	return ""
}

func mergeVariables(env map[string]any, variables map[string]any) {
	for key, value := range variables {
		if strings.Contains(key, ".") {
			formulatypes.SetNestedValue(env, key, value)
			continue
		}

		if merged, ok := mergeMapValues(env[key], value); ok {
			env[key] = merged
			continue
		}

		env[key] = value
	}
}

func mergeMapValues(existing, incoming any) (map[string]any, bool) {
	existingMap, ok := existing.(map[string]any)
	if !ok {
		return nil, false
	}

	incomingMap, ok := incoming.(map[string]any)
	if !ok {
		return nil, false
	}

	merged := make(map[string]any, len(existingMap))
	maps.Copy(merged, existingMap)

	for key, value := range incomingMap {
		if nested, ok := mergeMapValues(merged[key], value); ok {
			merged[key] = nested
			continue
		}
		merged[key] = value
	}

	return merged, true
}
