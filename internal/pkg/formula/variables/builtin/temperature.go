// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package builtin

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/formula/conversion"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// * Temperature-related variables

// * TemperatureDifferentialVar calculates the difference between min and max temperature
var TemperatureDifferentialVar = variables.NewVariableWithValidator(
	"temperature_differential",
	"Temperature difference between minimum and maximum requirements (in Fahrenheit)",
	formula.ValueTypeNumber,
	variables.SourceShipment,
	temperatureDifferentialResolver,
	temperatureDifferentialValidator,
)

// * TemperatureMinVar returns the minimum temperature requirement
var TemperatureMinVar = variables.NewVariable(
	"temperature_min",
	"Minimum required temperature in Fahrenheit",
	formula.ValueTypeNumber,
	variables.SourceShipment,
	func(ctx variables.VariableContext) (any, error) {
		return ctx.GetField("TemperatureMin")
	},
)

// * TemperatureMaxVar returns the maximum temperature requirement
var TemperatureMaxVar = variables.NewVariable(
	"temperature_max",
	"Maximum required temperature in Fahrenheit",
	formula.ValueTypeNumber,
	variables.SourceShipment,
	func(ctx variables.VariableContext) (any, error) {
		return ctx.GetField("TemperatureMax")
	},
)

// * RequiresTemperatureControlVar indicates if temperature control is required
var RequiresTemperatureControlVar = variables.NewVariable(
	"requires_temperature_control",
	"Whether the shipment requires temperature control",
	formula.ValueTypeBoolean,
	variables.SourceShipment,
	func(ctx variables.VariableContext) (any, error) {
		return ctx.GetComputed("computeRequiresTemperatureControl")
	},
)

// * temperatureDifferentialResolver calculates temperature differential
func temperatureDifferentialResolver(ctx variables.VariableContext) (any, error) {
	// * Try computed function first
	if diff, err := ctx.GetComputed("computeTemperatureDifferential"); err == nil {
		return diff, nil
	}

	// * Fallback to manual calculation
	minTemp, err := ctx.GetField("TemperatureMin")
	if err != nil {
		return 0.0, nil // No min temp, no differential
	}

	maxTemp, err := ctx.GetField("TemperatureMax")
	if err != nil {
		return 0.0, nil // No max temp, no differential
	}

	// * Convert to float64 and calculate
	minFloat, ok1 := conversion.ToFloat64(minTemp)
	maxFloat, ok2 := conversion.ToFloat64(maxTemp)

	if !ok1 || !ok2 {
		return 0.0, nil
	}

	return maxFloat - minFloat, nil
}

// * temperatureDifferentialValidator ensures the differential is non-negative
func temperatureDifferentialValidator(value any) error {
	if value == nil {
		return nil
	}

	diff, ok := conversion.ToFloat64(value)
	if !ok {
		return fmt.Errorf("temperature differential must be a number")
	}

	if diff < 0 {
		return fmt.Errorf("temperature differential cannot be negative")
	}

	return nil
}

// * RegisterTemperatureVariables registers all temperature-related variables
func RegisterTemperatureVariables(registry *variables.Registry) {
	registry.MustRegister(TemperatureDifferentialVar)
	registry.MustRegister(TemperatureMinVar)
	registry.MustRegister(TemperatureMaxVar)
	registry.MustRegister(RequiresTemperatureControlVar)
}
