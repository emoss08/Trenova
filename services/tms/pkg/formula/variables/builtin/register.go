package builtin

import "github.com/emoss08/trenova/pkg/formula/variables"

// * RegisterAll registers all built-in variables to the given registry
func RegisterAll(registry *variables.Registry) {
	RegisterTemperatureVariables(registry)
	RegisterHazmatVariables(registry)
}

// * RegisterDefaults registers all built-in variables to the default registry
func RegisterDefaults() {
	RegisterAll(variables.DefaultRegistry)
}
