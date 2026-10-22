package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentinvalidation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func newStateCoordinator() *shipmentstate.Coordinator {
	return shipmentstate.NewCoordinator()
}

func newTestCommercialCalculator(
	t *testing.T,
	freight decimal.Decimal,
	accessorialRepo accessorialchargeports.AccessorialChargeRepository,
) *shipmentcommercial.Calculator {
	t.Helper()

	return shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      ratingtest.NewFlatEngine(t, freight),
		AccessorialRepo: accessorialRepo,
	})
}

func newTestInvalidator(realtime agentports.RealtimeService) shipmentports.ShipmentInvalidator {
	return shipmentinvalidation.NewWithDependencies(realtime, nil, zap.NewNop())
}
