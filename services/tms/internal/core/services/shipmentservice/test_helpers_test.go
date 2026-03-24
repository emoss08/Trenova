package shipmentservice

import (
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
)

func newStateCoordinator() *shipmentstate.Coordinator {
	return shipmentstate.NewCoordinator()
}

func newTestCommercialCalculator(
	formula portservices.FormulaCalculator,
	accessorialRepo repositories.AccessorialChargeRepository,
) *shipmentcommercial.Calculator {
	return shipmentcommercial.New(shipmentcommercial.Params{
		Formula:         formula,
		AccessorialRepo: accessorialRepo,
	})
}
