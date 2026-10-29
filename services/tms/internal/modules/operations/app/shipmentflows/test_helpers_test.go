package shipmentflows

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentinvalidation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
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

// assembled wires the use-case services from the fields a test set, as New
// does from its params.
func assembled(s *rig) *rig {
	s.assemble()
	return s
}

func newTestInvalidator(
	realtime realtimeports.RealtimeService,
) shipmentinvalidationports.ShipmentInvalidator {
	return shipmentinvalidation.NewWithDependencies(realtime, nil, zap.NewNop())
}
