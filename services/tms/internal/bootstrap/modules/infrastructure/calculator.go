package infrastructure

import (
	"github.com/emoss08/trenova/pkg/calculator"
	"go.uber.org/fx"
)

var CalculatorsModule = fx.Module("calculators", fx.Provide(
	calculator.NewShipmentCalculator,
))
