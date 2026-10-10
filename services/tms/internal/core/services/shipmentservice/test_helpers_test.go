package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/services/rateengine/rateenginetest"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/shipmentinvalidation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"go.uber.org/zap"
)

func newStateCoordinator() *shipmentstate.Coordinator {
	return shipmentstate.NewCoordinator()
}

func newTestCommercialCalculator(
	t *testing.T,
	formula formulatemplateports.FormulaCalculator,
	accessorialRepo accessorialchargeports.AccessorialChargeRepository,
) *shipmentcommercial.Calculator {
	t.Helper()

	return shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      rateenginetest.NewFallbackEngine(t, formula),
		AccessorialRepo: accessorialRepo,
	})
}

func newTestInvalidator(realtime agentports.RealtimeService) shipmentports.ShipmentInvalidator {
	return shipmentinvalidation.NewWithDependencies(realtime, nil, zap.NewNop())
}
