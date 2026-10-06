package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/rateengine"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/shipmentinvalidation"
	"go.uber.org/zap"
)

func newStateCoordinator() *shipmentstate.Coordinator {
	return shipmentstate.NewCoordinator()
}

func newTestCommercialCalculator(
	t *testing.T,
	formula portservices.FormulaCalculator,
	accessorialRepo repositories.AccessorialChargeRepository,
) *shipmentcommercial.Calculator {
	t.Helper()

	return shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      rateengine.NewFallbackEngine(t, formula),
		AccessorialRepo: accessorialRepo,
	})
}

func newTestInvalidator(realtime services.RealtimeService) services.ShipmentInvalidator {
	return shipmentinvalidation.NewWithDependencies(realtime, nil, zap.NewNop())
}
