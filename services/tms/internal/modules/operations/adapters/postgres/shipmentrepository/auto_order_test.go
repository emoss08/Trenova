package shipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnAutoOrderCarriesThePurchaseOrderItWasCreatedWith(t *testing.T) {
	t.Parallel()

	repo := &repository{generator: testutil.TestSequenceGenerator{SingleValue: "ORD-1"}}
	entity := &shipment.Shipment{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		CustomerID:     pulid.MustNew("cus_"),
		OrderPONumber:  "8978300638",
	}

	created, err := repo.buildAutoOrder(t.Context(), entity, "", "")

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "ORD-1", created.OrderNumber)
	assert.Equal(t, "8978300638", created.PONumber)
}

func TestAShipmentOnAnOrderMakesNoAutoOrder(t *testing.T) {
	t.Parallel()

	repo := &repository{generator: testutil.TestSequenceGenerator{SingleValue: "ORD-1"}}
	entity := &shipment.Shipment{OrderID: pulid.MustNew("ord_"), OrderPONumber: "8978300638"}

	created, err := repo.buildAutoOrder(t.Context(), entity, "", "")

	require.NoError(t, err)
	assert.Nil(t, created)
}
