package builtin

import (
	"github.com/emoss08/trenova/pkg/formula/conversion"
	"github.com/emoss08/trenova/pkg/formula/variables"
	"github.com/emoss08/trenova/pkg/formulatypes"
)

var TemperatureDifferentialVar = variables.NewVariableWithValidator(
	"temperature_differential",
	"Temperature difference between minimum and maximum requirements (in Fahrenheit)",
	formulatypes.ValueTypeNumber,
	variables.SourceShipment,
	temperatureDifferentialResolver,
	temperatureDifferentialValidator,
)

var TemperatureMinVar = variables.NewVariable(
	"temperature_min",
	"Minimum required temperature in Fahrenheit",
	formulatypes.ValueTypeNumber,
	variables.SourceShipment,
	func(ctx variables.VariableContext) (any, error) {
		return ctx.GetField("TemperatureMin")
	},
)

var TemperatureMaxVar = variables.NewVariable(
	"temperature_max",
	"Maximum required temperature in Fahrenheit",
	formulatypes.ValueTypeNumber,
	variables.SourceShipment,
	func(ctx variables.VariableContext) (any, error) {
		return ctx.GetField("TemperatureMax")
	},
)

var RequiresTemperatureControlVar = variables.NewVariable(
	"requires_temperature_control",
	"Whether the shipment requires temperature control",
	formulatypes.ValueTypeBoolean,
	variables.SourceShipment,
	func(ctx variables.VariableContext) (any, error) {
		return ctx.GetComputed("computeRequiresTemperatureControl")
	},
)

func temperatureDifferentialResolver(ctx variables.VariableContext) (any, error) {
	if diff, err := ctx.GetComputed("computeTemperatureDifferential"); err == nil {
		return diff, nil
	}

	minTemp, err := ctx.GetField("TemperatureMin")
	if err != nil {
		return 0.0, err
	}

	maxTemp, err := ctx.GetField("TemperatureMax")
	if err != nil {
		return 0.0, err
	}

	minFloat, ok1 := conversion.ToFloat64(minTemp)
	maxFloat, ok2 := conversion.ToFloat64(maxTemp)

	if !ok1 || !ok2 {
		return 0.0, nil
	}

	return maxFloat - minFloat, nil
}

func temperatureDifferentialValidator(value any) error {
	if value == nil {
		return nil
	}

	diff, ok := conversion.ToFloat64(value)
	if !ok {
		return ErrTempDiffMustBeNumber
	}

	if diff < 0 {
		return ErrTempDiffCannotBeNegative
	}

	return nil
}

func RegisterTemperatureVariables(registry *variables.Registry) {
	registry.MustRegister(TemperatureDifferentialVar)
	registry.MustRegister(TemperatureMinVar)
	registry.MustRegister(TemperatureMaxVar)
	registry.MustRegister(RequiresTemperatureControlVar)
}
