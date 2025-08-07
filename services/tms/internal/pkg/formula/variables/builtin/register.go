/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package builtin

import "github.com/emoss08/trenova/internal/pkg/formula/variables"

// * RegisterAll registers all built-in variables to the given registry
func RegisterAll(registry *variables.Registry) {
	RegisterTemperatureVariables(registry)
	RegisterHazmatVariables(registry)
}

// * RegisterDefaults registers all built-in variables to the default registry
func RegisterDefaults() {
	RegisterAll(variables.DefaultRegistry)
}
