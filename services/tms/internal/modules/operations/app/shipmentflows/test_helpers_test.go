package shipmentflows

import (
	"testing"

	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentinvalidation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/fuelsurchargeportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type noBillingControl struct {
	tenantports.BillingControlRepository
}

func (noBillingControl) GetByOrgID(
	context.Context,
	pulid.ID,
) (*billingcontrol.BillingControl, error) {
	return nil, errortypes.NewNotFoundError("Billing control not found")
}

func newStateCoordinator() *shipmentstate.Coordinator {
	return shipmentstate.NewCoordinator()
}

func newCalculator(
	t *testing.T,
	engine ratequoteports.RateEngine,
	accessorialRepo accessorialchargeports.AccessorialChargeRepository,
) *shipmentcommercial.Calculator {
	t.Helper()

	fuel := fuelsurchargeportsmock.NewMockFuelSurchargeResolver(t)
	fuel.EXPECT().ResolveShipmentCharge(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	return shipmentcommercial.New(shipmentcommercial.Params{
		Logger:             zap.NewNop(),
		RateEngine:         engine,
		AccessorialRepo:    accessorialRepo,
		BillingControlRepo: noBillingControl{},
		FuelSurcharge:      fuel,
		Hydrator:           ratingtest.NoHydration{},
	})
}

func newTestCommercialCalculator(
	t *testing.T,
	freight decimal.Decimal,
	accessorialRepo accessorialchargeports.AccessorialChargeRepository,
) *shipmentcommercial.Calculator {
	t.Helper()

	return newCalculator(t, ratingtest.NewFlatEngine(t, freight), accessorialRepo)
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
	return shipmentinvalidation.New(shipmentinvalidation.Params{
		Realtime: realtime,
		Epochs:   &mocks.NoopShipmentBoardEpochBumper{},
		Logger:   zap.NewNop(),
	})
}
