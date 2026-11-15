package shipmenteffects

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func heldPayerShipment() (*shipment.Shipment, *customer.Customer) {
	tenant := plantest.Tenant()
	payer := &customer.Customer{
		ID:   pulid.MustNew("cus_"),
		Name: "Acme Foods",
		BillingProfile: &customer.CustomerBillingProfile{
			EnforceCreditLimit: true,
			CreditStatus:       customer.CreditStatusHold,
		},
	}
	return &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		CustomerID:     payer.ID,
	}, payer
}

func effectsWith(features platformcatalogports.FeatureGate, customers *customerportsmock.MockCustomerRepository) *Effects {
	return &Effects{l: zap.NewNop(), customerRepo: customers, features: features}
}

func TestACreditHoldDoesNotBlockAShipmentWithoutAccounting(t *testing.T) {
	t.Parallel()

	entity, _ := heldPayerShipment()
	customers := customerportsmock.NewMockCustomerRepository(t)

	err := effectsWith(plantest.BaseOnly(), customers).RefuseCreditHold(t.Context(), entity, nil)

	require.NoError(t, err, "the payer is not read: the mock fails on any call")
}

func TestACreditHoldBlocksAShipmentWithAccounting(t *testing.T) {
	t.Parallel()

	entity, payer := heldPayerShipment()
	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).Return(payer, nil).Once()

	err := effectsWith(plantest.AllFeatures(), customers).RefuseCreditHold(t.Context(), entity, nil)

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err), "the hold refuses the shipment as before")
}
