package workflowutils

import (
	"fmt"
	"regexp"
	"strings"
)

// VariableResolver handles interpolation of variables in workflow configurations.
// It supports the {{variable.path}} syntax for accessing nested fields in workflow state.
type VariableResolver struct {
	state map[string]any
}

// NewVariableResolver creates a new resolver with the given workflow state.
func NewVariableResolver(state map[string]any) *VariableResolver {
	return &VariableResolver{
		state: state,
	}
}

// variablePattern matches {{variable.path}} syntax
var variablePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ResolveString resolves all variables in a string.
// Variables are replaced with their values from the workflow state.
// Example: "Shipment {{trigger.shipmentId}} updated" -> "Shipment SHP-123 updated"
func (r *VariableResolver) ResolveString(input string) (string, error) {
	return variablePattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract the variable path (remove {{ and }})
		variablePath := strings.TrimSpace(match[2 : len(match)-2])

		// Resolve the variable
		value, err := r.resolveVariable(variablePath)
		if err != nil {
			// If variable not found, return the original placeholder
			return match
		}

		// Convert value to string
		return fmt.Sprintf("%v", value)
	}), nil
}

// ResolveConfig resolves all string variables in a configuration map.
// This recursively processes the config and replaces any string values containing variables.
func (r *VariableResolver) ResolveConfig(config map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(config))

	for key, value := range config {
		switch v := value.(type) {
		case string:
			// Resolve variables in string values
			resolved, err := r.ResolveString(v)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve variable in field %s: %w", key, err)
			}
			result[key] = resolved

		case map[string]any:
			// Recursively resolve nested maps
			resolved, err := r.ResolveConfig(v)
			if err != nil {
				return nil, err
			}
			result[key] = resolved

		case []any:
			// Resolve variables in array elements
			resolvedArray := make([]any, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					resolved, err := r.ResolveString(str)
					if err != nil {
						return nil, err
					}
					resolvedArray[i] = resolved
				} else {
					resolvedArray[i] = item
				}
			}
			result[key] = resolvedArray

		default:
			// Non-string values pass through unchanged
			result[key] = value
		}
	}

	return result, nil
}

// resolveVariable resolves a variable path like "trigger.shipmentId" to its value in the state.
// Supports nested paths using dot notation.
func (r *VariableResolver) resolveVariable(path string) (any, error) {
	parts := strings.Split(path, ".")

	var current any = r.state
	for i, part := range parts {
		// Try to access the current level as a map
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot access field %s: parent is not a map (path: %s)", part, strings.Join(parts[:i], "."))
		}

		// Get the value at this level
		value, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("field %s not found in workflow state (path: %s)", part, path)
		}

		current = value
	}

	return current, nil
}

// GetValue gets a value from the workflow state by path.
// This is a convenience method for direct access without string interpolation.
func (r *VariableResolver) GetValue(path string) (any, error) {
	return r.resolveVariable(path)
}

// GetString gets a string value from the workflow state by path.
func (r *VariableResolver) GetString(path string) (string, error) {
	value, err := r.resolveVariable(path)
	if err != nil {
		return "", err
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value at %s is not a string", path)
	}

	return str, nil
}

// MustResolveString is like ResolveString but panics on error.
// Useful for template resolution where errors should be caught during workflow validation.
func (r *VariableResolver) MustResolveString(input string) string {
	result, err := r.ResolveString(input)
	if err != nil {
		panic(fmt.Sprintf("failed to resolve variables in '%s': %v", input, err))
	}
	return result
}
